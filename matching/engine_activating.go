package matching

import (
	"fmt"

	"github.com/cryptonstudio/crypton-matching-engine/types/avl"
)

////////////////////////////////////////////////////////////////
// Activating stop orders
////////////////////////////////////////////////////////////////

func (e *Engine) activateAllStopOrders(ob *OrderBook) (bool, error) {
	activated := false

	for _, mode := range ob.spModes {
		// Try to activate buy stop orders
		buyActivated, err := e.activateStopOrders(ob, OrderSideBuy, ob.TopBuyStop(), mode)
		if err != nil {
			return false, fmt.Errorf("failed to activate buy stop orders: %w", err)
		}

		buyTrailingActivated, err := e.activateStopOrders(ob, OrderSideBuy, ob.TopTrailingBuyStop(), mode)
		if err != nil {
			return false, fmt.Errorf("failed to activate trailing buy stop orders: %w", err)
		}

		// Recalculate trailing buy stop orders
		e.recalculateTrailingStopPrice(ob, OrderSideSell, ob.TopAsk())

		// Try to activate sell stop orders
		sellActivated, err := e.activateStopOrders(ob, OrderSideSell, ob.TopSellStop(), mode)
		if err != nil {
			return false, fmt.Errorf("failed to activate sell stop orders: %w", err)
		}

		sellTrailingActivated, err := e.activateStopOrders(ob, OrderSideSell, ob.TopTrailingSellStop(), mode)
		if err != nil {
			return false, fmt.Errorf("failed to activate trailing sell stop orders: %w", err)
		}

		// Recalculate trailing sell stop orders
		e.recalculateTrailingStopPrice(ob, OrderSideBuy, ob.TopBid())

		activated = activated || buyActivated || buyTrailingActivated || sellActivated || sellTrailingActivated
	}

	return activated, nil
}

func (e *Engine) activateStopOrders(ob *OrderBook, side OrderSide, node *avl.Node[Uint, *PriceLevelL3], stopPriceMode StopPriceMode) (activated bool, err error) {
	activationPrice := ob.GetStopPrice(stopPriceMode)

	// Return if given price level node is nil
	if node == nil {
		return
	}
	priceLevel := node.Value()

	// Activate all stop orders
	for orderPtr := priceLevel.queue.Front(); orderPtr != nil && orderPtr.Value != nil; orderPtr = orderPtr.Next() {
		if orderPtr.Value.stopPriceMode != stopPriceMode {
			continue
		}

		// Check the arbitrage bid/ask prices
		var arbitrage bool

		stopPrice := orderPtr.Value.stopPrice
		if orderPtr.Value.takeProfit {
			if side == OrderSideBuy {
				arbitrage = stopPrice.GreaterThanOrEqualTo(activationPrice)
			} else {
				arbitrage = stopPrice.LessThanOrEqualTo(activationPrice)
			}
		} else {
			if side == OrderSideBuy {
				arbitrage = stopPrice.LessThanOrEqualTo(activationPrice)
			} else {
				arbitrage = stopPrice.GreaterThanOrEqualTo(activationPrice)
			}
		}

		if !arbitrage {
			continue
		}

		switch orderPtr.Value.orderType {
		case OrderTypeStop, OrderTypeTrailingStop:
			// Activate the stop order
			activated, err = e.activateStopOrder(ob, orderPtr.Value)
			if err != nil {
				return false, fmt.Errorf("failed to activate stop order: %w", err)
			}
		case OrderTypeStopLimit, OrderTypeTrailingStopLimit:
			// Activate the stop-limit order
			activated, err = e.activateStopLimitOrder(ob, orderPtr.Value)
			if err != nil {
				return false, fmt.Errorf("failed to activate stop-limit order: %w", err)
			}
		}
	}

	return
}

func (e *Engine) activateStopOrder(ob *OrderBook, order *Order) (bool, error) {
	// Delete the stop order from the order book
	_, err := ob.deleteOrder(ob.treeForOrder(order), order)
	if err != nil {
		return false, fmt.Errorf("failed to delete order: %w", err)
	}

	// Convert the stop order into the market order
	order.orderType = OrderTypeMarket
	order.price = NewZeroUint()
	order.stopPrice = NewZeroUint()
	order.timeInForce = OrderTimeInForceIOC

	// Call the corresponding handler
	e.handler.OnUpdateOrder(ob, order)

	// Match the market order
	e.matchMarketOrder(ob, order)

	if !order.IsExecuted() {
		// Call the corresponding handler
		e.handler.OnDeleteOrder(ob, order)

		// Erase the order
		ob.orders.Delete(order.id)

		// Release the order
		e.allocator.PutOrder(order)
	}

	return true, nil
}

func (e *Engine) activateStopLimitOrder(ob *OrderBook, order *Order) (bool, error) {
	// Check and delete linked orders (OCO)
	err := e.deleteLinkedOrder(ob, order, false)
	if err != nil {
		return false, fmt.Errorf("failed to delete linked order: %w", err)
	}

	// Delete the stop-limit order from the order book
	_, err = ob.deleteOrder(ob.treeForOrder(order), order)
	if err != nil {
		return false, fmt.Errorf("failed to delete order: %w", err)
	}

	// Convert the stop-limit order into the limit order
	order.orderType = OrderTypeLimit
	order.stopPrice = NewZeroUint()

	// Call the corresponding handler
	e.handler.OnUpdateOrder(ob, order)

	// Match the limit order
	err = e.matchLimitOrder(ob, order)
	if err != nil {
		return false, fmt.Errorf("failed to match limit order: %w", err)
	}

	// Delete remaining part in case of 'Immediate-Or-Cancel'/'Fill-Or-Kill' and exit.
	// If executed, handler has been already called.
	if (order.IsIOC() || order.IsFOK()) && !order.IsExecuted() {
		// Call the corresponding handler
		e.handler.OnDeleteOrder(ob, order)

		// Erase the order
		ob.orders.Delete(order.id)

		// Release the order
		e.allocator.PutOrder(order)
	}

	// Add remaining order in order book for GTC
	if order.IsGTC() && !order.IsExecuted() {
		// Add the new limit order into the order book
		priceLevelUpdate, err := ob.addOrder(ob.treeForOrder(order), order)
		if err != nil {
			return false, err
		}
		e.updatePriceLevel(ob, priceLevelUpdate)
	}

	return true, nil
}

