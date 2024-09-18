package matching

import "fmt"

// Engine is used to manage the market with orders, price levels and order books.
// Automatic orders matching can be enabled with EnableMatching() method or can be
// manually performed with Match() method.
// NOTE: The matching engine is thread safe only when created with multithread flag.
type Engine struct {
	handler Handler

	// Order books
	orderBooks      []*OrderBook
	orderBooksCount int

	// Automatic matching
	matching bool

	// Multi-thread mode
	multithread bool
}

// NewEngine creates and returns new Engine instance.
func NewEngine(handler Handler, multithread bool) *Engine {
	return &Engine{
		handler:     handler,
		orderBooks:  make([]*OrderBook, defaultReservedOrderBookSlots),
		multithread: multithread,
	}
}

// Start starts the matching engine.
func (e *Engine) Start() {}

// Stop stops the matching engine.
// It releases all internally used order books and cleans whole order book state.
func (e *Engine) Stop(forced bool) {

	// Close all order book tasks channels
	for i, c := 0, len(e.orderBooks); i < c; i++ {
		if e.orderBooks[i] != nil {
			close(e.orderBooks[i].chanTasks)
			if forced {
				close(e.orderBooks[i].chanForcedStop)
			}
		}
	}

	// Wait until everything is done
	for i, c := 0, len(e.orderBooks); i < c; i++ {
		if e.orderBooks[i] != nil {
			e.orderBooks[i].wg.Wait()
		}
	}

	// Clean all existing order books
	for i, c := 0, len(e.orderBooks); i < c; i++ {
		if e.orderBooks[i] != nil {
			e.orderBooks[i].Clean()
			e.orderBooks[i] = nil
		}
	}
	e.orderBooksCount = 0
}

////////////////////////////////////////////////////////////////
// Engine common
////////////////////////////////////////////////////////////////

// OrderBook returns the order book with given symbol id.
func (e *Engine) OrderBook(id uint32) *OrderBook {
	if int(id) >= len(e.orderBooks) {
		return nil
	}
	return e.orderBooks[id]
}

// OrderBooks returns total amount of currently existing order books.
func (e *Engine) OrderBooks() int {
	return e.orderBooksCount
}

// Orders returns total amount of currently existing orders.
func (e *Engine) Orders() int {
	orders := 0
	for i, c := 0, len(e.orderBooks); i < c; i++ {
		if e.orderBooks[i] != nil {
			orders += e.orderBooks[i].Size()
		}
	}
	return orders
}

// IsMatchingEnabled returns true if automatic matching is enabled.
func (e *Engine) IsMatchingEnabled() bool {
	return e.matching
}

// EnableMatching enables automatic matching.
func (e *Engine) EnableMatching() {
	e.matching = true
	e.Match()
}

// DisableMatching disables automatic matching.
func (e *Engine) DisableMatching() {
	e.matching = false
}

////////////////////////////////////////////////////////////////
// Order books management
////////////////////////////////////////////////////////////////

// AddOrderBook creates new order book and adds it to the engine.
func (e *Engine) AddOrderBook(symbol Symbol, marketPrice Uint, spModesConfig StopPriceModeConfig) (orderBook *OrderBook, err error) {
	if !symbol.Valid() {
		err = ErrInvalidSymbol
		return
	}

	// Ensure order books storage size
	newSize := len(e.orderBooks)
	for newSize <= int(symbol.id) {
		newSize *= 2
	}
	if newSize > len(e.orderBooks) {
		newOrderBooks := make([]*OrderBook, newSize)
		copy(newOrderBooks, e.orderBooks)
		e.orderBooks = newOrderBooks
	}

	// Ensure order book does not exist
	if e.orderBooks[symbol.id] != nil {
		err = ErrOrderBookDuplicate
		return
	}

	// Create order book
	orderBook = NewOrderBook(symbol, spModesConfig, defaultOrderBookTaskQueueSize)
	orderBook.marketPrice = marketPrice
	e.orderBooks[symbol.id] = orderBook
	e.orderBooksCount++

	// Call the corresponding handler
	e.handler.OnAddOrderBook(orderBook)

	// Run goroutine unique to the order book to perform order book specific tasks
	if e.multithread {
		go e.loopOrderBook(orderBook)
	}

	return
}

