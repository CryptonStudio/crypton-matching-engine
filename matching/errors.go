package matching

import (
	"errors"
)

// var (
// 	ErrInvalidID            = errors.New("invalid id")
// 	ErrInvalidName          = errors.New("invalid name")
// 	ErrInvalidPrecision     = errors.New("invalid precision")
// 	ErrInvalidPrice         = errors.New("invalid price")
// 	ErrInvalidQuantity      = errors.New("invalid quantity")
// 	ErrInvalidSide          = errors.New("invalid side")
// 	ErrInvalidAsset         = errors.New("invalid asset")
// 	ErrInvalidSymbol        = errors.New("invalid symbol")
// 	ErrInsufficientQuantity = errors.New("insufficient quantity to calculate price")
// 	ErrAssetExists          = errors.New("asset already exists")
// 	ErrSymbolExists         = errors.New("symbol already exists")
// 	ErrOrderExists          = errors.New("order already exists")
// 	ErrOrderNotExists       = errors.New("order does not exist")
// 	ErrOrderBookExists      = errors.New("order book already exists")
// )

// Errors used by the package.
var (
	ErrOrderBookDuplicate       = errors.New("order book is duplicated")
	ErrOrderBookNotFound        = errors.New("order book is not found")
	ErrOrderDuplicate           = errors.New("order is duplicated")
	ErrOrderNotFound            = errors.New("order is not found")
	ErrPriceLevelDuplicate      = errors.New("price level is duplicated")
	ErrPriceLevelNotFound       = errors.New("price level is not found")
	ErrInvalidOrderID           = errors.New("invalid order id")
	ErrInvalidOrderSide         = errors.New("invalid order side")
	ErrInvalidOrderType         = errors.New("invalid order type")
	ErrInvalidOrderParameter    = errors.New("invalid order parameter")
	ErrInvalidOrderQuantity     = errors.New("invalid order quantity")
	ErrForbiddenManualExecution = errors.New("manual execution is forbidden for automatically matching engine")
)
