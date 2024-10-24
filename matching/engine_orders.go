package matching

import (
	"fmt"
)

////////////////////////////////////////////////////////////////
// Adding new orders
////////////////////////////////////////////////////////////////

func (e *Engine) addLimitOrder(ob *OrderBook, order Order, recursive bool) error {
	// Check duplicate
	if _, ok := ob.orders.Get(order.id); ok {
		return ErrOrderDuplicate
	}

	// Create a new order
	newOrder := ob.allocator.GetOrder()
	*newOrder = order

	// Call the corresponding handler
	e.handler.OnAddOrder(ob, newOrder)

	// Automatic order matching
	if e.matching && !recursive {
		err := e.matchLimitOrder(ob, newOrder)
		if err != nil {
			return fmt.Errorf("failed to match limit order: %w", err)
		}
	}

	// Delete remaining part in case of 'Immediate-Or-Cancel'/'Fill-Or-Kill' and exit.
	// If executed, handler has been already called.
	if (newOrder.IsIOC() || newOrder.IsFOK()) && !newOrder.IsExecuted() {
		e.handler.OnDeleteOrder(ob, newOrder)
	}

	// Add remaining order in order book for GTC
	if newOrder.IsGTC() && !newOrder.IsExecuted() {
		// Set order to internal order storage
		ob.orders.Set(newOrder.id, newOrder)

		// Add the new limit order into the order book
		priceLevelUpdate, err := ob.addOrder(ob.treeForOrder(newOrder), newOrder)
		if err != nil {
			return err
		}
		e.handleUpdatePriceLevel(ob, priceLevelUpdate)
	}

	// Automatic order matching
	if e.matching && !recursive {
		err := e.match(ob)
		if err != nil {
			return fmt.Errorf("failed to match: %w", err)
		}
	}

	return nil
}

func (e *Engine) addMarketOrder(ob *OrderBook, order Order, recursive bool) error {
	// Check duplicate
	if _, ok := ob.orders.Get(order.id); ok {
		return ErrOrderDuplicate
	}

	newOrder := order

	newOrder.timeInForce = OrderTimeInForceIOC

	// Call the corresponding handler
	// Market order must be IOC
	e.handler.OnAddOrder(ob, &newOrder)

	// Automatic order matching
	if e.matching && !recursive {
		e.matchMarketOrder(ob, &newOrder)
	}

	if !newOrder.IsExecuted() {
		// Call the corresponding handler
		e.handler.OnDeleteOrder(ob, &newOrder)
	}

	// Automatic order matching
	if e.matching && !recursive {
		err := e.match(ob)
		if err != nil {
			return fmt.Errorf("failed to match: %w", err)
		}
	}

	return nil
}

func (e *Engine) addStopOrder(ob *OrderBook, order Order, recursive bool) error {
	// Check duplicate
	if _, ok := ob.orders.Get(order.id); ok {
		return ErrOrderDuplicate
	}

	// Create a new order
	newOrder := ob.allocator.GetOrder()
	*newOrder = order

	// Find the market price for further stop calculation
	marketPrice := ob.GetStopPrice(order.stopPriceMode)

	// If order isn't activated immediately we should specify if order is take profit or stop loss
	if newOrder.IsBuy() {
		newOrder.takeProfit = newOrder.stopPrice.LessThan(marketPrice)
	} else {
		newOrder.takeProfit = newOrder.stopPrice.GreaterThan(marketPrice)
	}

	// Recalculate stop price for trailing stop orders
	if newOrder.IsTrailingStop() {
		newOrder.stopPrice = ob.calculateTrailingStopPrice(newOrder)
	}

	// Set order to internal order storage
	ob.orders.Set(newOrder.id, newOrder)

	// Add the new stop order into the order book
	_, err := ob.addOrder(ob.treeForOrder(newOrder), newOrder)
	if err != nil {
		return err
	}

	// Call the corresponding handler
	e.handler.OnAddOrder(ob, newOrder)

	// Automatic order matching
	if !e.matching || recursive {
		return nil
	}

	// Check the market price
	arbitrage := newOrder.stopPrice.Equals(marketPrice)
	if arbitrage {
		// Call handler before further actions
		e.handler.OnActivateOrder(ob, newOrder)

		// delete linked order
		e.deleteLinkedOrder(ob, newOrder, true)

		// Delete the stop order from the order book
		_, err := ob.deleteOrder(ob.treeForOrder(newOrder), newOrder)
		if err != nil {
			return fmt.Errorf("failed to delete order: %w", err)
		}

		// Convert the stop order into the market order
		newOrder.orderType = OrderTypeMarket
		newOrder.price = NewZeroUint()
		newOrder.stopPrice = NewZeroUint()

		// Market order must be IOC
		newOrder.timeInForce = OrderTimeInForceIOC

		// Call the corresponding handler
		e.handler.OnUpdateOrder(ob, newOrder)

		// Match the market order
		e.matchMarketOrder(ob, newOrder)

		if !newOrder.IsExecuted() {
			// Call the corresponding handler
			e.handler.OnDeleteOrder(ob, newOrder)

			// Erase the order
			ob.orders.Delete(newOrder.id)

			// Release the order
			ob.allocator.PutOrder(newOrder)
		}
	}

	err = e.match(ob)
	if err != nil {
		return fmt.Errorf("failed to match: %w", err)
	}

	return nil
}