// DeleteOrderBook deletes order book from the engine.
func (e *Engine) DeleteOrderBook(id uint32) (orderBook *OrderBook, err error) {

	// Ensure order book exists
	if int(id) >= len(e.orderBooks) || e.orderBooks[id] == nil {
		err = ErrOrderBookNotFound
		return
	}

	orderBook = e.orderBooks[id]

	// Close order book tasks channel
	close(orderBook.chanTasks)

	// Wait until all order book tasks are performed
	orderBook.wg.Wait()

	// Call the corresponding handler
	e.handler.OnDeleteOrderBook(orderBook)

	// Clean and delete order book
	orderBook.Clean()
	e.orderBooks[id] = nil
	e.orderBooksCount--

	return
}

// GetMarketPriceForOrderBook return market price of given symbolID (last executed trade).
// NOTE: not concurrency safe, use when there is no matching process.
func (e *Engine) GetMarketPriceForOrderBook(symbolID uint32) (Uint, error) {
	orderBook := e.OrderBook(symbolID)
	if orderBook == nil {
		return Uint{}, ErrOrderBookNotFound
	}

	return orderBook.GetMarketPrice(), nil
}

// SetIndexMarkPricesForOrderBook sets index and prices for order book at the same time,
// it has ability to provoke matching iteration for disabled matching.
func (e *Engine) SetIndexMarkPricesForOrderBook(symbolID uint32, indexPrice Uint, markPrice Uint, iterate bool) error {
	ob := e.OrderBook(symbolID)
	if ob == nil {
		return ErrOrderBookNotFound
	}

	task := func(ob *OrderBook) error {
		ob.setIndexPrice(indexPrice)
		ob.setMarkPrice(markPrice)

		if e.matching || iterate {
			e.match(ob)
		}

		return nil
	}

	return e.performOrderBookTask(ob, task)
}

// SetMarkPrice sets the mark price for order book,
// it has ability to provoke matching iteration for disabled matching.
func (e *Engine) SetMarkPriceForOrderBook(symbolID uint32, price Uint, iterate bool) error {
	ob := e.OrderBook(symbolID)
	if ob == nil {
		return ErrOrderBookNotFound
	}

	task := func(ob *OrderBook) error {
		ob.setMarkPrice(price)

		if e.matching || iterate {
			e.match(ob)
		}

		return nil
	}

	return e.performOrderBookTask(ob, task)
}

// SetIndexPriceForOrderBook sets the index price for order book,
// it has ability to provoke matching iteration for disabled matching.
func (e *Engine) SetIndexPriceForOrderBook(symbolID uint32, price Uint, iterate bool) error {
	ob := e.OrderBook(symbolID)
	if ob == nil {
		return ErrOrderBookNotFound
	}

	task := func(ob *OrderBook) error {
		ob.setIndexPrice(price)

		if e.matching || iterate {
			e.match(ob)
		}

		return nil
	}

	return e.performOrderBookTask(ob, task)
}

////////////////////////////////////////////////////////////////
// Orders management
////////////////////////////////////////////////////////////////

// AddOrder adds new order to the engine.
func (e *Engine) AddOrder(order Order) error {
	// Get the valid order book for the order
	ob := e.OrderBook(order.symbolID)
	if ob == nil {
		return ErrOrderBookNotFound
	}

	// Change market slippage before validation
	if order.Type() == OrderTypeMarket || order.Type() == OrderTypeStop || order.Type() == OrderTypeTrailingStop {
		order.marketSlippage = Min(order.marketSlippage, ob.symbol.priceLimits.Max)
	}

	// Validate order parameters
	if err := order.Validate(ob); err != nil {
		return err
	}

	// Validate order parameters
	if err := order.CheckLocked(); err != nil {
		return err
	}

	task := func(ob *OrderBook) error {
		// Add the corresponding order type
		switch order.orderType {
		case OrderTypeLimit:
			return e.addLimitOrder(ob, order, false)
		case OrderTypeMarket:
			return e.addMarketOrder(ob, order, false)
		case OrderTypeStop, OrderTypeTrailingStop:
			return e.addStopOrder(ob, order, false)
		case OrderTypeStopLimit, OrderTypeTrailingStopLimit:
			return e.addStopLimitOrder(ob, order, false)
		default:
			return ErrInvalidOrderType
		}
	}

	return e.performOrderBookTask(ob, task)
}

