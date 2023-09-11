package itch

import (
	"errors"
)

func unmarshalSystemEventMessage(data []byte) (msg SystemEventMessage, err error) {
	if len(data) != 12 {
		err = errors.New("invalid size of the ITCH message type 'S' (SystemEventMessage)")
		return
	}
	msg.Type, data = readByte(data)
	msg.StockLocate, data = readUint16(data)
	msg.TrackingNumber, data = readUint16(data)
	msg.Timestamp, data = readTime(data)
	msg.EventCode, _ = readByte(data)
	return
}

func unmarshalStockDirectoryMessage(data []byte) (msg StockDirectoryMessage, err error) {
	if len(data) != 39 {
		err = errors.New("invalid size of the ITCH message type 'R' (StockDirectoryMessage)")
		return
	}
	msg.Type, data = readByte(data)
	msg.StockLocate, data = readUint16(data)
	msg.TrackingNumber, data = readUint16(data)
	msg.Timestamp, data = readTime(data)
	msg.Stock, data = readBytes8(data)
	msg.MarketCategory, data = readByte(data)
	msg.FinancialStatusIndicator, data = readByte(data)
	msg.RoundLotSize, data = readUint32(data)
	msg.RoundLotsOnly, data = readByte(data)
	msg.IssueClassification, data = readByte(data)
	msg.IssueSubType, data = readBytes2(data)
	msg.Authenticity, data = readByte(data)
	msg.ShortSaleThresholdIndicator, data = readByte(data)
	msg.IPOFlag, data = readByte(data)
	msg.LULDReferencePriceTier, data = readByte(data)
	msg.ETPFlag, data = readByte(data)
	msg.ETPLeverageFactor, data = readUint32(data)
	msg.InverseIndicator, _ = readByte(data)
	return
}

func unmarshalStockTradingActionMessage(data []byte) (msg StockTradingActionMessage, err error) {
	if len(data) != 25 {
		err = errors.New("invalid size of the ITCH message type 'H' (StockTradingActionMessage)")
		return
	}
	msg.Type, data = readByte(data)
	msg.StockLocate, data = readUint16(data)
	msg.TrackingNumber, data = readUint16(data)
	msg.Timestamp, data = readTime(data)
	msg.Stock, data = readBytes8(data)
	msg.TradingState, data = readByte(data)
	msg.Reserved, data = readByte(data)
	msg.Reason, _ = readByte(data)
	return
}

func unmarshalRegSHOMessage(data []byte) (msg RegSHOMessage, err error) {
	if len(data) != 20 {
		err = errors.New("invalid size of the ITCH message type 'Y' (RegSHOMessage)")
		return
	}
	msg.Type, data = readByte(data)
	msg.StockLocate, data = readUint16(data)
	msg.TrackingNumber, data = readUint16(data)
	msg.Timestamp, data = readTime(data)
	msg.Stock, data = readBytes8(data)
	msg.RegSHOAction, _ = readByte(data)
	return
}

func unmarshalMarketParticipantPositionMessage(data []byte) (msg MarketParticipantPositionMessage, err error) {
	if len(data) != 26 {
		err = errors.New("invalid size of the ITCH message type 'L' (MarketParticipantPositionMessage)")
		return
	}
	msg.Type, data = readByte(data)
	msg.StockLocate, data = readUint16(data)
	msg.TrackingNumber, data = readUint16(data)
	msg.Timestamp, data = readTime(data)
	msg.MPID, data = readBytes4(data)
	msg.Stock, data = readBytes8(data)
	msg.PrimaryMarketMaker, data = readByte(data)
	msg.MarketMakerMode, data = readByte(data)
	msg.MarketParticipantState, _ = readByte(data)
	return
}

func unmarshalMWCBDeclineMessage(data []byte) (msg MWCBDeclineMessage, err error) {
	if len(data) != 35 {
		err = errors.New("invalid size of the ITCH message type 'V' (MWCBDeclineMessage)")
		return
	}
	msg.Type, data = readByte(data)
	msg.StockLocate, data = readUint16(data)
	msg.TrackingNumber, data = readUint16(data)
	msg.Timestamp, data = readTime(data)
	msg.Level1, data = readUint64(data)
	msg.Level2, data = readUint64(data)
	msg.Level3, _ = readUint64(data)
	return
}

func unmarshalMWCBStatusMessage(data []byte) (msg MWCBStatusMessage, err error) {
	if len(data) != 12 {
		err = errors.New("invalid size of the ITCH message type 'W' (MWCBStatusMessage)")
		return
	}
	msg.Type, data = readByte(data)
	msg.StockLocate, data = readUint16(data)
	msg.TrackingNumber, data = readUint16(data)
	msg.Timestamp, data = readTime(data)
	msg.BreachedLevel, _ = readByte(data)
	return
}

