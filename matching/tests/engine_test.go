package matching_test

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	matching "github.com/cryptonstudio/crypton-matching-engine/matching"
	mockmatching "github.com/cryptonstudio/crypton-matching-engine/matching/mocks"
)

//nolint:maintidx
func TestBasic(t *testing.T) {
	const (
		symbolID   uint32 = 10
		orderID    uint64 = 100
		newOrderID uint64 = 101
	)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("add limit order", func(t *testing.T) {
		handler := mockmatching.NewMockHandler(ctrl)
		handler.EXPECT().OnAddOrderBook(gomock.Any())
		handler.EXPECT().OnAddOrder(gomock.Any(), gomock.Any())
		handler.EXPECT().OnAddPriceLevel(gomock.Any(), gomock.Any())
		handler.EXPECT().OnUpdateOrderBook(gomock.Any())

		engine := matching.NewEngine(handler, false)

		_, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0), matching.StopPriceModeConfig{Market: true})
		require.NoError(t, err)

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			orderID,
			matching.OrderSideBuy,
			matching.OrderTimeInForceGTC,
			matching.NewUint(10).Mul64(matching.UintPrecision),
			matching.NewUint(100).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewUint(1000).Mul64(matching.UintPrecision),
		))
		require.NoError(t, err)

		require.Equal(t, 1, engine.Orders())
	})

	t.Run("simple match", func(t *testing.T) {
		handler := mockmatching.NewMockHandler(ctrl)
		// order adding
		handler.EXPECT().OnAddOrderBook(gomock.Any()).Times(1)
		handler.EXPECT().OnAddOrder(gomock.Any(), gomock.Any()).Times(2)
		handler.EXPECT().OnAddPriceLevel(gomock.Any(), gomock.Any()).Times(2)
		handler.EXPECT().OnUpdateOrderBook(gomock.Any()).Times(2)
		// matching
		handler.EXPECT().OnDeletePriceLevel(gomock.Any(), gomock.Any()).Times(2)
		handler.EXPECT().OnUpdateOrderBook(gomock.Any()).Times(2)
		handler.EXPECT().OnDeleteOrder(gomock.Any(), gomock.Any()).Times(2)
		handler.EXPECT().OnExecuteOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(2)
		handler.EXPECT().OnExecuteTrade(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

		engine := matching.NewEngine(handler, false)

		_, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0), matching.StopPriceModeConfig{Market: true})
		require.NoError(t, err)

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			orderID,
			matching.OrderSideBuy,
			matching.OrderTimeInForceGTC,
			matching.NewUint(10).Mul64(matching.UintPrecision),
			matching.NewUint(100).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewUint(1000).Mul64(matching.UintPrecision),
		))
		require.NoError(t, err)

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			orderID+1,
			matching.OrderSideSell,
			matching.OrderTimeInForceGTC,
			matching.NewUint(10).Mul64(matching.UintPrecision),
			matching.NewUint(100).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewUint(100).Mul64(matching.UintPrecision),
		))
		require.NoError(t, err)

		engine.Match()
	})

	t.Run("simple match (changed order)", func(t *testing.T) {
		handler := mockmatching.NewMockHandler(ctrl)
		// order adding
		handler.EXPECT().OnAddOrderBook(gomock.Any()).Times(1)
		handler.EXPECT().OnAddOrder(gomock.Any(), gomock.Any()).Times(2)
		handler.EXPECT().OnAddPriceLevel(gomock.Any(), gomock.Any()).Times(2)
		handler.EXPECT().OnUpdateOrderBook(gomock.Any()).Times(2)
		// matching
		handler.EXPECT().OnDeletePriceLevel(gomock.Any(), gomock.Any()).Times(2)
		handler.EXPECT().OnUpdateOrderBook(gomock.Any()).Times(2)
		handler.EXPECT().OnDeleteOrder(gomock.Any(), gomock.Any()).Times(2)
		handler.EXPECT().OnExecuteOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(2)
		handler.EXPECT().OnExecuteTrade(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

		engine := matching.NewEngine(handler, false)

		_, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0), matching.StopPriceModeConfig{Market: true})
		require.NoError(t, err)

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			orderID,
			matching.OrderSideSell,
			matching.OrderTimeInForceGTC,
			matching.NewUint(10).Mul64(matching.UintPrecision),
			matching.NewUint(100).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewUint(100).Mul64(matching.UintPrecision),
		))
		require.NoError(t, err)

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			orderID+1,
			matching.OrderSideBuy,
			matching.OrderTimeInForceGTC,
			matching.NewUint(10).Mul64(matching.UintPrecision),
			matching.NewUint(100).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewUint(1000).Mul64(matching.UintPrecision),
		))
		require.NoError(t, err)

		engine.Match()
	})

	t.Run("partial match", func(t *testing.T) {
		handler := mockmatching.NewMockHandler(ctrl)
		// order adding
		handler.EXPECT().OnAddOrderBook(gomock.Any()).Times(1)
		handler.EXPECT().OnAddOrder(gomock.Any(), gomock.Any()).Times(2)
		handler.EXPECT().OnAddPriceLevel(gomock.Any(), gomock.Any()).Times(2)
		handler.EXPECT().OnUpdateOrderBook(gomock.Any()).Times(2)
		// matching
		handler.EXPECT().OnDeletePriceLevel(gomock.Any(), gomock.Any()).Times(1)
		handler.EXPECT().OnUpdatePriceLevel(gomock.Any(), gomock.Any()).Times(1)
		handler.EXPECT().OnUpdateOrderBook(gomock.Any()).Times(2)
		handler.EXPECT().OnDeleteOrder(gomock.Any(), gomock.Any()).Times(1)
		handler.EXPECT().OnExecuteOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(2)
		handler.EXPECT().OnExecuteTrade(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
		handler.EXPECT().OnUpdateOrder(gomock.Any(), gomock.Any()).Times(1)

		engine := matching.NewEngine(handler, false)

		_, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0), matching.StopPriceModeConfig{Market: true})
		require.NoError(t, err)

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			orderID,
			matching.OrderSideBuy,
			matching.OrderTimeInForceGTC,
			matching.NewUint(10).Mul64(matching.UintPrecision),
			matching.NewUint(100).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewUint(1000).Mul64(matching.UintPrecision),
		))
		require.NoError(t, err)

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			orderID+1,
			matching.OrderSideSell,
			matching.OrderTimeInForceGTC,
			matching.NewUint(10).Mul64(matching.UintPrecision),
			matching.NewUint(200).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewUint(200).Mul64(matching.UintPrecision),
		))
		require.NoError(t, err)

		engine.Match()
	})

	t.Run("partial match (changed sides)", func(t *testing.T) {
		handler := mockmatching.NewMockHandler(ctrl)
		// order adding
		handler.EXPECT().OnAddOrderBook(gomock.Any()).Times(1)
		handler.EXPECT().OnAddOrder(gomock.Any(), gomock.Any()).Times(2)
		handler.EXPECT().OnAddPriceLevel(gomock.Any(), gomock.Any()).Times(2)
		handler.EXPECT().OnUpdateOrderBook(gomock.Any()).Times(2)
		// matching
		handler.EXPECT().OnDeletePriceLevel(gomock.Any(), gomock.Any()).Times(1)
		handler.EXPECT().OnUpdatePriceLevel(gomock.Any(), gomock.Any()).Times(1)
		handler.EXPECT().OnUpdateOrderBook(gomock.Any()).Times(2)
		handler.EXPECT().OnDeleteOrder(gomock.Any(), gomock.Any()).Times(1)
		handler.EXPECT().OnExecuteOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(2)
		handler.EXPECT().OnExecuteTrade(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
		handler.EXPECT().OnUpdateOrder(gomock.Any(), gomock.Any()).Times(1)

		engine := matching.NewEngine(handler, false)

		_, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0), matching.StopPriceModeConfig{Market: true})
		require.NoError(t, err)

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			orderID,
			matching.OrderSideBuy,
			matching.OrderTimeInForceGTC,
			matching.NewUint(10).Mul64(matching.UintPrecision),
			matching.NewUint(200).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewUint(2000).Mul64(matching.UintPrecision),
		))
		require.NoError(t, err)

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			orderID+1,
			matching.OrderSideSell,
			matching.OrderTimeInForceGTC,
			matching.NewUint(10).Mul64(matching.UintPrecision),
			matching.NewUint(100).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewUint(100).Mul64(matching.UintPrecision),
		))
		require.NoError(t, err)

		engine.Match()
	})

	t.Run("reduce", func(t *testing.T) {
		handler := mockmatching.NewMockHandler(ctrl)
		// order adding
		handler.EXPECT().OnAddOrderBook(gomock.Any()).Times(1)
		handler.EXPECT().OnAddOrder(gomock.Any(), gomock.Any()).Times(1)
		handler.EXPECT().OnAddPriceLevel(gomock.Any(), gomock.Any()).Times(1)
		handler.EXPECT().OnUpdateOrderBook(gomock.Any()).Times(1)
		// reduce
		handler.EXPECT().OnUpdateOrder(gomock.Any(), gomock.Any()).Do(func(orderBook *matching.OrderBook, order *matching.Order) {
			require.True(t, order.RestQuantity().Equals(matching.NewUint(99).Mul64(matching.UintPrecision)))
		})
		handler.EXPECT().OnUpdatePriceLevel(gomock.Any(), gomock.Any()).Times(1)
		handler.EXPECT().OnUpdateOrderBook(gomock.Any()).Times(1)

		engine := matching.NewEngine(handler, false)

		_, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0), matching.StopPriceModeConfig{Market: true})
		require.NoError(t, err)

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			orderID,
			matching.OrderSideBuy,
			matching.OrderTimeInForceGTC,
			matching.NewUint(10).Mul64(matching.UintPrecision),
			matching.NewUint(100).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewUint(1000).Mul64(matching.UintPrecision),
		))
		require.NoError(t, err)

		err = engine.ReduceOrder(symbolID, orderID, matching.NewUint(1).Mul64(matching.UintPrecision))
		require.NoError(t, err)
	})

	t.Run("reduce too much", func(t *testing.T) {
		handler := mockmatching.NewMockHandler(ctrl)
		// order adding
		handler.EXPECT().OnAddOrderBook(gomock.Any()).Times(1)
		handler.EXPECT().OnAddOrder(gomock.Any(), gomock.Any()).Times(1)
		handler.EXPECT().OnAddPriceLevel(gomock.Any(), gomock.Any()).Times(1)
		handler.EXPECT().OnUpdateOrderBook(gomock.Any()).Times(1)
		// reduce
		handler.EXPECT().OnDeleteOrder(gomock.Any(), gomock.Any()).Do(func(orderBook *matching.OrderBook, order *matching.Order) {
			require.True(t, order.RestQuantity().Equals(matching.NewUint(0)))
		})
		handler.EXPECT().OnDeletePriceLevel(gomock.Any(), gomock.Any()).Times(1)
		handler.EXPECT().OnUpdateOrderBook(gomock.Any()).Times(1)

		engine := matching.NewEngine(handler, false)

		_, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0), matching.StopPriceModeConfig{Market: true})
		require.NoError(t, err)

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			orderID,
			matching.OrderSideBuy,
			matching.OrderTimeInForceGTC,
			matching.NewUint(10).Mul64(matching.UintPrecision),
			matching.NewUint(100).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewUint(1000).Mul64(matching.UintPrecision),
		))
		require.NoError(t, err)

		err = engine.ReduceOrder(symbolID, orderID, matching.NewUint(101).Mul64(matching.UintPrecision))
		require.NoError(t, err)
	})

	t.Run("mitigate to up", func(t *testing.T) {
		handler := mockmatching.NewMockHandler(ctrl)
		// order adding
		handler.EXPECT().OnAddOrderBook(gomock.Any()).Times(1)
		handler.EXPECT().OnAddOrder(gomock.Any(), gomock.Any()).Times(1)
		handler.EXPECT().OnAddPriceLevel(gomock.Any(), gomock.Any()).Times(1)
		handler.EXPECT().OnUpdateOrderBook(gomock.Any()).Times(1)
		// modify
		handler.EXPECT().OnUpdateOrder(gomock.Any(), gomock.Any()).Do(func(orderBook *matching.OrderBook, order *matching.Order) {
			require.True(t, order.Price().Equals(matching.NewUint(11).Mul64(matching.UintPrecision)))
			require.True(t, order.Quantity().Equals(matching.NewUint(101).Mul64(matching.UintPrecision)))
			require.True(t, order.RestQuantity().Equals(matching.NewUint(101).Mul64(matching.UintPrecision)))
		})
		handler.EXPECT().OnDeletePriceLevel(gomock.Any(), gomock.Any()).Do(
			func(orderBook *matching.OrderBook, update matching.PriceLevelUpdate) {
				require.True(t, update.Price.Equals(matching.NewUint(10).Mul64(matching.UintPrecision)))
			}).Times(1)
		handler.EXPECT().OnAddPriceLevel(gomock.Any(), gomock.Any()).Do(
			func(orderBook *matching.OrderBook, update matching.PriceLevelUpdate) {
				require.True(t, update.Price.Equals(matching.NewUint(11).Mul64(matching.UintPrecision)))
			}).Times(1)
		handler.EXPECT().OnUpdateOrderBook(gomock.Any()).Times(2) // delete prive level + add price level

		engine := matching.NewEngine(handler, false)

		_, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0), matching.StopPriceModeConfig{Market: true})
		require.NoError(t, err)

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			orderID,
			matching.OrderSideBuy,
			matching.OrderTimeInForceGTC,
			matching.NewUint(10).Mul64(matching.UintPrecision),
			matching.NewUint(100).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewUint(1000).Mul64(matching.UintPrecision),
		))
		require.NoError(t, err)

		err = engine.MitigateOrder(
			symbolID, orderID, matching.NewUint(11).Mul64(matching.UintPrecision),
			matching.NewUint(101).Mul64(matching.UintPrecision), matching.NewUint(1).Mul64(matching.UintPrecision),
		)
		require.NoError(t, err)
	})

	t.Run("mitigate to down", func(t *testing.T) {
		handler := mockmatching.NewMockHandler(ctrl)
		// order adding
		handler.EXPECT().OnAddOrderBook(gomock.Any()).Times(1)
		handler.EXPECT().OnAddOrder(gomock.Any(), gomock.Any()).Times(1)
		handler.EXPECT().OnAddPriceLevel(gomock.Any(), gomock.Any()).Times(1)
		handler.EXPECT().OnUpdateOrderBook(gomock.Any()).Times(1)
		// modify
		handler.EXPECT().OnUpdateOrder(gomock.Any(), gomock.Any()).Do(func(orderBook *matching.OrderBook, order *matching.Order) {
			require.True(t, order.Price().Equals(matching.NewUint(9).Mul64(matching.UintPrecision)))
			require.True(t, order.Quantity().Equals(matching.NewUint(90).Mul64(matching.UintPrecision)))
			require.True(t, order.RestQuantity().Equals(matching.NewUint(90).Mul64(matching.UintPrecision)))
		})
		handler.EXPECT().OnDeletePriceLevel(gomock.Any(), gomock.Any()).Do(
			func(orderBook *matching.OrderBook, update matching.PriceLevelUpdate) {
				require.True(t, update.Price.Equals(matching.NewUint(10).Mul64(matching.UintPrecision)))
			}).Times(1)
		handler.EXPECT().OnAddPriceLevel(gomock.Any(), gomock.Any()).Do(
			func(orderBook *matching.OrderBook, update matching.PriceLevelUpdate) {
				require.True(t, update.Price.Equals(matching.NewUint(9).Mul64(matching.UintPrecision)))
			}).Times(1)
		handler.EXPECT().OnUpdateOrderBook(gomock.Any()).Times(2)

		engine := matching.NewEngine(handler, false)

		_, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0), matching.StopPriceModeConfig{Market: true})
		require.NoError(t, err)

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			orderID,
			matching.OrderSideBuy,
			matching.OrderTimeInForceGTC,
			matching.NewUint(10).Mul64(matching.UintPrecision),
			matching.NewUint(100).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewUint(1000).Mul64(matching.UintPrecision),
		))
		require.NoError(t, err)

		err = engine.MitigateOrder(
			symbolID, orderID, matching.NewUint(9).Mul64(matching.UintPrecision),
			matching.NewUint(90).Mul64(matching.UintPrecision), matching.NewZeroUint(),
		)
		require.NoError(t, err)
	})

	t.Run("modify", func(t *testing.T) {
		handler := mockmatching.NewMockHandler(ctrl)
		// order adding
		handler.EXPECT().OnAddOrderBook(gomock.Any()).Times(1)
		handler.EXPECT().OnAddOrder(gomock.Any(), gomock.Any()).Times(1)
		handler.EXPECT().OnAddPriceLevel(gomock.Any(), gomock.Any()).Times(1)
		handler.EXPECT().OnUpdateOrderBook(gomock.Any()).Times(1)
		// modify
		handler.EXPECT().OnUpdateOrder(gomock.Any(), gomock.Any()).Do(func(orderBook *matching.OrderBook, order *matching.Order) {
			require.True(t, order.Price().Equals(matching.NewUint(11).Mul64(matching.UintPrecision)))
			require.True(t, order.Quantity().Equals(matching.NewUint(101).Mul64(matching.UintPrecision)))
			require.True(t, order.RestQuantity().Equals(matching.NewUint(101).Mul64(matching.UintPrecision)))
		})
		handler.EXPECT().OnDeletePriceLevel(gomock.Any(), gomock.Any()).Do(
			func(orderBook *matching.OrderBook, update matching.PriceLevelUpdate) {
				require.True(t, update.Price.Equals(matching.NewUint(10).Mul64(matching.UintPrecision)))
			}).Times(1)
		handler.EXPECT().OnAddPriceLevel(gomock.Any(), gomock.Any()).Do(
			func(orderBook *matching.OrderBook, update matching.PriceLevelUpdate) {
				require.True(t, update.Price.Equals(matching.NewUint(11).Mul64(matching.UintPrecision)))
			}).Times(1)
		handler.EXPECT().OnUpdateOrderBook(gomock.Any()).Times(2) // delete prive level + add price level

		engine := matching.NewEngine(handler, false)

		_, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0), matching.StopPriceModeConfig{Market: true})
		require.NoError(t, err)

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			orderID,
			matching.OrderSideBuy,
			matching.OrderTimeInForceGTC,
			matching.NewUint(10).Mul64(matching.UintPrecision),
			matching.NewUint(100).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewUint(1000).Mul64(matching.UintPrecision),
		))
		require.NoError(t, err)

		err = engine.ModifyOrder(
			symbolID, orderID, matching.NewUint(11).Mul64(matching.UintPrecision),
			matching.NewUint(101).Mul64(matching.UintPrecision),
		)
		require.NoError(t, err)
	})

	t.Run("replace", func(t *testing.T) {
		handler := mockmatching.NewMockHandler(ctrl)
		// order adding
		handler.EXPECT().OnAddOrderBook(gomock.Any()).Times(1)
		handler.EXPECT().OnAddOrder(gomock.Any(), gomock.Any()).Times(1)
		handler.EXPECT().OnAddPriceLevel(gomock.Any(), gomock.Any()).Times(1)
		// replace
		handler.EXPECT().OnDeletePriceLevel(gomock.Any(), gomock.Any()).Do(
			func(orderBook *matching.OrderBook, update matching.PriceLevelUpdate) {
				require.True(t, update.Price.Equals(matching.NewUint(10).Mul64(matching.UintPrecision)))
			}).Times(1)
		handler.EXPECT().OnUpdateOrderBook(gomock.Any()).Times(2) // delete prive level + add price level
		handler.EXPECT().OnDeleteOrder(gomock.Any(), gomock.Any()).Times(1)
		handler.EXPECT().OnAddOrder(gomock.Any(), gomock.Any()).Do(
			func(orderBook *matching.OrderBook, order *matching.Order) {
				require.True(t, order.Price().Equals(matching.NewUint(11).Mul64(matching.UintPrecision)))
			}).Times(1)
		handler.EXPECT().OnAddPriceLevel(gomock.Any(), gomock.Any()).Times(1)
		handler.EXPECT().OnUpdateOrderBook(gomock.Any()).Times(1)

		engine := matching.NewEngine(handler, false)

		_, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0), matching.StopPriceModeConfig{Market: true})
		require.NoError(t, err)

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			orderID,
			matching.OrderSideBuy,
			matching.OrderTimeInForceGTC,
			matching.NewUint(10).Mul64(matching.UintPrecision),
			matching.NewUint(100).Mul64(matching.UintPrecision),
			matching.NewZeroUint(),
			matching.NewUint(1000).Mul64(matching.UintPrecision),
		))
		require.NoError(t, err)

		err = engine.ReplaceOrder(
			symbolID, orderID, newOrderID, matching.NewUint(11).Mul64(matching.UintPrecision),
			matching.NewUint(101).Mul64(matching.UintPrecision),
		)
		require.NoError(t, err)
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