////////////////////////////////////////////////////////////////
// Recalculating trailing stop price
////////////////////////////////////////////////////////////////

func (e *Engine) recalculateTrailingStopPrice(ob *OrderBook, side OrderSide, node *avl.Node[Uint, *PriceLevelL3]) {
	if node == nil {
		return
	}

	var newTrailingPrice Uint

	// Check if we should skip the recalculation because of the market price goes to the wrong direction
	switch side {
	case OrderSideSell:
		oldTrailingPrice := ob.trailingAskPrice
		newTrailingPrice = ob.GetMarketTrailingStopPriceAsk()
		ob.trailingAskPrice = newTrailingPrice
		if newTrailingPrice.GreaterThanOrEqualTo(oldTrailingPrice) {
			return
		}
	case OrderSideBuy:
		oldTrailingPrice := ob.trailingBidPrice
		newTrailingPrice = ob.GetMarketTrailingStopPriceBid()
		ob.trailingBidPrice = newTrailingPrice
		if newTrailingPrice.LessThanOrEqualTo(oldTrailingPrice) {
			return
		}
	}

	// Recalculate trailing stop orders
	var previous *avl.Node[Uint, *PriceLevelL3]
	var current *avl.Node[Uint, *PriceLevelL3]
	if side == OrderSideBuy {
		current = ob.TopTrailingSellStop()
	} else {
		current = ob.TopTrailingBuyStop()
	}
	for current != nil {
		currentNext := current.NextRight()
		recalculated := false

		// Travel through orders at current price levels
		for orderPtr := current.Value().queue.Front(); orderPtr != nil; orderPtr = orderPtr.Next() {
			order := orderPtr.Value
			oldStopPrice := order.stopPrice
			newStopPrice := ob.calculateTrailingStopPrice(order)

			// Trailing distance for the order must be changed
			if !newStopPrice.Equals(oldStopPrice) {
				// Delete the order from the order book
				var tree *avl.Tree[Uint, *PriceLevelL3]
				if side == OrderSideBuy {
					tree = &ob.trailingBuyStop
				} else {
					tree = &ob.trailingSellStop
				}
				_, err := ob.deleteOrder(tree, order)
				if err != nil {
					return
				}

				// Update the stop order price
				switch order.orderType {
				case OrderTypeTrailingStop:
					order.stopPrice = newStopPrice
				case OrderTypeTrailingStopLimit:
					if order.price.GreaterThanOrEqualTo(order.stopPrice) {
						// diff >= 0
						diff := order.price.Sub(order.stopPrice)
						order.stopPrice = newStopPrice
						order.price = order.stopPrice.Add(diff)
					} else {
						// diff < 0
						diff := order.stopPrice.Sub(order.price)
						order.stopPrice = newStopPrice
						order.price = order.stopPrice.Sub(diff)
					}
				}

				// Call the corresponding handler
				e.handler.OnUpdateOrder(ob, order)

				// Add the new stop order into the order book
				_, err = ob.addOrder(tree, order)
				if err != nil {
					return
				}

				recalculated = true
			}
		}

		if recalculated {
			// Back to the previous stop price level
			if previous != nil {
				current = previous
			} else if side == OrderSideBuy {
				current = ob.TopTrailingSellStop()
			} else {
				current = ob.TopTrailingBuyStop()
			}
		} else {
			// Move to the next stop price level
			previous = current
			current = currentNext
		}
	}
}

func (ob *OrderBook) calculateTrailingStopPrice(order *Order) Uint {
	// Get the current market price
	var marketPrice Uint
	if order.IsBuy() {
		marketPrice = ob.GetMarketTrailingStopPriceAsk()
	} else {
		marketPrice = ob.GetMarketTrailingStopPriceBid()
	}
	trailingDistance := order.trailingDistance
	trailingStep := order.trailingStep

	// Convert percentage trailing values into absolute ones
	if trailingDistance.LessThanOrEqualTo(NewUint(10000)) {
		trailingDistance = trailingDistance.Mul(marketPrice).Div64(10000)
		trailingStep = trailingStep.Mul(marketPrice).Div64(10000)
	}

	oldPrice := order.stopPrice

	if order.IsBuy() {
		// Calculate a new stop price
		newPrice := NewMaxUint()
		if marketPrice.LessThan(NewMaxUint().Sub(trailingDistance)) {
			newPrice = marketPrice.Add(trailingDistance)
		}

		// If the new price is better and we get through the trailing step
		if newPrice.LessThan(oldPrice) {
			if oldPrice.Sub(newPrice).GreaterThanOrEqualTo(trailingStep) {
				return newPrice
			}
		}
	} else {
		// Calculate a new stop price
		newPrice := NewZeroUint()
		if marketPrice.GreaterThan(trailingDistance) {
			newPrice = marketPrice.Sub(trailingDistance)
		}

		// If the new price is better and we get through the trailing step
		if newPrice.GreaterThan(oldPrice) {
			if newPrice.Sub(oldPrice).GreaterThanOrEqualTo(trailingStep) {
				return newPrice
			}
		}
	}

	return oldPrice
}
