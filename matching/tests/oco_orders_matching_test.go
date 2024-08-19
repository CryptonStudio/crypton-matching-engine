package matching_test

import (
	"testing"

	matching "github.com/cryptonstudio/crypton-matching-engine/matching"
	mockmatching "github.com/cryptonstudio/crypton-matching-engine/matching/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

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

		err = engine.AddOrdersPair(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.OrderDirectionOpen,
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
				matching.OrderDirectionOpen,
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

		err = engine.AddOrdersPair(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.OrderDirectionOpen,
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
				matching.OrderDirectionOpen,
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

		err = engine.AddOrdersPair(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.OrderDirectionOpen,
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
				matching.OrderDirectionOpen,
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

		err = engine.AddOrdersPair(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideSell,
				matching.OrderDirectionClose,
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
				matching.OrderDirectionClose,
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

		err = engine.AddOrdersPair(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideSell,
				matching.OrderDirectionClose,
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
				matching.OrderDirectionClose,
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

		err = engine.AddOrdersPair(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.OrderDirectionOpen,
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
				matching.OrderDirectionOpen,
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

		err = engine.AddOrdersPair(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.OrderDirectionOpen,
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
				matching.OrderDirectionOpen,
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

		err = engine.AddOrdersPair(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.OrderDirectionOpen,
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
				matching.OrderDirectionOpen,
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

		err = engine.AddOrdersPair(
			matching.NewStopLimitOrder(
				symbolID,
				uint64(6),
				matching.OrderSideBuy,
				matching.OrderDirectionOpen,
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
				matching.OrderDirectionOpen,
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
			matching.OrderDirectionClose,
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
				matching.OrderDirectionOpen,
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
				matching.OrderDirectionOpen,
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
			quantity: matching.NewUint(10).Mul64(matching.UintPrecision),       // amount 10
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
			matching.OrderDirectionOpen,
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
			matching.OrderDirectionClose,
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
				require.True(t, makerOrderUpdate.Quantity.Equals(firstTrade.quantity),
					"%s != %s", makerOrderUpdate.Quantity.ToFloatString(), firstTrade.quantity.ToFloatString())
				require.True(t, takerOrderUpdate.Quantity.Equals(firstTrade.quantity),
					"%s != %s", takerOrderUpdate.Quantity.ToFloatString(), firstTrade.quantity.ToFloatString())

				quoteQty := firstTrade.quantity.Mul(firstTrade.price).Div64(matching.UintPrecision)
				require.True(t, makerOrderUpdate.QuoteQuantity.Equals(quoteQty),
					"%s != %s", makerOrderUpdate.QuoteQuantity.ToFloatString(), quoteQty.ToFloatString())
				require.True(t, takerOrderUpdate.QuoteQuantity.Equals(quoteQty),
					"%s != %s", takerOrderUpdate.QuoteQuantity.ToFloatString(), quoteQty.ToFloatString())

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
				matching.OrderDirectionClose,
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
				matching.OrderDirectionClose,
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
			matching.OrderDirectionOpen,
			matching.OrderTimeInForceGTC,
			secondTrade.limitPrice,
			secondTrade.quantity,
			matching.NewMaxUint(),
			matching.NewMaxUint(),
		))
		require.NoError(t, err)
	})
}
