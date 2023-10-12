package matching

import (
	"github.com/cryptonstudio/crypton-matching-engine/types/avl"
)

////////////////////////////////////////////////////////////////
// Matching order books
////////////////////////////////////////////////////////////////

// match matches crossed orders in given order book.
func (e *Engine) match(ob *OrderBook) {
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

				// Special case for 'All-Or-None' orders
				if orderBid.Value.IsAON() || orderAsk.Value.IsAON() {
					// Calculate the matching chain
					chain := e.calculateMatchingChainCross(ob, topBid, topAsk)
					if chain.IsZero() {
						// Matching is not available
						return
					}

					// Execute orders in the matching chain
					if orderBid.Value.IsAON() {
						price := orderBid.Value.price
						e.executeMatchingChain(ob, &ob.bids, topBid, price, chain)
						e.executeMatchingChain(ob, &ob.asks, topAsk, price, chain)
					} else {
						price := orderAsk.Value.price
						e.executeMatchingChain(ob, &ob.asks, topAsk, price, chain)
						e.executeMatchingChain(ob, &ob.bids, topBid, price, chain)
					}

					break
				}

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
				quantity, quoteQuantity := executingOrder.RestAvailableQuantities(price)

				// Ensure both orders are not executed already
				// TODO: Investigate the reason of occurring such cases!
				if orderBid.Value.RestAvailableQuantity(price).IsZero() {
					order := orderBid.Value
					orderBid = orderBid.Next()
					e.deleteOrder(ob, order, true)
					continue
				}
				if orderAsk.Value.RestAvailableQuantity(price).IsZero() {
					order := orderAsk.Value
					orderAsk = orderAsk.Next()
					e.deleteOrder(ob, order, true)
					continue
				}

				// Check and delete linked orders
				e.deleteLinkedOrder(ob, executingOrder, false)
				e.deleteLinkedOrder(ob, reducingOrder, false)

				// Call the corresponding handlers
				e.handler.OnExecuteOrder(ob, executingOrder, price, quantity)
				e.handler.OnExecuteOrder(ob, reducingOrder, price, quantity)

				if executingOrder.id < reducingOrder.id {
					e.handler.OnExecuteTrade(ob, executingOrder, reducingOrder, price, quantity)
				} else {
					e.handler.OnExecuteTrade(ob, reducingOrder, executingOrder, price, quantity)
				}

				// Update the corresponding market price
				ob.updateLastPrice(executingOrder.side, price)
				ob.updateMatchingPrice(executingOrder.side, price)

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
				e.deleteOrder(ob, executingOrder, true)

				// Update the corresponding market price
				ob.updateLastPrice(reducingOrder.side, price)
				ob.updateMatchingPrice(reducingOrder.side, price)

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
				if reducingOrder.RestAvailableQuantity(price).Equals(quantity) {
					if !swapped {
						orderAsk = orderAsk.Next()
					} else {
						orderBid = orderBid.Next()
					}
				}

				// Reduce the remaining order in the order book
				e.reduceOrder(ob, reducingOrder, quantity, true)
			}

			// Activate stop orders only if the current price level changed
			e.activateStopOrders(ob, OrderSideBuy, ob.TopBid(), ob.GetMarketPriceAsk())
			e.activateStopOrders(ob, OrderSideSell, ob.TopAsk(), ob.GetMarketPriceBid())
		}

		// Activate stop orders until there is something to activate
		if !e.activateAllStopOrders(ob) {
			break
		}
	}
}

////////////////////////////////////////////////////////////////
// Matching orders
////////////////////////////////////////////////////////////////

// matchLimitOrder matches given limit order in given order book.
func (e *Engine) matchLimitOrder(ob *OrderBook, order *Order) {

	// Match the limit order
	e.matchOrder(ob, order)
}

