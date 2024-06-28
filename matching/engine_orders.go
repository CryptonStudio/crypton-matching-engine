package matching

import (
	"fmt"
)

////////////////////////////////////////////////////////////////
// Adding new orders
////////////////////////////////////////////////////////////////

func (e *Engine) addLimitOrder(ob *OrderBook, order Order, recursive bool) error {
	// Create a new order
	newOrder := e.allocator.GetOrder()
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

	// Add a new order or delete remaining part in case of 'Immediate-Or-Cancel'/'Fill-Or-Kill' order
	if !newOrder.IsExecuted() && !newOrder.IsIOC() && !newOrder.IsFOK() {
		// Insert the order
		if ob.orders.Set(newOrder.id, newOrder); false {

			// Call the corresponding handler
			e.handler.OnDeleteOrder(ob, newOrder)

			// Release the order
			e.allocator.PutOrder(newOrder)

			return ErrOrderDuplicate
		}

		// Add the new limit order into the order book
		priceLevelUpdate, err := ob.addOrder(ob.treeForOrder(newOrder), newOrder)
		if err != nil {
			return err
		}
		e.updatePriceLevel(ob, priceLevelUpdate)
	} else {
		// Call the corresponding handler
		e.handler.OnDeleteOrder(ob, newOrder)
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
	newOrder := order

	// Call the corresponding handler
	e.handler.OnAddOrder(ob, &newOrder)

	// Automatic order matching
	if e.matching && !recursive {
		e.matchMarketOrder(ob, &newOrder)
	}

	// Call the corresponding handler
	e.handler.OnDeleteOrder(ob, &newOrder)

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
	// Create a new order
	newOrder := e.allocator.GetOrder()
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

	// Call the corresponding handler
	e.handler.OnAddOrder(ob, newOrder)

	// Automatic order matching
	if e.matching && !recursive {
		// Check the market price
		arbitrage := newOrder.stopPrice.Equals(marketPrice)
		if arbitrage {

			// Convert the stop order into the market order
			newOrder.orderType = OrderTypeMarket
			newOrder.price = NewZeroUint()
			newOrder.stopPrice = NewZeroUint()
			if newOrder.IsFOK() {
				newOrder.timeInForce = OrderTimeInForceFOK
			} else {
				newOrder.timeInForce = OrderTimeInForceIOC
			}

			// Call the corresponding handler
			e.handler.OnUpdateOrder(ob, newOrder)

			// Match the market order
			e.matchMarketOrder(ob, newOrder)

			// Call the corresponding handler
			e.handler.OnDeleteOrder(ob, newOrder)

			// Automatic order matching
			if e.matching && !recursive {
				err := e.match(ob)
				if err != nil {
					return fmt.Errorf("failed to match: %w", err)
				}
			}

			return nil
		}
	}

	// Add a new order or delete remaining part in case of 'Immediate-Or-Cancel'/'Fill-Or-Kill' order
	if !newOrder.IsExecuted() && !newOrder.IsIOC() && !newOrder.IsFOK() {

		// Insert the order
		if ob.orders.Set(newOrder.id, newOrder); false {

			// Call the corresponding handler
			e.handler.OnDeleteOrder(ob, newOrder)

			// Release the order
			e.allocator.PutOrder(newOrder)

			return ErrOrderDuplicate
		}

		// Add the new limit order into the order book
		_, err := ob.addOrder(ob.treeForOrder(newOrder), newOrder)
		if err != nil {
			return err
		}

	} else {
		// Call the corresponding handler
		e.handler.OnDeleteOrder(ob, newOrder)
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

func (e *Engine) addStopLimitOrder(ob *OrderBook, order Order, recursive bool) error {
	// Create a new order
	newOrder := e.allocator.GetOrder()
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

	// Automatic order matching
	if e.matching && !recursive {

		// Check the market price
		arbitrage := newOrder.stopPrice.Equals(engineStopPrice)
		if arbitrage {
			// Convert the stop-limit order into the limit order
			newOrder.orderType = OrderTypeLimit
			newOrder.stopPrice = NewZeroUint()

			// Call the corresponding handler
			e.handler.OnUpdateOrder(ob, newOrder)

			// Match the limit order
			err := e.matchLimitOrder(ob, newOrder)
			if err != nil {
				return fmt.Errorf("failed to match limit order: %w", err)
			}

			// Add a new limit order or delete remaining part in case of 'Immediate-Or-Cancel'/'Fill-Or-Kill' order
			if !newOrder.IsExecuted() && !newOrder.IsIOC() && !newOrder.IsFOK() {

				// Insert the order
				if ob.orders.Set(newOrder.id, newOrder); false {

					// Call the corresponding handler
					e.handler.OnDeleteOrder(ob, newOrder)

					// Release the order
					e.allocator.PutOrder(newOrder)

					return ErrOrderDuplicate
				}

				// Add the new limit order into the order book
				priceLevelUpdate, err := ob.addOrder(ob.treeForOrder(newOrder), newOrder)
				if err != nil {
					return err
				}
				e.updatePriceLevel(ob, priceLevelUpdate)
			} else {
				// Call the corresponding handler
				e.handler.OnDeleteOrder(ob, newOrder)
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
	}

	// Add a new order
	if !newOrder.IsExecuted() {

		// Insert the order
		if ob.orders.Set(newOrder.id, newOrder); false {

			// Call the corresponding handler
			e.handler.OnDeleteOrder(ob, newOrder)

			// Release the order
			e.allocator.PutOrder(newOrder)

			return ErrOrderDuplicate
		}

		// Add the new limit order into the order book
		_, err := ob.addOrder(ob.treeForOrder(newOrder), newOrder)
		if err != nil {
			return err
		}

	} else {
		// Call the corresponding handler
		e.handler.OnDeleteOrder(ob, newOrder)
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
// Reducing orders
////////////////////////////////////////////////////////////////

func (e *Engine) reduceOrder(ob *OrderBook, order *Order, quantity Uint, inOB, recursive bool) error {
	// Calculate the minimal possible order quantity to reduce
	quantity = Min(quantity, order.restQuantity)

	// Reduce the order rest quantity
	visible := order.VisibleQuantity()
	order.restQuantity = order.restQuantity.Sub(quantity)
	visible = visible.Sub(order.VisibleQuantity())

	// Call the corresponding handler
	if !order.IsExecuted() {
		e.handler.OnUpdateOrder(ob, order)

	} else {
		e.handler.OnDeleteOrder(ob, order)
	}

	if inOB {
		// Reduce the order in the order book
		priceLevelUpdate, err := ob.reduceOrder(ob.treeForOrder(order), order, quantity, visible)
		if err != nil {
			return err
		}
		if order.IsLimit() {
			e.updatePriceLevel(ob, priceLevelUpdate)
		}
	}

	// Delete the empty order
	if order.IsExecuted() {

		// Erase the order
		ob.orders.Delete(order.id)

		// Release the order
		e.allocator.PutOrder(order)
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
		e.updatePriceLevel(ob, priceLevelUpdate)
	}

	// Modify the order
	order.price = newPrice
	order.quantity = newQuantity
	order.restQuantity = newQuantity
	order.available = order.available.Add(additionalAmountToLock)

	// In-Flight Mitigation (IFM)
	if mitigate {
		// This calculation has the goal of preventing orders from being overfilled
		if newQuantity.GreaterThan(order.executedQuantity) {
			order.restQuantity = newQuantity.Sub(order.executedQuantity)
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
				e.updatePriceLevel(ob, priceLevelUpdate)
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
		e.allocator.PutOrder(order)
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
		e.updatePriceLevel(ob, priceLevelUpdate)
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
		if ob.orders.Set(order.id, order); false {

			// Call the corresponding handler
			e.handler.OnDeleteOrder(ob, order)

			// Release the order
			e.allocator.PutOrder(order)

			return ErrOrderDuplicate
		}

		// Add the modified order into the order book
		priceLevelUpdate, err := ob.addOrder(ob.treeForOrder(order), order)
		if err != nil {
			return err
		}
		if order.IsLimit() {
			e.updatePriceLevel(ob, priceLevelUpdate)
		}

	} else {

		// Call the corresponding handler
		e.handler.OnDeleteOrder(ob, order)

		// Release the order
		e.allocator.PutOrder(order)
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

func (e *Engine) deleteOrder(ob *OrderBook, order *Order, inOB bool, recursive bool) error {
	// Delete the order from the order book
	if inOB {
		priceLevelUpdate, err := ob.deleteOrder(ob.treeForOrder(order), order)
		if err != nil {
			return err
		}

		if order.IsLimit() {
			e.updatePriceLevel(ob, priceLevelUpdate)
		}
	}

	// Call the corresponding handler
	e.handler.OnDeleteOrder(ob, order)

	// Erase the order
	ob.orders.Delete(order.id)

	// Release the order
	e.allocator.PutOrder(order)

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
func (e *Engine) deleteLinkedOrder(ob *OrderBook, order *Order, inOB, recursive bool) error {
	if order.linkedOrderID == 0 {
		return nil
	}

	linkedOrder := ob.Order(order.linkedOrderID)
	if linkedOrder != nil {
		return e.deleteOrder(ob, linkedOrder, inOB, recursive)
	}

	return nil
}
