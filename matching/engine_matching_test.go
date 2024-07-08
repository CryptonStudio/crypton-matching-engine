package matching_test

import (
	"fmt"
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

	t.Run("buy market order by amount(quantity) in empty OB", func(t *testing.T) {
		handler := mockmatching.NewMockHandler(ctrl)

		engine := matching.NewEngine(handler, false)
		engine.EnableMatching()

		handler.EXPECT().OnAddOrderBook(gomock.Any()).AnyTimes()
		ob, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0), matching.StopPriceModeConfig{Market: true, Mark: true, Index: true})
		require.NoError(t, err)

		handler.EXPECT().OnAddOrder(ob, gomock.Any()).AnyTimes()
		handler.EXPECT().OnDeleteOrder(ob, gomock.Any())

		err = engine.AddOrder(matching.NewMarketOrder(symbolID, orderID,
			matching.OrderSideBuy,
			matching.OrderTimeInForceIOC,
			matching.NewUint(3).Mul64(matching.UintPrecision).Div64(2), // 1.5 amount
			matching.NewZeroUint(),
			matching.NewMaxUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)

		require.Equal(t, (*matching.Order)(nil), ob.Order(orderID)) // cancelled
	})

	t.Run("sell market order by amount(quantity) in empty OB", func(t *testing.T) {
		handler := mockmatching.NewMockHandler(ctrl)

		engine := matching.NewEngine(handler, false)
		engine.EnableMatching()

		handler.EXPECT().OnAddOrderBook(gomock.Any()).AnyTimes()
		ob, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0), matching.StopPriceModeConfig{Market: true, Mark: true, Index: true})
		require.NoError(t, err)

		handler.EXPECT().OnAddOrder(ob, gomock.Any()).AnyTimes()
		handler.EXPECT().OnDeleteOrder(ob, gomock.Any())

		err = engine.AddOrder(matching.NewMarketOrder(symbolID, orderID,
			matching.OrderSideSell,
			matching.OrderTimeInForceIOC,
			matching.NewUint(3).Mul64(matching.UintPrecision).Div64(2), // 1.5 amount
			matching.NewZeroUint(),
			matching.NewMaxUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)

		require.Equal(t, (*matching.Order)(nil), ob.Order(orderID)) // cancelled
	})

	t.Run("buy market order by amount(quantity), max slippage", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)

		err := engine.AddOrder(matching.NewMarketOrder(symbolID, orderID,
			matching.OrderSideBuy,
			matching.OrderTimeInForceIOC,
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

	t.Run("buy market order by amount(quantity), zero slippage", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)

		err := engine.AddOrder(matching.NewMarketOrder(symbolID, orderID,
			matching.OrderSideBuy,
			matching.OrderTimeInForceIOC,
			matching.NewUint(3).Mul64(matching.UintPrecision).Div64(2), // 1.5 amount
			matching.NewZeroUint(),
			matching.NewZeroUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)

		ob := engine.OrderBook(symbolID)
		require.Equal(t, (*matching.Order)(nil), ob.Order(3)) // #3 is full matched
		require.True(t,
			ob.Order(4).ExecutedQuantity().IsZero(), // not executed
		)
	})

	t.Run("buy market order by amount(quantity), easy slippage", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)

		err := engine.AddOrder(matching.NewMarketOrder(symbolID, orderID,
			matching.OrderSideBuy,
			matching.OrderTimeInForceIOC,
			matching.NewUint(3).Mul64(matching.UintPrecision).Div64(2), // 1.5 amount
			matching.NewZeroUint(),
			matching.NewUint(20).Mul64(matching.UintPrecision),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)

		ob := engine.OrderBook(symbolID)
		require.Equal(t, (*matching.Order)(nil), ob.Order(3)) // #3 is full matched
		require.True(t,
			ob.Order(4).ExecutedQuantity().Equals(matching.NewUint(matching.UintPrecision).Div64(2)), // #4 executed amount is 0.5
		)
	})

	t.Run("buy market order by total(quote), good locked", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()
		engine.Start()

		setupMarketState(t, engine, symbolID)

		dumpOB(t, engine)

		err := engine.AddOrder(matching.NewMarketOrder(symbolID, orderID,
			matching.OrderSideBuy,
			matching.OrderTimeInForceIOC,
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

	t.Run("buy market order by total(quote), overlocked", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()
		engine.Start()

		setupMarketState(t, engine, symbolID)

		dumpOB(t, engine)

		err := engine.AddOrder(matching.NewMarketOrder(symbolID, orderID,
			matching.OrderSideBuy,
			matching.OrderTimeInForceIOC,
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

	t.Run("sell market order by amount(quantity), max slippage", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)

		err := engine.AddOrder(matching.NewMarketOrder(symbolID, orderID,
			matching.OrderSideSell,
			matching.OrderTimeInForceIOC,
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
			fmt.Sprintf("have: %s, want: %s", ob.Order(1).ExecutedQuantity().ToFloatString(), matching.NewUint(matching.UintPrecision).Div64(2)),
		)
	})

	t.Run("sell market order by amount(quantity), zero slippage", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)

		err := engine.AddOrder(matching.NewMarketOrder(symbolID, orderID,
			matching.OrderSideSell,
			matching.OrderTimeInForceIOC,
			matching.NewUint(3).Mul64(matching.UintPrecision).Div64(2), // 1.5 amount
			matching.NewZeroUint(),
			matching.NewZeroUint(),
			matching.NewUint(3).Mul64(matching.UintPrecision).Div64(2),
		))
		require.NoError(t, err)

		ob := engine.OrderBook(symbolID)
		require.Equal(t, (*matching.Order)(nil), ob.Order(2)) // #2 is full matched
		require.True(t,
			ob.Order(1).ExecutedQuantity().IsZero(), // #1 no executed
		)
	})

	t.Run("sell market order by amount(quantity), easy slippage", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)

		err := engine.AddOrder(matching.NewMarketOrder(symbolID, orderID,
			matching.OrderSideSell,
			matching.OrderTimeInForceIOC,
			matching.NewUint(3).Mul64(matching.UintPrecision).Div64(2), // 1.5 amount
			matching.NewZeroUint(),
			matching.NewUint(20).Mul64(matching.UintPrecision),
			matching.NewUint(3).Mul64(matching.UintPrecision).Div64(2),
		))
		require.NoError(t, err)

		ob := engine.OrderBook(symbolID)
		require.Equal(t, (*matching.Order)(nil), ob.Order(2)) // #2 is full matched
		require.True(t,
			ob.Order(1).ExecutedQuantity().Equals(matching.NewUint(matching.UintPrecision).Div64(2)), // #1 executed amount is 0.5
			fmt.Sprintf(
				"have: %s, want: %s", ob.Order(1).ExecutedQuantity().ToFloatString(),
				matching.NewUint(matching.UintPrecision).Div64(2),
			),
		)
	})

	t.Run("sell market order by total(quote)", func(t *testing.T) {
		engine := matching.NewEngine(setupHandler(t), false)
		engine.EnableMatching()

		setupMarketState(t, engine, symbolID)

		err := engine.AddOrder(matching.NewMarketOrder(symbolID, orderID,
			matching.OrderSideSell,
			matching.OrderTimeInForceIOC,
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

		err = engine.AddOrdersPair(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.StopPriceModeMarket,
				matching.NewUint(40).Mul64(matching.UintPrecision), // stop-price 40
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewZeroUint(),
			),
			matching.NewLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
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

		err = engine.AddOrdersPair(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.StopPriceModeMarket,
				matching.NewUint(25).Mul64(matching.UintPrecision), // stop-price 25
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewZeroUint(),
			),
			matching.NewLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
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

		err = engine.AddOrdersPair(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.StopPriceModeMarket,
				matching.NewUint(35).Mul64(matching.UintPrecision), // stop-price 35
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewZeroUint(),
			),
			matching.NewLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
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

		err = engine.AddOrdersPair(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideSell,
				matching.OrderTimeInForceGTC,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.StopPriceModeMarket,
				matching.NewUint(35).Mul64(matching.UintPrecision), // stop-price 35
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewZeroUint(),
			),
			matching.NewLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideSell,
				matching.OrderTimeInForceGTC,
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

		err = engine.AddOrdersPair(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideSell,
				matching.OrderTimeInForceGTC,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.StopPriceModeMarket,
				matching.NewUint(25).Mul64(matching.UintPrecision), // stop-price 25
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewZeroUint(),
			),
			matching.NewLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideSell,
				matching.OrderTimeInForceGTC,
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

		err = engine.AddOrdersPair(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.StopPriceModeMarket,
				matching.NewUint(40).Mul64(matching.UintPrecision), // stop-price 40
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewZeroUint(),
			),
			matching.NewLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
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

		err = engine.AddOrdersPair(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.StopPriceModeMarket,
				matching.NewUint(40).Mul64(matching.UintPrecision), // stop-price 40
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewZeroUint(),
			),
			matching.NewLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
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

		err = engine.AddOrdersPair(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 35
				matching.StopPriceModeMarket,
				matching.NewUint(30).Mul64(matching.UintPrecision), // stop-price 30
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewZeroUint(),
			),
			matching.NewLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
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

		err = engine.AddOrdersPair(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
				matching.NewUint(40).Mul64(matching.UintPrecision), // price 40
				matching.StopPriceModeMarket,
				matching.NewUint(30).Mul64(matching.UintPrecision), // stop-price 30
				matching.NewUint(1).Mul64(matching.UintPrecision),  // amount 1
				matching.NewMaxUint(),
				matching.NewZeroUint(),
			),
			matching.NewLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
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
			matching.OrderTimeInForceGTC,
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
				matching.OrderTimeInForceGTC,
				matching.NewUint(30).Mul64(matching.UintPrecision), // price 30
				matching.StopPriceModeMarket,
				matching.NewUint(25).Mul64(matching.UintPrecision), // stop-price 25
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewZeroUint(),
			),
			matching.NewLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
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

	t.Run("buy OCO order, limit is executing, stop-limit is deleted (price < 1)", func(t *testing.T) {
		handler := mockmatching.NewMockHandler(ctrl)
		engine := matching.NewEngine(handler, false)
		engine.EnableMatching()

		// price level expectations
		handler.EXPECT().OnAddPriceLevel(gomock.Any(), gomock.Any()).Do(
			func(orderBook *matching.OrderBook, update matching.PriceLevelUpdate) {
				t.Logf("add price level for %s\n", update.Price.ToFloatString())
			}).AnyTimes()
		handler.EXPECT().OnUpdatePriceLevel(gomock.Any(), gomock.Any()).AnyTimes()
		handler.EXPECT().OnDeletePriceLevel(gomock.Any(), gomock.Any()).AnyTimes()

		// add order book
		handler.EXPECT().OnAddOrderBook(gomock.Any()).Times(1)
		ob, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0), matching.StopPriceModeConfig{Market: true})
		require.NoError(t, err)

		// ob updates expectations
		handler.EXPECT().OnUpdateOrderBook(gomock.Any()).AnyTimes()

		// execute one trade (2 orders)
		firstTrade := struct {
			price    matching.Uint
			quantity matching.Uint
		}{
			price:    matching.NewUint(13).Mul64(matching.UintPrecision / 100), // price 0.13
			quantity: matching.NewUint(1).Mul64(matching.UintPrecision),        // amount 10
		}
		handler.EXPECT().OnAddOrder(ob, gomock.Any()).Times(2)
		handler.EXPECT().OnExecuteOrder(
			ob, gomock.Any(), firstTrade.price, firstTrade.quantity,
			firstTrade.price.Mul(firstTrade.quantity).Div64(matching.UintPrecision)).Times(2)
		handler.EXPECT().OnExecuteTrade(
			ob, gomock.Any(), gomock.Any(), firstTrade.price, firstTrade.quantity,
			firstTrade.price.Mul(firstTrade.quantity).Div64(matching.UintPrecision)).Times(1)
		handler.EXPECT().OnDeleteOrder(gomock.Any(), gomock.Any()).Times(2)

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(1),
			matching.OrderSideBuy,
			matching.OrderTimeInForceGTC,
			firstTrade.price,
			firstTrade.quantity,
			matching.NewMaxUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(2),
			matching.OrderSideSell,
			matching.OrderTimeInForceGTC,
			firstTrade.price,
			firstTrade.quantity,
			matching.NewMaxUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)

		// mp 0.13
		require.True(t,
			ob.GetMarketPrice().Equals(matching.NewUint(13).Mul64(matching.UintPrecision/100)),
		)

		// trade OCO with limit
		secondTrade := struct {
			price      matching.Uint
			stopPrice  matching.Uint
			limitPrice matching.Uint
			quantity   matching.Uint
		}{
			// stop limit oco
			price:     matching.NewUint(12).Mul64(matching.UintPrecision / 100), // price 0.12
			stopPrice: matching.NewUint(12).Mul64(matching.UintPrecision / 100), // stop-price 0.12
			// price for limit orders to match
			limitPrice: matching.NewUint(13).Mul64(matching.UintPrecision / 100), // price 0.13
			quantity:   matching.NewUint(10).Mul64(matching.UintPrecision),       // amount 10
		}
		// 3 orders added -> 2 executions + 1 cancelled
		handler.EXPECT().OnAddOrder(ob, gomock.Any()).Times(3)
		handler.EXPECT().OnExecuteOrder(
			ob, gomock.Any(), secondTrade.limitPrice, secondTrade.quantity,
			secondTrade.limitPrice.Mul(secondTrade.quantity).Div64(matching.UintPrecision)).Do(
			func(orderBook *matching.OrderBook, orderID uint64, price matching.Uint, quantity matching.Uint, quoteQuantity matching.Uint) {
				t.Logf("order %d executed: price %s, qty %s, quoteQty %s\n",
					orderID,
					price.ToFloatString(), quantity.ToFloatString(),
					quoteQuantity.ToFloatString(),
				)
			}).Times(2)
		handler.EXPECT().OnExecuteTrade(
			ob, gomock.Any(), gomock.Any(), secondTrade.limitPrice, secondTrade.quantity,
			secondTrade.limitPrice.Mul(secondTrade.quantity).Div64(matching.UintPrecision)).Do(
			func(orderBook *matching.OrderBook, makerOrderUpdate matching.OrderUpdate, takerOrderUpdate matching.OrderUpdate, price matching.Uint, quantity matching.Uint, quoteQuantity matching.Uint) {
				makerOrderUpdate.Quantity.Equals(firstTrade.quantity)
				takerOrderUpdate.Quantity.Equals(firstTrade.quantity)

				quoteQty := firstTrade.quantity.Mul(firstTrade.price).Div64(matching.UintPrecision)
				makerOrderUpdate.QuoteQuantity.Equals(quoteQty)
				takerOrderUpdate.QuoteQuantity.Equals(quoteQty)

				t.Logf("trade qty: %s,maker executed: %d, taker executed: %d",
					quantity.ToFloatString(),
					makerOrderUpdate.ID,
					takerOrderUpdate.ID)
			}).Times(1)
		handler.EXPECT().OnDeleteOrder(gomock.Any(), gomock.Any()).Times(3)

		restLocked := secondTrade.quantity
		err = engine.AddOrdersPair(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(3),
				matching.OrderSideSell,
				matching.OrderTimeInForceGTC,
				secondTrade.price,
				matching.StopPriceModeMarket,
				secondTrade.stopPrice,
				secondTrade.quantity,
				matching.NewMaxUint(),
				matching.NewZeroUint(),
			),
			matching.NewLimitOrder(
				symbolID,
				uint64(4),
				matching.OrderSideSell,
				matching.OrderTimeInForceGTC,
				secondTrade.limitPrice,
				secondTrade.quantity,
				matching.NewMaxUint(),
				restLocked,
			),
		)
		require.NoError(t, err)

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			uint64(5),
			matching.OrderSideBuy,
			matching.OrderTimeInForceGTC,
			secondTrade.limitPrice,
			secondTrade.quantity,
			matching.NewMaxUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
	})
}

