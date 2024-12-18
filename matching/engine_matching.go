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

			itBid := topBid.Value().Iterator()
			itAsk := topAsk.Value().Iterator()
			itBid.Next()
			itAsk.Next()
			// Find the first order to execute and the first order to reduce
			// Execute crossed orders
			for itBid.Valid() && itAsk.Valid() {
				if itBid.Current().Value == nil || itAsk.Current().Value == nil {
					break
				}
				// Find the best order to execute and the best order to reduce
				executing, reducing, swapped := itBid.Current().Value, itAsk.Current().Value, false

				// Need to define price based on maker order,
				// define maker as order that has come earlier,
				// calculate price and call handler based on this.
				var price Uint
				if executing.id < reducing.id {
					price = getPriceForTrade(executing, reducing)
				} else {
					price = getPriceForTrade(reducing, executing)
				}

				// Define quantities for current execution.
				reducingQty, reducingQuoteQty := calcRestAvailableQuantities(reducing, price)
				executingQty, executingQuoteQty := calcRestAvailableQuantities(executing, price)
				quantity, quoteQuantity := executingQty, executingQuoteQty

				if reducingQty.LessThan(executingQty) {
					quantity, quoteQuantity = reducingQty, reducingQuoteQty
					executing, reducing, swapped = reducing, executing, true // swap
				}

				e.handler.OnExecuteOrder(ob, reducing.id, price, quantity, quoteQuantity)
				e.handler.OnExecuteOrder(ob, executing.id, price, quantity, quoteQuantity)

				if executing.id < reducing.id {
					e.handler.OnExecuteTrade(
						ob, OrderUpdate{executing.id, quantity, quoteQuantity}, OrderUpdate{reducing.id, quantity, quoteQuantity},
						price, quantity, quoteQuantity,
					)
				} else {
					e.handler.OnExecuteTrade(
						ob, OrderUpdate{reducing.id, quantity, quoteQuantity}, OrderUpdate{executing.id, quantity, quoteQuantity},
						price, quantity, quoteQuantity,
					)
				}

				// Execute orders
				reducingExecuted, err := e.executeOrder(ob, reducing, quantity, quoteQuantity)
				if err != nil {
					return fmt.Errorf("failed to execute order (id: %d): %w", reducing.ID(), err)
				}
				executingExecuted, err := e.executeOrder(ob, executing, quantity, quoteQuantity)
				if err != nil {
					return fmt.Errorf("failed to execute order (id: %d): %w", executing.ID(), err)
				}

				// Update common market price.
				ob.updateMarketPrice(price)

				// Cut remainders for orders.
				if !reducingExecuted {
					reducingExecuted = e.cutRemainders(ob, reducing)
				}
				if !executingExecuted {
					executingExecuted = e.cutRemainders(ob, executing)
				}

				// Check if executing is executed.
				if !executingExecuted {
					return ErrInternalExecutingOrderNotExecuted
				}

				// Next executing order.
				if !swapped {
					itBid.Next()
				} else {
					itAsk.Next()
				}

				// If reducing is executed too.
				if reducingExecuted {
					if !swapped {
						itAsk.Next()
					} else {
						itBid.Next()
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

		// Get top price from asks and max price for symbol.
		topPrice = ob.TopAsk().Value().Price()
		maxPrice := NewMaxUint()

		// Overflow protection for topPrice.Add(order.marketSlippage).
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
func (e *Engine) matchOrder(ob *OrderBook, taker *Order) error {
	// Special case for 'Fill-Or-Kill'
	if taker.IsFOK() {
		// Determine the best bid/ask price level
		var priceLevel *avl.Node[Uint, *PriceLevelL3]
		if taker.IsBuy() {
			priceLevel = ob.TopAsk()
		} else {
			priceLevel = ob.TopBid()
		}
		if priceLevel == nil {
			return nil
		}

		if !e.canExecuteChain(taker, priceLevel) {
			return nil
		}
	}

	// Start the matching from the top of the book
	for {
		// Determine the best bid/ask price level
		var priceLevel *avl.Node[Uint, *PriceLevelL3]
		if taker.IsBuy() {
			priceLevel = ob.TopAsk()
		} else {
			priceLevel = ob.TopBid()
		}
		if priceLevel == nil {
			return nil
		}

		// Check the arbitrage bid/ask prices
		if taker.IsEndByPrice(priceLevel.Value().Price()) {
			return nil
		}

		if taker.IsExecuted() {
			return nil
		}

		it := priceLevel.Value().Iterator()
		// Execute crossed orders
		for it.Next() {
			maker := it.Current().Value

			// Get the execution price and quantity of crossed order, executing is maker
			price := getPriceForTrade(maker, taker)

			makerQty, makerQuoteQty := calcRestAvailableQuantities(maker, price)
			qty, quoteQty := calcRestAvailableQuantities(taker, price)

			// Check if can't be matched at all (market with not enough available)
			if qty.IsZero() {
				return nil
			}

			// Choose less qty as qty for trade
			if makerQty.LessThan(qty) {
				qty = makerQty
				// need to calc like this because of crossed qty and price
				quoteQty = makerQuoteQty
			}

			// Calc quantities and call handlers
			e.handler.OnExecuteOrder(ob, taker.id, price, qty, quoteQty)
			e.handler.OnExecuteOrder(ob, maker.id, price, qty, quoteQty)
			e.handler.OnExecuteTrade(
				ob, OrderUpdate{maker.id, qty, quoteQty}, OrderUpdate{taker.id, qty, quoteQty},
				price, qty, quoteQty,
			)

			// Execute orders
			takerExecuted, err := e.executeOrder(ob, taker, qty, quoteQty)
			if err != nil {
				return fmt.Errorf("failed to execute order (id: %d): %w", taker.ID(), err)
			}
			makerExecuted, err := e.executeOrder(ob, maker, qty, quoteQty)
			if err != nil {
				return fmt.Errorf("failed to execute order (id: %d): %w", maker.ID(), err)
			}

			// Update common market price
			ob.updateMarketPrice(price)

			// Cut remainders for orders
			if !takerExecuted {
				takerExecuted = e.cutRemainders(ob, taker)
			}
			if !makerExecuted {
				_ = e.cutRemainders(ob, maker)
			}

			// Exit the loop if the order is executed
			if takerExecuted {
				return nil
			}
		}
	}
}

/////////////////////////////////////////////////////
// Matching chains
////////////////////////////////////////////////////////////////

// canExecuteChain have to be used for FOK orders to check if full execution is possible
// here we can only deal with limit type.
func (e *Engine) canExecuteChain(
	taker *Order,
	priceLevel *avl.Node[Uint, *PriceLevelL3],
) bool {
	required := taker.quantity

	// Travel through price levels
	for priceLevel != nil {
		if taker.IsEndByPrice(priceLevel.Value().Price()) {
			return false
		}

		// Travel through orders at current price levels
		it := priceLevel.Value().Iterator()
		for it.Next() {
			order := it.Current().Value
			if order == nil {
				break
			}
			quantity := order.restQuantity

			if required.LessThanOrEqualTo(quantity) {
				return true
			}

			required = required.Sub(quantity)
		}

		priceLevel = priceLevel.NextRight()
	}

	// Matching is not available
	return false
}

// getPrice ForTrade choses price for trade assuming that quote locking orders execution depends on price,
// so to guarantee execution quantity of limit orders, less price must be chosen.
func getPriceForTrade(maker *Order, taker *Order) Uint {
	switch {
	// Check market orders, they always executed by price of maker.
	case taker.IsMarket():
		return maker.price
	// Both locked in quote, so take min.
	case maker.IsLockingQuote() && taker.IsLockingQuote():
		return Min(maker.price, taker.price)
	// Maker only in quote.
	case maker.IsLockingQuote():
		return maker.price
	// Taker only in quote.
	case taker.IsLockingQuote():
		return taker.price
	}

	return maker.price
}
