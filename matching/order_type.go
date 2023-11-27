package matching

// OrderType is an enumeration of possible order types.
type OrderType uint8

const (
	// A limit order is an order to buy or sell a stock at a specific
	// price or better. A buy limit order can only be executed at the limit price or lower,
	// and a sell limit order can only be executed at the limit price or higher. A limit
	// order is not guaranteed to execute. A limit order can only be filled if the stock's
	// market price reaches the limit price. While limit orders do not guarantee execution,
	// they help ensure that an investor does not pay more than a predetermined price for a
	// stock.
	OrderTypeLimit OrderType = iota + 1

	// A market order is an order to buy or sell a stock at the best
	// available price. Generally, this type of order will be executed immediately. However,
	// the price at which a market order will be executed is not guaranteed. It is important
	// for investors to remember that the last-traded price is not necessarily the price at
	// which a market order will be executed. In fast-moving markets, the price at which a
	// market order will execute often deviates from the last-traded price or "real time"
	// quote.
	OrderTypeMarket

	// A stop order, also referred to as a stop-loss order, is an order
	// to buy or sell a stock once the price of the stock reaches a specified price, known
	// as the stop price. When the stop price is reached, a stop order becomes a market order.
	// A buy stop order is entered at a stop price above the current market price. Investors
	// generally use a buy stop order to limit a loss or to protect a profit on a stock that
	// they have sold short. A sell stop order is entered at a stop price below the current
	// market price. Investors generally use a sell stop order to limit a loss or to protect
	// a profit on a stock that they own.
	OrderTypeStop

	// A stop-limit order is an order to buy or sell a stock that
	// combines the features of a stop order and a limit order. Once the stop price is reached,
	// a stop-limit order becomes a limit order that will be executed at a specified price (or
	// better). The benefit of a stop-limit order is that the investor can control the price at
	// which the order can be executed.
	OrderTypeStopLimit

	// A trailing stop order is entered with a stop parameter
	// that creates a moving or trailing activation price, hence the name. This parameter
	// is entered as a percentage change or actual specific amount of rise (or fall) in the
	// security price. Trailing stop sell orders are used to maximize and protect profit as
	// a stock's price rises and limit losses when its price falls.
	OrderTypeTrailingStop

	// A trailing stop-limit order is similar to a trailing stop order.
	// Instead of selling at market price when triggered, the order becomes a limit order.
	OrderTypeTrailingStopLimit
)

func (ot OrderType) String() string {
	switch ot {
	case OrderTypeLimit:
		return "limit"
	case OrderTypeMarket:
		return "market"
	case OrderTypeStop:
		return "stop"
	case OrderTypeStopLimit:
		return "stop-limit"
	case OrderTypeTrailingStop:
		return "trailing-stop"
	case OrderTypeTrailingStopLimit:
		return "trailing-stop-limit"
	default:
		return "unknown"
	}
}

// OrderTimeInForce is an enumeration of possible order execution options.
type OrderTimeInForce uint8

const (
	// Good-Till-Cancelled (GTC) - A GTC order is an order to buy or sell a stock that
	// lasts until the order is completed or cancelled.
	OrderTimeInForceGTC OrderTimeInForce = iota + 1
	// Immediate-Or-Cancel (IOC) - An IOC order is an order to buy or sell a stock that
	// must be executed immediately. Any portion of the order that cannot be filled immediately
	// will be cancelled.
	OrderTimeInForceIOC
	// Fill-Or-Kill (FOK) - An FOK order is an order to buy or sell a stock that must
	// be executed immediately in its entirety; otherwise, the entire order will be cancelled
	// (i.e., no partial execution of the order is allowed).
	OrderTimeInForceFOK
	// All-Or-None (AON) - An AON order is an order to buy or sell a stock
	// that must be executed in its entirety, or not executed at all. AON orders that cannot be
	// executed immediately remain active until they are executed or cancelled.
	OrderTimeInForceAON
)

func (ot OrderTimeInForce) String() string {
	switch ot {
	case OrderTimeInForceGTC:
		return "good-till-cancelled"
	case OrderTimeInForceIOC:
		return "immediate-or-cancel"
	case OrderTimeInForceFOK:
		return "fill-or-kill"
	case OrderTimeInForceAON:
		return "all-or-none"
	default:
		return "unknown"
	}
}
