package matching

import (
	"sync"

	"github.com/tidwall/hashmap"

	"github.com/cryptonstudio/crypton-matching-engine/types/avl"
)

// Order book is used to store buy and sell orders in a price level order.
// NOTE: Not thread-safe.
type OrderBook struct {
	// Allocator used by the order book
	allocator *Allocator

	// Order book symbol
	symbol Symbol

	// Bid/Ask price levels
	bids avl.Tree[Uint, *PriceLevelL3]
	asks avl.Tree[Uint, *PriceLevelL3]

	// Buy/Sell stop orders price levels
	buyStop  avl.Tree[Uint, *PriceLevelL3]
	sellStop avl.Tree[Uint, *PriceLevelL3]

	// Buy/Sell trailing stop orders price levels
	trailingBuyStop  avl.Tree[Uint, *PriceLevelL3]
	trailingSellStop avl.Tree[Uint, *PriceLevelL3]

	// Allowed stop pride modes
	spModesConfig StopPriceModeConfig

	// Market last and trailing prices
	marketPrice Uint

	// Mark price
	markPrice      Uint
	markPriceMutex sync.RWMutex

	// Index price
	indexPrice      Uint
	indexPriceMutex sync.RWMutex

	lastBidPrice     Uint
	lastAskPrice     Uint
	matchingBidPrice Uint
	matchingAskPrice Uint
	trailingBidPrice Uint
	trailingAskPrice Uint

	// Last used update ID
	lastUpdateID uint64

	// Orders storage is internal for each order book
	orders *hashmap.Map[uint64, *Order]

	// Tasks to run in the single for the order book goroutine
	// Used total externally, stored in order book to avoid storing in separate container in matching engine
	chanTasks chan func(*OrderBook) error

	// Synchronization stuff
	chanForcedStop chan struct{} // for forced stop
	wg             sync.WaitGroup
}

// NewOrderBook creates and returns new OrderBook instance.
func NewOrderBook(allocator *Allocator, symbol Symbol, spModesConfig StopPriceModeConfig, taskQueueSize int) *OrderBook {
	newPriceLevelTree := func(allocator *Allocator) avl.Tree[Uint, *PriceLevelL3] {
		return avl.NewTreePooled[Uint, *PriceLevelL3](
			func(a, b Uint) int { return a.Cmp(b) },
			&allocator.priceLevelNodes,
		)
	}
	newPriceLevelReversedTree := func(allocator *Allocator) avl.Tree[Uint, *PriceLevelL3] {
		return avl.NewTreePooled[Uint, *PriceLevelL3](
			func(a, b Uint) int { return -a.Cmp(b) },
			&allocator.priceLevelNodes,
		)
	}

	return &OrderBook{
		allocator:        allocator,
		symbol:           symbol,
		bids:             newPriceLevelReversedTree(allocator),
		asks:             newPriceLevelTree(allocator),
		spModesConfig:    spModesConfig,
		buyStop:          newPriceLevelReversedTree(allocator),
		sellStop:         newPriceLevelTree(allocator),
		trailingBuyStop:  newPriceLevelReversedTree(allocator),
		trailingSellStop: newPriceLevelTree(allocator),
		marketPrice:      NewZeroUint(),
		lastBidPrice:     NewZeroUint(),
		lastAskPrice:     NewMaxUint(),
		matchingBidPrice: NewZeroUint(),
		matchingAskPrice: NewMaxUint(),
		trailingBidPrice: NewZeroUint(),
		trailingAskPrice: NewMaxUint(),
		orders:           hashmap.New[uint64, *Order](defaultReservedOrderSlots),
		chanTasks:        make(chan func(*OrderBook) error, taskQueueSize),
		chanForcedStop:   make(chan struct{}),
		wg:               sync.WaitGroup{},
	}
}

// Clean releases all internally used tree nodes and cleans whole order book state.
func (ob *OrderBook) Clean() {
	clean := func(v *PriceLevelL3) bool {
		ob.allocator.PutPriceLevel(v)
		return false
	}
	// Clean all price levels
	ob.bids.IteratePostOrder(clean)
	ob.asks.IteratePostOrder(clean)
	ob.buyStop.IteratePostOrder(clean)
	ob.sellStop.IteratePostOrder(clean)
	ob.trailingBuyStop.IteratePostOrder(clean)
	ob.trailingSellStop.IteratePostOrder(clean)
}

////////////////////////////////////////////////////////////////
// Order book symbol
////////////////////////////////////////////////////////////////

