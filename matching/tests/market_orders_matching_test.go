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
			matching.OrderDirectionOpen,
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
			matching.OrderDirectionClose,
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
			matching.OrderDirectionOpen,
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
			matching.OrderDirectionOpen,
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
			matching.OrderDirectionOpen,
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
			matching.OrderDirectionOpen,
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
			matching.OrderDirectionOpen,
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
			matching.OrderDirectionClose,
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
			matching.OrderDirectionClose,
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
			matching.OrderDirectionClose,
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
			matching.OrderDirectionClose,
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