// AddOrdersPair adds new orders pair (OCO orders) to the engine.
// First order should be stop-limit order and second one should be limit order.
// NOTE: lock all amount in limit order.
func (e *Engine) AddOrdersPair(stopLimitOrder Order, limitOrder Order) error {

	// Get the valid order book for the order
	ob := e.OrderBook(stopLimitOrder.symbolID)
	if ob == nil {
		return ErrOrderBookNotFound
	}

	// Validate orders parameters
	if err := stopLimitOrder.Validate(ob); err != nil {
		return err
	}
	if err := limitOrder.Validate(ob); err != nil {
		return err
	}

	// Check locked
	if err := CheckLockedOCO(&stopLimitOrder, &limitOrder); err != nil {
		return err
	}

	// Link OCO orders to each other
	stopLimitOrder.linkedOrderID = limitOrder.id
	limitOrder.linkedOrderID = stopLimitOrder.id

	task := func(ob *OrderBook) error {

		// Check market price
		if stopLimitOrder.IsBuy() {
			if stopLimitOrder.stopPrice.LessThan(ob.GetMarketPrice()) {
				return ErrBuyOCOStopPriceLessThanMarketPrice
			}
			if limitOrder.price.GreaterThan(ob.GetMarketPrice()) {
				return ErrBuyOCOLimitPriceGreaterThanMarketPrice
			}
		} else {
			if stopLimitOrder.stopPrice.GreaterThan(ob.GetMarketPrice()) {
				return ErrSellOCOStopPriceGreaterThanMarketPrice
			}
			if limitOrder.price.LessThan(ob.GetMarketPrice()) {
				return ErrSellOCOLimitPriceLessThanMarketPrice
			}
		}

		// Add limit order first (it has higher priority to be placed)
		err := e.addLimitOrder(ob, limitOrder, false)
		if err != nil {
			return err
		}

		limitOrderFromOB := ob.Order(limitOrder.id)

		// Check if limit order has been executed
		if limitOrderFromOB == nil || limitOrderFromOB.PartiallyExecuted() {
			// Imitation of order placing and cancellation
			e.handler.OnAddOrder(ob, &stopLimitOrder)
			e.handler.OnDeleteOrder(ob, &stopLimitOrder)
		} else {
			// Add stop-limit order
			err = e.addStopLimitOrder(ob, stopLimitOrder, false)
			if err != nil {
				return err
			}

			// Find stop-limit order in orderbook
			stopLimitOrderFromOB := ob.Order(stopLimitOrder.id)

			// Check if stop-limit order has been executed or activated
			if stopLimitOrderFromOB == nil || stopLimitOrderFromOB.Activated() {
				// check if order has been already deleted
				limitOrderFromOB = ob.Order(limitOrderFromOB.id)
				if limitOrderFromOB == nil {
					return nil
				}

				// Cancel limit order
				err := e.deleteOrder(ob, limitOrderFromOB, false)
				if err != nil {
					return fmt.Errorf("failed to delete order (id: %d): %w", limitOrderFromOB.ID(), err)
				}
			}
		}

		return nil
	}

	return e.performOrderBookTask(ob, task)
}