func unmarshalIPOQuotingMessage(data []byte) (msg IPOQuotingMessage, err error) {
	if len(data) != 28 {
		err = errors.New("invalid size of the ITCH message type 'K' (IPOQuotingMessage)")
		return
	}
	msg.Type, data = readByte(data)
	msg.StockLocate, data = readUint16(data)
	msg.TrackingNumber, data = readUint16(data)
	msg.Timestamp, data = readTime(data)
	msg.Stock, data = readBytes8(data)
	msg.IPOReleaseTime, data = readUint32(data)
	msg.IPOReleaseQualifier, data = readByte(data)
	msg.IPOPrice, _ = readUint32(data)
	return
}

func unmarshalAddOrderMessage(data []byte) (msg AddOrderMessage, err error) {
	if len(data) != 36 {
		err = errors.New("invalid size of the ITCH message type 'A' (AddOrderMessage)")
		return
	}
	msg.Type, data = readByte(data)
	msg.StockLocate, data = readUint16(data)
	msg.TrackingNumber, data = readUint16(data)
	msg.Timestamp, data = readTime(data)
	msg.OrderReferenceNumber, data = readUint64(data)
	msg.BuySellIndicator, data = readByte(data)
	msg.Shares, data = readUint32(data)
	msg.Stock, data = readBytes8(data)
	msg.Price, _ = readUint32(data)
	return
}

func unmarshalAddOrderMPIDMessage(data []byte) (msg AddOrderMPIDMessage, err error) {
	if len(data) != 40 {
		err = errors.New("invalid size of the ITCH message type 'F' (AddOrderMPIDMessage)")
		return
	}
	msg.Type, data = readByte(data)
	msg.StockLocate, data = readUint16(data)
	msg.TrackingNumber, data = readUint16(data)
	msg.Timestamp, data = readTime(data)
	msg.OrderReferenceNumber, data = readUint64(data)
	msg.BuySellIndicator, data = readByte(data)
	msg.Shares, data = readUint32(data)
	msg.Stock, data = readBytes8(data)
	msg.Price, data = readUint32(data)
	msg.Attribution, _ = readByte(data)
	return
}

func unmarshalOrderExecutedMessage(data []byte) (msg OrderExecutedMessage, err error) {
	if len(data) != 31 {
		err = errors.New("invalid size of the ITCH message type 'E' (OrderExecutedMessage)")
		return
	}
	msg.Type, data = readByte(data)
	msg.StockLocate, data = readUint16(data)
	msg.TrackingNumber, data = readUint16(data)
	msg.Timestamp, data = readTime(data)
	msg.OrderReferenceNumber, data = readUint64(data)
	msg.ExecutedShares, data = readUint32(data)
	msg.MatchNumber, _ = readUint64(data)
	return
}

func unmarshalOrderExecutedWithPriceMessage(data []byte) (msg OrderExecutedWithPriceMessage, err error) {
	if len(data) != 36 {
		err = errors.New("invalid size of the ITCH message type 'C' (OrderExecutedWithPriceMessage)")
		return
	}
	msg.Type, data = readByte(data)
	msg.StockLocate, data = readUint16(data)
	msg.TrackingNumber, data = readUint16(data)
	msg.Timestamp, data = readTime(data)
	msg.OrderReferenceNumber, data = readUint64(data)
	msg.ExecutedShares, data = readUint32(data)
	msg.MatchNumber, data = readUint64(data)
	msg.Printable, data = readByte(data)
	msg.ExecutionPrice, _ = readUint32(data)
	return
}

func unmarshalOrderCancelMessage(data []byte) (msg OrderCancelMessage, err error) {
	if len(data) != 23 {
		err = errors.New("invalid size of the ITCH message type 'X' (OrderCancelMessage)")
		return
	}
	msg.Type, data = readByte(data)
	msg.StockLocate, data = readUint16(data)
	msg.TrackingNumber, data = readUint16(data)
	msg.Timestamp, data = readTime(data)
	msg.OrderReferenceNumber, data = readUint64(data)
	msg.CanceledShares, _ = readUint32(data)
	return
}

func unmarshalOrderDeleteMessage(data []byte) (msg OrderDeleteMessage, err error) {
	if len(data) != 19 {
		err = errors.New("invalid size of the ITCH message type 'D' (OrderDeleteMessage)")
		return
	}
	msg.Type, data = readByte(data)
	msg.StockLocate, data = readUint16(data)
	msg.TrackingNumber, data = readUint16(data)
	msg.Timestamp, data = readTime(data)
	msg.OrderReferenceNumber, _ = readUint64(data)
	return
}

