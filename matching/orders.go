package matching

// NewLimitOrder creates new limit order.
func NewLimitOrder(
	symbolID uint32,
	orderID uint64,
	side OrderSide,
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
		timeInForce:  OrderTimeInForceGTC,
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
	quantity Uint,
	quoteQuantity Uint,
	slippage Uint,
	restLocked Uint,
) Order {
	return Order{
		id:              orderID,
		symbolID:        symbolID,
		orderType:       OrderTypeMarket,
		side:            side,
		timeInForce:     OrderTimeInForceGTC,
		quantity:        quantity,
		quoteQuantity:   quoteQuantity,
		marketSlippage:  slippage,
		available:       restLocked,
		restQuantity:    quantity,
		marketQuoteMode: quantity.IsZero() && !quoteQuantity.IsZero(),
	}
}

// NewStopOrder creates new stop order.
func NewStopOrder(
	symbolID uint32,
	orderID uint64,
	side OrderSide,
	stopPrice Uint,
	quantity Uint,
	quoteQuantity Uint,
	slippage Uint,
	restLocked Uint,
) Order {
	return Order{
		id:             orderID,
		symbolID:       symbolID,
		orderType:      OrderTypeStop,
		side:           side,
		timeInForce:    OrderTimeInForceGTC,
		stopPrice:      stopPrice,
		quantity:       quantity,
		quoteQuantity:  quoteQuantity,
		marketSlippage: slippage,
		available:      restLocked,
		restQuantity:   quantity,
	}
}

// NewStopLimitOrder creates new stop limit order.
func NewStopLimitOrder(
	symbolID uint32,
	orderID uint64,
	side OrderSide,
	price Uint,
	stopPrice Uint,
	quantity Uint,
	maxVisible Uint,
	restLocked Uint,
) Order {
	return Order{
		id:           orderID,
		symbolID:     symbolID,
		orderType:    OrderTypeStopLimit,
		side:         side,
		timeInForce:  OrderTimeInForceGTC,
		price:        price,
		stopPrice:    stopPrice,
		quantity:     quantity,
		maxVisible:   maxVisible,
		available:    restLocked,
		restQuantity: quantity,
	}
}

// NewTrailingStopOrder creates new stop order.
func NewTrailingStopOrder(
	symbolID uint32,
	orderID uint64,
	side OrderSide,
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
		timeInForce:      OrderTimeInForceGTC,
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
	price Uint,
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
		timeInForce:      OrderTimeInForceGTC,
		price:            price,
		stopPrice:        stopPrice,
		quantity:         quantity,
		maxVisible:       maxVisible,
		trailingDistance: trailingDistance,
		trailingStep:     trailingStep,
		available:        restLocked,
		restQuantity:     quantity,
	}
}
