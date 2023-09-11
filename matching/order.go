package matching

import (
	"github.com/cryptonstudio/crypton-matching/types/avl"
	"github.com/cryptonstudio/crypton-matching/types/list"
)

// Order contains information about an order.
// An order is an instruction to buy or sell on a trading venue such as a stock market,
// bond market, commodity market, or financial derivative market. These instructions can
// be simple or complicated, and can be sent to either a broker or directly to a trading
// venue via direct market access.
type Order struct {
	id          uint64
	symbolID    uint32
	orderType   OrderType
	side        OrderSide
	timeInForce OrderTimeInForce
	_           uint8 // required for struct bytes alignment

	price Uint

	stopPrice     Uint
	takeProfit    bool
	quantity      Uint
	quoteQuantity Uint

	// Order max visible quantity.
	// This property allows to prepare 'iceberg'/'hidden' orders with the following rules:
	// - maxVisible >= restQuantity - regular order
	// - maxVisible == 0            - 'hidden' order
	// - maxVisible < restQuantity  - 'iceberg' order
	// Supported only for limit and stop-limit orders!
	maxVisible Uint

	// Market order slippage.
	// Slippage is useful to protect market order from executions at prices
	// which are too far from the best price. If the slippage is provided
	// for market order its execution will be stopped when the price run
	// out of the market for the given slippage value. Zero slippage will
	// allow to execute market order only at the best price, non executed
	// part of the market order will be canceled.
	// Supported only for market and stop orders!
	marketSlippage Uint

	// Order trailing distance to market.
	// Value greater than 10000 represents absolute distance from the market.
	// Value less or equal to 10000 represents percentage distance from the market
	// with 0.01% precision (-1 means 0.01, -10000 means 100%).
	// Supported only for trailing stop orders!
	trailingDistance Uint

	// Order trailing step.
	// Value greater than 10000 represents absolute step from the market.
	// Value less or equal to 10000 represents percentage step from the market
	// with 0.01% precision (-1 means 0.01%, -10000 means 100%).
	// Supported only for trailing stop orders!
	trailingStep Uint

	available             Uint
	restQuantity          Uint
	executedQuantity      Uint
	executedQuoteQuantity Uint

	// Market order by using quote quantity.
	marketQuoteMode bool // by default is false, set to true only if quantity = 0 and quoteQuantity > 0

	// Pointer to the price level where the order is placed.
	priceLevel *avl.Node[Uint, *PriceLevelL3]

	// Pointer to the order queue where the order is placed.
	orderQueued *list.Element[*Order]
}

////////////////////////////////////////////////////////////////

// ID returns the order ID.
func (o *Order) ID() uint64 {
	return o.id
}

// SymbolID returns the symbol ID of the order.
func (o *Order) SymbolID() uint32 {
	return o.symbolID
}

////////////////////////////////////////////////////////////////

// Type returns the order type.
func (o *Order) Type() OrderType {
	return o.orderType
}

// IsLimit returns true if limit order.
func (o *Order) IsLimit() bool {
	return o.orderType == OrderTypeLimit
}

// IsMarket returns true if market order.
func (o *Order) IsMarket() bool {
	return o.orderType == OrderTypeMarket
}

// IsStop returns true if stop order.
func (o *Order) IsStop() bool {
	return o.orderType == OrderTypeStop
}

// IsStopLimit returns true if stop-limit order.
func (o *Order) IsStopLimit() bool {
	return o.orderType == OrderTypeStopLimit
}

// IsTrailingStop returns true if trailing stop order.
func (o *Order) IsTrailingStop() bool {
	return o.orderType == OrderTypeTrailingStop
}

// IsTrailingStopLimit returns true if trailing stop-limit order.
func (o *Order) IsTrailingStopLimit() bool {
	return o.orderType == OrderTypeTrailingStopLimit
}

////////////////////////////////////////////////////////////////

// Side returns the market side of the order.
func (o *Order) Side() OrderSide {
	return o.side
}

// IsBuy returns true if buy order.
func (o *Order) IsBuy() bool {
	return o.side == OrderSideBuy
}

// IsSell returns true if sell order.
func (o *Order) IsSell() bool {
	return o.side == OrderSideSell
}

////////////////////////////////////////////////////////////////

// TimeInForce returns the time in force option of the order.
func (o *Order) TimeInForce() OrderTimeInForce {
	return o.timeInForce
}

// IsGTC returns true if 'Good-Till-Cancelled' order.
func (o *Order) IsGTC() bool {
	return o.timeInForce == OrderTimeInForceGTC
}

// IsIOC returns true if 'Immediate-Or-Cancel' order.
func (o *Order) IsIOC() bool {
	return o.timeInForce == OrderTimeInForceIOC
}

