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

		_, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0))
		require.NoError(t, err)

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			orderID,
			matching.OrderSideBuy,
			matching.NewUint(10),
			matching.NewUint(100),
			matching.NewZeroUint(),
			matching.NewZeroUint(),
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
		handler.EXPECT().OnExecuteOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(2)
		handler.EXPECT().OnExecuteTrade(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

		engine := matching.NewEngine(handler, false)

		_, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0))
		require.NoError(t, err)

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			orderID,
			matching.OrderSideBuy,
			matching.NewUint(10),
			matching.NewUint(100),
			matching.NewZeroUint(),
			matching.NewUint(100),
		))
		require.NoError(t, err)

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			orderID+1,
			matching.OrderSideSell,
			matching.NewUint(10),
			matching.NewUint(100),
			matching.NewZeroUint(),
			matching.NewUint(100),
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
		handler.EXPECT().OnExecuteOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(2)
		handler.EXPECT().OnExecuteTrade(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
		handler.EXPECT().OnUpdateOrder(gomock.Any(), gomock.Any()).Times(1)

		engine := matching.NewEngine(handler, false)

		_, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0))
		require.NoError(t, err)

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			orderID,
			matching.OrderSideBuy,
			matching.NewUint(10),
			matching.NewUint(100),
			matching.NewZeroUint(),
			matching.NewUint(100),
		))
		require.NoError(t, err)

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			orderID+1,
			matching.OrderSideSell,
			matching.NewUint(10),
			matching.NewUint(200),
			matching.NewZeroUint(),
			matching.NewUint(200),
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
			require.True(t, order.RestQuantity().Equals(matching.NewUint(99)))
		})
		handler.EXPECT().OnUpdatePriceLevel(gomock.Any(), gomock.Any()).Times(1)
		handler.EXPECT().OnUpdateOrderBook(gomock.Any()).Times(1)

		engine := matching.NewEngine(handler, false)

		_, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0))
		require.NoError(t, err)

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			orderID,
			matching.OrderSideBuy,
			matching.NewUint(10),
			matching.NewUint(100),
			matching.NewZeroUint(),
			matching.NewZeroUint(),
		))
		require.NoError(t, err)

		err = engine.ReduceOrder(symbolID, orderID, matching.NewUint(1))
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

		_, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0))
		require.NoError(t, err)

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			orderID,
			matching.OrderSideBuy,
			matching.NewUint(10),
			matching.NewUint(100),
			matching.NewZeroUint(),
			matching.NewZeroUint(),
		))
		require.NoError(t, err)

		err = engine.ReduceOrder(symbolID, orderID, matching.NewUint(101))
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
			require.True(t, order.Price().Equals(matching.NewUint(11)))
			require.True(t, order.Quantity().Equals(matching.NewUint(101)))
			require.True(t, order.RestQuantity().Equals(matching.NewUint(101)))
		})
		handler.EXPECT().OnDeletePriceLevel(gomock.Any(), gomock.Any()).Do(
			func(orderBook *matching.OrderBook, update matching.PriceLevelUpdate) {
				require.True(t, update.Price.Equals(matching.NewUint(10)))
			}).Times(1)
		handler.EXPECT().OnAddPriceLevel(gomock.Any(), gomock.Any()).Do(
			func(orderBook *matching.OrderBook, update matching.PriceLevelUpdate) {
				require.True(t, update.Price.Equals(matching.NewUint(11)))
			}).Times(1)
		handler.EXPECT().OnUpdateOrderBook(gomock.Any()).Times(2) // delete prive level + add price level

		engine := matching.NewEngine(handler, false)

		_, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0))
		require.NoError(t, err)

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			orderID,
			matching.OrderSideBuy,
			matching.NewUint(10),
			matching.NewUint(100),
			matching.NewZeroUint(),
			matching.NewZeroUint(),
		))
		require.NoError(t, err)

		err = engine.MitigateOrder(symbolID, orderID, matching.NewUint(11), matching.NewUint(101), matching.NewZeroUint())
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
			require.True(t, order.Price().Equals(matching.NewUint(9)))
			require.True(t, order.Quantity().Equals(matching.NewUint(90)))
			require.True(t, order.RestQuantity().Equals(matching.NewUint(90)))
		})
		handler.EXPECT().OnDeletePriceLevel(gomock.Any(), gomock.Any()).Do(
			func(orderBook *matching.OrderBook, update matching.PriceLevelUpdate) {
				require.True(t, update.Price.Equals(matching.NewUint(10)))
			}).Times(1)
		handler.EXPECT().OnAddPriceLevel(gomock.Any(), gomock.Any()).Do(
			func(orderBook *matching.OrderBook, update matching.PriceLevelUpdate) {
				require.True(t, update.Price.Equals(matching.NewUint(9)))
			}).Times(1)
		handler.EXPECT().OnUpdateOrderBook(gomock.Any()).Times(2)

		engine := matching.NewEngine(handler, false)

		_, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0))
		require.NoError(t, err)

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			orderID,
			matching.OrderSideBuy,
			matching.NewUint(10),
			matching.NewUint(100),
			matching.NewZeroUint(),
			matching.NewZeroUint(),
		))
		require.NoError(t, err)

		err = engine.MitigateOrder(symbolID, orderID, matching.NewUint(9), matching.NewUint(90), matching.NewZeroUint())
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
			require.True(t, order.Price().Equals(matching.NewUint(11)))
			require.True(t, order.Quantity().Equals(matching.NewUint(101)))
			require.True(t, order.RestQuantity().Equals(matching.NewUint(101)))
		})
		handler.EXPECT().OnDeletePriceLevel(gomock.Any(), gomock.Any()).Do(
			func(orderBook *matching.OrderBook, update matching.PriceLevelUpdate) {
				require.True(t, update.Price.Equals(matching.NewUint(10)))
			}).Times(1)
		handler.EXPECT().OnAddPriceLevel(gomock.Any(), gomock.Any()).Do(
			func(orderBook *matching.OrderBook, update matching.PriceLevelUpdate) {
				require.True(t, update.Price.Equals(matching.NewUint(11)))
			}).Times(1)
		handler.EXPECT().OnUpdateOrderBook(gomock.Any()).Times(2) // delete prive level + add price level

		engine := matching.NewEngine(handler, false)

		_, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0))
		require.NoError(t, err)

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			orderID,
			matching.OrderSideBuy,
			matching.NewUint(10),
			matching.NewUint(100),
			matching.NewZeroUint(),
			matching.NewZeroUint(),
		))
		require.NoError(t, err)

		err = engine.ModifyOrder(symbolID, orderID, matching.NewUint(11), matching.NewUint(101))
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
				require.True(t, update.Price.Equals(matching.NewUint(10)))
			}).Times(1)
		handler.EXPECT().OnUpdateOrderBook(gomock.Any()).Times(2) // delete prive level + add price level
		handler.EXPECT().OnDeleteOrder(gomock.Any(), gomock.Any()).Times(1)
		handler.EXPECT().OnAddOrder(gomock.Any(), gomock.Any()).Do(
			func(orderBook *matching.OrderBook, order *matching.Order) {
				require.True(t, order.Price().Equals(matching.NewUint(11)))
			}).Times(1)
		handler.EXPECT().OnAddPriceLevel(gomock.Any(), gomock.Any()).Times(1)
		handler.EXPECT().OnUpdateOrderBook(gomock.Any()).Times(1)

		engine := matching.NewEngine(handler, false)

		_, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0))
		require.NoError(t, err)

		err = engine.AddOrder(matching.NewLimitOrder(
			symbolID,
			orderID,
			matching.OrderSideBuy,
			matching.NewUint(10),
			matching.NewUint(100),
			matching.NewZeroUint(),
			matching.NewUint(100),
		))
		require.NoError(t, err)

		err = engine.ReplaceOrder(symbolID, orderID, newOrderID, matching.NewUint(11), matching.NewUint(101))
		require.NoError(t, err)
	})
}