// AddTPSL adds new orders pair take-profit and stop-loss (OCO orders) to the engine.
// The first order should be take-profit order and the second order should be stop-loss.
// Based on stop-limit type.
// NOTE: lock all amount in take-profit order.
func (e *Engine) AddTPSL(tp Order, sl Order) error {
	// Get the valid order book for the order
	ob := e.OrderBook(tp.symbolID)
	if ob == nil {
		return ErrOrderBookNotFound
	}

	// Validate orders parameters
	if err := tp.Validate(ob); err != nil {
		return err
	}
	if err := sl.Validate(ob); err != nil {
		return err
	}

	// Check locked
	if err := CheckLockedTPSL(&tp, &sl); err != nil {
		return err
	}

	// Link OCO orders to each other
	tp.linkedOrderID = sl.id
	sl.linkedOrderID = tp.id

	task := func(ob *OrderBook) error {
		engineStopPrice := ob.GetStopPrice(tp.StopPriceMode())

		// Check engine price
		if tp.IsBuy() {
			if sl.stopPrice.LessThan(engineStopPrice) {
				return ErrBuySLStopPriceLessThanEnginePrice
			}
			if tp.stopPrice.GreaterThan(engineStopPrice) {
				return ErrBuyTPStopPriceGreaterThanEnginePrice
			}
		} else {
			if sl.stopPrice.GreaterThan(engineStopPrice) {
				return ErrSellSLStopPriceGreaterThanEnginePrice
			}
			if tp.stopPrice.LessThan(engineStopPrice) {
				return ErrSellTPStopPriceLessThanEnginePrice
			}
		}

		err := e.addStopLimitOrder(ob, tp, false)
		if err != nil {
			return err
		}

		tpFromOB := ob.Order(tp.id)

		// Check if tp order has been executed or activated
		if tpFromOB == nil || tpFromOB.Activated() {
			// Imitation of order placing and cancellation of sl linked order
			e.handler.OnAddOrder(ob, &sl)
			e.handler.OnDeleteOrder(ob, &sl)
		} else {
			// Add sl order
			err = e.addStopLimitOrder(ob, sl, false)
			if err != nil {
				return err
			}

			// Find sl order in orderbook
			slFromOB := ob.Order(sl.id)

			// Check if sl order has been executed or activated
			if slFromOB == nil || slFromOB.Activated() {
				// check if order has been already deleted
				tpFromOB = ob.Order(tp.id)
				if tpFromOB == nil {
					return nil
				}

				// Cancel tp linked order
				err := e.deleteOrder(ob, tpFromOB, false)
				if err != nil {
					return fmt.Errorf("failed to delete order (id: %d): %w", tpFromOB.ID(), err)
				}
			}
		}

		return nil
	}

	return e.performOrderBookTask(ob, task)
}

// AddTPSLMarket adds new orders pair take-profit and stop-limit (OCO orders) to the engine.
// The first order should be take-profit order and the second order should be stop-limit.
// Based on stop type.
// NOTE: lock all amount in take-profit order.
func (e *Engine) AddTPSLMarket(tp Order, sl Order) error {
	// Get the valid order book for the order
	ob := e.OrderBook(tp.symbolID)
	if ob == nil {
		return ErrOrderBookNotFound
	}

	// Validate orders parameters
	if err := tp.Validate(ob); err != nil {
		return err
	}
	if err := sl.Validate(ob); err != nil {
		return err
	}

	// Check locked
	if err := CheckLockedTPSL(&tp, &sl); err != nil {
		return err
	}

	// Link OCO orders to each other
	tp.linkedOrderID = sl.id
	sl.linkedOrderID = tp.id

	task := func(ob *OrderBook) error {
		engineStopPrice := ob.GetStopPrice(tp.StopPriceMode())

		// Check engine price
		if tp.IsBuy() {
			if sl.stopPrice.LessThan(engineStopPrice) {
				return ErrBuySLStopPriceLessThanEnginePrice
			}
			if tp.stopPrice.GreaterThan(engineStopPrice) {
				return ErrBuyTPStopPriceGreaterThanEnginePrice
			}
		} else {
			if sl.stopPrice.GreaterThan(engineStopPrice) {
				return ErrSellSLStopPriceGreaterThanEnginePrice
			}
			if tp.stopPrice.LessThan(engineStopPrice) {
				return ErrSellTPStopPriceLessThanEnginePrice
			}
		}

		err := e.addStopOrder(ob, tp, false)
		if err != nil {
			return err
		}

		tpFromOB := ob.Order(tp.id)

		// Check if tp order has been executed or activated
		if tpFromOB == nil || tpFromOB.Activated() {
			// Imitation of order placing and cancellation of sl linked order
			e.handler.OnAddOrder(ob, &sl)
			e.handler.OnDeleteOrder(ob, &sl)
		} else {
			// Add sl order
			err = e.addStopOrder(ob, sl, false)
			if err != nil {
				return err
			}

			// Find sl order in orderbook
			slFromOB := ob.Order(sl.id)

			// Check if sl order has been executed or activated
			if slFromOB == nil || slFromOB.Activated() {
				// check if order has been already deleted
				tpFromOB = ob.Order(tp.id)
				if tpFromOB == nil {
					return nil
				}

				// Cancel tp linked order
				err := e.deleteOrder(ob, tpFromOB, false)
				if err != nil {
					return fmt.Errorf("failed to delete order (id: %d): %w", tpFromOB.ID(), err)
				}
			}
		}

		return nil
	}

	return e.performOrderBookTask(ob, task)
}