// IsFOK returns true if 'Fill-Or-Kill' order.
func (o *Order) IsFOK() bool {
	return o.timeInForce == OrderTimeInForceFOK
}

// IsAON returns true if 'All-Or-None' order.
func (o *Order) IsAON() bool {
	return o.timeInForce == OrderTimeInForceAON
}

////////////////////////////////////////////////////////////////

// MaxVisibleQuantity returns maximum visible in an order book quantity of the order.
func (o *Order) MaxVisibleQuantity() Uint {
	return o.maxVisible
}

// IsHidden returns true if 'hidden' order.
func (o *Order) IsHidden() bool {
	return o.maxVisible.IsMax()
}

// Iceberg returns true if 'iceberg' order.
func (o *Order) IsIceberg() bool {
	return !o.IsHidden() && !o.maxVisible.IsZero()
}

////////////////////////////////////////////////////////////////

// MarketSlippage returns the slippage specified for the market order.
func (o *Order) MarketSlippage() Uint {
	return o.marketSlippage
}

// IsMarketSlippage returns true if slippage is specified for the market order.
func (o *Order) IsMarketSlippage() bool {
	return !o.marketSlippage.IsMax()
}

////////////////////////////////////////////////////////////////

// Price returns the order price.
func (o *Order) Price() Uint {
	return o.price
}

// StopPrice returns the order stop price.
func (o *Order) StopPrice() Uint {
	return o.stopPrice
}

// Quantity returns the order quantity.
func (o *Order) Quantity() Uint {
	return o.quantity
}

// QuoteQuantity returns optional quote quantity of the market order.
func (o *Order) QuoteQuantity() Uint {
	return o.quoteQuantity
}

////////////////////////////////////////////////////////////////

// RestQuantity returns order remaining quantity.
func (o *Order) RestQuantity() Uint {
	return o.restQuantity
}

// RestAvailableQuantity returns order remaining quantity which can be executed with specific price according to locked quantity.
func (o *Order) RestAvailableQuantity(price Uint) Uint {
	quote, _ := o.RestAvailableQuantities(price)
	return quote
}

// RestAvailableQuantities returns order remaining base and quote quantity which can be executed with specific price according to locked quantity.
func (o *Order) RestAvailableQuantities(price Uint) (Uint, Uint) {
	var restQuoteQuantity Uint
	var restQuantity Uint

	if o.marketQuoteMode {
		restQuoteQuantity = o.quoteQuantity.Sub(o.executedQuoteQuantity)
		restQuantity, _ = restQuoteQuantity.Mul64(UintPrecision).QuoRem(price)
	} else {
		restQuantity = o.restQuantity
		restQuoteQuantity = restQuantity.Mul(price).Div64(UintPrecision)
	}

	// cap rest quote quantity for buy order
	if o.IsBuy() && restQuoteQuantity.GreaterThan(o.available) {
		restQuoteQuantity = o.available
		restQuantity, _ = restQuoteQuantity.Mul64(UintPrecision).QuoRem(price)
	}

	// cap rest quantity for sell order
	if !o.IsBuy() && restQuantity.GreaterThan(o.available) {
		restQuantity = o.available
		restQuoteQuantity = restQuantity.Mul(price).Div64(UintPrecision)
	}

	return restQuantity, restQuoteQuantity
}

// VisibleQuantity returns order remaining visible quantity.
func (o *Order) VisibleQuantity() Uint {
	return Min(o.restQuantity, o.maxVisible)
}

// HiddenQuantity returns order remaining hidden quantity.
func (o *Order) HiddenQuantity() Uint {
	if o.restQuantity.GreaterThan(o.maxVisible) {
		return o.restQuantity.Sub(o.maxVisible)
	}
	return Uint{}
}

// ExecutedQuantity returns order executed base quantity.
func (o *Order) ExecutedQuantity() Uint {
	return o.executedQuantity
}

// ExecutedQuoteQuantity returns order executed quote quantity.
func (o *Order) ExecutedQuoteQuantity() Uint {
	return o.executedQuoteQuantity
}

// IsExecuted returns true if the order is completely executed.
func (o *Order) IsExecuted() bool {
	return o.restQuantity.IsZero()
}

// Available returns order available amount equivalents to rest user locked amount of asset.
func (o *Order) Available() Uint {
	return o.available
}

////////////////////////////////////////////////////////////////

// Validate returns error if the order fails to pass validation so can be used safely.
func (o *Order) Validate() error {

	// TODO: Implement validation very carefully!

	return nil
}

func (o *Order) RestoreExecution(executed, executedQuote, restQuantity Uint) {
	o.executedQuantity = executed
	o.executedQuoteQuantity = executedQuote
	o.restQuantity = restQuantity
}
