package matching

import (
	"fmt"
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
	spModes []StopPriceMode

	// Market last and trailing prices
	marketPrice Uint

	// Mark price
	markPrice Uint

	// Index price
	indexPrice Uint

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
func NewOrderBook(symbol Symbol, spModesConfig StopPriceModeConfig, taskQueueSize int) *OrderBook {
	// Prepare allocator
	// TODO: Test how GC behaves in both cases (with/without pool)
	allocator := NewAllocator(true)

	return &OrderBook{
		allocator:        allocator,
		symbol:           symbol,
		bids:             allocator.NewPriceLevelReversedTree(),
		asks:             allocator.NewPriceLevelTree(),
		spModes:          spModesConfig.Modes(),
		buyStop:          allocator.NewPriceLevelReversedTree(),
		sellStop:         allocator.NewPriceLevelTree(),
		trailingBuyStop:  allocator.NewPriceLevelReversedTree(),
		trailingSellStop: allocator.NewPriceLevelTree(),
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

// GetStopPrice is internal helper for matching.
func (ob *OrderBook) GetStopPrice(m StopPriceMode) Uint {
	switch m {
	case StopPriceModeMarket:
		return ob.GetMarketPrice()
	case StopPriceModeMark:
		return ob.GetMarkPrice()
	case StopPriceModeIndex:
		return ob.GetIndexPrice()
	}

	return NewZeroUint()
}

func (ob *OrderBook) GetMarketPrice() Uint {
	return ob.marketPrice
}

func (ob *OrderBook) updateMarketPrice(price Uint) {
	ob.marketPrice = price
}

func (ob *OrderBook) GetMarkPrice() Uint {
	return ob.markPrice
}

// setMarkPrice sets the mark price without new match iteration of the engine.
func (ob *OrderBook) setMarkPrice(price Uint) {
	ob.markPrice = price
}

func (ob *OrderBook) GetIndexPrice() Uint {
	return ob.indexPrice
}

// setIndexPrice sets the index price without new match iteration of the engine.
func (ob *OrderBook) setIndexPrice(price Uint) {
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
		err = ErrOrderTreeNotFound
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

	priceLevel := node.Value()

	if !order.IsVirtualOB() {
		// Update the price level volume
		priceLevel.volume = priceLevel.volume.Add(order.restQuantity)
		priceLevel.visible = priceLevel.visible.Add(order.VisibleQuantity())
	}

	// Enqueue the new order to the order queue of the price level
	order.orderQueued = priceLevel.queue.PushBack(order)

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
		err = ErrOrderTreeNotFound
		return
	}

	// Find the price level for the order
	node := order.priceLevel
	if node == nil {
		err = ErrPriceLevelNotFound
		return
	}

	priceLevel := node.Value()

	if !order.IsVirtualOB() {
		// Update the price level volume
		priceLevel.volume = priceLevel.volume.Sub(quantity)
		priceLevel.visible = priceLevel.visible.Sub(visible)
	}

	if order.IsExecuted() {
		// Dequeue the empty order from the order queue of the price level
		priceLevel.queue.Remove(order.orderQueued)
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
	if priceLevel.Orders() == 0 {
		err = ob.deletePriceLevel(tree, priceLevel.price)
		if err != nil {
			return
		}
		update.Kind = PriceLevelUpdateKindDelete
	}

	return
}

// deleteOrder deletes order from order book (real or virtual),
// real means order book with real volume (limit orders),
// virtual order books are used for activation of stop orders.
func (ob *OrderBook) deleteOrder(tree *avl.Tree[Uint, *PriceLevelL3], order *Order) (update PriceLevelUpdate, err error) {
	update.Kind = PriceLevelUpdateKindUpdate

	// Ensure the tree is specified
	if tree == nil {
		err = ErrOrderTreeNotFound
		return
	}

	// Find the price level for the order
	node := order.priceLevel
	if node == nil {
		err = ErrPriceLevelNotFound
		return
	}

	priceLevel := node.Value()

	if !order.IsVirtualOB() {
		// Update the price level volume
		priceLevel.volume = priceLevel.volume.Sub(order.restQuantity)
		priceLevel.visible = priceLevel.visible.Sub(order.VisibleQuantity())
	}

	// Dequeue the deleted order from the order queue of the price level
	priceLevel.queue.Remove(order.orderQueued)
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
	if priceLevel.Orders() == 0 {
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

// Debugging printer
func (ob *OrderBook) Debug() {
	fmt.Printf("\n\nDebugging: order book state\n\n")
	fmt.Printf("Market price: %s\n", ob.marketPrice.ToFloatString())
	fmt.Printf("Mark price: %s\n", ob.markPrice.ToFloatString())
	fmt.Printf("Index price: %s\n", ob.indexPrice.ToFloatString())

	orders := ob.orders.Values()

	fmt.Printf("\nOrders\n")
	for i := range orders {
		orders[i].Debug()
	}
}