// Symbol returns order book symbol.
func (ob *OrderBook) Symbol() Symbol {
	return ob.symbol
}

func (ob *OrderBook) UpdateSymbol(sym Symbol) error {
	if ob.symbol.id != sym.id {
		return ErrInvalidSymbol
	}

	if !sym.Valid() {
		return ErrInvalidSymbol
	}

	ob.symbol = sym
	return nil
}

////////////////////////////////////////////////////////////////
// Order book getters
////////////////////////////////////////////////////////////////

// IsEmpty returns true of the order book has no any orders.
func (ob *OrderBook) IsEmpty() bool {
	return ob.Size() == 0
}

// Size returns total amount of orders in the order book.
func (ob *OrderBook) Size() int {
	return ob.orders.Len()
}

// Order returns order with given id.
func (ob *OrderBook) Order(id uint64) *Order {
	if order, ok := ob.orders.Get(id); ok {
		return order
	}
	return nil
}

////////////////////////////////////////////////////////////////
// Top price levels getters
////////////////////////////////////////////////////////////////

// TopBid returns best buy order price level.
func (ob *OrderBook) TopBid() *avl.Node[Uint, *PriceLevelL3] {
	return ob.bids.MostLeft()
}

// TopAsk returns best sell order price level.
func (ob *OrderBook) TopAsk() *avl.Node[Uint, *PriceLevelL3] {
	return ob.asks.MostLeft()
}

// TopBuyStop returns best buy stop order price level.
func (ob *OrderBook) TopBuyStop() *avl.Node[Uint, *PriceLevelL3] {
	return ob.buyStop.MostLeft()
}

// TopSellStop returns best sell stop order price level.
func (ob *OrderBook) TopSellStop() *avl.Node[Uint, *PriceLevelL3] {
	return ob.sellStop.MostLeft()
}

// TopTrailingBuyStop returns best trailing buy stop order price level.
func (ob *OrderBook) TopTrailingBuyStop() *avl.Node[Uint, *PriceLevelL3] {
	return ob.trailingBuyStop.MostLeft()
}

// TopTrailingSellStop returns best trailing sell stop order price level.
func (ob *OrderBook) TopTrailingSellStop() *avl.Node[Uint, *PriceLevelL3] {
	return ob.trailingSellStop.MostLeft()
}

////////////////////////////////////////////////////////////////
// Price levels getters
////////////////////////////////////////////////////////////////

// GetBid returns buy order price level with given price.
func (ob *OrderBook) GetBid(price Uint) *avl.Node[Uint, *PriceLevelL3] {
	// TODO: Firstly check 4-16 most left nodes, then used binary search from tree root
	return ob.bids.Find(price)
}

// GetAsk returns sell order price level with given price.
func (ob *OrderBook) GetAsk(price Uint) *avl.Node[Uint, *PriceLevelL3] {
	// TODO: Firstly check 4-16 most left nodes, then used binary search from tree root
	return ob.asks.Find(price)
}

// GetBuyStop returns buy stop order price level with given price.
func (ob *OrderBook) GetBuyStop(price Uint) *avl.Node[Uint, *PriceLevelL3] {
	return ob.buyStop.Find(price)
}

// GetSellStop returns sell stop order price level with given price.
func (ob *OrderBook) GetSellStop(price Uint) *avl.Node[Uint, *PriceLevelL3] {
	return ob.sellStop.Find(price)
}

// GetTrailingBuyStop returns trailing buy stop order price level with given price.
func (ob *OrderBook) GetTrailingBuyStop(price Uint) *avl.Node[Uint, *PriceLevelL3] {
	return ob.trailingBuyStop.Find(price)
}

// GetTrailingSellStop returns trailing sell stop order price level with given price.
func (ob *OrderBook) GetTrailingSellStop(price Uint) *avl.Node[Uint, *PriceLevelL3] {
	return ob.trailingSellStop.Find(price)
}

////////////////////////////////////////////////////////////////
// Market, Mark and Index methods
// Stop price is one of follows depending on stopPriceMode
////////////////////////////////////////////////////////////////

func (ob *OrderBook) GetStopPrice(m StopPriceMode) Uint {
	switch m {
	case StopPriceModeMarket:
		return ob.marketPrice
	case StopPriceModeMark:
		return ob.markPrice
	case StopPriceModeIndex:
		return ob.indexPrice
	}

	return NewZeroUint()
}

func (ob *OrderBook) GetMarketPrice() Uint {
	return ob.marketPrice
}

