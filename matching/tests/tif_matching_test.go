package matching_test

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	matching "github.com/cryptonstudio/crypton-matching-engine/matching"
	mockmatching "github.com/cryptonstudio/crypton-matching-engine/matching/mocks"
)

func TestTimeInForce(t *testing.T) {
	const (
		symbolID   uint32 = 10
		orderID    uint64 = 100
		newOrderID uint64 = 101
	)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	setupHandler := func(t *testing.T) matching.Handler {
		handler := mockmatching.NewMockHandler(ctrl)
		setupMockHandler(t, handler)
		return handler
	}

	// GTC
	t.Run("GTC - create, match, cancel", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		ob, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0), matching.StopPriceModeConfig{Market: true, Mark: true, Index: true})
		require.NoError(t, err)

		// place in empty OB
		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
			matching.OrderDirectionOpen,
			matching.OrderTimeInForceGTC,
			matching.NewUint(30).Mul64(matching.UintPrecision), // price 30
			matching.NewUint(10).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
		require.Equal(t, matching.OrderTypeLimit, ob.Order(5).Type()) // check: order is placed

		// match part of the order
		partQty := matching.NewUint(5).Mul64(matching.UintPrecision)
		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(6),
			matching.OrderSideSell,
			matching.OrderDirectionClose,
			matching.OrderTimeInForceGTC,
			matching.NewUint(30).Mul64(matching.UintPrecision), // price 30
			partQty,
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
		require.Equal(t, matching.OrderTypeLimit, ob.Order(5).Type())   // check: order is still placed
		require.True(t, ob.Order(5).ExecutedQuantity().Equals(partQty)) // check: partQty is executed

		// cancel the order
		err = engine.DeleteOrder(symbolID, 5)
		require.NoError(t, err)
		require.Nil(t, ob.Order(5)) // check: order is cancelled
	})

	// IOC
	t.Run("IOC - empty OB", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		ob, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0), matching.StopPriceModeConfig{Market: true, Mark: true, Index: true})
		require.NoError(t, err)

		// place in empty OB
		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
			matching.OrderDirectionOpen,
			matching.OrderTimeInForceIOC,
			matching.NewUint(30).Mul64(matching.UintPrecision), // price 30
			matching.NewUint(10).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
		require.Nil(t, ob.Order(5)) // check: order is cancelled
	})

	t.Run("IOC - prepared OB for partial match", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		ob, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0), matching.StopPriceModeConfig{Market: true, Mark: true, Index: true})
		require.NoError(t, err)

		// prepare OB with GTC
		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
			matching.OrderDirectionOpen,
			matching.OrderTimeInForceGTC,
			matching.NewUint(30).Mul64(matching.UintPrecision), // price 30
			matching.NewUint(5).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
		require.Equal(t, matching.OrderTypeLimit, ob.Order(5).Type()) // check: order is placed

		// place in prepared OB
		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(6),
			matching.OrderSideSell,
			matching.OrderDirectionClose,
			matching.OrderTimeInForceIOC,
			matching.NewUint(30).Mul64(matching.UintPrecision), // price 30
			matching.NewUint(10).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
		require.Nil(t, ob.Order(5)) // check: gtc order is executed
		require.Nil(t, ob.Order(6)) // check: ioc order is cancelled
	})

	t.Run("IOC - prepared OB for full match", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		ob, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0), matching.StopPriceModeConfig{Market: true, Mark: true, Index: true})
		require.NoError(t, err)

		// prepare OB with GTC
		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
			matching.OrderDirectionOpen,
			matching.OrderTimeInForceGTC,
			matching.NewUint(30).Mul64(matching.UintPrecision), // price 30
			matching.NewUint(15).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
		require.Equal(t, matching.OrderTypeLimit, ob.Order(5).Type()) // check: order is placed

		// place in prepared OB
		partQty := matching.NewUint(5).Mul64(matching.UintPrecision)
		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(6),
			matching.OrderSideSell,
			matching.OrderDirectionClose,
			matching.OrderTimeInForceIOC,
			matching.NewUint(30).Mul64(matching.UintPrecision), // price 30
			partQty,
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
		require.True(t, ob.Order(5).ExecutedQuantity().Equals(partQty)) // check: partQty is executed
		require.Nil(t, ob.Order(6))                                     // check: ioc order is executed
	})

	// FOK
	t.Run("FOK - empty OB", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		ob, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0), matching.StopPriceModeConfig{Market: true, Mark: true, Index: true})
		require.NoError(t, err)

		// place in empty OB
		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
			matching.OrderDirectionOpen,
			matching.OrderTimeInForceFOK,
			matching.NewUint(30).Mul64(matching.UintPrecision), // price 30
			matching.NewUint(10).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
		require.Nil(t, ob.Order(5)) // check: order is cancelled
	})

	t.Run("FOK - prepared OB for partial match", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		ob, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0), matching.StopPriceModeConfig{Market: true, Mark: true, Index: true})
		require.NoError(t, err)

		// prepare OB with GTC
		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
			matching.OrderDirectionOpen,
			matching.OrderTimeInForceGTC,
			matching.NewUint(30).Mul64(matching.UintPrecision), // price 30
			matching.NewUint(5).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
		require.Equal(t, matching.OrderTypeLimit, ob.Order(5).Type()) // check: order is placed

		// place in prepared OB
		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(6),
			matching.OrderSideSell,
			matching.OrderDirectionClose,
			matching.OrderTimeInForceFOK,
			matching.NewUint(30).Mul64(matching.UintPrecision), // price 30
			matching.NewUint(10).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
		require.True(t, ob.Order(5).ExecutedQuantity().IsZero()) // check: gtc order is not executed
		require.Nil(t, ob.Order(6))                              // check: fok order is cancelled
	})

	t.Run("FOK - prepared OB for full match", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		ob, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0), matching.StopPriceModeConfig{Market: true, Mark: true, Index: true})
		require.NoError(t, err)

		// prepare OB with GTC
		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
			matching.OrderDirectionOpen,
			matching.OrderTimeInForceGTC,
			matching.NewUint(30).Mul64(matching.UintPrecision), // price 30
			matching.NewUint(15).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
		require.Equal(t, matching.OrderTypeLimit, ob.Order(5).Type()) // check: order is placed

		// place in prepared OB
		partQty := matching.NewUint(5).Mul64(matching.UintPrecision)
		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(6),
			matching.OrderSideSell,
			matching.OrderDirectionClose,
			matching.OrderTimeInForceFOK,
			matching.NewUint(30).Mul64(matching.UintPrecision), // price 30
			partQty,
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
		require.True(t, ob.Order(5).ExecutedQuantity().Equals(partQty)) // check: partQty is executed
		require.Nil(t, ob.Order(6))                                     // check: fok order is executed
	})
}

