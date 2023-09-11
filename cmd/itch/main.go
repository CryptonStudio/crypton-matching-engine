package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cryptonstudio/crypton-matching/providers/nasdaq/itch"
)

const filePath = "./.stash/itch/01302019.NASDAQ_ITCH50"

var _ itch.Handler = &ITCH{}

func main() {

	// Create ITCH data processor
	itchHandler := &ITCH{}
	processor, err := itch.NewProcessor(itchHandler)
	if err != nil {
		log.Fatal(err)
	}

	// Run reading ITCH data from file
	timeStart := time.Now()
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	err = processor.Process(file)
	if err != nil {
		log.Fatal(err)
	}
	timeElapsed := time.Since(timeStart)
	msgCountTotal := 0
	for i := 0; i < 256; i++ {
		msgCount := itchHandler.messages[i]
		msgCountTotal += msgCount
		if msgCount > 0 {
			fmt.Printf("Message %c: %d\n", byte(i), msgCount)
		}
	}
	fmt.Printf("Total message count: %d\n", msgCountTotal)
	fmt.Printf("Processed file. Time elapsed: %f s.\n", timeElapsed.Seconds())
}

////////////////////////////////////////////////////////////////

type ITCH struct {
	messages [256]int
}

func (h *ITCH) OnSystemEventMessage(msg itch.SystemEventMessage) error {
	h.messages[msg.Type]++
	return nil
}

func (h *ITCH) OnStockDirectoryMessage(msg itch.StockDirectoryMessage) error {
	h.messages[msg.Type]++
	return nil
}

func (h *ITCH) OnStockTradingActionMessage(msg itch.StockTradingActionMessage) error {
	h.messages[msg.Type]++
	return nil
}

func (h *ITCH) OnRegSHOMessage(msg itch.RegSHOMessage) error {
	h.messages[msg.Type]++
	return nil
}

func (h *ITCH) OnMarketParticipantPositionMessage(msg itch.MarketParticipantPositionMessage) error {
	h.messages[msg.Type]++
	return nil
}

func (h *ITCH) OnMWCBDeclineMessage(msg itch.MWCBDeclineMessage) error {
	h.messages[msg.Type]++
	return nil
}

func (h *ITCH) OnMWCBStatusMessage(msg itch.MWCBStatusMessage) error {
	h.messages[msg.Type]++
	return nil
}

func (h *ITCH) OnIPOQuotingMessage(msg itch.IPOQuotingMessage) error {
	h.messages[msg.Type]++
	return nil
}

func (h *ITCH) OnAddOrderMessage(msg itch.AddOrderMessage) error {
	h.messages[msg.Type]++
	return nil
}

func (h *ITCH) OnAddOrderMPIDMessage(msg itch.AddOrderMPIDMessage) error {
	h.messages[msg.Type]++
	return nil
}

func (h *ITCH) OnOrderExecutedMessage(msg itch.OrderExecutedMessage) error {
	h.messages[msg.Type]++
	return nil
}

func (h *ITCH) OnOrderExecutedWithPriceMessage(msg itch.OrderExecutedWithPriceMessage) error {
	h.messages[msg.Type]++
	return nil
}

func (h *ITCH) OnOrderCancelMessage(msg itch.OrderCancelMessage) error {
	h.messages[msg.Type]++
	return nil
}

func (h *ITCH) OnOrderDeleteMessage(msg itch.OrderDeleteMessage) error {
	h.messages[msg.Type]++
	return nil
}

func (h *ITCH) OnOrderReplaceMessage(msg itch.OrderReplaceMessage) error {
	h.messages[msg.Type]++
	return nil
}

func (h *ITCH) OnTradeMessage(msg itch.TradeMessage) error {
	h.messages[msg.Type]++
	return nil
}

func (h *ITCH) OnCrossTradeMessage(msg itch.CrossTradeMessage) error {
	h.messages[msg.Type]++
	return nil
}

func (h *ITCH) OnBrokenTradeMessage(msg itch.BrokenTradeMessage) error {
	h.messages[msg.Type]++
	return nil
}

func (h *ITCH) OnNOIIMessage(msg itch.NOIIMessage) error {
	h.messages[msg.Type]++
	return nil
}

func (h *ITCH) OnRPIIMessage(msg itch.RPIIMessage) error {
	h.messages[msg.Type]++
	return nil
}

func (h *ITCH) OnLULDAuctionCollarMessage(msg itch.LULDAuctionCollarMessage) error {
	h.messages[msg.Type]++
	return nil
}

func (h *ITCH) OnUnknownMessage(msg itch.UnknownMessage) error {
	h.messages[msg.Type]++
	return nil
}
