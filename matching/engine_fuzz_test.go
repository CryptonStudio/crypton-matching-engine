package matching_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"testing"
	"unsafe"

	matching "github.com/cryptonstudio/crypton-matching-engine/matching"
	mockmatching "github.com/cryptonstudio/crypton-matching-engine/matching/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func FuzzAllOrders(f *testing.F) {

	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, a []byte) {
		testAllOrders(t, a)
	})
}

func TestFailedExample(t *testing.T) {
	testAllOrders(t, []byte("0\x000a\x02\x0000010\x00\x01\x02\x01\x010000000000\x02\x01\x01\x010000000000\x03\x01\x01\x010000000000"))
}

func testAllOrders(t *testing.T, a []byte) {
	data, err := parseBytesToData(a)
	if err != nil {
		return
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	setupHandler := func(t *testing.T) matching.Handler {
		handler := mockmatching.NewMockHandler(ctrl)
		setupMockHandler(t, handler)
		// to skip oco invalid pairs
		handler.EXPECT().OnError(gomock.Any(), gomock.Any()).AnyTimes()
		return handler
	}

	engine := matching.NewEngine(setupHandler(t), false)
	engine.EnableMatching()

	_, err = engine.AddOrderBook(data.symbol,
		matching.NewUint(0),
		matching.StopPriceModeConfig{Market: true},
	)
	require.NoError(t, err)

	defer func() {
		// recover from panic if one occurred. Set err to nil otherwise.
		if recover() != nil {
			t.Logf("stacktrace from panic:\n%s\n", string(debug.Stack()))
			t.Log("engine set:\n")
			t.Log(data.String() + "n")
			t.Fail()
		}
	}()

	t.Log(data.String())

	for _, oo := range data.ordersSequence {
		if len(oo.orders) == 1 {
			err = engine.AddOrder(oo.orders[0])
		}
		if len(oo.orders) == 2 {
			switch {
			case oo.orderType == orderTypeOCO:
				err = engine.AddOrdersPair(oo.orders[0], oo.orders[1])
			case oo.orderType == orderTypeTPSL:
				err = engine.AddTPSL(oo.orders[0], oo.orders[1])
			}
		}

		if err != nil && !errors.Is(err, matching.ErrInvalidOrderPrice) &&
			!errors.Is(err, matching.ErrInvalidOrderQuantity) &&
			!errors.Is(err, matching.ErrInvalidOrderQuoteQuantity) &&
			!errors.Is(err, matching.ErrInvalidMarketSlippage) &&
			!errors.Is(err, matching.ErrInvalidOrderStopPrice) &&
			!errors.Is(err, matching.ErrBuyOCOStopPriceLessThanMarketPrice) &&
			!errors.Is(err, matching.ErrBuyOCOLimitPriceGreaterThanMarketPrice) &&
			!errors.Is(err, matching.ErrSellOCOStopPriceGreaterThanMarketPrice) &&
			!errors.Is(err, matching.ErrSellOCOLimitPriceLessThanMarketPrice) &&
			!errors.Is(err, matching.ErrSellSLStopPriceGreaterThanEnginePrice) &&
			!errors.Is(err, matching.ErrSellTPStopPriceLessThanEnginePrice) &&
			!errors.Is(err, matching.ErrBuySLStopPriceLessThanEnginePrice) &&
			!errors.Is(err, matching.ErrBuyTPStopPriceGreaterThanEnginePrice) {
			t.Logf("error: %s", err)
			t.FailNow()
		}
	}
}

// Data parsing:
// 2 bytes for 'float' numbers
// 1 byte for enums
const orderTypeOCO matching.OrderType = 255
const orderTypeTPSL matching.OrderType = 254

type allDataForFuzz struct {
	symbol         matching.Symbol
	ordersSequence []sequenceItem
}

type sequenceItem struct {
	orderType matching.OrderType
	orders    []matching.Order
}

func (a allDataForFuzz) validate() error {
	if len(a.ordersSequence) > 0 {
		if a.ordersSequence[0].orderType == orderTypeTPSL {
			return errors.New("TPSL order should be second record")
		}
	}
	return nil
}

func (a allDataForFuzz) String() string {
	lines := []string{}
	lines = append(lines, fmt.Sprintf("symbol: price.min=%s, price.max=%s, price.step=%s, lot.min=%s, lot.max=%s, lot.step=%s",
		a.symbol.PriceLimits().Min.ToFloatString(),
		a.symbol.PriceLimits().Max.ToFloatString(),
		a.symbol.PriceLimits().Step.ToFloatString(),
		a.symbol.LotSizeLimits().Min.ToFloatString(),
		a.symbol.LotSizeLimits().Max.ToFloatString(),
		a.symbol.LotSizeLimits().Step.ToFloatString(),
	))
	for _, oo := range a.ordersSequence {
		if len(oo.orders) == 2 {
			lines = append(lines, "orders pair:")
		}
		for i := range oo.orders {
			lines = append(lines, fmt.Sprintf("id=%d type=%s side=%s, tif=%s, price=%s, stop price=%s, quantity=%s quoteQuant=%s availableQty=%s restQty=%s",
				oo.orders[i].ID(),
				oo.orders[i].Type().String(),
				oo.orders[i].Side().String(),
				oo.orders[i].TimeInForce().String(),
				oo.orders[i].Price().ToFloatString(),
				oo.orders[i].StopPrice().ToFloatString(),
				oo.orders[i].Quantity().ToFloatString(),
				oo.orders[i].QuoteQuantity().ToFloatString(),
				oo.orders[i].Available().ToFloatString(),
				oo.orders[i].RestQuantity().ToFloatString(),
			))
		}
	}

	return strings.Join(lines, "\n")
}

func uint16Uint(v uint16) matching.Uint {
	return matching.NewUint(uint64(v)).Mul64(matching.UintPrecision).Div64(1000)
}

type symbolConfig struct {
	PriceMin  uint16
	PriceMax  uint16
	PriceStep uint16
	LotMin    uint16
	LotMax    uint16
	LotStep   uint16
}

const symbolConfigSize = int(unsafe.Sizeof(symbolConfig{}))

type orderData struct {
	Type        matching.OrderType
	Side        matching.OrderSide
	TIF         matching.OrderTimeInForce
	ModQuote    uint8
	Price       uint16
	Quantity    uint16
	StopPrice   uint16
	Slippage    uint16
	Visible     uint16
	TpStopPrice uint16
	TpPrice     uint16
	SlStopPrice uint16
	SlPrice     uint16
}

const orderDataSize = int(unsafe.Sizeof(orderData{}))

func parseBytesToData(inp []byte) (allDataForFuzz, error) {
	if len(inp) <= symbolConfigSize {
		return allDataForFuzz{}, errors.New("invalid input length")
	}
	if (len(inp)-symbolConfigSize)%orderDataSize != 0 {
		return allDataForFuzz{}, errors.New("invalid input length")
	}

	// 2 orders at least
	if (len(inp)-symbolConfigSize)/orderDataSize < 2 {
		return allDataForFuzz{}, errors.New("need more orders")
	}

	const symbolID = 1

	buf := bytes.NewReader(inp)

	var cfg symbolConfig
	err := binary.Read(buf, binary.BigEndian, &cfg)
	if err != nil {
		return allDataForFuzz{}, err
	}

	var result allDataForFuzz

	result.symbol = matching.NewSymbolWithLimits(
		symbolID,
		"a",
		matching.Limits{
			Min:  uint16Uint(cfg.PriceMin),
			Max:  uint16Uint(cfg.PriceMax),
			Step: uint16Uint(cfg.PriceStep),
		},
		matching.Limits{
			Min:  uint16Uint(cfg.LotMin),
			Max:  uint16Uint(cfg.LotMax),
			Step: uint16Uint(cfg.LotStep),
		},
	)
	if !result.symbol.Valid() {
		return allDataForFuzz{}, matching.ErrInvalidSymbol
	}

	id := uint64(0)
	for j := symbolConfigSize; j < len(inp); j += orderDataSize {
		id++
		var dt orderData
		err := binary.Read(buf, binary.BigEndian, &dt)
		if err != nil {
			return allDataForFuzz{}, err
		}

		if !(dt.Type == matching.OrderTypeLimit || dt.Type == matching.OrderTypeStopLimit ||
			dt.Type == matching.OrderTypeMarket || dt.Type == matching.OrderTypeStop ||
			dt.Type == orderTypeOCO || dt.Type == orderTypeTPSL) {
			return allDataForFuzz{}, matching.ErrInvalidOrderType
		}

		if !(dt.Side == matching.OrderSideBuy || dt.Side == matching.OrderSideSell) {
			return allDataForFuzz{}, matching.ErrInvalidOrderSide
		}

		if !(dt.TIF == matching.OrderTimeInForceGTC || dt.TIF == matching.OrderTimeInForceIOC ||
			dt.TIF == matching.OrderTimeInForceFOK) {
			return allDataForFuzz{}, errors.New("invalid time-in-force")
		}

		// for market orders: 0 - use quantity as quantity, 1 - use as quote quantity
		if !(dt.ModQuote == 0 || dt.ModQuote == 1) {
			return allDataForFuzz{}, errors.New("invalid mod quote")
		}

		price := uint16Uint(dt.Price)
		quantity := uint16Uint(dt.Quantity)
		stopPrice := uint16Uint(dt.StopPrice)
		slippage := uint16Uint(dt.Slippage)
		visible := uint16Uint(dt.Visible)
		tpStopPrice := uint16Uint(dt.TpStopPrice)
		tpPrice := uint16Uint(dt.TpPrice)
		slStopPrice := uint16Uint(dt.SlStopPrice)
		slPrice := uint16Uint(dt.SlPrice)

		if quantity.IsZero() {
			return allDataForFuzz{}, matching.ErrInvalidOrderQuantity
		}
		if (dt.Type == matching.OrderTypeStopLimit || dt.Type == orderTypeOCO) && price.Equals(stopPrice) {
			return allDataForFuzz{}, matching.ErrInvalidOrderStopPrice
		}

		restLocked := quantity
		if dt.Side == matching.OrderSideBuy && (dt.Type == matching.OrderTypeLimit || dt.Type == matching.OrderTypeStopLimit) {
			restLocked = quantity.Mul(price).Div64(matching.UintPrecision)
		}

		switch dt.Type {
		case matching.OrderTypeLimit:
			result.ordersSequence = append(result.ordersSequence, sequenceItem{
				dt.Type,
				[]matching.Order{matching.NewLimitOrder(
					symbolID, id, dt.Side, dt.TIF, price, quantity, visible, restLocked,
				)}})
		case matching.OrderTypeStopLimit:
			result.ordersSequence = append(result.ordersSequence, sequenceItem{
				dt.Type,
				[]matching.Order{matching.NewStopLimitOrder(
					symbolID, id, dt.Side, dt.TIF, price, matching.StopPriceModeMarket, stopPrice, quantity, visible, restLocked,
				)}})
		case matching.OrderTypeMarket:
			var q, qq matching.Uint
			if dt.ModQuote == 0 {
				q = quantity
			} else {
				qq = quantity
			}
			result.ordersSequence = append(result.ordersSequence, sequenceItem{
				dt.Type,
				[]matching.Order{matching.NewMarketOrder(
					symbolID, id, dt.Side, matching.OrderTimeInForceIOC, q,
					qq, slippage, restLocked,
				)}})
		case matching.OrderTypeStop:
			var q, qq matching.Uint
			if dt.ModQuote == 0 {
				q = quantity
			} else {
				qq = quantity
			}
			result.ordersSequence = append(result.ordersSequence, sequenceItem{
				dt.Type,
				[]matching.Order{matching.NewStopOrder(
					symbolID, id, dt.Side, matching.OrderTimeInForceIOC,
					matching.StopPriceModeMarket, stopPrice, q,
					qq, slippage, restLocked,
				)}})
		case orderTypeOCO:
			restLocked := quantity
			if dt.Side == matching.OrderSideBuy {
				restLocked = matching.Max(quantity.Mul(price).Div64(matching.UintPrecision), quantity.Mul(stopPrice).Div64(matching.UintPrecision))
			}

			id1 := id
			id++
			id2 := id
			result.ordersSequence = append(result.ordersSequence, sequenceItem{
				dt.Type,
				[]matching.Order{
					matching.NewStopLimitOrder(
						symbolID,
						id1,
						dt.Side,
						dt.TIF,
						price,
						matching.StopPriceModeMarket,
						stopPrice,
						quantity,
						visible,
						matching.NewZeroUint(),
					),
					matching.NewLimitOrder(
						symbolID,
						id2,
						dt.Side,
						dt.TIF,
						price,
						quantity,
						visible,
						restLocked,
					),
				}})
		case orderTypeTPSL:
			restLocked := quantity
			if dt.Side == matching.OrderSideBuy {
				restLocked = matching.Max(quantity.Mul(tpPrice).Div64(matching.UintPrecision), quantity.Mul(slPrice).Div64(matching.UintPrecision))
			}

			id1 := id
			id++
			id2 := id
			result.ordersSequence = append(result.ordersSequence, sequenceItem{
				dt.Type,
				[]matching.Order{
					matching.NewStopLimitOrder(
						symbolID,
						id1,
						dt.Side,
						dt.TIF,
						tpPrice,
						matching.StopPriceModeMarket,
						tpStopPrice,
						quantity,
						visible,
						restLocked,
					),
					matching.NewStopLimitOrder(
						symbolID,
						id2,
						dt.Side,
						dt.TIF,
						slPrice,
						matching.StopPriceModeMarket,
						slStopPrice,
						quantity,
						visible,
						matching.NewZeroUint(),
					),
				}})
		}
	}

	if err = result.validate(); err != nil {
		return allDataForFuzz{}, err
	}

	return result, nil
}