func (e *Engine) addStopLimitOrder(ob *OrderBook, order Order, recursive bool) error {
	// Check duplicate
	if _, ok := ob.orders.Get(order.id); ok {
		return ErrOrderDuplicate
	}

	// Create a new order
	newOrder := ob.allocator.GetOrder()
	*newOrder = order

	// Find the market price for further stop calculation
	engineStopPrice := ob.GetStopPrice(order.stopPriceMode)

	// If order isn't activated immediately we should specify if order is take profit or stop loss
	if newOrder.IsBuy() {
		newOrder.takeProfit = newOrder.stopPrice.LessThan(engineStopPrice)
	} else {
		newOrder.takeProfit = newOrder.stopPrice.GreaterThan(engineStopPrice)
	}

	// Recalculate stop price for trailing stop orders
	if newOrder.IsTrailingStopLimit() {
		if newOrder.price.GreaterThanOrEqualTo(newOrder.stopPrice) {
			// diff >= 0
			diff := newOrder.price.Sub(newOrder.stopPrice)
			newOrder.stopPrice = ob.calculateTrailingStopPrice(newOrder)
			newOrder.price = newOrder.stopPrice.Add(diff)
		} else {
			// diff < 0
			diff := newOrder.stopPrice.Sub(newOrder.price)
			newOrder.stopPrice = ob.calculateTrailingStopPrice(newOrder)
			newOrder.price = newOrder.stopPrice.Sub(diff)
		}
	}

	// Call the corresponding handler
	e.handler.OnAddOrder(ob, newOrder)

	// Check the market price
	arbitrage := newOrder.stopPrice.Equals(engineStopPrice)
	if !arbitrage {
		// Set order to internal order storage
		ob.orders.Set(newOrder.id, newOrder)

		// Add the new stop-limit order into the virtual order book
		_, err := ob.addOrder(ob.treeForOrder(newOrder), newOrder)
		if err != nil {
			return err
		}
	} else {
		// Call handler before further actions
		e.handler.OnActivateOrder(ob, newOrder)

		// delete linked order
		e.deleteLinkedOrder(ob, newOrder, true)

		// Convert the stop-limit order into the limit order
		newOrder.orderType = OrderTypeLimit
		newOrder.stopPrice = NewZeroUint()

		// Call the corresponding handler
		e.handler.OnUpdateOrder(ob, newOrder)

		// Automatic order matching
		if e.matching && !recursive {
			err := e.matchLimitOrder(ob, newOrder)
			if err != nil {
				return fmt.Errorf("failed to match limit order: %w", err)
			}
		}

		// Delete remaining part in case of 'Immediate-Or-Cancel'/'Fill-Or-Kill' and exit.
		// If executed, handler has been already called.
		if (newOrder.IsIOC() || newOrder.IsFOK()) && !newOrder.IsExecuted() {
			e.handler.OnDeleteOrder(ob, newOrder)
		}

		// Add remaining order in order book for GTC
		if newOrder.IsGTC() && !newOrder.IsExecuted() {
			// Set order to internal order storage
			ob.orders.Set(newOrder.id, newOrder)

			// Add the new limit order into the order book
			priceLevelUpdate, err := ob.addOrder(ob.treeForOrder(newOrder), newOrder)
			if err != nil {
				return err
			}
			e.handleUpdatePriceLevel(ob, priceLevelUpdate)
		}
	}

	// Automatic order matching
	if e.matching && !recursive {
		err := e.match(ob)
		if err != nil {
			return fmt.Errorf("failed to match: %w", err)
		}
	}
	return nil
}

