package matching_test

import (
	"testing"

	matching "github.com/cryptonstudio/crypton-matching-engine/matching"
	mockmatching "github.com/cryptonstudio/crypton-matching-engine/matching/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestStopLimitOrdersMatching(t *testing.T) {
	const (
		symbolID uint32 = 10
	)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	setupHandler := func(t *testing.T) matching.Handler {
		handler := mockmatching.NewMockHandler(ctrl)
		setupMockHandler(t, handler)
		return handler
	}

	t.Run("buy (stop price == market price)", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)
		ob := engine.OrderBook(symbolID)

		err := engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
			matching.OrderDirectionOpen,
			matching.OrderTimeInForceGTC,
			matching.NewUint(30).Mul64(matching.UintPrecision), // price 30
			matching.NewUint(1).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))

		require.NoError(t, err)
		require.True(t,
			ob.GetMarketPrice().Equals(matching.NewUint(30).Mul64(matching.UintPrecision)),
		)

		stopLimitOrderID := uint64(6)
		err = engine.AddOrder(
			matching.NewStopLimitOrder(
				symbolID,
				stopLimitOrderID,
				matching.OrderSideBuy,
				matching.OrderDirectionOpen,
				matching.OrderTimeInForceGTC,
				matching.NewUint(5).Mul64(matching.UintPrecision), // price 5
				matching.StopPriceModeMarket,
				matching.NewUint(30).Mul64(matching.UintPrecision), // stop price 30
				matching.NewUint(3).Mul64(matching.UintPrecision),
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
		)

		require.NoError(t, err)
		require.Equal(t, matching.OrderTypeLimit, ob.Order(stopLimitOrderID).Type())
	})

	t.Run("sell (stop price == market price)", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)
		ob := engine.OrderBook(symbolID)

		err := engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
			matching.OrderDirectionOpen,
			matching.OrderTimeInForceGTC,
			matching.NewUint(30).Mul64(matching.UintPrecision), // price 30
			matching.NewUint(1).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))

		require.NoError(t, err)
		require.True(t,
			ob.GetMarketPrice().Equals(matching.NewUint(30).Mul64(matching.UintPrecision)),
		)

		stopLimitOrderID := uint64(6)
		err = engine.AddOrder(
			matching.NewStopLimitOrder(
				symbolID,
				stopLimitOrderID,
				matching.OrderSideSell,
				matching.OrderDirectionClose,
				matching.OrderTimeInForceGTC,
				matching.NewUint(50).Mul64(matching.UintPrecision), // price 50
				matching.StopPriceModeMarket,
				matching.NewUint(30).Mul64(matching.UintPrecision), // stop price 30
				matching.NewUint(3).Mul64(matching.UintPrecision),
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
		)

		require.NoError(t, err)
		require.Equal(t, matching.OrderTypeLimit, ob.Order(stopLimitOrderID).Type())
	})

	t.Run("buy stop-loss", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)
		ob := engine.OrderBook(symbolID)

		err := engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
			matching.OrderDirectionOpen,
			matching.OrderTimeInForceGTC,
			matching.NewUint(30).Mul64(matching.UintPrecision), // price 30
			matching.NewUint(1).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
		require.True(t,
			ob.GetMarketPrice().Equals(matching.NewUint(30).Mul64(matching.UintPrecision)),
		)

		stopLimitOrderID := uint64(6)
		err = engine.AddOrder(
			matching.NewStopLimitOrder(
				symbolID,
				stopLimitOrderID,
				matching.OrderSideBuy,
				matching.OrderDirectionOpen,
				matching.OrderTimeInForceGTC,
				matching.NewUint(5).Mul64(matching.UintPrecision), // price 5
				matching.StopPriceModeMarket,
				matching.NewUint(35).Mul64(matching.UintPrecision), // stop price 35
				matching.NewUint(3).Mul64(matching.UintPrecision),
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
		)
		require.NoError(t, err)
		require.Equal(t, matching.OrderTypeStopLimit, ob.Order(stopLimitOrderID).Type())
		require.False(t, ob.Order(stopLimitOrderID).IsTakeProfit())

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(7),
			matching.OrderSideBuy,
			matching.OrderDirectionOpen,
			matching.OrderTimeInForceGTC,
			matching.NewUint(40).Mul64(matching.UintPrecision), // price 40
			matching.NewUint(1).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
		require.True(t,
			ob.GetMarketPrice().Equals(matching.NewUint(40).Mul64(matching.UintPrecision)),
		)
		require.Equal(t, matching.OrderTypeLimit, ob.Order(stopLimitOrderID).Type())
	})

	t.Run("sell stop-loss", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)
		ob := engine.OrderBook(symbolID)

		err := engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
			matching.OrderDirectionOpen,
			matching.OrderTimeInForceGTC,
			matching.NewUint(30).Mul64(matching.UintPrecision), // price 30
			matching.NewUint(1).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
		require.True(t,
			ob.GetMarketPrice().Equals(matching.NewUint(30).Mul64(matching.UintPrecision)),
		)

		stopLimitOrderID := uint64(6)
		err = engine.AddOrder(
			matching.NewStopLimitOrder(
				symbolID,
				stopLimitOrderID,
				matching.OrderSideSell,
				matching.OrderDirectionClose,
				matching.OrderTimeInForceGTC,
				matching.NewUint(50).Mul64(matching.UintPrecision), // price 50
				matching.StopPriceModeMarket,
				matching.NewUint(20).Mul64(matching.UintPrecision), // stop price 25
				matching.NewUint(3).Mul64(matching.UintPrecision),
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
		)
		require.NoError(t, err)
		require.Equal(t, matching.OrderTypeStopLimit, ob.Order(stopLimitOrderID).Type())
		require.False(t, ob.Order(stopLimitOrderID).IsTakeProfit())

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(7),
			matching.OrderSideSell,
			matching.OrderDirectionClose,
			matching.OrderTimeInForceGTC,
			matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
			matching.NewUint(1).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
		require.True(t,
			ob.GetMarketPrice().Equals(matching.NewUint(20).Mul64(matching.UintPrecision)),
		)
		require.Equal(t, matching.OrderTypeLimit, ob.Order(stopLimitOrderID).Type())
	})

	t.Run("buy take-profit", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)
		ob := engine.OrderBook(symbolID)

		err := engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
			matching.OrderDirectionOpen,
			matching.OrderTimeInForceGTC,
			matching.NewUint(30).Mul64(matching.UintPrecision), // price 30
			matching.NewUint(1).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
		require.True(t,
			ob.GetMarketPrice().Equals(matching.NewUint(30).Mul64(matching.UintPrecision)),
		)

		stopLimitOrderID := uint64(6)
		err = engine.AddOrder(
			matching.NewStopLimitOrder(
				symbolID,
				stopLimitOrderID,
				matching.OrderSideBuy,
				matching.OrderDirectionOpen,
				matching.OrderTimeInForceGTC,
				matching.NewUint(5).Mul64(matching.UintPrecision), // price 5
				matching.StopPriceModeMarket,
				matching.NewUint(25).Mul64(matching.UintPrecision), // stop price 25
				matching.NewUint(3).Mul64(matching.UintPrecision),
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
		)
		require.NoError(t, err)
		require.Equal(t, matching.OrderTypeStopLimit, ob.Order(stopLimitOrderID).Type())
		require.True(t, ob.Order(stopLimitOrderID).IsTakeProfit())

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(7),
			matching.OrderSideSell,
			matching.OrderDirectionClose,
			matching.OrderTimeInForceGTC,
			matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
			matching.NewUint(1).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
		require.True(t,
			ob.GetMarketPrice().Equals(matching.NewUint(20).Mul64(matching.UintPrecision)),
		)
		require.Equal(t, matching.OrderTypeLimit, ob.Order(stopLimitOrderID).Type())
	})

	t.Run("sell take-profit", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)
		ob := engine.OrderBook(symbolID)

		err := engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
			matching.OrderDirectionOpen,
			matching.OrderTimeInForceGTC,
			matching.NewUint(30).Mul64(matching.UintPrecision), // price 30
			matching.NewUint(1).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
		require.True(t,
			ob.GetMarketPrice().Equals(matching.NewUint(30).Mul64(matching.UintPrecision)),
		)

		stopLimitOrderID := uint64(6)
		err = engine.AddOrder(
			matching.NewStopLimitOrder(
				symbolID,
				stopLimitOrderID,
				matching.OrderSideSell,
				matching.OrderDirectionClose,
				matching.OrderTimeInForceGTC,
				matching.NewUint(50).Mul64(matching.UintPrecision), // price 50
				matching.StopPriceModeMarket,
				matching.NewUint(35).Mul64(matching.UintPrecision), // stop price 35
				matching.NewUint(3).Mul64(matching.UintPrecision),
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
		)
		require.NoError(t, err)
		require.Equal(t, matching.OrderTypeStopLimit, ob.Order(stopLimitOrderID).Type())
		require.True(t, ob.Order(stopLimitOrderID).IsTakeProfit())

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(7),
			matching.OrderSideBuy,
			matching.OrderDirectionOpen,
			matching.OrderTimeInForceGTC,
			matching.NewUint(40).Mul64(matching.UintPrecision), // price 40
			matching.NewUint(1).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
		require.True(t,
			ob.GetMarketPrice().Equals(matching.NewUint(40).Mul64(matching.UintPrecision)),
		)
		require.Equal(t, matching.OrderTypeLimit, ob.Order(stopLimitOrderID).Type())
	})

	t.Run("buy take-profit (mark price)", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)

		ob := engine.OrderBook(symbolID)

		err := engine.SetMarkPriceForOrderBook(symbolID, matching.NewUint(30).Mul64(matching.UintPrecision), false)
		require.NoError(t, err)
		require.True(t,
			ob.GetMarkPrice().Equals(matching.NewUint(30).Mul64(matching.UintPrecision)),
		)

		stopLimitOrderID := uint64(6)
		err = engine.AddOrder(
			matching.NewStopLimitOrder(
				symbolID,
				stopLimitOrderID,
				matching.OrderSideBuy,
				matching.OrderDirectionOpen,
				matching.OrderTimeInForceGTC,
				matching.NewUint(5).Mul64(matching.UintPrecision), // price 5
				matching.StopPriceModeMark,
				matching.NewUint(25).Mul64(matching.UintPrecision), // stop price 25
				matching.NewUint(3).Mul64(matching.UintPrecision),
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
		)
		require.NoError(t, err)
		require.Equal(t, matching.OrderTypeStopLimit, ob.Order(stopLimitOrderID).Type())
		require.True(t, ob.Order(stopLimitOrderID).IsTakeProfit())

		err = engine.SetMarkPriceForOrderBook(symbolID, matching.NewUint(20).Mul64(matching.UintPrecision), true)
		require.NoError(t, err)
		require.True(t,
			ob.GetMarkPrice().Equals(matching.NewUint(20).Mul64(matching.UintPrecision)),
		)
		require.Equal(t, matching.OrderTypeLimit, ob.Order(stopLimitOrderID).Type())
	})

	t.Run("sell take-profit (mark price)", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)

		ob := engine.OrderBook(symbolID)

		err := engine.SetMarkPriceForOrderBook(symbolID, matching.NewUint(20).Mul64(matching.UintPrecision), false)
		require.NoError(t, err)
		require.True(t,
			ob.GetMarkPrice().Equals(matching.NewUint(20).Mul64(matching.UintPrecision)),
		)

		stopLimitOrderID := uint64(6)
		err = engine.AddOrder(
			matching.NewStopLimitOrder(
				symbolID,
				stopLimitOrderID,
				matching.OrderSideBuy,
				matching.OrderDirectionOpen,
				matching.OrderTimeInForceGTC,
				matching.NewUint(5).Mul64(matching.UintPrecision), // price 5
				matching.StopPriceModeMark,
				matching.NewUint(25).Mul64(matching.UintPrecision), // stop price 25
				matching.NewUint(3).Mul64(matching.UintPrecision),
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
		)
		require.NoError(t, err)
		require.Equal(t, matching.OrderTypeStopLimit, ob.Order(stopLimitOrderID).Type())
		require.True(t, !ob.Order(stopLimitOrderID).IsTakeProfit())

		err = engine.SetMarkPriceForOrderBook(symbolID, matching.NewUint(30).Mul64(matching.UintPrecision), true)
		require.NoError(t, err)
		require.True(t,
			ob.GetMarkPrice().Equals(matching.NewUint(30).Mul64(matching.UintPrecision)),
		)
		require.Equal(t, matching.OrderTypeLimit, ob.Order(stopLimitOrderID).Type())
	})

	t.Run("buy take-profit (index price)", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)

		ob := engine.OrderBook(symbolID)

		err := engine.SetIndexPriceForOrderBook(symbolID, matching.NewUint(30).Mul64(matching.UintPrecision), false)
		require.NoError(t, err)
		require.True(t,
			ob.GetIndexPrice().Equals(matching.NewUint(30).Mul64(matching.UintPrecision)),
		)

		stopLimitOrderID := uint64(6)
		err = engine.AddOrder(
			matching.NewStopLimitOrder(
				symbolID,
				stopLimitOrderID,
				matching.OrderSideBuy,
				matching.OrderDirectionOpen,
				matching.OrderTimeInForceGTC,
				matching.NewUint(5).Mul64(matching.UintPrecision), // price 5
				matching.StopPriceModeIndex,
				matching.NewUint(25).Mul64(matching.UintPrecision), // stop price 25
				matching.NewUint(3).Mul64(matching.UintPrecision),
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
		)
		require.NoError(t, err)
		require.Equal(t, matching.OrderTypeStopLimit, ob.Order(stopLimitOrderID).Type())
		require.True(t, ob.Order(stopLimitOrderID).IsTakeProfit())

		err = engine.SetIndexPriceForOrderBook(symbolID, matching.NewUint(20).Mul64(matching.UintPrecision), true)
		require.NoError(t, err)
		require.True(t,
			ob.GetIndexPrice().Equals(matching.NewUint(20).Mul64(matching.UintPrecision)),
		)
		require.Equal(t, matching.OrderTypeLimit, ob.Order(stopLimitOrderID).Type())
	})

	t.Run("sell take-profit (index price)", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)

		ob := engine.OrderBook(symbolID)

		err := engine.SetIndexPriceForOrderBook(symbolID, matching.NewUint(20).Mul64(matching.UintPrecision), false)
		require.NoError(t, err)
		require.True(t,
			ob.GetIndexPrice().Equals(matching.NewUint(20).Mul64(matching.UintPrecision)),
		)

		stopLimitOrderID := uint64(6)
		err = engine.AddOrder(
			matching.NewStopLimitOrder(
				symbolID,
				stopLimitOrderID,
				matching.OrderSideBuy,
				matching.OrderDirectionOpen,
				matching.OrderTimeInForceGTC,
				matching.NewUint(5).Mul64(matching.UintPrecision), // price 5
				matching.StopPriceModeIndex,
				matching.NewUint(25).Mul64(matching.UintPrecision), // stop price 25
				matching.NewUint(3).Mul64(matching.UintPrecision),
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
		)
		require.NoError(t, err)
		require.Equal(t, matching.OrderTypeStopLimit, ob.Order(stopLimitOrderID).Type())
		require.True(t, !ob.Order(stopLimitOrderID).IsTakeProfit())

		err = engine.SetIndexPriceForOrderBook(symbolID, matching.NewUint(30).Mul64(matching.UintPrecision), true)
		require.NoError(t, err)
		require.True(t,
			ob.GetIndexPrice().Equals(matching.NewUint(30).Mul64(matching.UintPrecision)),
		)
		require.Equal(t, matching.OrderTypeLimit, ob.Order(stopLimitOrderID).Type())
	})
}