func (ob *OrderBook) GetMarkPrice() Uint {
	ob.markPriceMutex.RLock()
	defer ob.markPriceMutex.RUnlock()

	return ob.markPrice
}

// setMarkPrice sets the mark price without new match iteration of the engine.
func (ob *OrderBook) setMarkPrice(price Uint) {
	ob.markPriceMutex.Lock()
	defer ob.markPriceMutex.Unlock()

	ob.markPrice = price
}

func (ob *OrderBook) GetIndexPrice() Uint {
	ob.indexPriceMutex.RLock()
	defer ob.indexPriceMutex.RUnlock()

	return ob.indexPrice
}

// setIndexPrice sets the index price without new match iteration of the engine.
func (ob *OrderBook) setIndexPrice(price Uint) {
	ob.indexPriceMutex.Lock()
	defer ob.indexPriceMutex.Unlock()

	ob.indexPrice = price
}

// TODO: check later trailing prices
func (ob *OrderBook) GetMarketTrailingStopPriceBid() Uint {
	lastPrice, topPrice := ob.lastBidPrice, NewZeroUint()
	if top := ob.TopBid(); top != nil {
		topPrice = top.Value().Price()
	}
	return Min(lastPrice, topPrice)
}

func (ob *OrderBook) GetMarketTrailingStopPriceAsk() Uint {
	lastPrice, topPrice := ob.lastAskPrice, NewMaxUint()
	if top := ob.TopAsk(); top != nil {
		topPrice = top.Value().Price()
	}
	return Max(lastPrice, topPrice)
}

////////////////////////////////////////////////////////////////
// Orders management
////////////////////////////////////////////////////////////////

func (ob *OrderBook) addOrder(tree *avl.Tree[Uint, *PriceLevelL3], order *Order) (update PriceLevelUpdate, err error) {
	update.Kind = PriceLevelUpdateKindUpdate

	// Ensure the tree is specified
	if tree == nil {
		err = ErrInvalidOrderType
		return
	}

	// Find the price level for the order
	// TODO: Firstly check 4-16 most left nodes, then used binary search from tree root
	node := tree.Find(order.price)

	// Create a new price level if no one found
	if node == nil {
		node, err = ob.addPriceLevel(tree, order.price)
		if err != nil {
			return
		}
		update.Kind = PriceLevelUpdateKindAdd
	}

	// Update the price level volume
	priceLevel := node.Value()
	priceLevel.volume = priceLevel.volume.Add(order.restQuantity)
	priceLevel.visible = priceLevel.visible.Add(order.VisibleQuantity())

	// Enqueue the new order to the order queue of the price level
	order.orderQueued = priceLevel.queue.PushBack(order)
	priceLevel.orders++

	// Cache the price level in the given order
	order.priceLevel = node

	// Price level was changed so prepare update object
	update = PriceLevelUpdate{
		Kind:    update.Kind,
		Side:    order.side,
		Price:   priceLevel.Price(),
		Volume:  priceLevel.Volume(),
		Visible: priceLevel.Visible(),
		Orders:  priceLevel.Orders(),
		Top:     tree.MostLeft() != nil && node.Key().Equals(tree.MostLeft().Key()),
	}

	return
}

func (ob *OrderBook) reduceOrder(tree *avl.Tree[Uint, *PriceLevelL3], order *Order, quantity Uint, visible Uint) (update PriceLevelUpdate, err error) {
	update.Kind = PriceLevelUpdateKindUpdate

	// Determine the thee to work from the order
	if tree == nil {
		err = ErrInvalidOrderType
		return
	}

	// Find the price level for the order
	node := order.priceLevel
	if node == nil {
		err = ErrPriceLevelNotFound
		return
	}

	// Update the price level volume
	priceLevel := node.Value()
	priceLevel.volume = priceLevel.volume.Sub(quantity)
	priceLevel.visible = priceLevel.visible.Sub(visible)

	if order.IsExecuted() {
		// Dequeue the empty order from the order queue of the price level
		priceLevel.queue.Remove(order.orderQueued)
		priceLevel.orders--
		order.orderQueued = nil

		// Clear the price level cache in the given order
		order.priceLevel = nil
	}

	// Price level was changed so prepare update object
	update = PriceLevelUpdate{
		Kind:    update.Kind,
		Side:    order.side,
		Price:   priceLevel.Price(),
		Volume:  priceLevel.Volume(),
		Visible: priceLevel.Visible(),
		Orders:  priceLevel.Orders(),
		Top:     tree.MostLeft() != nil && node.Key().Equals(tree.MostLeft().Key()),
	}

	// Delete the empty price level
	if priceLevel.volume.IsZero() {
		err = ob.deletePriceLevel(tree, priceLevel.price)
		if err != nil {
			return
		}
		update.Kind = PriceLevelUpdateKindDelete
	}

	return
}

