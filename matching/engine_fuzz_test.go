package matching_test

import (
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"testing"

	matching "github.com/cryptonstudio/crypton-matching-engine/matching"
	mockmatching "github.com/cryptonstudio/crypton-matching-engine/matching/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

/*
func FuzzLimitTimeInForce(f *testing.F) {
	symbolIDs := []uint32{1, 2, 3}

	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, a []byte) {
		if len(a) == 0 {
			return
		}
		// 0: order side, 1: TIF, 2: price, 3: quantity
		if len(a)%4 != 0 {
			return
		}

		var orders []matching.Order

		for i := 0; i < len(a); i += 4 {
			side := matching.OrderSide(a[i])
			if !(side == matching.OrderSideBuy || side == matching.OrderSideSell) {
				return
			}
			tif := matching.OrderTimeInForce(a[i+1])
			if !(tif == matching.OrderTimeInForceGTC || tif == matching.OrderTimeInForceIOC ||
				tif == matching.OrderTimeInForceFOK) {
				return
			}
			price := matching.NewUint(uint64(a[i+2])).Mul64(matching.UintPrecision).Div64(10)
			quantity := matching.NewUint(uint64(a[i+3])).Mul64(matching.UintPrecision).Div64(10)
			restLocked := quantity
			if side == matching.OrderSideBuy {
				restLocked = quantity.Mul(price).Div64(matching.UintPrecision)
			}

			for j := range symbolIDs {
				orders = append(orders, matching.NewLimitOrder(
					symbolIDs[j], uint64(j*len(a)+i+1), side, tif, price, quantity, matching.NewZeroUint(), restLocked,
				))
			}
		}

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		setupHandler := func(t *testing.T) matching.Handler {
			handler := mockmatching.NewMockHandler(ctrl)
			setupMockHandler(t, handler)
			return handler
		}

		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		// BTC_USDT
		symbols := getSymbolsWithLimits()
		_, err := engine.AddOrderBook(matching.NewSymbolWithLimits(
			symbolIDs[0],
			symbols[0].Name,
			symbols[0].PriceLimits,
			symbols[0].LotSizeLimits),
			matching.NewUint(0),
			matching.StopPriceModeConfig{Market: true},
		)
		require.NoError(t, err)

		// ETH_USDT
		_, err = engine.AddOrderBook(matching.NewSymbolWithLimits(
			symbolIDs[1],
			symbols[1].Name,
			symbols[1].PriceLimits,
			symbols[1].LotSizeLimits),
			matching.NewUint(0),
			matching.StopPriceModeConfig{Market: true},
		)
		require.NoError(t, err)

		// simple OB
		_, err = engine.AddOrderBook(matching.NewSymbol(symbolIDs[2], ""), matching.NewUint(0), matching.StopPriceModeConfig{Market: true})
		require.NoError(t, err)

		defer func() {
			// recover from panic if one occurred. Set err to nil otherwise.
			if recover() != nil {
				t.Logf("orders set:\n")
				for i := range orders {
					t.Logf("id=%d side=%s, tif=%s, price=%s, quantity=%s\n",
						orders[i].ID(),
						orders[i].Side().String(),
						orders[i].TimeInForce().String(),
						orders[i].Price().ToFloatString(),
						orders[i].Quantity().ToFloatString(),
					)
				}
				t.Fail()
			}
		}()

		for i := range orders {
			engine.AddOrder(orders[i])
		}
	})
}
*/

func FuzzAllOrders(f *testing.F) {

	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, a []byte) {
		testAllOrders(t, a)
	})
}

func TestFailedExample(t *testing.T) {
	testAllOrders(t, []byte("0\x03000\x000001\x02\x02\x01\x01\x01\x010\x0300000000\x02\x02\x01\x010000000000"))
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
		if len(oo) == 1 {
			engine.AddOrder(oo[0])
		}
		if len(oo) == 2 {
			engine.AddOrdersPair(oo[0], oo[1])
		}
	}
}

// Data parsing:
// 2 bytes for 'float' numbers
// 1 byte for enums
// symbol: price min, max, step; lot min,max,step (=12b)
// order: type, side, tif, mod_quote, price, (quote)quantity, stop price, slippage, visible (=14b)
type allDataForFuzz struct {
	symbol         matching.Symbol
	ordersSequence [][]matching.Order
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
		if len(oo) == 2 {
			lines = append(lines, "OCO pair:")
		}
		for i := range oo {
			lines = append(lines, fmt.Sprintf("id=%d type=%s side=%s, tif=%s, price=%s, stop price=%s, quantity=%s quoteQuant=%s availableQty=%s restQty=%s",
				oo[i].ID(),
				oo[i].Type().String(),
				oo[i].Side().String(),
				oo[i].TimeInForce().String(),
				oo[i].Price().ToFloatString(),
				oo[i].StopPrice().ToFloatString(),
				oo[i].Quantity().ToFloatString(),
				oo[i].QuoteQuantity().ToFloatString(),
				oo[i].Available().ToFloatString(),
				oo[i].RestQuantity().ToFloatString(),
			))
		}
	}

	return strings.Join(lines, "\n")
}

