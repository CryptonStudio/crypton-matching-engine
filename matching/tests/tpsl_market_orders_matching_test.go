package matching_test

import (
	"testing"

	matching "github.com/cryptonstudio/crypton-matching-engine/matching"
	mockmatching "github.com/cryptonstudio/crypton-matching-engine/matching/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestTPSLMarketOrders(t *testing.T) {
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

	t.Run("buy both are placed", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)

		ob := engine.OrderBook(symbolID)

		err := engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
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

		err = engine.AddTPSLMarket(
			matching.NewStopOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.OrderTimeInForceIOC,
				matching.StopPriceModeMarket,
				matching.NewUint(25).Mul64(matching.UintPrecision), // stop-price 25
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewZeroUint(),
				matching.NewZeroUint(),
				matching.NewMaxUint(),
			),
			matching.NewStopOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.OrderTimeInForceIOC,
				matching.StopPriceModeMarket,
				matching.NewUint(40).Mul64(matching.UintPrecision), // stop-price 40
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewZeroUint(),
				matching.NewZeroUint(),
				matching.NewZeroUint(),
			),
		)
		require.NoError(t, err)
		require.Equal(t, matching.OrderTypeStop, ob.Order(6).Type()) // tp order is placed
		require.True(t, ob.Order(6).IsTakeProfit())

		require.Equal(t, matching.OrderTypeStop, ob.Order(7).Type()) // sl order is placed
		require.False(t, ob.Order(7).IsTakeProfit())
	})

	t.Run("buy SL stop price is less than market price", func(t *testing.T) {
		// in place to test onError() calls
		handler := mockmatching.NewMockHandler(ctrl)
		setupMockHandler(t, handler)

		engine := matching.NewEngine(handler, false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)
		ob := engine.OrderBook(symbolID)

		handler.EXPECT().OnError(ob, matching.ErrBuySLStopPriceLessThanEnginePrice)

		err := engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
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

		err = engine.AddTPSLMarket(
			matching.NewStopOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.OrderTimeInForceIOC,
				matching.StopPriceModeMarket,
				matching.NewUint(25).Mul64(matching.UintPrecision), // stop-price 25
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewZeroUint(),
				matching.NewZeroUint(),
				matching.NewMaxUint(),
			),
			matching.NewStopOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.OrderTimeInForceIOC,
				matching.StopPriceModeMarket,
				matching.NewUint(25).Mul64(matching.UintPrecision), // stop-price 25
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewZeroUint(),
				matching.NewZeroUint(),
				matching.NewZeroUint(),
			),
		)
		require.ErrorIs(t, err, matching.ErrBuySLStopPriceLessThanEnginePrice)
	})

	t.Run("buy TP stop price is greater than market price", func(t *testing.T) {
		// in place to test onError() calls
		handler := mockmatching.NewMockHandler(ctrl)
		setupMockHandler(t, handler)

		engine := matching.NewEngine(handler, false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)
		ob := engine.OrderBook(symbolID)

		handler.EXPECT().OnError(ob, matching.ErrBuyTPStopPriceGreaterThanEnginePrice)

		err := engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
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

		err = engine.AddTPSLMarket(
			matching.NewStopOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.OrderTimeInForceIOC,
				matching.StopPriceModeMarket,
				matching.NewUint(35).Mul64(matching.UintPrecision), // stop-price 35
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewZeroUint(),
				matching.NewZeroUint(),
				matching.NewMaxUint(),
			),
			matching.NewStopOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.OrderTimeInForceIOC,
				matching.StopPriceModeMarket,
				matching.NewUint(40).Mul64(matching.UintPrecision), // stop-price 40
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewZeroUint(),
				matching.NewZeroUint(),
				matching.NewZeroUint(),
			),
		)
		require.ErrorIs(t, err, matching.ErrBuyTPStopPriceGreaterThanEnginePrice)
	})

	t.Run("sell SL stop price is greater than market price", func(t *testing.T) {
		// in place to test onError() calls
		handler := mockmatching.NewMockHandler(ctrl)
		setupMockHandler(t, handler)

		engine := matching.NewEngine(handler, false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)
		ob := engine.OrderBook(symbolID)

		handler.EXPECT().OnError(ob, matching.ErrSellSLStopPriceGreaterThanEnginePrice)

		err := engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
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

		err = engine.AddTPSLMarket(
			matching.NewStopOrder(
				symbolID,
				uint64(6),
				matching.OrderSideSell,
				matching.OrderTimeInForceIOC,
				matching.StopPriceModeMarket,
				matching.NewUint(40).Mul64(matching.UintPrecision), // stop-price 40
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewZeroUint(),
				matching.NewZeroUint(),
				matching.NewMaxUint(),
			),
			matching.NewStopOrder(
				symbolID,
				uint64(7),
				matching.OrderSideSell,
				matching.OrderTimeInForceIOC,
				matching.StopPriceModeMarket,
				matching.NewUint(40).Mul64(matching.UintPrecision), // stop-price 40
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewZeroUint(),
				matching.NewZeroUint(),
				matching.NewZeroUint(),
			),
		)
		require.ErrorIs(t, err, matching.ErrSellSLStopPriceGreaterThanEnginePrice)
	})

	t.Run("sell TP stop price is less than market price", func(t *testing.T) {
		// in place to test onError() calls
		handler := mockmatching.NewMockHandler(ctrl)
		setupMockHandler(t, handler)

		engine := matching.NewEngine(handler, false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)
		ob := engine.OrderBook(symbolID)

		handler.EXPECT().OnError(ob, matching.ErrSellTPStopPriceLessThanEnginePrice)

		err := engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
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

		err = engine.AddTPSLMarket(
			matching.NewStopOrder(
				symbolID,
				uint64(6),
				matching.OrderSideSell,
				matching.OrderTimeInForceIOC,
				matching.StopPriceModeMarket,
				matching.NewUint(20).Mul64(matching.UintPrecision), // stop-price 20
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewZeroUint(),
				matching.NewZeroUint(),
				matching.NewMaxUint(),
			),
			matching.NewStopOrder(
				symbolID,
				uint64(7),
				matching.OrderSideSell,
				matching.OrderTimeInForceIOC,
				matching.StopPriceModeMarket,
				matching.NewUint(20).Mul64(matching.UintPrecision), // stop-price 20
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewZeroUint(),
				matching.NewZeroUint(),
				matching.NewZeroUint(),
			),
		)
		require.ErrorIs(t, err, matching.ErrSellTPStopPriceLessThanEnginePrice)
	})

	t.Run("buy TP is deleted manually, SL is deleted automatically", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)

		ob := engine.OrderBook(symbolID)

		err := engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
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

		err = engine.AddTPSLMarket(
			matching.NewStopOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.OrderTimeInForceIOC,
				matching.StopPriceModeMarket,
				matching.NewUint(25).Mul64(matching.UintPrecision), // stop-price 25
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewZeroUint(),
				matching.NewZeroUint(),
				matching.NewMaxUint(),
			),
			matching.NewStopOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.OrderTimeInForceIOC,
				matching.StopPriceModeMarket,
				matching.NewUint(40).Mul64(matching.UintPrecision), // stop-price 40
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewZeroUint(),
				matching.NewZeroUint(),
				matching.NewZeroUint(),
			),
		)
		require.NoError(t, err)
		require.Equal(t, matching.OrderTypeStop, ob.Order(6).Type()) // tp order is placed
		require.Equal(t, matching.OrderTypeStop, ob.Order(7).Type()) // sl order is placed

		err = engine.DeleteOrder(symbolID, 6)
		require.NoError(t, err)
		require.Equal(t, (*matching.Order)(nil), ob.Order(6)) // tp order is deleted
		require.Equal(t, (*matching.Order)(nil), ob.Order(7)) // sl order is deleted
	})

	t.Run("buy SL is deleted manually, TP is deleted automatically", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)

		ob := engine.OrderBook(symbolID)

		err := engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
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

		err = engine.AddTPSLMarket(
			matching.NewStopOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.OrderTimeInForceIOC,
				matching.StopPriceModeMarket,
				matching.NewUint(25).Mul64(matching.UintPrecision), // stop-price 25
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewZeroUint(),
				matching.NewZeroUint(),
				matching.NewMaxUint(),
			),
			matching.NewStopOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.OrderTimeInForceIOC,
				matching.StopPriceModeMarket,
				matching.NewUint(40).Mul64(matching.UintPrecision), // stop-price 40
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewZeroUint(),
				matching.NewZeroUint(),
				matching.NewZeroUint(),
			),
		)
		require.NoError(t, err)
		require.Equal(t, matching.OrderTypeStop, ob.Order(6).Type()) // tp order is placed
		require.Equal(t, matching.OrderTypeStop, ob.Order(7).Type()) // sl order is placed

		err = engine.DeleteOrder(symbolID, 7)
		require.NoError(t, err)
		require.Equal(t, (*matching.Order)(nil), ob.Order(6)) // tp order is deleted
		require.Equal(t, (*matching.Order)(nil), ob.Order(7)) // sl order is deleted
	})

	t.Run("buy TP is activated and fully executed immediately, SL is deleted", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)

		ob := engine.OrderBook(symbolID)

		err := engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
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

		err = engine.AddTPSLMarket(
			matching.NewStopOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.OrderTimeInForceIOC,
				matching.StopPriceModeMarket,
				matching.NewUint(30).Mul64(matching.UintPrecision), // stop-price 30
				matching.NewUint(1).Mul64(matching.UintPrecision),  // amount 1
				matching.NewZeroUint(),
				matching.NewZeroUint(),
				matching.NewMaxUint(),
			),
			matching.NewStopOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.OrderTimeInForceIOC,
				matching.StopPriceModeMarket,
				matching.NewUint(40).Mul64(matching.UintPrecision), // stop-price 40
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewZeroUint(),
				matching.NewZeroUint(),
				matching.NewZeroUint(),
			),
		)
		require.NoError(t, err)
		require.Equal(t, (*matching.Order)(nil), ob.Order(6)) // tp is fully executed
		require.Equal(t, (*matching.Order)(nil), ob.Order(7)) // sl is deleted
	})

	t.Run("buy SL is activated and fully executed immediately, TP is deleted", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)

		ob := engine.OrderBook(symbolID)

		err := engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
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

		err = engine.AddTPSLMarket(
			matching.NewStopOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.OrderTimeInForceIOC,
				matching.StopPriceModeMarket,
				matching.NewUint(20).Mul64(matching.UintPrecision), // stop-price 20
				matching.NewUint(1).Mul64(matching.UintPrecision),  // amount 1
				matching.NewZeroUint(),
				matching.NewZeroUint(),
				matching.NewMaxUint(),
			),
			matching.NewStopOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.OrderTimeInForceIOC,
				matching.StopPriceModeMarket,
				matching.NewUint(30).Mul64(matching.UintPrecision), // stop-price 30
				matching.NewUint(1).Mul64(matching.UintPrecision),  // amount 1
				matching.NewZeroUint(),
				matching.NewZeroUint(),
				matching.NewZeroUint(),
			),
		)
		require.NoError(t, err)
		require.Equal(t, (*matching.Order)(nil), ob.Order(7)) // sl is fully executed
		require.Equal(t, (*matching.Order)(nil), ob.Order(6)) // tp is deleted
	})
}
