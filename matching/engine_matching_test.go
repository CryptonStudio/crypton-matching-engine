package matching_test

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	matching "github.com/cryptonstudio/crypton-matching-engine/matching"
	mockmatching "github.com/cryptonstudio/crypton-matching-engine/matching/mocks"
)

func TestMarketOrdersMatching(t *testing.T) {
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

	dumpOB := func(t *testing.T, engine *matching.Engine) {
		ob := engine.OrderBook(symbolID)

		mostLeft := ob.TopBid()
		for mostLeft != nil {
			t.Logf("bid price level: %s\n", mostLeft.Key().ToFloatString())
			mostLeft = mostLeft.NextRight()
		}

		mostLeft = ob.TopAsk()
		for mostLeft != nil {
			t.Logf("ask price level: %s\n", mostLeft.Key().ToFloatString())
			mostLeft = mostLeft.NextRight()
		}
	}

	t.Run("1-buy market order by amount(quantity)", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)

		err := engine.AddOrder(matching.NewMarketOrder(symbolID, orderID,
			matching.OrderSideBuy,
			matching.NewUint(3).Mul64(matching.UintPrecision).Div64(2), // 1.5 amount
			matching.NewZeroUint(),
			matching.NewMaxUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)

		ob := engine.OrderBook(symbolID)
		require.Equal(t, (*matching.Order)(nil), ob.Order(3)) // #3 is full matched
		require.True(t,
			ob.Order(4).ExecutedQuantity().Equals(matching.NewUint(matching.UintPrecision).Div64(2)), // #4 executed amount is 0.5
		)
	})

	t.Run("2-buy market order by total(quote), good locked", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()
		engine.Start()

		setupMarketState(t, engine, symbolID)

		dumpOB(t, engine)

		err := engine.AddOrder(matching.NewMarketOrder(symbolID, orderID,
			matching.OrderSideBuy,
			matching.NewZeroUint(),
			matching.NewUint(40).Mul64(matching.UintPrecision),
			matching.NewMaxUint(),
			matching.NewUint(40).Mul64(matching.UintPrecision),
		))
		require.NoError(t, err)

		ob := engine.OrderBook(symbolID)
		require.Equal(t, (*matching.Order)(nil), ob.Order(3))              // #3 is full matched
		rest := matching.NewUint(matching.UintPrecision).Div64(4).Mul64(3) // 0.75
		require.True(t,
			ob.Order(4).RestQuantity().Equals(rest), // #4 executed amount is 0.25, rest is 0.75
			"result is %s, executed quote %s, executed %s",
			ob.Order(4).RestQuantity().ToFloatString(),
			ob.Order(4).ExecutedQuoteQuantity().ToFloatString(),
			ob.Order(4).ExecutedQuantity().ToFloatString(),
		)
	})

	t.Run("2-buy market order by total(quote), overlocked", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()
		engine.Start()

		setupMarketState(t, engine, symbolID)

		dumpOB(t, engine)

		err := engine.AddOrder(matching.NewMarketOrder(symbolID, orderID,
			matching.OrderSideBuy,
			matching.NewZeroUint(),
			matching.NewUint(40).Mul64(matching.UintPrecision),
			matching.NewMaxUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)

		ob := engine.OrderBook(symbolID)
		require.Equal(t, (*matching.Order)(nil), ob.Order(3))              // #3 is full matched
		rest := matching.NewUint(matching.UintPrecision).Div64(4).Mul64(3) // 0.75
		require.True(t,
			ob.Order(4).RestQuantity().Equals(rest), // #4 executed amount is 0.25, rest is 0.75
			"result is %s, executed quote %s, executed %s",
			ob.Order(4).RestQuantity().ToFloatString(),
			ob.Order(4).ExecutedQuoteQuantity().ToFloatString(),
			ob.Order(4).ExecutedQuantity().ToFloatString(),
		)
	})

	t.Run("3-sell market order by amount(quantity)", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)

		err := engine.AddOrder(matching.NewMarketOrder(symbolID, orderID,
			matching.OrderSideSell,
			matching.NewUint(3).Mul64(matching.UintPrecision).Div64(2), // 1.5 amount
			matching.NewZeroUint(),
			matching.NewMaxUint(),
			matching.NewUint(3).Mul64(matching.UintPrecision).Div64(2),
		))
		require.NoError(t, err)

		ob := engine.OrderBook(symbolID)
		require.Equal(t, (*matching.Order)(nil), ob.Order(2)) // #2 is full matched
		require.True(t,
			ob.Order(1).ExecutedQuantity().Equals(matching.NewUint(matching.UintPrecision).Div64(2)), // #1 executed amount is 0.5
		)
	})

	t.Run("4-sell market order by total(quote)", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)

		err := engine.AddOrder(matching.NewMarketOrder(symbolID, orderID,
			matching.OrderSideSell,
			matching.NewZeroUint(),
			matching.NewUint(25).Mul64(matching.UintPrecision),
			matching.NewMaxUint(),
			matching.NewUint(25).Mul64(matching.UintPrecision),
		))
		require.NoError(t, err)

		ob := engine.OrderBook(symbolID)
		require.Equal(t, (*matching.Order)(nil), ob.Order(2)) // #2 is full matched
		rest := matching.NewUint(matching.UintPrecision).Div64(2)
		require.True(t,
			ob.Order(1).RestQuantity().Equals(rest), // #1 executed amount is 0.5, rest is 0.5
			"result is %s", ob.Order(1).RestQuantity().ToFloatString(),
		)
	})

}

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
				matching.NewUint(5).Mul64(matching.UintPrecision),  // price 5
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
				matching.NewUint(50).Mul64(matching.UintPrecision), // price 50
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
				matching.NewUint(5).Mul64(matching.UintPrecision),  // price 5
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
				matching.NewUint(50).Mul64(matching.UintPrecision), // price 50
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
				matching.NewUint(5).Mul64(matching.UintPrecision),  // price 5
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
				matching.NewUint(50).Mul64(matching.UintPrecision), // price 50
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
}

