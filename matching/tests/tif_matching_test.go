package matching_test

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	matching "github.com/cryptonstudio/crypton-matching-engine/matching"
	mockmatching "github.com/cryptonstudio/crypton-matching-engine/matching/mocks"
)

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
			matching.OrderDirectionOpen,
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
			matching.OrderDirectionClose,
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
			matching.OrderDirectionOpen,
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
			matching.OrderDirectionOpen,
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
			matching.OrderDirectionClose,
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
			matching.OrderDirectionOpen,
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
			matching.OrderDirectionClose,
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
			matching.OrderDirectionOpen,
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
			matching.OrderDirectionOpen,
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
			matching.OrderDirectionClose,
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
			matching.OrderDirectionOpen,
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
			matching.OrderDirectionClose,
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