////////////////////////////////////////////////////////////////
// Executing orders
////////////////////////////////////////////////////////////////

// executeOrder processes the fact of order execution.
// bool flag is true when order is executed/deleted.
func (e *Engine) executeOrder(ob *OrderBook, order *Order, qty Uint, quoteQty Uint) (bool, error) {
	// Decrease the order available quantity
	if order.IsLockingQuote() {
		order.SubAvailable(quoteQty)
	} else {
		order.SubAvailable(qty)
	}

	// Increase the order executed quantity
	order.AddExecutedQuantity(qty)
	order.AddExecutedQuoteQuantity(quoteQty)

	// Check and delete linked orders
	err := e.deleteLinkedOrder(ob, order, true)
	if err != nil {
		return false, fmt.Errorf("failed to delete linked order (id: %d): %w", order.ID(), err)
	}

	// Check market mode
	if order.marketQuoteMode {
		executed := false
		order.SubRestQuoteQuantity(quoteQty)
		if !order.IsExecuted() {
			e.handler.OnUpdateOrder(ob, order)
		} else {
			executed = true
			e.handler.OnDeleteOrder(ob, order)
			e.deleteOrder(ob, order, true)
		}

		return executed, nil
	}

	// Reduce the order rest quantities
	visible := order.VisibleQuantity()
	order.SubRestQuantity(qty)
	visible = visible.Sub(order.VisibleQuantity())
	executed := false

	if !order.IsExecuted() {
		e.handler.OnUpdateOrder(ob, order)
	} else {
		executed = true
		e.handler.OnDeleteOrder(ob, order)
	}

	// TODO: this part is copy from deleteOrder but without matching. Need unify.

	if order.priceLevel != nil {
		// Reduce the order in the order book
		priceLevelUpdate, err := ob.reduceOrder(ob.treeForOrder(order), order, qty, visible)
		if err != nil {
			return false, err
		}

		e.handleUpdatePriceLevel(ob, priceLevelUpdate)
	}

	// Delete the empty order
	if order.IsExecuted() {
		// Erase the order
		ob.orders.Delete(order.id)

		// Release the order
		ob.allocator.PutOrder(order)
	}

	return executed, nil
}

////////////////////////////////////////////////////////////////
// Reducing orders
////////////////////////////////////////////////////////////////

func (e *Engine) reduceOrder(ob *OrderBook, order *Order, quantity Uint, recursive bool) error {
	// Calculate the minimal possible order quantity to reduce
	quantity = Min(quantity, order.restQuantity)

	// Reduce the order rest quantity
	visible := order.VisibleQuantity()
	order.SubRestQuantity(quantity)
	visible = visible.Sub(order.VisibleQuantity())

	// Call the corresponding handler
	if !order.IsExecuted() {
		e.handler.OnUpdateOrder(ob, order)
	} else {
		e.handler.OnDeleteOrder(ob, order)
	}

	if order.priceLevel != nil {
		// Reduce the order in the order book
		priceLevelUpdate, err := ob.reduceOrder(ob.treeForOrder(order), order, quantity, visible)
		if err != nil {
			return err
		}

		e.handleUpdatePriceLevel(ob, priceLevelUpdate)
	}

	// Delete the empty order
	if order.IsExecuted() {
		// Erase the order
		ob.orders.Delete(order.id)

		// Release the order
		ob.allocator.PutOrder(order)
	}

	// Automatic order matching
	if e.matching && !recursive {
		err := e.match(ob)
		if err != nil {
			return fmt.Errorf("failed to match: %w", err)
		}
	}

	return nil
}