func TestOCOOrders(t *testing.T) {
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

	t.Run("buy, both are placed", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)

		ob := engine.OrderBook(symbolID)

		err := engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
			matching.NewUint(30).Mul64(matching.UintPrecision), // price 30
			matching.NewUint(1).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
		require.True(t,
			ob.GetMarketPrice().Equals(matching.NewUint(30).Mul64(matching.UintPrecision)),
		)

		err = engine.AddOrdersPair(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.NewUint(40).Mul64(matching.UintPrecision), // stop-price 40
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
			matching.NewLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.NewUint(15).Mul64(matching.UintPrecision), // price 15
				matching.NewUint(1).Mul64(matching.UintPrecision),  // amount 1
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
		)
		require.NoError(t, err)
		require.Equal(t, matching.OrderTypeStopLimit, ob.Order(6).Type()) // stop-limit order is placed
		require.Equal(t, matching.OrderTypeLimit, ob.Order(7).Type())     // limit is placed
	})

	t.Run("buy OCO stop price is less than market price", func(t *testing.T) {
		// in place to test onError() calls
		handler := mockmatching.NewMockHandler(ctrl)
		setupMockHandler(t, handler)

		engine := matching.NewEngine(handler, false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)
		ob := engine.OrderBook(symbolID)

		handler.EXPECT().OnError(ob, matching.ErrBuyOCOStopPriceLessThanMarketPrice)

		err := engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
			matching.NewUint(30).Mul64(matching.UintPrecision), // price 30
			matching.NewUint(1).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
		require.True(t,
			ob.GetMarketPrice().Equals(matching.NewUint(30).Mul64(matching.UintPrecision)),
		)

		err = engine.AddOrdersPair(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.NewUint(25).Mul64(matching.UintPrecision), // stop-price 25
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
			matching.NewLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.NewUint(15).Mul64(matching.UintPrecision), // price 15
				matching.NewUint(1).Mul64(matching.UintPrecision),  // amount 1
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
		)
		require.ErrorIs(t, err, matching.ErrBuyOCOStopPriceLessThanMarketPrice)
	})

	t.Run("buy OCO limit order price is greater than market price", func(t *testing.T) {
		// in place to test onError() calls
		handler := mockmatching.NewMockHandler(ctrl)
		setupMockHandler(t, handler)

		engine := matching.NewEngine(handler, false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)
		ob := engine.OrderBook(symbolID)

		handler.EXPECT().OnError(ob, matching.ErrBuyOCOLimitPriceGreaterThanMarketPrice)

		err := engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
			matching.NewUint(30).Mul64(matching.UintPrecision), // price 30
			matching.NewUint(1).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
		require.True(t,
			ob.GetMarketPrice().Equals(matching.NewUint(30).Mul64(matching.UintPrecision)),
		)

		err = engine.AddOrdersPair(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.NewUint(35).Mul64(matching.UintPrecision), // stop-price 35
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
			matching.NewLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.NewUint(32).Mul64(matching.UintPrecision), // price 32
				matching.NewUint(1).Mul64(matching.UintPrecision),  // amount 1
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
		)
		require.ErrorIs(t, err, matching.ErrBuyOCOLimitPriceGreaterThanMarketPrice)
	})

	t.Run("sell OCO stop price is greater than market price", func(t *testing.T) {
		// in place to test onError() calls
		handler := mockmatching.NewMockHandler(ctrl)
		setupMockHandler(t, handler)

		engine := matching.NewEngine(handler, false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)
		ob := engine.OrderBook(symbolID)

		handler.EXPECT().OnError(ob, matching.ErrSellOCOStopPriceGreaterThanMarketPrice)

		err := engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
			matching.NewUint(30).Mul64(matching.UintPrecision), // price 30
			matching.NewUint(1).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
		require.True(t,
			ob.GetMarketPrice().Equals(matching.NewUint(30).Mul64(matching.UintPrecision)),
		)

		err = engine.AddOrdersPair(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideSell,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.NewUint(35).Mul64(matching.UintPrecision), // stop-price 35
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
			matching.NewLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideSell,
				matching.NewUint(32).Mul64(matching.UintPrecision), // price 32
				matching.NewUint(1).Mul64(matching.UintPrecision),  // amount 1
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
		)
		require.ErrorIs(t, err, matching.ErrSellOCOStopPriceGreaterThanMarketPrice)
	})

	t.Run("sell OCO limit order price is less than market price", func(t *testing.T) {
		// in place to test onError() calls
		handler := mockmatching.NewMockHandler(ctrl)
		setupMockHandler(t, handler)

		engine := matching.NewEngine(handler, false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)
		ob := engine.OrderBook(symbolID)

		handler.EXPECT().OnError(ob, matching.ErrSellOCOLimitPriceLessThanMarketPrice)

		err := engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
			matching.NewUint(30).Mul64(matching.UintPrecision), // price 30
			matching.NewUint(1).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
		require.True(t,
			ob.GetMarketPrice().Equals(matching.NewUint(30).Mul64(matching.UintPrecision)),
		)

		err = engine.AddOrdersPair(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideSell,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.NewUint(25).Mul64(matching.UintPrecision), // stop-price 25
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
			matching.NewLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideSell,
				matching.NewUint(28).Mul64(matching.UintPrecision), // price 28
				matching.NewUint(1).Mul64(matching.UintPrecision),  // amount 1
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
		)
		require.ErrorIs(t, err, matching.ErrSellOCOLimitPriceLessThanMarketPrice)
	})

	t.Run("buy, stop-limit is deleted manually, limit is deleted automatically", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)

		ob := engine.OrderBook(symbolID)

		err := engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
			matching.NewUint(30).Mul64(matching.UintPrecision), // price 30
			matching.NewUint(1).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
		require.True(t,
			ob.GetMarketPrice().Equals(matching.NewUint(30).Mul64(matching.UintPrecision)),
		)

		err = engine.AddOrdersPair(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.NewUint(40).Mul64(matching.UintPrecision), // stop-price 40
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
			matching.NewLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.NewUint(15).Mul64(matching.UintPrecision), // price 15
				matching.NewUint(1).Mul64(matching.UintPrecision),  // amount 1
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
		)
		require.NoError(t, err)
		require.Equal(t, matching.OrderTypeStopLimit, ob.Order(6).Type()) // stop-limit order is placed
		require.Equal(t, matching.OrderTypeLimit, ob.Order(7).Type())     // limit is placed

		err = engine.DeleteOrder(symbolID, 6)
		require.NoError(t, err)
		require.Equal(t, (*matching.Order)(nil), ob.Order(6)) // stop-limit is is deleted
		require.Equal(t, (*matching.Order)(nil), ob.Order(7)) // limit order is deleted
	})

	t.Run("buy, limit is deleted manually, stop-limit is deleted automatically", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)

		ob := engine.OrderBook(symbolID)

		err := engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
			matching.NewUint(30).Mul64(matching.UintPrecision), // price 30
			matching.NewUint(1).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
		require.True(t,
			ob.GetMarketPrice().Equals(matching.NewUint(30).Mul64(matching.UintPrecision)),
		)

		err = engine.AddOrdersPair(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.NewUint(40).Mul64(matching.UintPrecision), // stop-price 40
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
			matching.NewLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.NewUint(15).Mul64(matching.UintPrecision), // price 15
				matching.NewUint(1).Mul64(matching.UintPrecision),  // amount 1
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
		)
		require.NoError(t, err)
		require.Equal(t, matching.OrderTypeStopLimit, ob.Order(6).Type()) // stop-limit order is placed
		require.Equal(t, matching.OrderTypeLimit, ob.Order(7).Type())     // limit is placed

		err = engine.DeleteOrder(symbolID, 7)
		require.NoError(t, err)
		require.Equal(t, (*matching.Order)(nil), ob.Order(6)) // stop-limit is is deleted
		require.Equal(t, (*matching.Order)(nil), ob.Order(7)) // limit order is deleted
	})

	t.Run("buy OCO order, stop-limit is activated immediately, limit is deleted", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)

		ob := engine.OrderBook(symbolID)

		err := engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
			matching.NewUint(30).Mul64(matching.UintPrecision), // price 30
			matching.NewUint(1).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
		require.True(t,
			ob.GetMarketPrice().Equals(matching.NewUint(30).Mul64(matching.UintPrecision)),
		)

		err = engine.AddOrdersPair(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 35
				matching.NewUint(30).Mul64(matching.UintPrecision), // stop-price 30
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
			matching.NewLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.NewUint(15).Mul64(matching.UintPrecision), // price 15
				matching.NewUint(1).Mul64(matching.UintPrecision),  // amount 1
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
		)
		require.NoError(t, err)
		require.Equal(t, matching.OrderTypeLimit, ob.Order(6).Type()) // stop-limit order is activated
		require.Equal(t, (*matching.Order)(nil), ob.Order(7))         // limit order is deleted
	})

	t.Run("buy OCO order, stop-limit is activated and fully executed immediately, limit is deleted", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)

		ob := engine.OrderBook(symbolID)

		err := engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
			matching.NewUint(30).Mul64(matching.UintPrecision), // price 30
			matching.NewUint(1).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
		require.True(t,
			ob.GetMarketPrice().Equals(matching.NewUint(30).Mul64(matching.UintPrecision)),
		)

		err = engine.AddOrdersPair(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.NewUint(40).Mul64(matching.UintPrecision), // price 40
				matching.NewUint(30).Mul64(matching.UintPrecision), // stop-price 30
				matching.NewUint(1).Mul64(matching.UintPrecision),  // amount 1
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
			matching.NewLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.NewUint(15).Mul64(matching.UintPrecision), // price 15
				matching.NewUint(1).Mul64(matching.UintPrecision),  // amount 1
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
		)
		require.NoError(t, err)
		require.Equal(t, (*matching.Order)(nil), ob.Order(6)) // stop-limit is fully executed
		require.Equal(t, (*matching.Order)(nil), ob.Order(7)) // limit order is deleted
	})

	t.Run("buy OCO order, limit is executing, stop-limit is deleted", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)

		ob := engine.OrderBook(symbolID)

		err := engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideSell,
			matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
			matching.NewUint(10).Mul64(matching.UintPrecision), // amount 10
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
		require.True(t,
			ob.GetMarketPrice().Equals(matching.NewUint(20).Mul64(matching.UintPrecision)),
		)

		err = engine.AddOrdersPair(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.NewUint(30).Mul64(matching.UintPrecision), // price 30
				matching.NewUint(25).Mul64(matching.UintPrecision), // stop-price 25
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
			matching.NewLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.NewUint(15).Mul64(matching.UintPrecision), // amount 15
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
		)
		require.NoError(t, err)
		require.False(t, ob.Order(7).ExecutedQuantity().IsZero()) // limit order is executing
		require.Equal(t, (*matching.Order)(nil), ob.Order(6))     // stop-limit is deleted
	})
}

