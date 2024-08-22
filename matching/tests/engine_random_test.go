package matching_test

import (
	"errors"
	"math/rand"
	"testing"
	"time"

	matching "github.com/cryptonstudio/crypton-matching-engine/matching"
	"github.com/stretchr/testify/require"
)

func TestMemoryDmg(t *testing.T) {
	t.SkipNow()
	const N = 10_000
	symIDS := []uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	symbols := generateSymbols(symIDS)

	orders, err := generateOrderSequence(1, N, symIDS)
	require.NoError(t, err)

	stor := newFuzzStorage(t)
	for _, oo := range orders {
		if len(oo.orders) == 1 {
			stor.addOrder(oo.orders[0], 0)
		}
		if len(oo.orders) == 2 {
			stor.addOrder(oo.orders[0], oo.orders[1].ID())
			stor.addOrder(oo.orders[1], oo.orders[0].ID())
		}
	}

	engine := matching.NewEngine(stor, true)
	engine.EnableMatching()

	for _, s := range symbols {
		_, err = engine.AddOrderBook(s,
			matching.NewUint(0),
			matching.StopPriceModeConfig{Market: true, Mark: false, Index: false},
		)
		require.NoError(t, err)
	}

	for _, oo := range orders {
		if len(oo.orders) == 1 {
			err = engine.AddOrder(oo.orders[0])
		}
		if len(oo.orders) == 2 {
			switch {
			case oo.orderType == orderTypeOCO:
				err = engine.AddOrdersPair(oo.orders[0], oo.orders[1])
			case oo.orderType == orderTypeTPSLLimit:
				err = engine.AddTPSL(oo.orders[0], oo.orders[1])
			case oo.orderType == orderTypeTPSLMarket:
				err = engine.AddTPSLMarket(oo.orders[0], oo.orders[1])
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

	time.Sleep(time.Second * 10)

	for _, id := range symIDS {
		ob := engine.OrderBook(id)
		for orderPtr := ob.TopAsk().Value().Queue().Front(); orderPtr != nil; {
			orderPtrNext := orderPtr.Next()
			order := orderPtr.Value
			require.True(t, !order.Available().IsZero(), "symbol %d, order %d avail=0", id, order.ID())
			require.Equal(t, matching.OrderSideSell, order.Side(), "symbol %d, order %d side", id, order.ID())
			orderPtr = orderPtrNext
		}
		for orderPtr := ob.TopBid().Value().Queue().Front(); orderPtr != nil; {
			orderPtrNext := orderPtr.Next()
			order := orderPtr.Value
			require.True(t, !order.Available().IsZero(), "symbol %d, order %d avail=0", id, order.ID())
			require.Equal(t, matching.OrderSideSell, order.Side(), "symbol %d, order %d side", id, order.ID())
			orderPtr = orderPtrNext
		}
	}

	engine.Stop(true)
	time.Sleep(time.Second * 10)
}

func generateSymbols(ids []uint32) []matching.Symbol {
	var result []matching.Symbol

	for _, id := range ids {
		result = append(result, matching.NewSymbolWithLimits(
			id,
			"a",
			matching.Limits{
				Min:  matching.NewUint(100000000000),
				Max:  matching.NewUint(100000000000000000),
				Step: matching.NewUint(100000000000),
			},
			matching.Limits{
				Min:  matching.NewUint(1000000000),
				Max:  matching.NewUint(900000000000000000),
				Step: matching.NewUint(1000000000),
			},
		))
	}

	return result
}

func randomChoice[T any](list []T) T {
	var empty T
	if len(list) == 0 {
		return empty
	}

	return list[rand.Intn(len(list))]
}

func randomRange[T int | int64 | uint64 | uint32](down, up T) T {
	return T(rand.Int63n(int64(up-down)) + int64(down))
}

func generateOrderSequence(id uint64, n int, symbols []uint32) ([]sequenceItem, error) {
	var result []sequenceItem
	for j := 0; j < n; j += 1 {
		id++

		symbolID := randomChoice(symbols)
		side := randomChoice([]matching.OrderSide{matching.OrderSideBuy, matching.OrderSideSell})
		typ := randomChoice([]matching.OrderType{
			matching.OrderTypeLimit,
			// matching.OrderTypeStopLimit,
			// matching.OrderTypeMarket,
			// matching.OrderTypeStop,
			// orderTypeOCO,
			// orderTypeTPSLLimit,
			// orderTypeTPSLMarket,
		})
		dir := randomChoice([]matching.OrderDirection{matching.OrderDirectionClose, matching.OrderDirectionOpen})
		tif := randomChoice([]matching.OrderTimeInForce{matching.OrderTimeInForceFOK, matching.OrderTimeInForceGTC, matching.OrderTimeInForceIOC})
		priceMode := randomChoice([]matching.StopPriceMode{matching.StopPriceModeIndex, matching.StopPriceModeMark, matching.StopPriceModeMarket})
		tpMode := randomChoice([]matching.StopPriceMode{matching.StopPriceModeIndex, matching.StopPriceModeMark, matching.StopPriceModeMarket})
		slMode := randomChoice([]matching.StopPriceMode{matching.StopPriceModeIndex, matching.StopPriceModeMark, matching.StopPriceModeMarket})
		modQuote := randomChoice([]uint8{0, 1})

		price := matching.NewUint(randomRange[uint64](1, 1000000)).Mul64(matching.UintPrecision).Div64(100)
		quantity := matching.NewUint(randomRange[uint64](1, 1000000)).Mul64(matching.UintPrecision).Div64(100)
		stopPrice := matching.NewUint(randomRange[uint64](1, 1000000)).Mul64(matching.UintPrecision).Div64(100000)
		slippage := matching.NewUint(randomRange[uint64](1, 1000000)).Mul64(matching.UintPrecision).Div64(100000)
		visible := matching.NewUint(randomRange[uint64](1, 1000000)).Mul64(matching.UintPrecision).Div64(100000)
		tpPrice := matching.NewUint(randomRange[uint64](1, 1000000)).Mul64(matching.UintPrecision).Div64(100000)
		slPrice := matching.NewUint(randomRange[uint64](1, 1000000)).Mul64(matching.UintPrecision).Div64(100000)
		tpStopPrice := matching.NewUint(randomRange[uint64](1, 1000000)).Mul64(matching.UintPrecision).Div64(100000)
		slStopPrice := matching.NewUint(randomRange[uint64](1, 1000000)).Mul64(matching.UintPrecision).Div64(100000)
		tpQuantity := matching.NewUint(randomRange[uint64](1, 1000000)).Mul64(matching.UintPrecision).Div64(100000)
		slQuantity := matching.NewUint(randomRange[uint64](1, 1000000)).Mul64(matching.UintPrecision).Div64(100000)
		tpSlippage := matching.NewUint(randomRange[uint64](1, 1000000)).Mul64(matching.UintPrecision).Div64(100000)
		slSlippage := matching.NewUint(randomRange[uint64](1, 1000000)).Mul64(matching.UintPrecision).Div64(100000)

		if (typ == matching.OrderTypeStopLimit || typ == orderTypeOCO) && price.Equals(stopPrice) {
			return nil, matching.ErrInvalidOrderStopPrice
		}

		restLocked := quantity
		if dir == matching.OrderDirectionOpen {
			if typ == matching.OrderTypeLimit || typ == matching.OrderTypeStopLimit {
				restLocked = quantity.Mul(price).Div64(matching.UintPrecision)
			}
			if typ == orderTypeOCO {
				restLocked = matching.Max(quantity.Mul(price).Div64(matching.UintPrecision), quantity.Mul(stopPrice).Div64(matching.UintPrecision))
			}
			if typ == orderTypeTPSLLimit {
				restLocked = matching.Max(quantity.Mul(tpPrice).Div64(matching.UintPrecision), quantity.Mul(slPrice).Div64(matching.UintPrecision))
			}
		}

		switch typ {
		case matching.OrderTypeLimit:
			result = append(result, sequenceItem{
				typ,
				[]matching.Order{matching.NewLimitOrder(
					symbolID, id, side, dir, tif, price, quantity, matching.NewMaxUint(), restLocked,
				)}})
		case matching.OrderTypeStopLimit:
			result = append(result, sequenceItem{
				typ,
				[]matching.Order{matching.NewStopLimitOrder(
					symbolID, id, side, dir, tif, price, priceMode, stopPrice, quantity, visible, restLocked,
				)}})
		case matching.OrderTypeMarket:
			result = append(result, sequenceItem{
				typ,
				[]matching.Order{matching.NewMarketOrder(
					symbolID, id, side, dir, matching.OrderTimeInForceIOC, modQQ(modQuote, 0, quantity),
					modQQ(modQuote, 1, quantity), slippage, restLocked,
				)}})
		case matching.OrderTypeStop:
			result = append(result, sequenceItem{
				typ,
				[]matching.Order{matching.NewStopOrder(symbolID, id, side, dir, matching.OrderTimeInForceIOC,
					priceMode, stopPrice, modQQ(modQuote, 0, quantity),
					modQQ(modQuote, 1, quantity), slippage, restLocked,
				)}})
		case orderTypeOCO:
			id1 := id
			id++
			id2 := id
			result = append(result, sequenceItem{
				typ,
				[]matching.Order{
					matching.NewStopLimitOrder(symbolID, id1, side, dir, tif, price,
						priceMode, stopPrice,
						quantity, visible, matching.NewZeroUint(),
					),
					matching.NewLimitOrder(symbolID, id2, side, dir, tif, price,
						quantity, visible, restLocked,
					),
				}})
		case orderTypeTPSLLimit:
			id1 := id
			id++
			id2 := id
			result = append(result, sequenceItem{
				typ,
				[]matching.Order{
					matching.NewStopLimitOrder(symbolID, id1, side, dir, tif, tpPrice,
						tpMode, tpStopPrice,
						quantity, visible, restLocked,
					),
					matching.NewStopLimitOrder(symbolID, id2, side, dir, tif, slPrice,
						slMode, slStopPrice,
						quantity, visible, matching.NewZeroUint(),
					),
				}})
		case orderTypeTPSLMarket:
			restLocked = matching.Max(tpQuantity, slQuantity)

			id1 := id
			id++
			id2 := id
			result = append(result, sequenceItem{
				typ,
				[]matching.Order{
					matching.NewStopOrder(symbolID, id1, side, dir, matching.OrderTimeInForceIOC,
						tpMode, tpStopPrice,
						modQQ(modQuote, 0, tpQuantity),
						modQQ(modQuote, 1, tpQuantity),
						tpSlippage, restLocked,
					),
					matching.NewStopOrder(symbolID, id2, side, dir, matching.OrderTimeInForceIOC,
						slMode, slStopPrice,
						modQQ(modQuote, 0, slQuantity),
						modQQ(modQuote, 1, slQuantity),
						slSlippage, matching.NewZeroUint(),
					),
				}})
		}
	}

	return result, nil
}