////////////////////////////////////////////////////////////////
// Modifying orders
////////////////////////////////////////////////////////////////

func (e *Engine) modifyOrder(ob *OrderBook, order *Order, newPrice Uint, newQuantity Uint, additionalAmountToLock Uint, mitigate bool, recursive bool) error {
	// Delete the order from the order book
	priceLevelUpdate, err := ob.deleteOrder(ob.treeForOrder(order), order)
	if err != nil {
		return err
	}
	if order.IsLimit() {
		e.handleUpdatePriceLevel(ob, priceLevelUpdate)
	}

	// Modify the order
	order.price = newPrice
	order.quantity = newQuantity
	order.restQuantity = newQuantity
	order.AddAvailable(additionalAmountToLock)

	// In-Flight Mitigation (IFM)
	if mitigate {
		// This calculation has the goal of preventing orders from being overfilled
		if newQuantity.GreaterThan(order.executedQuantity) {
			order.SubRestQuantity(order.executedQuantity)
		} else {
			order.restQuantity = NewZeroUint()
		}
	}

	// Update the order
	if !order.IsExecuted() {

		// Call the corresponding handler
		e.handler.OnUpdateOrder(ob, order)

		// Automatic order matching
		if e.matching && !recursive {
			err := e.matchLimitOrder(ob, order)
			if err != nil {
				return fmt.Errorf("failed to match limit order: %w", err)
			}
		}

		// Add non empty order into the order book
		if !order.IsExecuted() {
			// Add the modified order into the order book
			priceLevelUpdate, err := ob.addOrder(ob.treeForOrder(order), order)
			if err != nil {
				return err
			}
			if order.IsLimit() {
				e.handleUpdatePriceLevel(ob, priceLevelUpdate)
			}
		}

	}

	// Delete the empty order
	if order.IsExecuted() {
		// Call the corresponding handler
		e.handler.OnDeleteOrder(ob, order)

		// Erase the order
		ob.orders.Delete(order.id)

		// Release the order
		ob.allocator.PutOrder(order)
	}

	// Automatic order matching
	if e.matching && !recursive {
		err := e.match(ob)
		if err != nil {
			return fmt.Errorf("failed to match: %w", err)
		}
	}

	return nil
}

////////////////////////////////////////////////////////////////
// Replacing orders
////////////////////////////////////////////////////////////////

func (e *Engine) replaceOrder(ob *OrderBook, order *Order, newID uint64, newPrice Uint, newQuantity Uint, recursive bool) error {
	// Delete the old order from the order book
	priceLevelUpdate, err := ob.deleteOrder(ob.treeForOrder(order), order)
	if err != nil {
		return err
	}
	if order.IsLimit() {
		e.handleUpdatePriceLevel(ob, priceLevelUpdate)
	}

	// Call the corresponding handler
	e.handler.OnDeleteOrder(ob, order)

	// Erase the order
	ob.orders.Delete(order.id)

	// Update the order with new values
	order.id = newID
	order.price = newPrice
	order.quantity = newQuantity
	order.executedQuantity = NewZeroUint()
	order.executedQuoteQuantity = NewZeroUint()
	order.restQuantity = newQuantity

	// Call the corresponding handler
	e.handler.OnAddOrder(ob, order)

	// Automatic order matching
	if e.matching && !recursive {
		err := e.matchLimitOrder(ob, order)
		if err != nil {
			return fmt.Errorf("failed to match limit order: %w", err)
		}
	}

	// Add the order
	if !order.IsExecuted() {
		// Insert the order
		ob.orders.Set(order.id, order)

		// Add the modified order into the order book
		priceLevelUpdate, err := ob.addOrder(ob.treeForOrder(order), order)
		if err != nil {
			return err
		}
		if order.IsLimit() {
			e.handleUpdatePriceLevel(ob, priceLevelUpdate)
		}

	} else {
		// Call the corresponding handler
		e.handler.OnDeleteOrder(ob, order)

		// Release the order
		ob.allocator.PutOrder(order)
	}

	// Automatic order matching
	if e.matching && !recursive {
		err := e.match(ob)
		if err != nil {
			return fmt.Errorf("failed to match: %w", err)
		}
	}

	return nil
}

