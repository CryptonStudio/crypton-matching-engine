package itch

import (
	"testing"
)

func BenchmarkUnmarshalMessages(b *testing.B) {
	data := [64]byte{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		unmarshalSystemEventMessage(data[:12])
		unmarshalStockDirectoryMessage(data[:39])
		unmarshalStockTradingActionMessage(data[:25])
		unmarshalRegSHOMessage(data[:20])
		unmarshalMarketParticipantPositionMessage(data[:26])
		unmarshalMWCBDeclineMessage(data[:35])
		unmarshalMWCBStatusMessage(data[:12])
		unmarshalIPOQuotingMessage(data[:28])
		unmarshalAddOrderMessage(data[:36])
		unmarshalAddOrderMPIDMessage(data[:40])
		unmarshalOrderExecutedMessage(data[:31])
		unmarshalOrderExecutedWithPriceMessage(data[:36])
		unmarshalOrderCancelMessage(data[:23])
		unmarshalOrderDeleteMessage(data[:19])
		unmarshalOrderReplaceMessage(data[:35])
		unmarshalTradeMessage(data[:44])
		unmarshalCrossTradeMessage(data[:40])
		unmarshalBrokenTradeMessage(data[:19])
		unmarshalNOIIMessage(data[:50])
		unmarshalRPIIMessage(data[:20])
		unmarshalLULDAuctionCollarMessage(data[:35])
		unmarshalUnknownMessage(data[:1])
		unmarshalUnknownMessage(data[:2])
		unmarshalUnknownMessage(data[:3])
		unmarshalUnknownMessage(data[:64])
	}
}