func TestTPSLOrders(t *testing.T) {
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

		err = engine.AddTPSL(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.StopPriceModeMarket,
				matching.NewUint(25).Mul64(matching.UintPrecision), // stop-price 25
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
			matching.NewStopLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.StopPriceModeMarket,
				matching.NewUint(40).Mul64(matching.UintPrecision), // stop-price 40
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewZeroUint(),
			),
		)
		require.NoError(t, err)
		require.Equal(t, matching.OrderTypeStopLimit, ob.Order(6).Type()) // tp order is placed
		require.True(t, ob.Order(6).IsTakeProfit())

		require.Equal(t, matching.OrderTypeStopLimit, ob.Order(7).Type()) // sl order is placed
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

		err = engine.AddTPSL(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.StopPriceModeMarket,
				matching.NewUint(25).Mul64(matching.UintPrecision), // stop-price 25
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
			matching.NewStopLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.StopPriceModeMarket,
				matching.NewUint(25).Mul64(matching.UintPrecision), // stop-price 25
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
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

		err = engine.AddTPSL(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.StopPriceModeMarket,
				matching.NewUint(35).Mul64(matching.UintPrecision), // stop-price 35
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
			matching.NewStopLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.StopPriceModeMarket,
				matching.NewUint(40).Mul64(matching.UintPrecision), // stop-price 40
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
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

		err = engine.AddTPSL(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideSell,
				matching.OrderTimeInForceGTC,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.StopPriceModeMarket,
				matching.NewUint(40).Mul64(matching.UintPrecision), // stop-price 40
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
			matching.NewStopLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideSell,
				matching.OrderTimeInForceGTC,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.StopPriceModeMarket,
				matching.NewUint(40).Mul64(matching.UintPrecision), // stop-price 40
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
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

		err = engine.AddTPSL(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideSell,
				matching.OrderTimeInForceGTC,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.StopPriceModeMarket,
				matching.NewUint(20).Mul64(matching.UintPrecision), // stop-price 20
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
			matching.NewStopLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideSell,
				matching.OrderTimeInForceGTC,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.StopPriceModeMarket,
				matching.NewUint(20).Mul64(matching.UintPrecision), // stop-price 20
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
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

		err = engine.AddTPSL(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.StopPriceModeMarket,
				matching.NewUint(25).Mul64(matching.UintPrecision), // stop-price 25
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
			matching.NewStopLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.StopPriceModeMarket,
				matching.NewUint(40).Mul64(matching.UintPrecision), // stop-price 40
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewZeroUint(),
			),
		)
		require.NoError(t, err)
		require.Equal(t, matching.OrderTypeStopLimit, ob.Order(6).Type()) // tp order is placed
		require.Equal(t, matching.OrderTypeStopLimit, ob.Order(7).Type()) // sl order is placed

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

		err = engine.AddTPSL(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.StopPriceModeMarket,
				matching.NewUint(25).Mul64(matching.UintPrecision), // stop-price 25
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
			matching.NewStopLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.StopPriceModeMarket,
				matching.NewUint(40).Mul64(matching.UintPrecision), // stop-price 40
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewZeroUint(),
			),
		)
		require.NoError(t, err)
		require.Equal(t, matching.OrderTypeStopLimit, ob.Order(6).Type()) // tp order is placed
		require.Equal(t, matching.OrderTypeStopLimit, ob.Order(7).Type()) // sl order is placed

		err = engine.DeleteOrder(symbolID, 7)
		require.NoError(t, err)
		require.Equal(t, (*matching.Order)(nil), ob.Order(6)) // tp order is deleted
		require.Equal(t, (*matching.Order)(nil), ob.Order(7)) // sl order is deleted
	})

	t.Run("buy TP is activated immediately, SL is deleted", func(t *testing.T) {
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

		err = engine.AddTPSL(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.StopPriceModeMarket,
				matching.NewUint(30).Mul64(matching.UintPrecision), // stop-price 30
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
			matching.NewStopLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.StopPriceModeMarket,
				matching.NewUint(40).Mul64(matching.UintPrecision), // stop-price 40
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewZeroUint(),
			),
		)
		require.NoError(t, err)
		require.Equal(t, matching.OrderTypeLimit, ob.Order(6).Type()) // tp order is activated
		require.Equal(t, (*matching.Order)(nil), ob.Order(7))         // sl order is deleted
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

		err = engine.AddTPSL(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
				matching.NewUint(40).Mul64(matching.UintPrecision), // price 40
				matching.StopPriceModeMarket,
				matching.NewUint(30).Mul64(matching.UintPrecision), // stop-price 30
				matching.NewUint(1).Mul64(matching.UintPrecision),  // amount 1
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
			matching.NewStopLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.StopPriceModeMarket,
				matching.NewUint(40).Mul64(matching.UintPrecision), // stop-price 40
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewZeroUint(),
			),
		)
		require.NoError(t, err)
		require.Equal(t, (*matching.Order)(nil), ob.Order(6)) // tp is fully executed
		require.Equal(t, (*matching.Order)(nil), ob.Order(7)) // sl is deleted
	})

	t.Run("buy SL is activated immediately, TP is deleted", func(t *testing.T) {
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

		err = engine.AddTPSL(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.StopPriceModeMarket,
				matching.NewUint(20).Mul64(matching.UintPrecision), // stop-price 20
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
			matching.NewStopLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
				matching.NewUint(20).Mul64(matching.UintPrecision), // price 20
				matching.StopPriceModeMarket,
				matching.NewUint(30).Mul64(matching.UintPrecision), // stop-price 30
				matching.NewUint(3).Mul64(matching.UintPrecision),  // amount 3
				matching.NewMaxUint(),
				matching.NewZeroUint(),
			),
		)
		require.NoError(t, err)
		require.Equal(t, matching.OrderTypeLimit, ob.Order(7).Type()) // sl order is activated
		require.Equal(t, (*matching.Order)(nil), ob.Order(6))         // tp order is deleted
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

		err = engine.AddTPSL(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
				matching.NewUint(40).Mul64(matching.UintPrecision), // price 40
				matching.StopPriceModeMarket,
				matching.NewUint(20).Mul64(matching.UintPrecision), // stop-price 20
				matching.NewUint(1).Mul64(matching.UintPrecision),  // amount 1
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			),
			matching.NewStopLimitOrder(
				symbolID,
				uint64(7),
				matching.OrderSideBuy,
				matching.OrderTimeInForceGTC,
				matching.NewUint(40).Mul64(matching.UintPrecision), // price 40
				matching.StopPriceModeMarket,
				matching.NewUint(30).Mul64(matching.UintPrecision), // stop-price 30
				matching.NewUint(1).Mul64(matching.UintPrecision),  // amount 1
				matching.NewMaxUint(),
				matching.NewZeroUint(),
			),
		)
		require.NoError(t, err)
		require.Equal(t, (*matching.Order)(nil), ob.Order(7)) // sl is fully executed
		require.Equal(t, (*matching.Order)(nil), ob.Order(6)) // tp is deleted
	})
}

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

