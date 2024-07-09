package matching

// NewLimitOrder creates new limit order.
func NewLimitOrder(
	symbolID uint32,
	orderID uint64,
	side OrderSide,
	timeInForce OrderTimeInForce,
	price Uint,
	quantity Uint,
	maxVisible Uint,
	restLocked Uint,
) Order {
	return Order{
		id:           orderID,
		symbolID:     symbolID,
		orderType:    OrderTypeLimit,
		side:         side,
		timeInForce:  timeInForce,
		price:        price,
		quantity:     quantity,
		maxVisible:   maxVisible,
		available:    restLocked,
		restQuantity: quantity,
	}
}

// NewMarketOrder creates new market order.
func NewMarketOrder(
	symbolID uint32,
	orderID uint64,
	side OrderSide,
	timeInForce OrderTimeInForce,
	quantity Uint,
	quoteQuantity Uint,
	slippage Uint,
	restLocked Uint,
) Order {
	return Order{
		id:                orderID,
		symbolID:          symbolID,
		orderType:         OrderTypeMarket,
		side:              side,
		timeInForce:       timeInForce,
		quantity:          quantity,
		quoteQuantity:     quoteQuantity,
		marketSlippage:    slippage,
		available:         restLocked,
		restQuantity:      quantity,
		restQuoteQuantity: quoteQuantity,
		marketQuoteMode:   quantity.IsZero() && !quoteQuantity.IsZero(),
	}
}

// NewStopOrder creates new stop order.
func NewStopOrder(
	symbolID uint32,
	orderID uint64,
	side OrderSide,
	timeInForce OrderTimeInForce,
	stopPriceMode StopPriceMode,
	stopPrice Uint,
	quantity Uint,
	quoteQuantity Uint,
	slippage Uint,
	restLocked Uint,
) Order {
	return Order{
		id:                orderID,
		symbolID:          symbolID,
		orderType:         OrderTypeStop,
		side:              side,
		timeInForce:       timeInForce,
		stopPriceMode:     stopPriceMode,
		stopPrice:         stopPrice,
		quantity:          quantity,
		quoteQuantity:     quoteQuantity,
		marketSlippage:    slippage,
		available:         restLocked,
		restQuantity:      quantity,
		restQuoteQuantity: quoteQuantity,
		marketQuoteMode:   quantity.IsZero() && !quoteQuantity.IsZero(),
	}
}

// NewStopLimitOrder creates new stop limit order.
func NewStopLimitOrder(
	symbolID uint32,
	orderID uint64,
	side OrderSide,
	timeInForce OrderTimeInForce,
	price Uint,
	stopPriceMode StopPriceMode,
	stopPrice Uint,
	quantity Uint,
	maxVisible Uint,
	restLocked Uint,
) Order {
	return Order{
		id:            orderID,
		symbolID:      symbolID,
		orderType:     OrderTypeStopLimit,
		side:          side,
		timeInForce:   timeInForce,
		price:         price,
		stopPriceMode: stopPriceMode,
		stopPrice:     stopPrice,
		quantity:      quantity,
		maxVisible:    maxVisible,
		available:     restLocked,
		restQuantity:  quantity,
	}
}

// NewTrailingStopOrder creates new stop order.
func NewTrailingStopOrder(
	symbolID uint32,
	orderID uint64,
	side OrderSide,
	timeInForce OrderTimeInForce,
	stopPriceMode StopPriceMode,
	stopPrice Uint,
	quantity Uint,
	quoteQuantity Uint,
	slippage Uint,
	trailingDistance Uint,
	trailingStep Uint,
	restLocked Uint,
) Order {
	return Order{
		id:               orderID,
		symbolID:         symbolID,
		orderType:        OrderTypeTrailingStop,
		side:             side,
		timeInForce:      timeInForce,
		stopPriceMode:    stopPriceMode,
		stopPrice:        stopPrice,
		quantity:         quantity,
		quoteQuantity:    quoteQuantity,
		marketSlippage:   slippage,
		trailingDistance: trailingDistance,
		trailingStep:     trailingStep,
		available:        restLocked,
		restQuantity:     quantity,
	}
}

// NewTrailingStopLimitOrder creates new stop limit order.
func NewTrailingStopLimitOrder(
	symbolID uint32,
	orderID uint64,
	side OrderSide,
	timeInForce OrderTimeInForce,
	price Uint,
	stopPriceMode StopPriceMode,
	stopPrice Uint,
	quantity Uint,
	maxVisible Uint,
	trailingDistance Uint,
	trailingStep Uint,
	restLocked Uint,
) Order {
	return Order{
		id:               orderID,
		symbolID:         symbolID,
		orderType:        OrderTypeTrailingStopLimit,
		side:             side,
		timeInForce:      timeInForce,
		price:            price,
		stopPrice:        stopPrice,
		stopPriceMode:    stopPriceMode,
		quantity:         quantity,
		maxVisible:       maxVisible,
		trailingDistance: trailingDistance,
		trailingStep:     trailingStep,
		available:        restLocked,
		restQuantity:     quantity,
	}
}