// matchMarketOrder matches given market order in given order book.
func (e *Engine) matchMarketOrder(ob *OrderBook, order *Order) {

	// Calculate acceptable maker order price with optional slippage value
	var topPrice Uint
	if order.IsBuy() {
		// Check if there is nothing to buy
		if ob.TopAsk() == nil {
			return
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
			return
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
	e.matchOrder(ob, order)
}

// matchOrder matches given order in given order book.
func (e *Engine) matchOrder(ob *OrderBook, order *Order) {

	// Start the matching from the top of the book
	for {
		// Determine the best bid/ask price level
		var tree *avl.Tree[Uint, *PriceLevelL3]
		var priceLevel *avl.Node[Uint, *PriceLevelL3]
		if order.IsBuy() {
			tree = &ob.asks
			priceLevel = ob.TopAsk()
		} else {
			tree = &ob.bids
			priceLevel = ob.TopBid()
		}
		if priceLevel == nil {
			return
		}

		// Check the arbitrage bid/ask prices
		if order.IsBuy() {
			if order.price.LessThan(priceLevel.Value().Price()) {
				return
			}
		} else {
			if order.price.GreaterThan(priceLevel.Value().Price()) {
				return
			}
		}

		// Special case for 'Fill-Or-Kill' and 'All-Or-None' orders
		if order.IsFOK() || order.IsAON() {

			// Calculate the matching chain
			chain := e.calculateMatchingChain(ob, order, priceLevel)
			if chain.IsZero() {
				// Matching is not available
				return
			}

			// Execute orders in the matching chain
			// TODO: Re-implement matching chains
			e.executeMatchingChain(ob, tree, priceLevel, order.price, chain)

			// Call the corresponding handlers
			// TODO: Call OnExecuteTrade handler!
			// NOTE: To do that it is necessary to re-implement executeMatchingChain() method
			e.handler.OnExecuteOrder(ob, order, order.price, order.restQuantity)

			// Update the corresponding market price
			ob.updateLastPrice(order.side, order.price)
			ob.updateMatchingPrice(order.side, order.price)

			// Increase the order executed quantity
			quantity := order.restQuantity
			quoteQuantity := quantity.Mul(order.price).Div64(UintPrecision)
			if order.IsBuy() {
				order.available = order.available.Sub(quoteQuantity)
			} else {
				order.available = order.available.Sub(quantity)
			}
			order.executedQuantity = order.executedQuantity.Add(quantity)
			order.executedQuoteQuantity = order.executedQuoteQuantity.Add(quoteQuantity)
			order.restQuantity = NewZeroUint()

			return
		}

		// Execute crossed orders
		for orderPtr := priceLevel.Value().queue.Front(); !order.IsExecuted() && orderPtr != nil; {
			orderPtrNext := orderPtr.Next()
			executingOrder := orderPtr.Value

			// Get the execution price and quantity
			price := executingOrder.price
			executingQuantity, orderQuantity := executingOrder.RestAvailableQuantity(price), order.RestAvailableQuantity(price)
			quantity := Min(executingQuantity, orderQuantity)
			quoteQuantity := quantity.Mul(price).Div64(UintPrecision)

			// Ensure both orders are not executed already
			// TODO: Investigate the reason of occurring such cases!
			if executingQuantity.IsZero() {
				e.deleteOrder(ob, executingOrder, true)
				orderPtr = orderPtrNext
				continue
			}
			if orderQuantity.IsZero() {
				return
			}

			// Special case for 'All-Or-None' orders
			if executingOrder.IsAON() && executingQuantity.GreaterThan(orderQuantity) {
				// Matching is not available
				return
			}

			// Check and delete linked orders
			e.deleteLinkedOrder(ob, executingOrder, false)
			e.deleteLinkedOrder(ob, order, false)

			// Call the corresponding handlers
			e.handler.OnExecuteOrder(ob, executingOrder, price, quantity)
			e.handler.OnExecuteOrder(ob, order, price, quantity)
			if executingOrder.id < order.id {
				e.handler.OnExecuteTrade(ob, executingOrder, order, price, quantity)
			} else {
				e.handler.OnExecuteTrade(ob, order, executingOrder, price, quantity)
			}

			// Update the corresponding market price
			ob.updateLastPrice(executingOrder.side, price)
			ob.updateMatchingPrice(executingOrder.side, price)

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
			e.reduceOrder(ob, executingOrder, quantity, true)

			// Update the corresponding market price
			ob.updateLastPrice(order.side, price)
			ob.updateMatchingPrice(order.side, price)

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
				return
			}

			// Move to the next order to execute at the same price level
			orderPtr = orderPtrNext
		}
	}
}

////////////////////////////////////////////////////////////////
// Matching chains
////////////////////////////////////////////////////////////////

func (e *Engine) executeMatchingChain(
	ob *OrderBook,
	tree *avl.Tree[Uint, *PriceLevelL3],
	node *avl.Node[Uint, *PriceLevelL3],
	price Uint,
	volume Uint,
) {

	// Execute all orders in the matching chain
	for !volume.IsZero() && node != nil {
		nodeNext := node.NextRight()

		// Execute all orders in the current price level
		for orderPtr := node.Value().queue.Front(); !volume.IsZero() && orderPtr != nil; {
			orderPtrNext := orderPtr.Next()
			order := orderPtr.Value

			var quantity, quoteQuantity Uint

			// Execute order
			if order.IsAON() {
				// Get the execution quantity
				quantity, quoteQuantity = order.RestAvailableQuantities(price)

				// Call the corresponding handler
				// TODO: Call OnExecuteTrade handler!
				// NOTE: To do that it is necessary to re-implement executeMatchingChain() method
				e.handler.OnExecuteOrder(ob, order, price, quantity)

				// Update the corresponding market price
				ob.updateLastPrice(order.side, price)
				ob.updateMatchingPrice(order.side, price)

				// Increase the order executed quantity
				order.executedQuantity = order.executedQuantity.Add(quantity)
				order.executedQuoteQuantity = order.executedQuoteQuantity.Add(quoteQuantity)

				// Delete the executing order from the order book
				e.deleteOrder(ob, order, true)
			} else {
				// Get the execution quantity
				quantity = Min(order.RestAvailableQuantity(price), volume)
				quoteQuantity := quantity.Mul(price).Div64(UintPrecision)

				// Call the corresponding handler
				// TODO: Call OnExecuteTrade handler!
				// NOTE: To do that it is necessary to re-implement executeMatchingChain() method
				e.handler.OnExecuteOrder(ob, order, price, quantity)

				// Update the corresponding market price
				ob.updateLastPrice(order.side, price)
				ob.updateMatchingPrice(order.side, price)

				// Increase the order executed quantity
				order.executedQuantity = order.executedQuantity.Add(quantity)
				order.executedQuoteQuantity = order.executedQuoteQuantity.Add(quoteQuantity)

				// Reduce the executing order in the order book
				e.reduceOrder(ob, order, quantity, true)
			}

			// Reduce the execution chain
			volume = volume.Sub(quantity)

			// Move to the next order to execute at the same price level
			orderPtr = orderPtrNext
		}

		// Switch to the next price level
		node = nodeNext
	}
}

func (e *Engine) calculateMatchingChain(
	ob *OrderBook,
	order *Order,
	node *avl.Node[Uint, *PriceLevelL3],
) Uint {
	available, volume := NewZeroUint(), order.restQuantity

	// Travel through price levels
	for node != nil {
		nodeNext := node.NextRight()
		priceLevel := node.Value()

		// Check the arbitrage bid/ask prices
		if order.IsBuy() {
			if order.price.GreaterThan(priceLevel.price) {
				return NewZeroUint()
			}
		} else {
			if order.price.LessThan(priceLevel.price) {
				return NewZeroUint()
			}
		}

		// Travel through orders at current price levels
		for orderPtr := priceLevel.queue.Front(); orderPtr != nil; {
			orderPtrNext := orderPtr.Next()
			order := orderPtr.Value
			need := volume.Sub(available)
			quantity := order.RestAvailableQuantity(order.price)
			if !order.IsAON() {
				quantity = Min(quantity, need)
			}
			available = available.Add(quantity)

			// Matching is possible, return the chain size
			if volume.Equals(available) {
				return available
			}

			// Matching is not possible
			if volume.LessThan(available) {
				return NewZeroUint()
			}

			// Move to the next order to calculate at the same price level
			orderPtr = orderPtrNext
		}

		// Switch to the next price level
		node = nodeNext
	}

	// Matching is not available
	return NewZeroUint()
}

func (e *Engine) calculateMatchingChainCross(
	ob *OrderBook,
	bid *avl.Node[Uint, *PriceLevelL3],
	ask *avl.Node[Uint, *PriceLevelL3],
) Uint {
	longest, shortest := bid, ask
	longestOrder, shortestOrder := bid.Value().queue.Front(), ask.Value().queue.Front()
	required := longestOrder.Value.restQuantity
	available := NewZeroUint()

	// Find the initial longest order chain
	if longestOrder.Value.IsAON() && shortestOrder.Value.IsAON() {
		// Choose the longest 'All-Or-None' order
		if shortestOrder.Value.restQuantity.GreaterThan(longestOrder.Value.restQuantity) {
			required = shortestOrder.Value.restQuantity
			longest, shortest = shortest, longest                     // swap
			longestOrder, shortestOrder = shortestOrder, longestOrder // swap
		}
	} else if shortestOrder.Value.IsAON() {
		// Make single 'All-Or-None' order to be the longest
		required = shortestOrder.Value.restQuantity
		longest, shortest = shortest, longest                     // swap
		longestOrder, shortestOrder = shortestOrder, longestOrder // swap
	}

	// Travel through price levels
	for longest != nil && shortest != nil {
		longestNext := longest.NextRight()
		shortestNext := shortest.NextRight()

		// Travel through orders at current price levels
		for longestOrder != nil && shortestOrder != nil {
			need := required.Sub(available)
			quantity := shortestOrder.Value.restQuantity
			if !shortestOrder.Value.IsAON() {
				quantity = Min(shortestOrder.Value.restQuantity, need)
			}
			available = available.Add(quantity)

			// Matching is possible, return the chain size
			if required.Equals(available) {
				return available
			}

			// Swap longest and shortest chains
			if required.LessThan(available) {
				next := longestOrder.Next()
				longestOrder = shortestOrder
				shortestOrder = next
				required, available = available, required // swap
				continue
			}

			// Take the next order
			shortestOrder = shortestOrder.Next()
		}

		// Switch to the next longest price level
		if longestOrder == nil {
			longest = longestNext
			if longest != nil {
				longestOrder = longest.Value().queue.Front()
			}
		}

		// Switch to the next shortest price level
		if shortestOrder == nil {
			shortest = shortestNext
			if shortest != nil {
				shortestOrder = shortest.Value().queue.Front()
			}
		}
	}

	// Matching is not available
	return NewZeroUint()
}