func unmarshalOrderReplaceMessage(data []byte) (msg OrderReplaceMessage, err error) {
	if len(data) != 35 {
		err = errors.New("invalid size of the ITCH message type 'U' (OrderReplaceMessage)")
		return
	}
	msg.Type, data = readByte(data)
	msg.StockLocate, data = readUint16(data)
	msg.TrackingNumber, data = readUint16(data)
	msg.Timestamp, data = readTime(data)
	msg.OriginalOrderReferenceNumber, data = readUint64(data)
	msg.NewOrderReferenceNumber, data = readUint64(data)
	msg.Shares, data = readUint32(data)
	msg.Price, _ = readUint32(data)
	return
}

func unmarshalTradeMessage(data []byte) (msg TradeMessage, err error) {
	if len(data) != 44 {
		err = errors.New("invalid size of the ITCH message type 'P' (TradeMessage)")
		return
	}
	msg.Type, data = readByte(data)
	msg.StockLocate, data = readUint16(data)
	msg.TrackingNumber, data = readUint16(data)
	msg.Timestamp, data = readTime(data)
	msg.OrderReferenceNumber, data = readUint64(data)
	msg.BuySellIndicator, data = readByte(data)
	msg.Shares, data = readUint32(data)
	msg.Stock, data = readBytes8(data)
	msg.Price, data = readUint32(data)
	msg.MatchNumber, _ = readUint64(data)
	return
}

func unmarshalCrossTradeMessage(data []byte) (msg CrossTradeMessage, err error) {
	if len(data) != 40 {
		err = errors.New("invalid size of the ITCH message type 'Q' (CrossTradeMessage)")
		return
	}
	msg.Type, data = readByte(data)
	msg.StockLocate, data = readUint16(data)
	msg.TrackingNumber, data = readUint16(data)
	msg.Timestamp, data = readTime(data)
	msg.Shares, data = readUint64(data)
	msg.Stock, data = readBytes8(data)
	msg.CrossPrice, data = readUint32(data)
	msg.MatchNumber, data = readUint64(data)
	msg.CrossType, _ = readByte(data)
	return
}

func unmarshalBrokenTradeMessage(data []byte) (msg BrokenTradeMessage, err error) {
	if len(data) != 19 {
		err = errors.New("invalid size of the ITCH message type 'B' (BrokenTradeMessage)")
		return
	}
	msg.Type, data = readByte(data)
	msg.StockLocate, data = readUint16(data)
	msg.TrackingNumber, data = readUint16(data)
	msg.Timestamp, data = readTime(data)
	msg.MatchNumber, _ = readUint64(data)
	return
}

func unmarshalNOIIMessage(data []byte) (msg NOIIMessage, err error) {
	if len(data) != 50 {
		err = errors.New("invalid size of the ITCH message type 'I' (NOIIMessage)")
		return
	}
	msg.Type, data = readByte(data)
	msg.StockLocate, data = readUint16(data)
	msg.TrackingNumber, data = readUint16(data)
	msg.Timestamp, data = readTime(data)
	msg.PairedShares, data = readUint64(data)
	msg.ImbalanceShares, data = readUint64(data)
	msg.ImbalanceDirection, data = readByte(data)
	msg.Stock, data = readBytes8(data)
	msg.FarPrice, data = readUint32(data)
	msg.NearPrice, data = readUint32(data)
	msg.CurrentReferencePrice, data = readUint32(data)
	msg.CrossType, data = readByte(data)
	msg.PriceVariationIndicator, _ = readByte(data)
	return
}

func unmarshalRPIIMessage(data []byte) (msg RPIIMessage, err error) {
	if len(data) != 20 {
		err = errors.New("invalid size of the ITCH message type 'N' (RPIIMessage)")
		return
	}
	msg.Type, data = readByte(data)
	msg.StockLocate, data = readUint16(data)
	msg.TrackingNumber, data = readUint16(data)
	msg.Timestamp, data = readTime(data)
	msg.Stock, data = readBytes8(data)
	msg.InterestFlag, _ = readByte(data)
	return
}

func unmarshalLULDAuctionCollarMessage(data []byte) (msg LULDAuctionCollarMessage, err error) {
	if len(data) != 35 {
		err = errors.New("invalid size of the ITCH message type 'J' (LULDAuctionCollarMessage)")
		return
	}
	msg.Type, data = readByte(data)
	msg.StockLocate, data = readUint16(data)
	msg.TrackingNumber, data = readUint16(data)
	msg.Timestamp, data = readTime(data)
	msg.Stock, data = readBytes8(data)
	msg.AuctionCollarReferencePrice, data = readUint32(data)
	msg.UpperAuctionCollarPrice, data = readUint32(data)
	msg.LowerAuctionCollarPrice, data = readUint32(data)
	msg.AuctionCollarExtension, _ = readUint32(data)
	return
}

func unmarshalUnknownMessage(data []byte) (msg UnknownMessage, err error) {
	if len(data) < 1 {
		err = errors.New("invalid size of the unknown ITCH message")
		return
	}
	msg.Type, _ = readByte(data)
	return
}