////////////////////////////////////////////////////////////////
// Deleting orders
////////////////////////////////////////////////////////////////

func (e *Engine) deleteOrder(ob *OrderBook, order *Order, recursive bool) error {
	// Delete the order from the order book
	if order.priceLevel != nil {
		priceLevelUpdate, err := ob.deleteOrder(ob.treeForOrder(order), order)
		if err != nil {
			return err
		}

		if !order.IsVirtualOB() {
			e.handleUpdatePriceLevel(ob, priceLevelUpdate)
		}
	}

	// Call the corresponding handler
	e.handler.OnDeleteOrder(ob, order)

	// Erase the order
	ob.orders.Delete(order.id)

	// Release the order
	ob.allocator.PutOrder(order)

	// Automatic order matching
	if e.matching && !recursive {
		err := e.match(ob)
		if err != nil {
			return fmt.Errorf("failed to match: %w", err)
		}
	}

	return nil
}

// Checks linked OCO order and deletes if it exists
func (e *Engine) deleteLinkedOrder(ob *OrderBook, order *Order, recursive bool) error {
	if order.linkedOrderID == 0 {
		return nil
	}

	linkedOrder := ob.Order(order.linkedOrderID)
	if linkedOrder != nil {
		// transfer available
		if !linkedOrder.available.IsZero() {
			order.AddAvailable(linkedOrder.available)
		}

		// reset link
		order.linkedOrderID = 0

		return e.deleteOrder(ob, linkedOrder, recursive)
	}

	return nil
}

// cutRemainders cuts parts of orders which are less than configured steps,
// bool flag is true when order is executed/deleted.
func (e *Engine) cutRemainders(ob *OrderBook, order *Order) bool {
	restQuantity, restQuoteQuantity, executed := order.RestQuantity(), order.RestQuoteQuantity(), false

	switch {
	case
		// Check rest quantities.
		!restQuantity.IsZero() && restQuantity.LessThan(ob.symbol.lotSizeLimits.Step),
		!restQuoteQuantity.IsZero() && restQuoteQuantity.LessThan(ob.symbol.quoteLotSizeLimits.Step),
		// Check locked quantities.
		order.IsLockingBase() && order.Available().LessThan(ob.symbol.lotSizeLimits.Step),
		order.IsLockingQuote() && order.Available().LessThan(ob.symbol.quoteLotSizeLimits.Step):

		// Delete order.
		e.deleteOrder(ob, order, true)
		executed = true
	}

	return executed
}

// calcRestAvailableQuantities calculate quantities for order with specified price.
func calcRestAvailableQuantities(order *Order, price Uint) (Uint, Uint) {
	// Calc rest quantities.

	restQuantity, restQuoteQuantity := order.RestQuantity(), order.RestQuoteQuantity()
	switch {
	case restQuantity.IsZero() && restQuoteQuantity.IsZero():
		return NewZeroUint(), NewZeroUint()
	case restQuantity.IsZero():
		restQuantity, restQuoteQuantity = calcQuantitiesFromQuoteAndPrice(restQuoteQuantity, price)
	case restQuoteQuantity.IsZero():
		restQuoteQuantity = restQuantity.Mul(price).Div64(UintPrecision)
	}

	// Check available.

	switch {
	// Available in base
	case order.IsLockingBase() && order.Available().LessThan(restQuantity):
		restQuantity = order.Available()
		restQuoteQuantity = order.Available().Mul(price).Div64(UintPrecision)
	// Available in quote
	case order.IsLockingQuote() && order.Available().LessThan(restQuoteQuantity):
		restQuantity, restQuoteQuantity = calcQuantitiesFromQuoteAndPrice(order.Available(), price)
	}

	return restQuantity, restQuoteQuantity
}

func calcQuantitiesFromQuoteAndPrice(quoteQuantity Uint, price Uint) (Uint, Uint) {
	quantity, _ := quoteQuantity.Mul64(UintPrecision).QuoRem(price)
	return quantity, quoteQuantity
}
