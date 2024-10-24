package matching

//go:generate mockgen -destination=mocks/interfaces.go -package=mockmatching . Handler
type Handler interface {

	// Order book handlers
	OnAddOrderBook(orderBook *OrderBook)
	OnUpdateOrderBook(orderBook *OrderBook)
	OnDeleteOrderBook(orderBook *OrderBook)

	// Price level handlers
	OnAddPriceLevel(orderBook *OrderBook, update PriceLevelUpdate)
	OnUpdatePriceLevel(orderBook *OrderBook, update PriceLevelUpdate)
	OnDeletePriceLevel(orderBook *OrderBook, update PriceLevelUpdate)

	// Orders handlers
	OnAddOrder(orderBook *OrderBook, order *Order)
	OnActivateOrder(orderBook *OrderBook, order *Order)
	OnUpdateOrder(orderBook *OrderBook, order *Order)
	OnDeleteOrder(orderBook *OrderBook, order *Order)

	// Matching handlers
	OnExecuteOrder(orderBook *OrderBook, orderID uint64, price Uint, quantity Uint, quoteQuantity Uint)
	OnExecuteTrade(orderBook *OrderBook, makerOrderUpdate OrderUpdate, takerOrderUpdate OrderUpdate, price Uint, quantity Uint, quoteQuantity Uint)

	// Errors handler
	OnError(orderBook *OrderBook, err error)
}
