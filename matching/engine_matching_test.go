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

	setupMarketState := func(t *testing.T, engine *matching.Engine) {
		_, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""))
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

	setupHandler := func(t *testing.T) matching.Handler {
		handler := mockmatching.NewMockHandler(ctrl)
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

		setupMarketState(t, engine)

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

		setupMarketState(t, engine)

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

		setupMarketState(t, engine)

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

		setupMarketState(t, engine)

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

		setupMarketState(t, engine)

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
