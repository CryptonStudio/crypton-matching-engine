package matching

import (
	"errors"
	"fmt"
)

// Errors used by the package.
var (
	ErrOrderBookDuplicate        = errors.New("order book is duplicated")
	ErrOrderBookNotFound         = errors.New("order book is not found")
	ErrOrderDuplicate            = errors.New("order is duplicated")
	ErrOrderNotFound             = errors.New("order is not found")
	ErrPriceLevelDuplicate       = errors.New("price level is duplicated")
	ErrPriceLevelNotFound        = errors.New("price level is not found")
	ErrInvalidSymbol             = errors.New("invalid symbol")
	ErrInvalidOrderID            = errors.New("invalid order id")
	ErrInvalidOrderSide          = errors.New("invalid order side")
	ErrInvalidOrderDirection     = errors.New("invalid order direction")
	ErrInvalidOrderType          = errors.New("invalid order type")
	ErrInvalidOrderPrice         = errors.New("invalid order price")
	ErrInvalidOrderStopPrice     = errors.New("invalid order stop price")
	ErrInvalidOrderQuantity      = errors.New("invalid order quantity")
	ErrInvalidOrderQuoteQuantity = errors.New("invalid order quote quantity")
	ErrInvalidMarketSlippage     = errors.New("invalid market slippage")
	ErrForbiddenManualExecution  = errors.New("manual execution is forbidden for automatically matching engine")
	ErrOrderTreeNotFound         = errors.New("order tree not found")
	ErrNotEnoughLockedAmount     = errors.New("not enough locked amount for order")

	// OCO
	ErrBuyOCOStopPriceLessThanMarketPrice     = errors.New("stop price must be greater than market price (buy OCO order)")
	ErrBuyOCOLimitPriceGreaterThanMarketPrice = errors.New("limit order price must be less than market price (buy OCO order)")
	ErrSellOCOStopPriceGreaterThanMarketPrice = errors.New("stop price must be less than market price (sell OCO order)")
	ErrSellOCOLimitPriceLessThanMarketPrice   = errors.New("limit order price must be greater than market price (sell OCO order)")
	ErrOCOStopLimitNotZeroLocked              = errors.New("stop limit order locked must be zero: locked amount must be in limit order")

	// TPSL
	ErrBuySLStopPriceLessThanEnginePrice     = errors.New("stop price must be greater than engine price (buy stop-loss order)")
	ErrBuyTPStopPriceGreaterThanEnginePrice  = errors.New("stop price must be less than engine price (buy take-profit order)")
	ErrSellSLStopPriceGreaterThanEnginePrice = errors.New("stop price must be less than engine price (sell stop-loss order)")
	ErrSellTPStopPriceLessThanEnginePrice    = errors.New("stop price must be greater than engine price (sell take-profit order)")
	ErrSLNotZeroLocked                       = errors.New("stop limit order locked must be zero: locked amount must be in take profit")

	// Internal Errors
	ErrInternalExecutingOrderNotExecuted = errors.New("internal matching error: executing order not executed")
)

// OrderError is returned by a failed order book task and passed to Handler.OnError,
// so the handler knows which order the operation was about.
// For OrderOpAddPair OrderID is the stop-limit order and LinkedOrderID is the limit order.
type OrderError struct {
	Op            OrderOp
	OrderID       uint64
	LinkedOrderID uint64
	Err           error
}

func (e *OrderError) Error() string {
	if e.LinkedOrderID != 0 {
		return fmt.Sprintf("%s order %d (linked %d): %v", e.Op, e.OrderID, e.LinkedOrderID, e.Err)
	}
	return fmt.Sprintf("%s order %d: %v", e.Op, e.OrderID, e.Err)
}

func (e *OrderError) Unwrap() error {
	return e.Err
}

func withOrderError(task func(ob *OrderBook) error, op OrderOp, orderID, linkedOrderID uint64) func(ob *OrderBook) error {
	return func(ob *OrderBook) error {
		err := task(ob)
		if err != nil {
			return &OrderError{Op: op, OrderID: orderID, LinkedOrderID: linkedOrderID, Err: err}
		}
		return nil
	}
}
