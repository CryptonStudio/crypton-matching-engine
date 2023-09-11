package itch

import (
	"time"
)

type SystemEventMessage struct {
	Type           byte
	StockLocate    uint16
	TrackingNumber uint16
	Timestamp      time.Time
	EventCode      byte
}

type StockDirectoryMessage struct {
	Type                        byte
	StockLocate                 uint16
	TrackingNumber              uint16
	Timestamp                   time.Time
	Stock                       [8]byte
	MarketCategory              byte
	FinancialStatusIndicator    byte
	RoundLotSize                uint32
	RoundLotsOnly               byte
	IssueClassification         byte
	IssueSubType                [2]byte
	Authenticity                byte
	ShortSaleThresholdIndicator byte
	IPOFlag                     byte
	LULDReferencePriceTier      byte
	ETPFlag                     byte
	ETPLeverageFactor           uint32
	InverseIndicator            byte
}

type StockTradingActionMessage struct {
	Type           byte
	StockLocate    uint16
	TrackingNumber uint16
	Timestamp      time.Time
	Stock          [8]byte
	TradingState   byte
	Reserved       byte
	Reason         byte
}

type RegSHOMessage struct {
	Type           byte
	StockLocate    uint16
	TrackingNumber uint16
	Timestamp      time.Time
	Stock          [8]byte
	RegSHOAction   byte
}

type MarketParticipantPositionMessage struct {
	Type                   byte
	StockLocate            uint16
	TrackingNumber         uint16
	Timestamp              time.Time
	MPID                   [4]byte
	Stock                  [8]byte
	PrimaryMarketMaker     byte
	MarketMakerMode        byte
	MarketParticipantState byte
}

type MWCBDeclineMessage struct {
	Type           byte
	StockLocate    uint16
	TrackingNumber uint16
	Timestamp      time.Time
	Level1         uint64
	Level2         uint64
	Level3         uint64
}

type MWCBStatusMessage struct {
	Type           byte
	StockLocate    uint16
	TrackingNumber uint16
	Timestamp      time.Time
	BreachedLevel  byte
}

type IPOQuotingMessage struct {
	Type                byte
	StockLocate         uint16
	TrackingNumber      uint16
	Timestamp           time.Time
	Stock               [8]byte
	IPOReleaseTime      uint32
	IPOReleaseQualifier byte
	IPOPrice            uint32
}

type AddOrderMessage struct {
	Type                 byte
	StockLocate          uint16
	TrackingNumber       uint16
	Timestamp            time.Time
	OrderReferenceNumber uint64
	BuySellIndicator     byte
	Shares               uint32
	Stock                [8]byte
	Price                uint32
}

type AddOrderMPIDMessage struct {
	Type                 byte
	StockLocate          uint16
	TrackingNumber       uint16
	Timestamp            time.Time
	OrderReferenceNumber uint64
	BuySellIndicator     byte
	Shares               uint32
	Stock                [8]byte
	Price                uint32
	Attribution          byte
}

type OrderExecutedMessage struct {
	Type                 byte
	StockLocate          uint16
	TrackingNumber       uint16
	Timestamp            time.Time
	OrderReferenceNumber uint64
	ExecutedShares       uint32
	MatchNumber          uint64
}

type OrderExecutedWithPriceMessage struct {
	Type                 byte
	StockLocate          uint16
	TrackingNumber       uint16
	Timestamp            time.Time
	OrderReferenceNumber uint64
	ExecutedShares       uint32
	MatchNumber          uint64
	Printable            byte
	ExecutionPrice       uint32
}

type OrderCancelMessage struct {
	Type                 byte
	StockLocate          uint16
	TrackingNumber       uint16
	Timestamp            time.Time
	OrderReferenceNumber uint64
	CanceledShares       uint32
}

type OrderDeleteMessage struct {
	Type                 byte
	StockLocate          uint16
	TrackingNumber       uint16
	Timestamp            time.Time
	OrderReferenceNumber uint64
}

type OrderReplaceMessage struct {
	Type                         byte
	StockLocate                  uint16
	TrackingNumber               uint16
	Timestamp                    time.Time
	OriginalOrderReferenceNumber uint64
	NewOrderReferenceNumber      uint64
	Shares                       uint32
	Price                        uint32
}

type TradeMessage struct {
	Type                 byte
	StockLocate          uint16
	TrackingNumber       uint16
	Timestamp            time.Time
	OrderReferenceNumber uint64
	BuySellIndicator     byte
	Shares               uint32
	Stock                [8]byte
	Price                uint32
	MatchNumber          uint64
}

type CrossTradeMessage struct {
	Type           byte
	StockLocate    uint16
	TrackingNumber uint16
	Timestamp      time.Time
	Shares         uint64
	Stock          [8]byte
	CrossPrice     uint32
	MatchNumber    uint64
	CrossType      byte
}

type BrokenTradeMessage struct {
	Type           byte
	StockLocate    uint16
	TrackingNumber uint16
	Timestamp      time.Time
	MatchNumber    uint64
}

type NOIIMessage struct {
	Type                    byte
	StockLocate             uint16
	TrackingNumber          uint16
	Timestamp               time.Time
	PairedShares            uint64
	ImbalanceShares         uint64
	ImbalanceDirection      byte
	Stock                   [8]byte
	FarPrice                uint32
	NearPrice               uint32
	CurrentReferencePrice   uint32
	CrossType               byte
	PriceVariationIndicator byte
}

type RPIIMessage struct {
	Type           byte
	StockLocate    uint16
	TrackingNumber uint16
	Timestamp      time.Time
	Stock          [8]byte
	InterestFlag   byte
}

type LULDAuctionCollarMessage struct {
	Type                        byte
	StockLocate                 uint16
	TrackingNumber              uint16
	Timestamp                   time.Time
	Stock                       [8]byte
	AuctionCollarReferencePrice uint32
	UpperAuctionCollarPrice     uint32
	LowerAuctionCollarPrice     uint32
	AuctionCollarExtension      uint32
}

type UnknownMessage struct {
	Type byte
}
