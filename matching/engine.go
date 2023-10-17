package matching

// Engine is used to manage the market with orders, price levels and order books.
// Automatic orders matching can be enabled with EnableMatching() method or can be
// manually performed with Match() method.
// NOTE: The matching engine is thread safe only when created with multithread flag.
type Engine struct {
	handler Handler

	// Allocator used by all order books
	allocator *Allocator

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
		allocator:   NewAllocator(),
		orderBooks:  make([]*OrderBook, defaultReservedOrderBookSlots),
		multithread: multithread,
	}
}

// Start starts the matching engine.
func (e *Engine) Start() {
}

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
			// TODO: Size() is not orders but price levels!
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
func (e *Engine) AddOrderBook(symbol Symbol) (orderBook *OrderBook, err error) {

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

	// Prepare allocator
	// TODO: Maybe it is not better to use single allocator for all order books
	// TODO: Test how GC behaves in both cases (single allocator or own allocator for each order book)
	// allocator := NewAllocator()
	allocator := e.allocator

	// Create order book
	orderBook = NewOrderBook(allocator, symbol, defaultOrderBookTaskQueueSize)
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
func (e *Engine) GetMarketPriceForOrderBook(symbolID uint32) (Uint, error) {

	orderBook := e.OrderBook(symbolID)
	if orderBook == nil {
		return Uint{}, ErrOrderBookNotFound
	}

	return orderBook.GetMarketPrice(), nil
}

////////////////////////////////////////////////////////////////
// Orders management
////////////////////////////////////////////////////////////////

// AddOrder adds new order to the engine.
func (e *Engine) AddOrder(order Order) error {

	// Validate order parameters
	if err := order.Validate(); err != nil {
		return err
	}

	// Get the valid order book for the order
	ob := e.OrderBook(order.symbolID)
	if ob == nil {
		return ErrOrderBookNotFound
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
func (e *Engine) AddOrdersPair(stopLimitOrder Order, limitOrder Order) error {

	// Validate orders parameters
	if err := stopLimitOrder.Validate(); err != nil {
		return err
	}
	if err := limitOrder.Validate(); err != nil {
		return err
	}

	// Link OCO orders to each other
	stopLimitOrder.linkedOrderID = limitOrder.id
	limitOrder.linkedOrderID = stopLimitOrder.id

	// Get the valid order book for the order
	ob := e.OrderBook(stopLimitOrder.symbolID)
	if ob == nil {
		return ErrOrderBookNotFound
	}

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

		// Add stop-limit order first
		err := e.addStopLimitOrder(ob, stopLimitOrder, false)
		if err != nil {
			return err
		}

		// Find stop-limit order in orderbook
		stopLimitOrderFromOB := ob.Order(stopLimitOrder.id)

		// Check if stop-limit order has been executed or activated
		if stopLimitOrderFromOB == nil || stopLimitOrderFromOB.Type() == OrderTypeLimit {
			// Imitation of order placing and cancellation
			e.handler.OnAddOrder(ob, &limitOrder)
			e.handler.OnDeleteOrder(ob, &limitOrder)
		} else {
			// Add limit order
			err = e.addLimitOrder(ob, limitOrder, false)
			if err != nil {
				return err
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
		e.deleteLinkedOrder(ob, order, false)

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
		orderQuantity := order.RestAvailableQuantity(order.price)
		quantity = Min(quantity, orderQuantity)
		quoteQuantity := quantity.Mul(order.price).Div64(UintPrecision)

		// Call the corresponding handler
		e.handler.OnExecuteOrder(ob, order, order.price, quantity)

		// Update the corresponding market price
		ob.updateLastPrice(order.side, order.price)
		ob.updateMatchingPrice(order.side, order.price)

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
			e.updatePriceLevel(ob, priceLevelUpdate)
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
			e.allocator.PutOrder(order)
		}

		// Automatic order matching
		if e.matching {
			e.match(ob)
		}

		// Reset matching price
		ob.resetMatchingPrice()

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
		orderQuantity := order.RestAvailableQuantity(price)
		quantity = Min(quantity, orderQuantity)
		quoteQuantity := quantity.Mul(price).Div64(UintPrecision)

		// Call the corresponding handler
		e.handler.OnExecuteOrder(ob, order, price, quantity)

		// Update the corresponding market price
		ob.updateLastPrice(order.side, price)
		ob.updateMatchingPrice(order.side, price)

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
			e.updatePriceLevel(ob, priceLevelUpdate)
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
			e.allocator.PutOrder(order)
		}

		// Automatic order matching
		if e.matching {
			e.match(ob)
		}

		// Reset matching price
		ob.resetMatchingPrice()

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
		e.match(ob)
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

func (e *Engine) updatePriceLevel(ob *OrderBook, update PriceLevelUpdate) {
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
