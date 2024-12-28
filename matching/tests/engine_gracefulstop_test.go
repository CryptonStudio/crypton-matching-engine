package matching_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	matching "github.com/cryptonstudio/crypton-matching-engine/matching"
	"github.com/stretchr/testify/require"
)

func TestWriteAfterStop(t *testing.T) {
	for range 100 {
		wtch := newWatchHandler()
		engine := matching.NewEngine(wtch, true)

		_, err := engine.AddOrderBook(matching.NewSymbolWithLimits(
			1, "",
			matching.Limits{
				Min:  matching.NewUint(matching.UintPrecision),
				Max:  matching.NewUint(matching.UintPrecision).Mul64(1000),
				Step: matching.NewUint(matching.UintPrecision),
			},
			matching.Limits{
				Min:  matching.NewUint(matching.UintPrecision),
				Max:  matching.NewUint(matching.UintPrecision).Mul64(1000),
				Step: matching.NewUint(matching.UintPrecision),
			},
		),
			matching.NewUint(0),
			matching.StopPriceModeConfig{Market: true, Mark: true, Index: true},
		)
		require.NoError(t, err)

		for i := range 100 {
			engine.AddOrder(matching.NewLimitOrder(1, uint64(i+1), //nolint:errcheck
				matching.OrderSideBuy,
				matching.OrderDirectionOpen,
				matching.OrderTimeInForceGTC,
				matching.NewUint(matching.UintPrecision).Mul64(2),
				matching.NewUint(matching.UintPrecision).Mul64(2),
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			))
			engine.AddOrder(matching.NewLimitOrder(1, uint64(1000*i+1), //nolint:errcheck
				matching.OrderSideSell,
				matching.OrderDirectionOpen,
				matching.OrderTimeInForceGTC,
				matching.NewUint(matching.UintPrecision).Mul64(2),
				matching.NewUint(matching.UintPrecision).Mul64(3),
				matching.NewMaxUint(),
				matching.NewMaxUint(),
			))
		}
		engine.EnableMatching()
		engine.Start()
		engine.Stop(false)
		wtch.start()
		time.Sleep(time.Millisecond * 10)
		require.Equal(t, int64(0), wtch.counter.Load())
	}
}

// fuzzStorage implements Handler and it's need for check unlocking after matching.
type watchHandler struct {
	mx      sync.Mutex
	do      atomic.Bool
	counter atomic.Int64
}

func newWatchHandler() *watchHandler {
	return &watchHandler{}
}

func (wh *watchHandler) start() {
	wh.do.Store(true)
}

func (wh *watchHandler) inc() {
	wh.mx.Lock()
	time.Sleep(time.Microsecond)
	defer wh.mx.Unlock()
	if wh.do.Load() {
		wh.counter.Add(1)
	}
}

func (wh *watchHandler) OnAddOrderBook(orderBook *matching.OrderBook) {
	wh.inc()
}
func (wh *watchHandler) OnUpdateOrderBook(orderBook *matching.OrderBook) {
	wh.inc()
}

func (wh *watchHandler) OnDeleteOrderBook(orderBook *matching.OrderBook) {
	wh.inc()
}

func (wh *watchHandler) OnAddPriceLevel(orderBook *matching.OrderBook, update matching.PriceLevelUpdate) {
	wh.inc()
}
func (wh *watchHandler) OnUpdatePriceLevel(orderBook *matching.OrderBook, update matching.PriceLevelUpdate) {
	wh.inc()
}
func (wh *watchHandler) OnDeletePriceLevel(orderBook *matching.OrderBook, update matching.PriceLevelUpdate) {
	wh.inc()
}

func (wh *watchHandler) OnAddOrder(orderBook *matching.OrderBook, order *matching.Order) {
	wh.inc()
}
func (wh *watchHandler) OnActivateOrder(orderBook *matching.OrderBook, order *matching.Order) {
	wh.inc()
}
func (wh *watchHandler) OnUpdateOrder(orderBook *matching.OrderBook, order *matching.Order) {
	wh.inc()
}
func (wh *watchHandler) OnDeleteOrder(orderBook *matching.OrderBook, order *matching.Order) {
	wh.inc()
}

func (wh *watchHandler) OnExecuteOrder(orderBook *matching.OrderBook, orderID uint64, price matching.Uint, quantity matching.Uint, quoteQuantity matching.Uint) {
	wh.inc()
}
func (wh *watchHandler) OnExecuteTrade(orderBook *matching.OrderBook, makerOrderUpdate matching.OrderUpdate,
	takerOrderUpdate matching.OrderUpdate, price matching.Uint, quantity matching.Uint, quoteQuantity matching.Uint) {
	wh.inc()
}

func (wh *watchHandler) OnError(orderBook *matching.OrderBook, err error) {
	wh.inc()
}
