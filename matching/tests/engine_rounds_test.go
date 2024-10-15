package matching_test

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	matching "github.com/cryptonstudio/crypton-matching-engine/matching"
	mockmatching "github.com/cryptonstudio/crypton-matching-engine/matching/mocks"
)

func TestCalculationRound(t *testing.T) {
	const (
		symbolID   uint32 = 10
		orderID    uint64 = 100
		newOrderID uint64 = 101
	)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("add limit order", func(t *testing.T) {
		handler := mockmatching.NewMockHandler(ctrl)
		handler.EXPECT().OnAddOrderBook(gomock.Any()).AnyTimes()
		handler.EXPECT().OnAddOrder(gomock.Any(), gomock.Any()).AnyTimes()
		handler.EXPECT().OnAddPriceLevel(gomock.Any(), gomock.Any()).AnyTimes()
		handler.EXPECT().OnUpdatePriceLevel(gomock.Any(), gomock.Any()).AnyTimes()
		handler.EXPECT().OnUpdateOrderBook(gomock.Any()).AnyTimes()
		handler.EXPECT().OnUpdateOrder(gomock.Any(), gomock.Any()).AnyTimes()
		handler.EXPECT().OnDeleteOrder(gomock.Any(), gomock.Any()).AnyTimes()
		handler.EXPECT().OnExecuteOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		handler.EXPECT().OnExecuteTrade(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		engine := matching.NewEngine(handler, false)

		_, err := engine.AddOrderBook(matching.NewSymbolWithLimits(
			symbolID,
			"",
			matching.Limits{
				Min:  matching.NewUint(48).Mul64(matching.UintPrecision).Div64(100),
				Max:  matching.NewUint(255).Mul64(matching.UintPrecision).Div64(100),
				Step: matching.NewUint(1).Mul64(matching.UintPrecision).Div64(100),
			},
			matching.Limits{
				Min:  matching.NewUint(48).Mul64(matching.UintPrecision).Div64(100),
				Max:  matching.NewUint(49).Mul64(matching.UintPrecision).Div64(100),
				Step: matching.NewUint(1).Mul64(matching.UintPrecision).Div64(100),
			},
		),
			matching.NewUint(0), matching.StopPriceModeConfig{Market: true, Mark: true, Index: true})
		require.NoError(t, err)

		engine.SetIndexPriceForOrderBook(symbolID, matching.NewUint(48).Mul64(matching.UintPrecision).Div64(100), false) //nolint:errcheck
		engine.SetMarkPriceForOrderBook(symbolID, matching.NewUint(48).Mul64(matching.UintPrecision).Div64(100), false)  //nolint:errcheck

		// 1. limit
		{
			price := matching.NewUint(207).Mul64(matching.UintPrecision).Div64(100)
			quantity := matching.NewUint(48).Mul64(matching.UintPrecision).Div64(100)
			restLocked := quantity.Mul(price).Div64(matching.UintPrecision)
			order := matching.NewLimitOrder(
				symbolID,
				1,
				matching.OrderSideBuy,
				matching.OrderDirectionOpen,
				matching.OrderTimeInForceGTC,
				price,
				quantity,
				matching.NewMaxUint(),
				restLocked,
			)
			require.NoError(t, order.CheckLocked())
			err = engine.AddOrder(order)
			require.NoError(t, err)
		}

		// 2. market
		{
			price := matching.NewUint(48).Mul64(matching.UintPrecision).Div64(100)
			quantity := matching.NewUint(50).Mul64(matching.UintPrecision).Div64(100)
			order := matching.NewStopOrder(
				symbolID,
				2,
				matching.OrderSideSell,
				matching.OrderDirectionOpen,
				matching.OrderTimeInForceIOC,
				matching.StopPriceModeIndex,
				price,
				quantity,
				matching.NewZeroUint(),
				price,
				quantity,
			)
			err = engine.AddOrder(order)
			require.NoError(t, err)
		}

		// 3. market
		{
			price := matching.NewUint(48).Mul64(matching.UintPrecision).Div64(100)
			quantity := matching.NewUint(32).Mul64(matching.UintPrecision).Div64(100)
			order := matching.NewStopOrder(
				symbolID,
				3,
				matching.OrderSideSell,
				matching.OrderDirectionOpen,
				matching.OrderTimeInForceIOC,
				matching.StopPriceModeIndex,
				price,
				quantity,
				matching.NewZeroUint(),
				price,
				quantity,
			)
			err = engine.AddOrder(order)
			require.NoError(t, err)
		}

		// activation
		err = engine.SetIndexMarkPricesForOrderBook(symbolID,
			matching.NewUint(48).Mul64(matching.UintPrecision).Div64(100),
			matching.NewUint(48).Mul64(matching.UintPrecision).Div64(100),
			true)
		require.NoError(t, err)

		ob := engine.OrderBook(symbolID)
		limit := ob.Order(1)
		require.NoError(t, limit.CheckLocked())
	})
}
