package matching

// OrderSide is an enumeration of possible trading sides (buy/sell).
type OrderSide uint8

const (
	// OrderSideBuy represents market side which includes only buy orders (bids).
	OrderSideBuy OrderSide = iota + 1
	// OrderSideSell represents market side which includes only sell orders (asks).
	OrderSideSell
)

func (os OrderSide) String() string {
	switch os {
	case OrderSideBuy:
		return "buy"
	case OrderSideSell:
		return "sell"
	default:
		return "unknown"
	}
}
