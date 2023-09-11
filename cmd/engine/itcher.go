package main

import (
	"fmt"
	"time"

	"github.com/cryptonstudio/crypton-matching-engine/matching"
	"github.com/cryptonstudio/crypton-matching-engine/providers/nasdaq/itch"
)

type ITCH struct {
	messages [256]int
	handled  int
	errors   int
	engine   *matching.Engine
}

func (h *ITCH) OnSystemEventMessage(msg itch.SystemEventMessage) error {
	h.messages[msg.Type]++
	return nil
}

func (h *ITCH) OnStockDirectoryMessage(msg itch.StockDirectoryMessage) error {
	h.messages[msg.Type]++
	h.handled++
	symbol := matching.NewSymbol(uint32(msg.StockLocate), string(msg.Stock[:]))
	_, err := h.engine.AddOrderBook(symbol)
	if err != nil {
		h.errors++
		// fmt.Printf("ERROR: Unable to add order book %s (%d)\n", string(msg.Stock[:]), msg.StockLocate)
		return err
	}
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
	h.handled++
	var side matching.OrderSide
	if msg.BuySellIndicator == 'B' {
		side = matching.OrderSideBuy
	} else {
		side = matching.OrderSideSell
	}
	// TODO: Implement matching.New*Order functions instead!
	order := matching.NewLimitOrder(
		uint32(msg.StockLocate),
		msg.OrderReferenceNumber,
		side,
		matching.NewUint(uint64(msg.Price)),
		matching.NewUint(uint64(msg.Shares)),
		matching.NewUint(uint64(msg.Shares)),
		matching.NewUint(uint64(msg.Shares)), // TODO
	)
	err := h.engine.AddOrder(order)
	if err != nil {
		h.errors++
		// fmt.Printf("ERROR: Unable to add order %s (%d)\n", string(msg.Stock[:]), msg.OrderReferenceNumber)
		return err
	}
	return nil
}

func (h *ITCH) OnAddOrderMPIDMessage(msg itch.AddOrderMPIDMessage) error {
	h.messages[msg.Type]++
	h.handled++
	var side matching.OrderSide
	if msg.BuySellIndicator == 'B' {
		side = matching.OrderSideBuy
	} else {
		side = matching.OrderSideSell
	}
	// TODO: Implement matching.New*Order functions instead!
	order := matching.NewLimitOrder(
		uint32(msg.StockLocate),
		msg.OrderReferenceNumber,
		side,
		matching.NewUint(uint64(msg.Price)),
		matching.NewUint(uint64(msg.Shares)),
		matching.NewUint(uint64(msg.Shares)),
		matching.NewUint(uint64(msg.Shares)), // TODO
	)
	err := h.engine.AddOrder(order)
	if err != nil {
		h.errors++
		// fmt.Printf("ERROR: Unable to add order %s (%d)\n", string(msg.Stock[:]), msg.OrderReferenceNumber)
		return err
	}
	return nil
}

func (h *ITCH) OnOrderExecutedMessage(msg itch.OrderExecutedMessage) error {
	h.messages[msg.Type]++
	if !autoMatching {
		h.handled++
		err := h.engine.ExecuteOrder(uint32(msg.StockLocate), msg.OrderReferenceNumber, matching.NewUint(uint64(msg.ExecutedShares)))
		if err != nil {
			h.errors++
			fmt.Printf("ERROR: Unable to execute order (%d)\n", msg.OrderReferenceNumber)
			return err
		}
	}
	return nil
}

func (h *ITCH) OnOrderExecutedWithPriceMessage(msg itch.OrderExecutedWithPriceMessage) error {
	h.messages[msg.Type]++
	if !autoMatching {
		h.handled++
		err := h.engine.ExecuteOrderByPrice(
			uint32(msg.StockLocate),
			msg.OrderReferenceNumber,
			matching.NewUint(uint64(msg.ExecutionPrice)),
			matching.NewUint(uint64(msg.ExecutedShares)),
		)
		if err != nil {
			h.errors++
			fmt.Printf("ERROR: Unable to execute order with price (%d)\n", msg.OrderReferenceNumber)
			return err
		}
	}
	return nil
}

func (h *ITCH) OnOrderCancelMessage(msg itch.OrderCancelMessage) error {
	h.messages[msg.Type]++
	h.handled++
	err := h.engine.ReduceOrder(uint32(msg.StockLocate), msg.OrderReferenceNumber, matching.NewUint(uint64(msg.CanceledShares)))
	if err != nil {
		h.errors++
		// fmt.Printf("ERROR: Unable to cancel order (%d)\n", msg.OrderReferenceNumber)
		return err
	}
	return nil
}

func (h *ITCH) OnOrderDeleteMessage(msg itch.OrderDeleteMessage) error {
	h.messages[msg.Type]++
	h.handled++
	err := h.engine.DeleteOrder(uint32(msg.StockLocate), msg.OrderReferenceNumber)
	if err != nil {
		h.errors++
		// fmt.Printf("ERROR: Unable to delete order (%d)\n", msg.OrderReferenceNumber)
		return err
	}
	return nil
}

func (h *ITCH) OnOrderReplaceMessage(msg itch.OrderReplaceMessage) error {
	h.messages[msg.Type]++
	h.handled++
	err := h.engine.ReplaceOrder(
		uint32(msg.StockLocate),
		msg.OriginalOrderReferenceNumber,
		msg.NewOrderReferenceNumber,
		matching.NewUint(uint64(msg.Price)),
		matching.NewUint(uint64(msg.Shares)),
	)
	if err != nil {
		h.errors++
		// fmt.Printf("ERROR: Unable to replace order (%d)\n", msg.OriginalOrderReferenceNumber)
		return err
	}
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

func (h *ITCH) PrintStatistics(elapsed time.Duration) {
	fmt.Printf("ITCH PROCESSOR HANDLER:\n")
	msgCountTotal := 0
	for i := 0; i < 256; i++ {
		msgCount := h.messages[i]
		msgCountTotal += msgCount
		// if msgCount > 0 {
		// 	fmt.Printf("Message %c: %d\n", byte(i), msgCount)
		// }
	}
	fmt.Printf("Errors %22d\n", h.errors)
	fmt.Printf("Handled messages %12d\n", h.handled)
	fmt.Printf("Total message %15d\n", msgCountTotal)
}
