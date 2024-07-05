package matching

import (
	"fmt"

	"github.com/cryptonstudio/crypton-matching-engine/types/avl"
)

////////////////////////////////////////////////////////////////
// Matching order books
////////////////////////////////////////////////////////////////

// match matches crossed orders in given order book.
func (e *Engine) match(ob *OrderBook) error {
	// Matching loop
	for {
		for {
			// Find the best bid/ask price level
			topBid, topAsk := ob.TopBid(), ob.TopAsk()

			// Continue only if there are crossed orders
			if topBid == nil || topAsk == nil || topBid.Value().Price().LessThan(topAsk.Value().Price()) {
				break
			}

			// Find the first order to execute and the first order to reduce
			orderBid := topBid.Value().queue.Front()
			orderAsk := topAsk.Value().queue.Front()

			// Execute crossed orders
			for orderBid != nil && orderAsk != nil && orderBid.Value != nil && orderAsk.Value != nil {
				// Find the best order to execute and the best order to reduce
				executing, reducing, swapped := orderBid.Value, orderAsk.Value, false

				// Check quantity
				if executing.restQuantity.GreaterThan(reducing.restQuantity) {
					executing, reducing, swapped = reducing, executing, true // swap
				}

				// Need to define price based on maker order,
				// define maker as order that has come earlier,
				// calculate price and call handler based on this.
				price := reducing.price
				if executing.id < reducing.id {
					price = executing.price
				}

				// Get the execution quantities
				quantity, quoteQuantity := executing.restQuantity, executing.restQuantity.Mul(price).Div64(UintPrecision)

				// Call handlers
				e.handler.OnExecuteOrder(ob, executing, price, quantity, quoteQuantity)
				e.handler.OnExecuteOrder(ob, reducing, price, quantity, quoteQuantity)
				if executing.id < reducing.id {
					e.handler.OnExecuteTrade(ob, executing, reducing, price, quantity, quoteQuantity)
				} else {
					e.handler.OnExecuteTrade(ob, reducing, executing, price, quantity, quoteQuantity)
				}

				// Execute orders
				err := e.executeOrder(ob, reducing, quantity, quoteQuantity)
				if err != nil {
					return fmt.Errorf("failed to execute order (id: %d): %w", reducing.ID(), err)
				}
				err = e.executeOrder(ob, executing, quantity, quoteQuantity)
				if err != nil {
					return fmt.Errorf("failed to execute order (id: %d): %w", executing.ID(), err)
				}

				// Update common market price
				ob.updateMarketPrice(price)

				// Next executing order
				if !swapped {
					orderBid = orderBid.Next()
				} else {
					orderAsk = orderAsk.Next()
				}

				// If reducing is executed too
				if reducing.IsExecuted() {
					if !swapped {
						orderAsk = orderAsk.Next()
					} else {
						orderBid = orderBid.Next()
					}
				}
			}

			// Activate stop orders only if the current price level changed
			for _, mode := range ob.spModes {
				_, err := e.activateStopOrders(ob, OrderSideBuy, ob.TopBid(), mode)
				if err != nil {
					return fmt.Errorf("failed to activate buy stop orders: %w", err)
				}

				_, err = e.activateStopOrders(ob, OrderSideSell, ob.TopAsk(), mode)
				if err != nil {
					return fmt.Errorf("failed to activate sell stop orders: %w", err)
				}
			}
		}

		activated, err := e.activateAllStopOrders(ob)
		if err != nil {
			return fmt.Errorf("failed to activate all stop orders: %w", err)
		}

		// Activate stop orders until there is something to activate
		if !activated {
			break
		}
	}

	return nil
}

////////////////////////////////////////////////////////////////
// Matching orders
////////////////////////////////////////////////////////////////

// matchLimitOrder matches given limit order in given order book.
func (e *Engine) matchLimitOrder(ob *OrderBook, order *Order) error {
	// Match the limit order
	err := e.matchOrder(ob, order)
	if err != nil {
		return fmt.Errorf("failed to match order: %w", err)
	}

	return nil
}

