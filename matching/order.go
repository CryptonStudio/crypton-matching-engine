package matching

import (
	"fmt"

	"github.com/cryptonstudio/crypton-matching-engine/types/avl"
	"github.com/cryptonstudio/crypton-matching-engine/types/list"
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
	direction   OrderDirection
	timeInForce OrderTimeInForce

	_ uint8 // required for struct bytes alignment

	price Uint

	stopPrice     Uint
	stopPriceMode StopPriceMode
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

	// Base trading quantities,
	// mutable through the trading activity,
	// use setters with specified limits.
	available             Uint
	restQuantity          Uint
	restQuoteQuantity     Uint
	executedQuantity      Uint
	executedQuoteQuantity Uint

	// Market order by using quote quantity.
	marketQuoteMode bool // by default is false, set to true only if quantity = 0 and quoteQuantity > 0

	// Linked order in OCO order pair (used for OCO orders only)
	linkedOrderID uint64

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

// IsTrailingStop returns true if trailing stop order.
func (o *Order) IsTakeProfit() bool {
	return o.takeProfit
}

////////////////////////////////////////////////////////////////

// Side returns the market side of the order.
func (o *Order) Side() OrderSide {
	return o.side
}

// Direction returns the market direction of the order.
func (o *Order) Direction() OrderDirection {
	return o.direction
}

// IsBuy returns true if buy order.
func (o *Order) IsBuy() bool {
	return o.side == OrderSideBuy
}

// IsSell returns true if sell order.
func (o *Order) IsSell() bool {
	return o.side == OrderSideSell
}

// IsVirtualOB shows if the order uses virtual order book
func (o *Order) IsVirtualOB() bool {
	switch o.orderType {
	case OrderTypeStop, OrderTypeStopLimit, OrderTypeTrailingStop, OrderTypeTrailingStopLimit:
		return true
	}

	return false
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

// StopPriceMode returns the order stop price mode.
func (o *Order) StopPriceMode() StopPriceMode {
	return o.stopPriceMode
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

func (o *Order) SubRestQuantity(v Uint) {
	o.restQuantity = o.restQuantity.Sub(v)
}

// RestQuoteQuantity returns remaining quote quantity.
// Must be used only for market quote mode.
func (o *Order) RestQuoteQuantity() Uint {
	return o.restQuoteQuantity
}

func (o *Order) SubRestQuoteQuantity(v Uint) {
	o.restQuoteQuantity = o.restQuoteQuantity.Sub(v)
}

// Available returns order available amount equivalents to rest user locked amount of asset.
func (o *Order) Available() Uint {
	return o.available
}

func (o *Order) AddAvailable(v Uint) {
	o.available = o.available.Add(v)
}

func (o *Order) SubAvailable(v Uint) {
	o.available = o.available.Sub(v)
}

// ExecutedQuantity returns order executed base quantity.
func (o *Order) ExecutedQuantity() Uint {
	return o.executedQuantity
}

func (o *Order) AddExecutedQuantity(v Uint) {
	o.executedQuantity = o.executedQuantity.Add(v)
}

// ExecutedQuoteQuantity returns order executed quote quantity.
func (o *Order) ExecutedQuoteQuantity() Uint {
	return o.executedQuoteQuantity
}

func (o *Order) AddExecutedQuoteQuantity(v Uint) {
	o.executedQuoteQuantity = o.executedQuoteQuantity.Add(v)
}

// IsExecuted returns true if the order is completely executed.
// It covers cases
// - when restQuantity is zero
// - when restQuoteQuantity (marketQuoteMode)
// - when check after deleting (cleaned order)
func (o *Order) IsExecuted() bool {
	if o.marketQuoteMode {
		return o.restQuoteQuantity.IsZero()
	}

	return o.restQuantity.IsZero()
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

// Linked order in OCO order pair (used for OCO orders only)
func (o *Order) LinkedOrderID() uint64 {
	return o.linkedOrderID
}

////////////////////////////////////////////////////////////////

// Validate returns error if the order fails to pass validation so can be used safely.
func (o *Order) Validate(ob *OrderBook) error {
	// Validate order ID
	if o.id == 0 {
		return ErrInvalidOrderID
	}

	// Validate order type
	switch o.orderType {
	case OrderTypeLimit:
	case OrderTypeMarket:
	case OrderTypeStop:
	case OrderTypeStopLimit:
	case OrderTypeTrailingStop:
	case OrderTypeTrailingStopLimit:
	default:
		return ErrInvalidOrderType
	}

	switch o.side {
	case OrderSideBuy:
	case OrderSideSell:
	default:
		return ErrInvalidOrderSide
	}

	// Validate price (if necessary)
	switch o.orderType {
	case OrderTypeLimit, OrderTypeStopLimit, OrderTypeTrailingStopLimit:
		if o.price.LessThan(ob.symbol.priceLimits.Min) {
			return ErrInvalidOrderPrice
		}
		if o.price.GreaterThan(ob.symbol.priceLimits.Max) {
			return ErrInvalidOrderPrice
		}
		_, rem := o.price.QuoRem(ob.symbol.priceLimits.Step)
		if !rem.IsZero() {
			return ErrInvalidOrderPrice
		}
	}

	// Validate stop price (if necessary)
	switch o.orderType {
	case OrderTypeStop, OrderTypeStopLimit, OrderTypeTrailingStop, OrderTypeTrailingStopLimit:
		if o.stopPrice.LessThan(ob.symbol.priceLimits.Min) {
			return ErrInvalidOrderStopPrice
		}
		if o.stopPrice.GreaterThan(ob.symbol.priceLimits.Max) {
			return ErrInvalidOrderStopPrice
		}
		_, rem := o.stopPrice.QuoRem(ob.symbol.priceLimits.Step)
		if !rem.IsZero() {
			return ErrInvalidOrderStopPrice
		}
	}

	// Validate quantity (if necessary)
	switch o.orderType {
	case OrderTypeLimit, OrderTypeStopLimit, OrderTypeTrailingStopLimit:
		if o.quantity.LessThan(ob.symbol.lotSizeLimits.Min) {
			return ErrInvalidOrderQuantity
		}
		if o.quantity.GreaterThan(ob.symbol.lotSizeLimits.Max) {
			return ErrInvalidOrderQuantity
		}
		_, rem := o.quantity.QuoRem(ob.symbol.lotSizeLimits.Step)
		if !rem.IsZero() {
			return ErrInvalidOrderQuantity
		}
	}
	if !o.quantity.IsZero() && !o.quoteQuantity.IsZero() {
		return ErrInvalidOrderQuantity
	}
	if o.quantity.IsZero() {
		if o.quoteQuantity.LessThan(ob.symbol.quoteLotSizeLimits.Min) {
			return ErrInvalidOrderQuoteQuantity
		}
		if o.quoteQuantity.GreaterThan(ob.symbol.quoteLotSizeLimits.Max) {
			return ErrInvalidOrderQuoteQuantity
		}
		_, rem := o.quoteQuantity.QuoRem(ob.symbol.quoteLotSizeLimits.Step)
		if !rem.IsZero() {
			return ErrInvalidOrderQuoteQuantity
		}
	}

	// Validate slippage by price limits step
	if !o.marketSlippage.IsZero() {
		_, rem := o.marketSlippage.QuoRem(ob.symbol.priceLimits.Step)
		if !rem.IsZero() {
			return ErrInvalidMarketSlippage
		}
	}

	return nil
}

func (o *Order) IsLockingBase() bool {
	return o.direction == OrderDirectionClose
}

func (o *Order) IsLockingQuote() bool {
	return o.direction == OrderDirectionOpen
}

// CheckLocked checks locked quantity,
// all combinations of orders need exact minimum locked amount,
// except Buy Market Base, Sell Market Quote.
func (o *Order) CheckLocked() error {
	var needLocked Uint
	// Close Limit, Close Stop-limit, Close Market, Close Stop
	if o.IsLockingBase() && !o.quantity.IsZero() {
		needLocked = o.restQuantity
	}
	// Open Limit, Open Stop-limit
	if o.IsLockingQuote() && !o.quantity.IsZero() && !o.price.IsZero() {
		needLocked = o.restQuantity.Mul(o.price).Div64(UintPrecision)
	}
	// Open Market Quote, Open Stop Quote
	if o.IsLockingQuote() && !o.quoteQuantity.IsZero() {
		needLocked = o.restQuoteQuantity
	}

	if o.available.LessThan(needLocked) {
		return ErrNotEnoughLockedAmount
	}

	return nil
}

// CheckLockedOCO calculates and validates available for OCO orders pair
func CheckLockedOCO(stopLimit *Order, limit *Order) error {
	if !stopLimit.available.IsZero() {
		return ErrOCOStopLimitNotZeroLocked
	}

	needLocked := limit.quantity
	if limit.IsLockingQuote() {
		price := Max(stopLimit.price, limit.price)
		needLocked = limit.quantity.Mul(price).Div64(UintPrecision)
	}

	if limit.available.LessThan(needLocked) {
		return ErrNotEnoughLockedAmount
	}

	return nil
}

// CheckLockedTPSL calculates and validates available for TP/SL orders pair
func CheckLockedTPSL(tp *Order, sl *Order) error {
	if !sl.available.IsZero() {
		return ErrSLNotZeroLocked
	}

	needLocked := tp.quantity
	if tp.IsLockingQuote() {
		price := Max(tp.price, sl.price)
		needLocked = tp.quantity.Mul(price).Div64(UintPrecision)
	}

	if tp.available.LessThan(needLocked) {
		return ErrNotEnoughLockedAmount
	}

	return nil
}

// CheckLockedTPSL calculates and validates available for market TP/SL orders pair.
// Only in quote mode
func CheckLockedTPSLMarket(tp *Order, sl *Order) error {
	if !sl.available.IsZero() {
		return ErrSLNotZeroLocked
	}

	// Do not validate for Buy Market Base.
	if tp.Side() == OrderSideBuy {
		return nil
	}

	needLocked := tp.quantity

	if tp.available.LessThan(needLocked) {
		return ErrNotEnoughLockedAmount
	}

	return nil
}

func (o *Order) RestoreExecution(executed, executedQuote, restQuantity Uint) {
	o.executedQuantity = executed
	o.executedQuoteQuantity = executedQuote
	// don't rest quote, because stop order (market) not activated yet
	o.restQuantity = restQuantity
}

////////////////////////////////////////////////////////////////

func (o *Order) Activated() bool {
	return o.Type() == OrderTypeLimit || o.Type() == OrderTypeMarket
}

func (o *Order) PartiallyExecuted() bool {
	return !o.ExecutedQuantity().IsZero()
}

////////////////////////////////////////////////////////////////

// Clean cleans the order, use before put order to the pool
func (o *Order) Clean() {
	o.id = 0
	o.symbolID = 0
	o.orderType = 0
	o.side = 0
	o.timeInForce = 0
	o.price = NewZeroUint()
	o.stopPrice = NewZeroUint()
	o.stopPriceMode = 0
	o.takeProfit = false
	o.quantity = NewZeroUint()
	o.quoteQuantity = NewZeroUint()
	o.maxVisible = NewZeroUint()
	o.marketSlippage = NewZeroUint()
	o.trailingDistance = NewZeroUint()
	o.trailingStep = NewZeroUint()
	o.available = NewZeroUint()
	o.restQuantity = NewZeroUint()
	o.restQuoteQuantity = NewZeroUint()
	o.executedQuantity = NewZeroUint()
	o.executedQuoteQuantity = NewZeroUint()
	o.marketQuoteMode = false
	o.linkedOrderID = 0
	o.priceLevel = nil
	o.orderQueued = nil
}

// Debugging printer
func (o *Order) Debug() {
	fmt.Printf(
		"Order id=%d type=%s side=%s tif=%s price=%s qty=%s quoteQty=%s restQty=%s availQty=%s execQty=%s execQuoteQty=%s executed=%t priceLevel=%p\n",
		o.ID(),
		o.Type().String(),
		o.Side().String(),
		o.TimeInForce().String(),
		o.Price().ToFloatString(),
		o.Quantity().ToFloatString(),
		o.QuoteQuantity().ToFloatString(),
		o.RestQuantity().ToFloatString(),
		o.Available().ToFloatString(),
		o.ExecutedQuantity().ToFloatString(),
		o.ExecutedQuoteQuantity().ToFloatString(),
		o.IsExecuted(),
		o.priceLevel,
	)
}
