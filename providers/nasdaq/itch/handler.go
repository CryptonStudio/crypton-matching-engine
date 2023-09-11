package itch

type Handler interface {
	OnSystemEventMessage(msg SystemEventMessage) error
	OnStockDirectoryMessage(msg StockDirectoryMessage) error
	OnStockTradingActionMessage(msg StockTradingActionMessage) error
	OnRegSHOMessage(msg RegSHOMessage) error
	OnMarketParticipantPositionMessage(msg MarketParticipantPositionMessage) error
	OnMWCBDeclineMessage(msg MWCBDeclineMessage) error
	OnMWCBStatusMessage(msg MWCBStatusMessage) error
	OnIPOQuotingMessage(msg IPOQuotingMessage) error
	OnAddOrderMessage(msg AddOrderMessage) error
	OnAddOrderMPIDMessage(msg AddOrderMPIDMessage) error
	OnOrderExecutedMessage(msg OrderExecutedMessage) error
	OnOrderExecutedWithPriceMessage(msg OrderExecutedWithPriceMessage) error
	OnOrderCancelMessage(msg OrderCancelMessage) error
	OnOrderDeleteMessage(msg OrderDeleteMessage) error
	OnOrderReplaceMessage(msg OrderReplaceMessage) error
	OnTradeMessage(msg TradeMessage) error
	OnCrossTradeMessage(msg CrossTradeMessage) error
	OnBrokenTradeMessage(msg BrokenTradeMessage) error
	OnNOIIMessage(msg NOIIMessage) error
	OnRPIIMessage(msg RPIIMessage) error
	OnLULDAuctionCollarMessage(msg LULDAuctionCollarMessage) error
	OnUnknownMessage(msg UnknownMessage) error
}