// ReduceOrder reduces the order by the given quantity.
func (e *Engine) ReduceOrder(symbolID uint32, orderID uint64, quantity Uint) error {
	// Get the valid order book for the order
	ob := e.OrderBook(symbolID)
	if ob == nil {
		return ErrOrderBookNotFound
	}

	task := func(ob *OrderBook) error {

		// Get the order by given id
		order := ob.Order(orderID)
		if order == nil {
			return ErrOrderNotFound
		}

		// Reduce the order
		return e.reduceOrder(ob, order, quantity, false)
	}

	return e.performOrderBookTask(ob, task)
}

// ModifyOrder modifies the order with the given new price and quantity.
func (e *Engine) ModifyOrder(symbolID uint32, orderID uint64, newPrice Uint, newQuantity Uint) error {

	// Get the valid order book for the order
	ob := e.OrderBook(symbolID)
	if ob == nil {
		return ErrOrderBookNotFound
	}

	task := func(ob *OrderBook) error {

		// Get the order by given id
		order := ob.Order(orderID)
		if order == nil {
			return ErrOrderNotFound
		}

		// Modify the order
		return e.modifyOrder(ob, order, newPrice, newQuantity, NewZeroUint(), false, false)
	}

	return e.performOrderBookTask(ob, task)
}

// MitigateOrder mitigates the order with the given new price and quantity.
func (e *Engine) MitigateOrder(symbolID uint32, orderID uint64, newPrice Uint, newQuantity Uint, additionalAmountToLock Uint) error {

	// Get the valid order book for the order
	ob := e.OrderBook(symbolID)
	if ob == nil {
		return ErrOrderBookNotFound
	}

	task := func(ob *OrderBook) error {

		// Get the order by given id
		order := ob.Order(orderID)
		if order == nil {
			return ErrOrderNotFound
		}

		// Mitigate the order
		return e.modifyOrder(ob, order, newPrice, newQuantity, additionalAmountToLock, true, false)
	}

	return e.performOrderBookTask(ob, task)
}

// ReplaceOrder replaces the order with a new one.
func (e *Engine) ReplaceOrder(symbolID uint32, orderID uint64, newID uint64, newPrice Uint, newQuantity Uint) error {

	// Get the valid order book for the order
	ob := e.OrderBook(symbolID)
	if ob == nil {
		return ErrOrderBookNotFound
	}

	task := func(ob *OrderBook) error {

		// Get the order by given id
		order := ob.Order(orderID)
		if order == nil {
			return ErrOrderNotFound
		}
		if !order.IsLimit() {
			// Only limit orders can be replaced
			return ErrInvalidOrderType
		}

		// Replace the order with new one
		return e.replaceOrder(ob, order, newID, newPrice, newQuantity, false)
	}

	return e.performOrderBookTask(ob, task)
}