func (ob *OrderBook) deleteOrder(tree *avl.Tree[Uint, *PriceLevelL3], order *Order) (update PriceLevelUpdate, err error) {
	update.Kind = PriceLevelUpdateKindUpdate

	// Ensure the tree is specified
	if tree == nil {
		err = ErrInvalidOrderType
		return
	}

	// Find the price level for the order
	node := order.priceLevel
	if node == nil {
		err = ErrPriceLevelNotFound
		return
	}

	// Update the price level volume
	priceLevel := node.Value()
	priceLevel.volume = priceLevel.volume.Sub(order.restQuantity)
	priceLevel.visible = priceLevel.visible.Sub(order.VisibleQuantity())

	// Dequeue the deleted order from the order queue of the price level
	priceLevel.queue.Remove(order.orderQueued)
	priceLevel.orders--
	order.orderQueued = nil

	// Clear the price level cache in the given order
	order.priceLevel = nil

	// Price level was changed so prepare update object
	update = PriceLevelUpdate{
		Kind:    update.Kind,
		Side:    order.side,
		Price:   priceLevel.Price(),
		Volume:  priceLevel.Volume(),
		Visible: priceLevel.Visible(),
		Orders:  priceLevel.Orders(),
		Top:     tree.MostLeft() != nil && node.Key().Equals(tree.MostLeft().Key()),
	}

	// Delete the empty price level
	if priceLevel.volume.IsZero() {
		err = ob.deletePriceLevel(tree, priceLevel.price)
		if err != nil {
			return
		}
		update.Kind = PriceLevelUpdateKindDelete
	}

	return
}

////////////////////////////////////////////////////////////////
// Price levels management
////////////////////////////////////////////////////////////////

func (ob *OrderBook) addPriceLevel(tree *avl.Tree[Uint, *PriceLevelL3], price Uint) (*avl.Node[Uint, *PriceLevelL3], error) {
	priceLevel := ob.allocator.GetPriceLevel()
	priceLevel.price = price
	node, err := tree.Add(price, priceLevel)
	if err != nil {
		return nil, ErrPriceLevelDuplicate
	}
	return node, nil
}

func (ob *OrderBook) deletePriceLevel(tree *avl.Tree[Uint, *PriceLevelL3], price Uint) error {
	priceLevel, err := tree.Remove(price)
	if err != nil {
		return ErrPriceLevelNotFound
	}
	ob.allocator.PutPriceLevel(priceLevel)
	return err
}

// //////////////////////////////////////////////////////////////
// Market prices management
// //////////////////////////////////////////////////////////////

func (ob *OrderBook) updateMarketPrice(price Uint) {
	ob.marketPrice = price
}

func (ob *OrderBook) updateLastPrice(side OrderSide, price Uint) {
	if side == OrderSideBuy {
		ob.lastBidPrice = price
	} else {
		ob.lastAskPrice = price
	}
}

func (ob *OrderBook) updateMatchingPrice(side OrderSide, price Uint) {
	if side == OrderSideBuy {
		ob.matchingBidPrice = price
	} else {
		ob.matchingAskPrice = price
	}
}

func (ob *OrderBook) resetMatchingPrice() {
	ob.matchingBidPrice = NewZeroUint()
	ob.matchingAskPrice = NewMaxUint()
}

////////////////////////////////////////////////////////////////
// Internal helpers
////////////////////////////////////////////////////////////////

func (ob *OrderBook) treeForOrder(order *Order) *avl.Tree[Uint, *PriceLevelL3] {
	switch order.orderType {
	case OrderTypeLimit:
		if order.IsBuy() {
			return &ob.bids
		} else {
			return &ob.asks
		}
	case OrderTypeMarket:
	case OrderTypeStop, OrderTypeStopLimit:
		if order.IsBuy() {
			return &ob.buyStop
		} else {
			return &ob.sellStop
		}
	case OrderTypeTrailingStop, OrderTypeTrailingStopLimit:
		if order.IsBuy() {
			return &ob.trailingBuyStop
		} else {
			return &ob.trailingSellStop
		}
	}
	return nil
}