// This function is helper to define base bids and asks (not recommended to modify)
func setupMarketState(t *testing.T, engine *matching.Engine, symbolID uint32) {
	_, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0))
	require.NoError(t, err)

	pricesAndSides := []struct {
		id    uint64
		price uint64
		side  matching.OrderSide
	}{
		{1, 10, matching.OrderSideBuy},
		{2, 20, matching.OrderSideBuy},
		{3, 30, matching.OrderSideSell},
		{4, 40, matching.OrderSideSell},
	}

	for _, ps := range pricesAndSides {
		err := engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			ps.id,
			ps.side,
			matching.NewUint(ps.price).Mul64(matching.UintPrecision),
			matching.NewUint(1).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
	}

	require.Equal(t, 4, engine.Orders())
}

func setupMockHandler(t *testing.T, handler *mockmatching.MockHandler) {
	handler.EXPECT().OnAddOrderBook(gomock.Any()).AnyTimes()
	handler.EXPECT().OnAddOrder(gomock.Any(), gomock.Any()).AnyTimes()
	handler.EXPECT().OnDeleteOrder(gomock.Any(), gomock.Any()).AnyTimes()
	handler.EXPECT().OnUpdateOrder(gomock.Any(), gomock.Any()).AnyTimes()
	handler.EXPECT().OnAddPriceLevel(gomock.Any(), gomock.Any()).Do(
		func(orderBook *matching.OrderBook, update matching.PriceLevelUpdate) {
			t.Logf("add price level for %s\n", update.Price.ToFloatString())
		}).AnyTimes()
	handler.EXPECT().OnUpdatePriceLevel(gomock.Any(), gomock.Any()).AnyTimes()
	handler.EXPECT().OnDeletePriceLevel(gomock.Any(), gomock.Any()).AnyTimes()
	handler.EXPECT().OnUpdateOrderBook(gomock.Any()).AnyTimes()
	handler.EXPECT().OnExecuteOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Do(
		func(orderBook *matching.OrderBook, order *matching.Order, price matching.Uint, quantity matching.Uint) {
			t.Logf("order %d (order baseq = %s) executed: price %s, quantity %s\n",
				order.ID(), order.RestQuantity().ToFloatString(),
				price.ToFloatString(), quantity.ToFloatString())
		}).AnyTimes()
	handler.EXPECT().OnExecuteTrade(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
}