func b2Uint(inp []byte) matching.Uint {
	return matching.NewUint(uint64(inp[0]) + 256*uint64(inp[1])).Mul64(matching.UintPrecision).Div64(1000)
}

func parseBytesToData(inp []byte) (allDataForFuzz, error) {
	if len(inp) <= 12 {
		return allDataForFuzz{}, errors.New("invalid input length")
	}
	if (len(inp)-12)%14 != 0 {
		return allDataForFuzz{}, errors.New("invalid input length")
	}

	// fake type to generate OCO case
	const orderTypeOCO matching.OrderType = 255
	const symbolID = 1

	var result allDataForFuzz

	result.symbol = matching.NewSymbolWithLimits(
		symbolID,
		"a",
		matching.Limits{
			Min:  b2Uint(inp[0:2]),
			Max:  b2Uint(inp[2:4]),
			Step: b2Uint(inp[4:6]),
		},
		matching.Limits{
			Min:  b2Uint(inp[6:8]),
			Max:  b2Uint(inp[8:10]),
			Step: b2Uint(inp[10:12]),
		},
	)
	if !result.symbol.Valid() {
		return allDataForFuzz{}, matching.ErrInvalidSymbol
	}

	id := uint64(0)
	for j := 12; j < len(inp); j += 14 {
		id++
		orderType := matching.OrderType(inp[j])
		if !(orderType == matching.OrderTypeLimit || orderType == matching.OrderTypeStopLimit ||
			orderType == matching.OrderTypeMarket || orderType == matching.OrderTypeStop || orderType == orderTypeOCO) {
			return allDataForFuzz{}, matching.ErrInvalidOrderType
		}

		side := matching.OrderSide(inp[j+1])
		if !(side == matching.OrderSideBuy || side == matching.OrderSideSell) {
			return allDataForFuzz{}, matching.ErrInvalidOrderSide
		}

		tif := matching.OrderTimeInForce(inp[j+2])
		if !(tif == matching.OrderTimeInForceGTC || tif == matching.OrderTimeInForceIOC ||
			tif == matching.OrderTimeInForceFOK) {
			return allDataForFuzz{}, errors.New("invalid time-in-force")
		}

		// for market orders: 0 - use quantity as quantity, 1 - use as quote quantity
		mod_quote := inp[j+3]
		if !(mod_quote == 0 || mod_quote == 1) {
			return allDataForFuzz{}, errors.New("invalid mod quote")
		}

		price := b2Uint(inp[j+4 : j+6])
		quantity := b2Uint(inp[j+6 : j+8])
		stopPrice := b2Uint(inp[j+8 : j+10])
		slippage := b2Uint(inp[j+10 : j+12])
		visible := b2Uint(inp[j+12 : j+14])

		if quantity.IsZero() {
			return allDataForFuzz{}, matching.ErrInvalidOrderQuantity
		}
		if (orderType == matching.OrderTypeStopLimit || orderType == orderTypeOCO) && price.Equals(stopPrice) {
			return allDataForFuzz{}, matching.ErrInvalidOrderStopPrice
		}

		restLocked := quantity
		if side == matching.OrderSideBuy && (orderType == matching.OrderTypeLimit || orderType == matching.OrderTypeStopLimit) {
			restLocked = quantity.Mul(price).Div64(matching.UintPrecision)
		}

		switch orderType {
		case matching.OrderTypeLimit:
			result.ordersSequence = append(result.ordersSequence, []matching.Order{matching.NewLimitOrder(
				symbolID, id, side, tif, price, quantity, visible, restLocked,
			)})
		case matching.OrderTypeStopLimit:
			result.ordersSequence = append(result.ordersSequence, []matching.Order{matching.NewStopLimitOrder(
				symbolID, id, side, tif, price, matching.StopPriceModeMarket, stopPrice, quantity, visible, restLocked,
			)})
		case matching.OrderTypeMarket:
			var q, qq matching.Uint
			if mod_quote == 0 {
				q = quantity
			} else {
				qq = quantity
			}
			result.ordersSequence = append(result.ordersSequence, []matching.Order{matching.NewMarketOrder(
				symbolID, id, side, matching.OrderTimeInForceIOC, q,
				qq, slippage, restLocked,
			)})
		case matching.OrderTypeStop:
			var q, qq matching.Uint
			if mod_quote == 0 {
				q = quantity
			} else {
				qq = quantity
			}
			result.ordersSequence = append(result.ordersSequence, []matching.Order{matching.NewStopOrder(
				symbolID, id, side, matching.OrderTimeInForceIOC,
				matching.StopPriceModeMarket, stopPrice, q,
				qq, slippage, restLocked,
			)})
		case orderTypeOCO:
			id1 := id
			id++
			id2 := id
			result.ordersSequence = append(result.ordersSequence, []matching.Order{
				matching.NewStopLimitOrder(
					symbolID,
					id1,
					side,
					tif,
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
					side,
					tif,
					price,
					quantity,
					visible,
					restLocked,
				),
			})
		}
	}

	return result, nil
}
