package matching_test

import (
	"testing"
	"time"

	matching "github.com/cryptonstudio/crypton-matching-engine/matching"
	mockmatching "github.com/cryptonstudio/crypton-matching-engine/matching/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestOrderError(t *testing.T) {
	const symbolID uint32 = 10

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testCases := []struct {
		name          string
		run           func(engine *matching.Engine) error
		op            matching.OrderOp
		orderID       uint64
		linkedOrderID uint64
		cause         error
	}{
		{
			name: "add",
			run: func(engine *matching.Engine) error {
				return engine.AddOrder(matching.NewLimitOrder(
					symbolID,
					uint64(1),
					matching.OrderSideBuy,
					matching.OrderDirectionOpen,
					matching.OrderTimeInForceGTC,
					matching.NewUint(10).Mul64(matching.UintPrecision),
					matching.NewUint(1).Mul64(matching.UintPrecision),
					matching.NewZeroUint(),
					matching.NewMaxUint(),
				))
			},
			op:      matching.OrderOpAdd,
			orderID: 1,
			cause:   matching.ErrOrderDuplicate,
		},
		{
			name: "add pair",
			run: func(engine *matching.Engine) error {
				return engine.AddOrdersPair(
					matching.NewStopLimitOrder(
						symbolID,
						uint64(6),
						matching.OrderSideSell,
						matching.OrderDirectionClose,
						matching.OrderTimeInForceGTC,
						matching.NewUint(20).Mul64(matching.UintPrecision),
						matching.StopPriceModeMarket,
						matching.NewUint(35).Mul64(matching.UintPrecision),
						matching.NewUint(3).Mul64(matching.UintPrecision),
						matching.NewMaxUint(),
						matching.NewZeroUint(),
					),
					matching.NewLimitOrder(
						symbolID,
						uint64(7),
						matching.OrderSideSell,
						matching.OrderDirectionClose,
						matching.OrderTimeInForceGTC,
						matching.NewUint(32).Mul64(matching.UintPrecision),
						matching.NewUint(1).Mul64(matching.UintPrecision),
						matching.NewMaxUint(),
						matching.NewMaxUint(),
					),
				)
			},
			op:            matching.OrderOpAddPair,
			orderID:       6,
			linkedOrderID: 7,
			cause:         matching.ErrSellOCOStopPriceGreaterThanMarketPrice,
		},
		{
			name: "delete",
			run: func(engine *matching.Engine) error {
				return engine.DeleteOrder(symbolID, 999)
			},
			op:      matching.OrderOpDelete,
			orderID: 999,
			cause:   matching.ErrOrderNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := mockmatching.NewMockHandler(ctrl)
			setupMockHandler(t, handler)

			engine := matching.NewEngine(handler, false)
			engine.EnableMatching()

			setupMarketState(t, engine, symbolID)
			ob := engine.OrderBook(symbolID)

			var handled error
			handler.EXPECT().OnError(ob, gomock.Any()).Do(func(_ *matching.OrderBook, err error) {
				handled = err
			})

			err := tc.run(engine)

			for _, got := range []error{handled, err} {
				var orderErr *matching.OrderError
				require.ErrorAs(t, got, &orderErr)
				require.Equal(t, tc.op, orderErr.Op)
				require.Equal(t, tc.orderID, orderErr.OrderID)
				require.Equal(t, tc.linkedOrderID, orderErr.LinkedOrderID)
				require.ErrorIs(t, got, tc.cause)
			}
		})
	}
}

func TestOrderErrorAsync(t *testing.T) {
	const symbolID uint32 = 10

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler := mockmatching.NewMockHandler(ctrl)
	handler.EXPECT().OnAddOrderBook(gomock.Any()).AnyTimes()

	engine := matching.NewEngine(handler, true)
	defer engine.Stop(false)
	engine.EnableMatching()
	_, err := engine.AddOrderBook(matching.NewSymbol(symbolID, ""), matching.NewUint(0), matching.StopPriceModeConfig{Market: true})
	require.NoError(t, err)
	ob := engine.OrderBook(symbolID)

	handled := make(chan error, 1)
	handler.EXPECT().OnError(ob, gomock.Any()).Do(func(_ *matching.OrderBook, err error) {
		handled <- err
	})

	require.NoError(t, engine.DeleteOrder(symbolID, 999))

	var got error
	select {
	case got = <-handled:
	case <-time.After(5 * time.Second):
		t.Fatal("OnError is not called")
	}

	var orderErr *matching.OrderError
	require.ErrorAs(t, got, &orderErr)
	require.Equal(t, matching.OrderOpDelete, orderErr.Op)
	require.Equal(t, uint64(999), orderErr.OrderID)
	require.ErrorIs(t, got, matching.ErrOrderNotFound)
}