// DeleteOrder deletes the order from the engine.
func (e *Engine) DeleteOrder(symbolID uint32, orderID uint64) error {

	// Get the valid order book for the order
	ob := e.OrderBook(symbolID)
	if ob == nil {
		return ErrOrderBookNotFound
	}

	task := func(ob *OrderBook) error {

		// Get the order by given id
		order := ob.Order(orderID)
		if order == nil {
			return ErrOrderNotFound
		}

		// Delete linked order if it exists
		err := e.deleteLinkedOrder(ob, order, false)
		if err != nil {
			return fmt.Errorf("failed to delete linked order (id: %d): %w", order.ID(), err)
		}

		// Delete the order
		return e.deleteOrder(ob, order, false)
	}

	return e.performOrderBookTask(ob, task)
}

// ExecuteOrder executes the order by the given quantity.
func (e *Engine) ExecuteOrder(symbolID uint32, orderID uint64, quantity Uint) error {
	if e.matching {
		return ErrForbiddenManualExecution
	}

	// Get the valid order book for the order
	ob := e.OrderBook(symbolID)
	if ob == nil {
		return ErrOrderBookNotFound
	}

	task := func(ob *OrderBook) (err error) {
		// Get the order by given id
		order := ob.Order(orderID)
		if order == nil {
			return ErrOrderNotFound
		}

		// Calculate the minimal possible order quantity to execute
		orderQuantity := order.RestQuantity()
		quantity = Min(quantity, orderQuantity)
		quoteQuantity := quantity.Mul(order.price).Div64(UintPrecision)

		// Call the corresponding handler
		e.handler.OnExecuteOrder(ob, order.id, order.price, quantity, quoteQuantity)

		// Update the common market price
		ob.updateMarketPrice(order.price)

		visible := order.VisibleQuantity()

		// Decrease the order available quantity
		if order.IsBuy() {
			order.available = order.available.Sub(quoteQuantity)
		} else {
			order.available = order.available.Sub(quantity)
		}

		// Increase the order executed quantity1
		order.executedQuantity = order.executedQuantity.Add(quantity)
		order.executedQuoteQuantity = order.executedQuoteQuantity.Add(quoteQuantity)

		// Reduce the order leaves quantity
		order.restQuantity = orderQuantity.Sub(quantity)

		visible = visible.Sub(order.VisibleQuantity())

		// Reduce the order in the order book
		priceLevelUpdate, err := ob.reduceOrder(ob.treeForOrder(order), order, quantity, visible)
		if err != nil {
			return err
		}
		if order.IsLimit() {
			e.handleUpdatePriceLevel(ob, priceLevelUpdate)
		}

		// Update the order or delete the empty order
		if !order.IsExecuted() {

			// Call the corresponding handler
			e.handler.OnUpdateOrder(ob, order)

		} else {

			// Call the corresponding handler
			e.handler.OnDeleteOrder(ob, order)

			// Erase the order
			ob.orders.Delete(order.id)

			// Release the order
			ob.allocator.PutOrder(order)
		}

		// Automatic order matching
		if e.matching {
			err := e.match(ob)
			if err != nil {
				return fmt.Errorf("failed to match: %w", err)
			}
		}

		return
	}

	return e.performOrderBookTask(ob, task)
}