// This function is helper to define base bids and asks (not recommended to modify)
func setupMarketState(t *testing.T, engine *matching.Engine, symbolID uint32) {
	_, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0), matching.StopPriceModeConfig{Market: true, Mark: true, Index: true})
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
			matching.OrderTimeInForceGTC,
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
	handler.EXPECT().OnDeleteOrder(gomock.Any(), gomock.Any()).Do(
		func(orderBook *matching.OrderBook, order *matching.Order) {
			if order.ID() == 0 {
				panic("order id is 0")
			}
		}).AnyTimes()
	handler.EXPECT().OnUpdateOrder(gomock.Any(), gomock.Any()).AnyTimes()
	handler.EXPECT().OnAddPriceLevel(gomock.Any(), gomock.Any()).Do(
		func(orderBook *matching.OrderBook, update matching.PriceLevelUpdate) {
			t.Logf("add price level for %s\n", update.Price.ToFloatString())
		}).AnyTimes()
	handler.EXPECT().OnUpdatePriceLevel(gomock.Any(), gomock.Any()).AnyTimes()
	handler.EXPECT().OnDeletePriceLevel(gomock.Any(), gomock.Any()).AnyTimes()
	handler.EXPECT().OnUpdateOrderBook(gomock.Any()).AnyTimes()
	handler.EXPECT().OnExecuteOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Do(
		func(orderBook *matching.OrderBook, orderID uint64, price matching.Uint, quantity matching.Uint, quoteQuantity matching.Uint) {
			t.Logf("order %d executed: price %s, qty %s, quoteQty %s\n",
				orderID,
				price.ToFloatString(), quantity.ToFloatString(),
				quoteQuantity.ToFloatString(),
			)
		}).AnyTimes()
	handler.EXPECT().OnExecuteTrade(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
}