func TestTimeInForceFOK(t *testing.T) {
	const (
		symbolID uint32 = 10
	)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// GTC state: prices 29 (#1), 30 (#2), 31 (#3) with amount 5
	// FOK price 30 (#100), cases:
	// 1. buy, amount 9, should execute, #1 is executed full, #2 is executed 4
	// 2. buy, amount 11, should not execute, all GTC are not executed
	// 3. sell, amount 9, should execute, #3 is executed full, #2 is executed 4
	// 4. sell, amount 11, should not execute, all GTC are not executed
	gtcState := []struct {
		price   matching.Uint
		orderID uint64
	}{
		{matching.NewUint(29).Mul64(matching.UintPrecision), 1},
		{matching.NewUint(30).Mul64(matching.UintPrecision), 2},
		{matching.NewUint(31).Mul64(matching.UintPrecision), 3},
	}

	testCases := []struct {
		name      string
		fokSide   matching.OrderSide
		amount    matching.Uint
		gtcResult []string // coded states and amounts
	}{
		{"1:buy,9", matching.OrderSideBuy, matching.NewUint(9).Mul64(matching.UintPrecision), []string{"", "4", "0"}},
		{"2:buy,11", matching.OrderSideBuy, matching.NewUint(11).Mul64(matching.UintPrecision), []string{"0", "0", "0"}},
		{"3:sell,9", matching.OrderSideSell, matching.NewUint(9).Mul64(matching.UintPrecision), []string{"0", "4", ""}},
		{"4:sell,11", matching.OrderSideSell, matching.NewUint(11).Mul64(matching.UintPrecision), []string{"0", "0", "0"}},
	}

	handler := mockmatching.NewMockHandler(ctrl)
	setupMockHandler(t, handler)

	engine := matching.NewEngine(handler, false)
	engine.EnableMatching()

	ob, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0), matching.StopPriceModeConfig{Market: true, Mark: true, Index: true})
	require.NoError(t, err)
	handler.EXPECT().OnError(ob, gomock.Any()).AnyTimes()

	for _, tc := range testCases {
		for _, g := range gtcState {
			engine.DeleteOrder(symbolID, g.orderID) //nolint:errcheck
		}
		ob.Clean()

		gtcSide := matching.OrderSideSell
		if tc.fokSide == matching.OrderSideSell {
			gtcSide = matching.OrderSideBuy
		}
		// prepare OB with GTC
		for _, g := range gtcState {
			err = engine.AddOrder(matching.NewLimitOrder(
				symbolID,
				g.orderID,
				gtcSide,
				matching.OrderDirectionOpen,
				matching.OrderTimeInForceGTC,
				g.price,
				matching.NewUint(5).Mul64(matching.UintPrecision),
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			))
			require.NoError(t, err, tc.name)
			require.Equal(t, matching.OrderTypeLimit, ob.Order(g.orderID).Type(), tc.name) // check: order is placed
		}
		// place in prepared OB
		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(100),
			tc.fokSide,
			matching.OrderDirectionOpen,
			matching.OrderTimeInForceFOK,
			matching.NewUint(30).Mul64(matching.UintPrecision),
			tc.amount,
			matching.NewMaxUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err, tc.name)

		// check
		require.Nil(t, ob.Order(100)) // check: fok order is cancelled/executed

		for i := range gtcState {
			id := gtcState[i].orderID
			if tc.gtcResult[i] == "" {
				require.Nil(t, ob.Order(id), tc.name+fmt.Sprintf(" order #%d not nil", id))
			} else {
				expect, _ := matching.NewUintFromFloatString(tc.gtcResult[i])
				actual := ob.Order(id).ExecutedQuantity()
				require.True(t, actual.Equals(expect),
					tc.name+fmt.Sprintf(" order #%d, expect=%s, actual=%s",
						id, expect.ToFloatString(), actual.ToFloatString()),
				)
			}
		}
	}
}