// ExecuteOrderByPrice executes the order by the given price and quantity.
func (e *Engine) ExecuteOrderByPrice(symbolID uint32, orderID uint64, price Uint, quantity Uint) error {
	if e.matching {
		return ErrForbiddenManualExecution
	}

	// Get the valid order book for the order
	ob := e.OrderBook(symbolID)
	if ob == nil {
		return ErrOrderBookNotFound
	}

	task := func(ob *OrderBook) (err error) {

		// Get the order by given id
		order := ob.Order(orderID)
		if order == nil {
			return ErrOrderNotFound
		}

		// Calculate the minimal possible order quantity to execute
		orderQuantity := order.RestQuantity()
		quantity = Min(quantity, orderQuantity)
		quoteQuantity := quantity.Mul(price).Div64(UintPrecision)

		// Call the corresponding handler
		e.handler.OnExecuteOrder(ob, order.id, price, quantity, quoteQuantity)

		// Update the common market price
		ob.updateMarketPrice(order.price)

		visible := order.VisibleQuantity()

		// Decrease the order available quantity
		if order.IsBuy() {
			order.available = order.available.Sub(quoteQuantity)
		} else {
			order.available = order.available.Sub(quantity)
		}

		// Increase the order executed quantity1
		order.executedQuantity = order.executedQuantity.Add(quantity)
		order.executedQuoteQuantity = order.executedQuoteQuantity.Add(quoteQuantity)

		// Reduce the order leaves quantity
		order.restQuantity = orderQuantity.Sub(quantity)

		visible = visible.Sub(order.VisibleQuantity())

		// Reduce the order in the order book
		priceLevelUpdate, err := ob.reduceOrder(ob.treeForOrder(order), order, quantity, visible)
		if err != nil {
			return err
		}
		if order.IsLimit() {
			e.handleUpdatePriceLevel(ob, priceLevelUpdate)
		}

		// Update the order or delete the empty order
		if !order.IsExecuted() {

			// Call the corresponding handler
			e.handler.OnUpdateOrder(ob, order)

		} else {

			// Call the corresponding handler
			e.handler.OnDeleteOrder(ob, order)

			// Erase the order
			ob.orders.Delete(order.id)

			// Release the order
			ob.allocator.PutOrder(order)
		}

		// Automatic order matching
		if e.matching {
			err := e.match(ob)
			if err != nil {
				return fmt.Errorf("failed to match: %w", err)
			}
		}

		return
	}

	return e.performOrderBookTask(ob, task)
}

////////////////////////////////////////////////////////////////
// Matching
////////////////////////////////////////////////////////////////

// Match matches crossed orders in all order books.
// Method will match all crossed orders in each order book. Buy orders will be
// matched with sell orders at arbitrage price starting from the top of the book.
// Matched orders will be executed with deleted form the order book. After the
// matching operation each order book will have the top (best) bid price guarantied
// less than the top (best) ask price!
func (e *Engine) Match() {
	task := func(ob *OrderBook) error {
		err := e.match(ob)
		if err != nil {
			return fmt.Errorf("failed to match: %w", err)
		}
		return nil
	}
	for i, c := 0, len(e.orderBooks); i < c; i++ {
		if e.orderBooks[i] != nil {
			e.performOrderBookTask(e.orderBooks[i], task)
		}
	}
}

////////////////////////////////////////////////////////////////
// Loops
////////////////////////////////////////////////////////////////

// loopOrderBook is unique for order book goroutine separately working with given order book and performing enqueued tasks.
func (e *Engine) loopOrderBook(ob *OrderBook) {
	ob.wg.Add(1)
	defer ob.wg.Done()

	// Loop over order book tasks from the queue
	for {
		select {
		case task, ok := <-ob.chanTasks:
			if !ok {
				return
			}
			// Perform task
			if err := task(ob); err != nil {
				// Call the corresponding handler
				// TODO: Make handled errors more informative
				e.handler.OnError(ob, err)
			}
		case <-ob.chanForcedStop:
			return
		}
	}
}

////////////////////////////////////////////////////////////////
// Internal helpers
////////////////////////////////////////////////////////////////

func (e *Engine) handleUpdatePriceLevel(ob *OrderBook, update PriceLevelUpdate) {
	update.ID = ob.lastUpdateID
	ob.lastUpdateID++ // no need to use atomic.AddUint64()
	switch update.Kind {
	case PriceLevelUpdateKindAdd:
		e.handler.OnAddPriceLevel(ob, update)
	case PriceLevelUpdateKindUpdate:
		e.handler.OnUpdatePriceLevel(ob, update)
	case PriceLevelUpdateKindDelete:
		e.handler.OnDeletePriceLevel(ob, update)
	}
	e.handler.OnUpdateOrderBook(ob)
}

func (e *Engine) performOrderBookTask(ob *OrderBook, task func(ob *OrderBook) error) error {
	if e.multithread {
		ob.chanTasks <- task
		return nil
	} else {
		err := task(ob)
		if err != nil {
			// Call the corresponding handler
			e.handler.OnError(ob, err)
		}
		return err
	}
}