// matchMarketOrder matches given market order in given order book.
func (e *Engine) matchMarketOrder(ob *OrderBook, order *Order) error {
	var topPrice Uint
	if order.IsBuy() {
		// Check if there is nothing to buy
		if ob.TopAsk() == nil {
			return nil
		}

		topPrice = ob.TopAsk().Value().Price()
		maxPrice := ob.symbol.priceLimits.Max
		if topPrice.GreaterThan(maxPrice.Sub(order.marketSlippage)) {
			order.price = maxPrice
		} else {
			order.price = topPrice.Add(order.marketSlippage)
		}
	} else {
		// Check if there is nothing to sell
		if ob.TopBid() == nil {
			return nil
		}

		topPrice = ob.TopBid().Value().Price()
		if topPrice.LessThan(order.marketSlippage) {
			order.price = NewZeroUint()
		} else {
			order.price = topPrice.Sub(order.marketSlippage)
		}
	}

	// Match the market order
	err := e.matchOrder(ob, order)
	if err != nil {
		return fmt.Errorf("failed to match order: %w", err)
	}

	return nil
}

// matchOrder matches given order in given order book.
func (e *Engine) matchOrder(ob *OrderBook, order *Order) error {
	// Start the matching from the top of the book
	for {
		// Determine the best bid/ask price level
		var priceLevel *avl.Node[Uint, *PriceLevelL3]
		if order.IsBuy() {
			priceLevel = ob.TopAsk()
		} else {
			priceLevel = ob.TopBid()
		}
		if priceLevel == nil {
			return nil
		}

		// Check the arbitrage bid/ask prices
		if order.IsBuy() {
			if order.price.LessThan(priceLevel.Value().Price()) {
				return nil
			}
		} else {
			if order.price.GreaterThan(priceLevel.Value().Price()) {
				return nil
			}
		}

		// Special case for 'Fill-Or-Kill'
		if order.IsFOK() && !e.canExecuteChain(order, priceLevel) {
			return nil
		}

		if order.IsExecuted() {
			return nil
		}

		// Execute crossed orders
		for orderPtr := priceLevel.Value().queue.Front(); orderPtr != nil; {
			orderPtrNext := orderPtr.Next()
			executingOrder := orderPtr.Value

			// Get the execution price and quantity of crossed order, executing is maker
			price, execQty := executingOrder.price, executingOrder.restQuantity
			qty, quoteQty := calcExecutingForTaker(order, price)
			// Check if can't be matched at all (market with not enough available)
			if qty.IsZero() {
				return nil
			}

			// Choose less qty as qty for trade
			if execQty.LessThan(qty) {
				qty = execQty
				quoteQty = execQty.Mul(price).Div64(UintPrecision)
			}

			// Call the trade handlers
			e.handler.OnExecuteOrder(ob, executingOrder, price, qty, quoteQty)
			e.handler.OnExecuteOrder(ob, order, price, qty, quoteQty)
			e.handler.OnExecuteTrade(ob, executingOrder, order, price, qty, quoteQty)

			// Execute orders
			err := e.executeOrder(ob, order, qty, quoteQty)
			if err != nil {
				return fmt.Errorf("failed to execute order (id: %d): %w", order.ID(), err)
			}
			err = e.executeOrder(ob, executingOrder, qty, quoteQty)
			if err != nil {
				return fmt.Errorf("failed to execute order (id: %d): %w", executingOrder.ID(), err)
			}

			// Update common market price
			ob.updateMarketPrice(price)

			// Exit the loop if the order is executed
			if order.IsExecuted() {
				return nil
			}

			// Move to the next order to execute at the same price level
			orderPtr = orderPtrNext
		}
	}
}

////////////////////////////////////////////////////////////////
// Matching chains
////////////////////////////////////////////////////////////////

// canExecuteChain have to be used for FOK orders to check if full execution is possible
// here we can only deal with limit type.
func (e *Engine) canExecuteChain(
	reducingOrder *Order,
	executing *avl.Node[Uint, *PriceLevelL3],
) bool {
	required := reducingOrder.quantity

	// Travel through price levels
	for executing != nil {
		// Take first order of price level
		executingOrder := executing.Value().queue.Front()

		// Travel through orders at current price levels
		for executingOrder != nil && executingOrder.Value != nil {
			quantity := executingOrder.Value.restQuantity

			if required.LessThanOrEqualTo(quantity) {
				return true
			}

			required = required.Sub(quantity)

			// Take the next order
			executingOrder = executingOrder.Next()
		}

		executing = executing.NextRight()
	}

	// Matching is not available
	return false
}
