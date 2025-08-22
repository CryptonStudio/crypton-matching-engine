package main

import (
	"fmt"
	"sync/atomic"

	"github.com/cryptonstudio/crypton-matching-engine/matching"
)

type Matcher struct {
	orderBookUpdates  [3]uint64
	priceLevelUpdates [3]uint64
	orderUpdates      [3]uint64
	executeUpdates    [2]uint64
	errors            uint64
	totalUpdates      uint64
}

func (m *Matcher) OnActivateOrder(orderBook *matching.OrderBook, order *matching.Order) {}

func (m *Matcher) OnAddOrderBook(orderBook *matching.OrderBook) {
	atomic.AddUint64(&m.orderBookUpdates[0], 1)
	atomic.AddUint64(&m.totalUpdates, 1)
}

func (m *Matcher) OnUpdateOrderBook(orderBook *matching.OrderBook) {
	atomic.AddUint64(&m.orderBookUpdates[1], 1)
	atomic.AddUint64(&m.totalUpdates, 1)
}

func (m *Matcher) OnDeleteOrderBook(orderBook *matching.OrderBook) {
	atomic.AddUint64(&m.orderBookUpdates[2], 1)
	atomic.AddUint64(&m.totalUpdates, 1)
}

func (m *Matcher) OnAddPriceLevel(orderBook *matching.OrderBook, update matching.PriceLevelUpdate) {
	atomic.AddUint64(&m.priceLevelUpdates[0], 1)
	atomic.AddUint64(&m.totalUpdates, 1)
}

func (m *Matcher) OnUpdatePriceLevel(orderBook *matching.OrderBook, update matching.PriceLevelUpdate) {
	atomic.AddUint64(&m.priceLevelUpdates[1], 1)
	atomic.AddUint64(&m.totalUpdates, 1)
}

func (m *Matcher) OnDeletePriceLevel(orderBook *matching.OrderBook, update matching.PriceLevelUpdate) {
	atomic.AddUint64(&m.priceLevelUpdates[2], 1)
	atomic.AddUint64(&m.totalUpdates, 1)
}

func (m *Matcher) OnAddOrder(orderBook *matching.OrderBook, order *matching.Order) {
	atomic.AddUint64(&m.orderUpdates[0], 1)
	atomic.AddUint64(&m.totalUpdates, 1)
}

func (m *Matcher) OnUpdateOrder(orderBook *matching.OrderBook, order *matching.Order) {
	atomic.AddUint64(&m.orderUpdates[1], 1)
	atomic.AddUint64(&m.totalUpdates, 1)
}

func (m *Matcher) OnDeleteOrder(orderBook *matching.OrderBook, order *matching.Order) {
	atomic.AddUint64(&m.orderUpdates[2], 1)
	atomic.AddUint64(&m.totalUpdates, 1)
}

func (m *Matcher) OnExecuteOrder(orderBook *matching.OrderBook, orderID uint64, price matching.Uint, quantity matching.Uint, quoteQuantity matching.Uint) {
	atomic.AddUint64(&m.executeUpdates[0], 1)
	atomic.AddUint64(&m.totalUpdates, 1)
}

func (m *Matcher) OnExecuteTrade(orderBook *matching.OrderBook, makerOrderID matching.OrderUpdate, takerOrderID matching.OrderUpdate, price matching.Uint, quantity matching.Uint, quoteQuantity matching.Uint) {
	atomic.AddUint64(&m.executeUpdates[1], 1)
	atomic.AddUint64(&m.totalUpdates, 1)
}

func (m *Matcher) OnError(orderBook *matching.OrderBook, err error) {
	atomic.AddUint64(&m.errors, 1)
}

func (m *Matcher) PrintStatistics() {
	fmt.Printf("MATCHING ENGINE HANDLER:\n")
	fmt.Printf("Order book adds %13d\n", m.orderBookUpdates[0])
	fmt.Printf("Order book updates %10d\n", m.orderBookUpdates[1])
	fmt.Printf("Order book deletes %10d\n", m.orderBookUpdates[2])
	fmt.Printf("Price level adds %12d\n", m.priceLevelUpdates[0])
	fmt.Printf("Price level updates %9d\n", m.priceLevelUpdates[1])
	fmt.Printf("Price level deletes %9d\n", m.priceLevelUpdates[2])
	fmt.Printf("Order adds %18d\n", m.orderUpdates[0])
	fmt.Printf("Order updates %15d\n", m.orderUpdates[1])
	fmt.Printf("Order deletes %15d\n", m.orderUpdates[2])
	fmt.Printf("Executed orders %13d\n", m.executeUpdates[0])
	fmt.Printf("Executed trades %13d\n", m.executeUpdates[1])
	fmt.Printf("Errors %22d\n", m.errors)
	fmt.Printf("Total calls %17d\n", m.totalUpdates)
}
