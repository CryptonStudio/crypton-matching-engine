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
		// Check the arbitrage bid/ask prices
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
				executingOrder, reducingOrder, swapped := orderBid.Value, orderAsk.Value, false
				if executingOrder.restQuantity.GreaterThan(reducingOrder.restQuantity) {
					executingOrder, reducingOrder, swapped = reducingOrder, executingOrder, true // swap
				}

				// Get the execution price and quantity
				var price Uint
				if executingOrder.id < reducingOrder.id {
					price = executingOrder.price
				} else {
					price = reducingOrder.price
				}
				quantity, quoteQuantity := executingOrder.RestAvailableQuantities(price, ob.symbol.lotSizeLimits.Step)

				// Ensure both orders are not executed already
				if orderBid.Value.RestAvailableQuantity(price, ob.symbol.lotSizeLimits.Step).IsZero() {
					order := orderBid.Value
					orderBid = orderBid.Next()
					err := e.deleteOrder(ob, order, true, true)
					if err != nil {
						return fmt.Errorf("failed to delete order (id: %d): %w", order.ID(), err)
					}

					continue
				}
				if orderAsk.Value.RestAvailableQuantity(price, ob.symbol.lotSizeLimits.Step).IsZero() {
					order := orderAsk.Value
					orderAsk = orderAsk.Next()
					err := e.deleteOrder(ob, order, true, true)
					if err != nil {
						return fmt.Errorf("failed to delete order (id: %d): %w", order.ID(), err)
					}

					continue
				}

				// Check and delete linked orders
				err := e.deleteLinkedOrder(ob, executingOrder, true, false)
				if err != nil {
					return fmt.Errorf("failed to delete linked order (id: %d): %w", executingOrder.ID(), err)
				}

				err = e.deleteLinkedOrder(ob, reducingOrder, true, false)
				if err != nil {
					return fmt.Errorf("failed to delete linked order (id: %d): %w", reducingOrder.ID(), err)
				}

				// Call the corresponding handlers
				e.handler.OnExecuteOrder(ob, executingOrder, price, quantity)
				e.handler.OnExecuteOrder(ob, reducingOrder, price, quantity)

				if executingOrder.id < reducingOrder.id {
					e.handler.OnExecuteTrade(ob, executingOrder, reducingOrder, price, quantity)
				} else {
					e.handler.OnExecuteTrade(ob, reducingOrder, executingOrder, price, quantity)
				}

				// Update common market price
				ob.updateMarketPrice(price)

				// Decrease the order available quantity
				if executingOrder.IsBuy() {
					executingOrder.available = executingOrder.available.Sub(quoteQuantity)
				} else {
					executingOrder.available = executingOrder.available.Sub(quantity)
				}

				// Increase the order executed quantity
				executingOrder.executedQuantity = executingOrder.executedQuantity.Add(quantity)
				executingOrder.executedQuoteQuantity = executingOrder.executedQuoteQuantity.Add(quoteQuantity)

				// Move to the next orders pair at the same price level
				if !swapped {
					orderBid = orderBid.Next()
				} else {
					orderAsk = orderAsk.Next()
				}

				// Delete the executing order from the order book
				err = e.deleteOrder(ob, executingOrder, true, true)
				if err != nil {
					return fmt.Errorf("failed to delete order (id: %d): %w", executingOrder.ID(), err)
				}

				// Decrease the order available quantity
				if reducingOrder.IsBuy() {
					reducingOrder.available = reducingOrder.available.Sub(quoteQuantity)
				} else {
					reducingOrder.available = reducingOrder.available.Sub(quantity)
				}

				// Increase the order executed quantity
				reducingOrder.executedQuantity = reducingOrder.executedQuantity.Add(quantity)
				reducingOrder.executedQuoteQuantity = reducingOrder.executedQuoteQuantity.Add(quoteQuantity)

				// Move to the next orders pair at the same price level
				if reducingOrder.RestAvailableQuantity(price, ob.symbol.lotSizeLimits.Step).Equals(quantity) {
					if !swapped {
						orderAsk = orderAsk.Next()
					} else {
						orderBid = orderBid.Next()
					}
				}

				// Reduce the remaining order in the order book
				err = e.reduceOrder(ob, reducingOrder, quantity, true, true)
				if err != nil {
					return fmt.Errorf("failed to reduce order (id: %d): %w", reducingOrder.ID(), err)
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

	// Calculate acceptable maker order price with optional slippage value
	var topPrice Uint
	if order.IsBuy() {
		// Check if there is nothing to buy
		if ob.TopAsk() == nil {
			return nil
		}

		topPrice = ob.TopAsk().Value().Price()
		if topPrice.GreaterThan(NewMaxUint().Sub(order.marketSlippage)) {
			order.price = NewMaxUint()
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

	// Fill rest quantity with correct value
	if !order.quoteQuantity.IsZero() {
		order.restQuantity, _ = order.quoteQuantity.Mul64(UintPrecision).QuoRem(topPrice)
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
		if order.IsFOK() {
			canExecuteTakerChain := e.canExecuteChain(order.quantity, priceLevel)

			if canExecuteTakerChain {
				err := e.executeTakerChain(ob, order, priceLevel)
				if err != nil {
					return fmt.Errorf("failed to execute taker chain: %w", err)
				}
			}

			return nil
		}

		// Execute crossed orders
		for orderPtr := priceLevel.Value().queue.Front(); !order.IsExecuted() && orderPtr != nil; {
			orderPtrNext := orderPtr.Next()
			executingOrder := orderPtr.Value

			// Get the execution price and quantity
			price := executingOrder.price
			executingQuantity := executingOrder.RestAvailableQuantity(price, ob.symbol.lotSizeLimits.Step)
			orderQuantity := order.RestAvailableQuantity(price, ob.symbol.lotSizeLimits.Step)
			quantity := Min(executingQuantity, orderQuantity)
			quoteQuantity := quantity.Mul(price).Div64(UintPrecision)

			// Ensure both orders are not executed already
			// TODO: Investigate the reason of occurring such cases!
			if executingQuantity.IsZero() {
				err := e.deleteOrder(ob, executingOrder, true, true)
				if err != nil {
					return fmt.Errorf("failed to delete order (id: %d): %w", executingOrder.ID(), err)
				}

				orderPtr = orderPtrNext
				continue
			}
			if orderQuantity.IsZero() {
				return nil
			}

			// Check and delete linked orders
			err := e.deleteLinkedOrder(ob, executingOrder, true, false)
			if err != nil {
				return fmt.Errorf("failed to delete linked order (id: %d): %w", executingOrder.ID(), err)
			}

			err = e.deleteLinkedOrder(ob, order, true, false)
			if err != nil {
				return fmt.Errorf("failed to delete linked order (id: %d): %w", order.ID(), err)
			}

			// Call the corresponding handlers
			e.handler.OnExecuteOrder(ob, executingOrder, price, quantity)
			e.handler.OnExecuteOrder(ob, order, price, quantity)
			if executingOrder.id < order.id {
				e.handler.OnExecuteTrade(ob, executingOrder, order, price, quantity)
			} else {
				e.handler.OnExecuteTrade(ob, order, executingOrder, price, quantity)
			}

			// Update common market price
			ob.updateMarketPrice(price)

			// Decrease the order available quantity
			if executingOrder.IsBuy() {
				executingOrder.available = executingOrder.available.Sub(quoteQuantity)
			} else {
				executingOrder.available = executingOrder.available.Sub(quantity)
			}

			// Increase the order executed quantity
			executingOrder.executedQuantity = executingOrder.executedQuantity.Add(quantity)
			executingOrder.executedQuoteQuantity = executingOrder.executedQuoteQuantity.Add(quoteQuantity)

			// Reduce the executing order in the order book
			err = e.reduceOrder(ob, executingOrder, quantity, true, true)
			if err != nil {
				return fmt.Errorf("failed to reduce order (id: %d): %w", executingOrder.ID(), err)
			}

			// Decrease the order available quantity
			if order.IsBuy() {
				order.available = order.available.Sub(quoteQuantity)
			} else {
				order.available = order.available.Sub(quantity)
			}

			// Increase the order executed quantity
			order.executedQuantity = order.executedQuantity.Add(quantity)
			order.executedQuoteQuantity = order.executedQuoteQuantity.Add(quoteQuantity)

			// Reduce the order leaves quantity
			order.restQuantity = orderQuantity.Sub(quantity)

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
func (e *Engine) canExecuteChain(
	required Uint,
	executing *avl.Node[Uint, *PriceLevelL3],
) bool {
	// Travel through price levels
	for executing != nil {
		// Take first order of price level
		executingOrder := executing.Value().queue.Front()

		// Travel through orders at current price levels
		for executingOrder != nil {
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

// executeTakerChain have to be used for FOK orders to be fully executed
func (e *Engine) executeTakerChain(
	ob *OrderBook,
	reducingOrder *Order,
	node *avl.Node[Uint, *PriceLevelL3],
) error {
	price := reducingOrder.price
	volume := reducingOrder.quantity
	// We can update market price one time because all trades will be at the same price
	ob.updateMarketPrice(price)

	// Execute all orders in the matching chain
	for node != nil {
		nodeNext := node.NextRight()

		// Execute all orders in the current price level
		for orderPtr := node.Value().queue.Front(); orderPtr != nil; {
			orderPtrNext := orderPtr.Next()
			order := orderPtr.Value

			quantity, quoteQuantity := order.RestAvailableQuantities(price, ob.symbol.lotSizeLimits.Step)

			if quantity.GreaterThanOrEqualTo(volume) {
				// calc quote volume
				quoteVolume := volume.Mul(price).Div64(UintPrecision)

				// executing order only reduced
				order.executedQuantity = order.executedQuantity.Add(volume)
				order.executedQuoteQuantity = order.executedQuoteQuantity.Add(quoteVolume)
				e.handler.OnExecuteOrder(ob, order, price, volume)
				err := e.reduceOrder(ob, order, volume, true, true)
				if err != nil {
					return fmt.Errorf("failed to reduce order (id: %d): %w", order.ID(), err)
				}

				// reducing order (FOK) is now fully executed
				reducingOrder.executedQuantity = reducingOrder.executedQuantity.Add(volume)
				reducingOrder.executedQuoteQuantity = reducingOrder.executedQuoteQuantity.Add(quoteVolume)
				e.handler.OnExecuteOrder(ob, reducingOrder, price, volume)
				err = e.deleteOrder(ob, reducingOrder, false, true)
				if err != nil {
					return fmt.Errorf("failed to delete order (id: %d): %w", order.ID(), err)
				}

				// NOTE: FOK order is taker
				e.handler.OnExecuteTrade(ob, order, reducingOrder, price, volume)

				return nil
			}

			// Reduce the execution chain
			volume = volume.Sub(quantity)

			// fully executed executing order
			order.executedQuantity = order.executedQuantity.Add(quantity)
			order.executedQuoteQuantity = order.executedQuoteQuantity.Add(quoteQuantity)
			e.handler.OnExecuteOrder(ob, order, price, quantity)
			err := e.deleteOrder(ob, order, true, true)
			if err != nil {
				return fmt.Errorf("failed to delete order (id: %d): %w", order.ID(), err)
			}

			// reduce reducing order
			reducingOrder.executedQuantity = reducingOrder.executedQuantity.Add(quantity)
			reducingOrder.executedQuoteQuantity = reducingOrder.executedQuoteQuantity.Add(quantity)
			e.handler.OnExecuteOrder(ob, reducingOrder, price, quantity)
			err = e.reduceOrder(ob, reducingOrder, quantity, false, true)
			if err != nil {
				return fmt.Errorf("failed to reduce order (id: %d): %w", order.ID(), err)
			}

			// NOTE: FOK order is taker
			e.handler.OnExecuteTrade(ob, order, reducingOrder, price, quantity)

			// Move to the next order to execute at the same price level
			orderPtr = orderPtrNext
		}

		// Switch to the next price level
		node = nodeNext
	}

	return nil
}
