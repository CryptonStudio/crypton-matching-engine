package matching_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"testing"
	"unsafe"

	matching "github.com/cryptonstudio/crypton-matching-engine/matching"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func FuzzChainOrders(f *testing.F) {
	f.Add([]byte{}, []byte{}, []byte{})

	f.Fuzz(func(t *testing.T, state, sellChain, buyChain []byte) {
		testChainOrders(t, state, sellChain, buyChain)
	})
}

func testChainOrders(t *testing.T, stateRaw, sellChainRaw, buyChainRaw []byte) {
	state, err := parseChainState(stateRaw)
	if err != nil {
		return
	}

	sellChain, err := parseOrderSequence(matching.OrderSideSell, 1, sellChainRaw)
	if err != nil {
		return
	}

	buyChain, err := parseOrderSequence(matching.OrderSideBuy, 1000, buyChainRaw)
	if err != nil {
		return
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	orders := mixSequences(state.popOrder, sellChain, buyChain)

	stor := newFuzzStorage(t)
	for _, oo := range orders {
		if len(oo.orders) == 1 {
			stor.addOrder(oo.orders[0], 0)
		}
		if len(oo.orders) == 2 {
			stor.addOrder(oo.orders[0], oo.orders[1].ID())
			stor.addOrder(oo.orders[1], oo.orders[0].ID())
		}
	}

	engine := matching.NewEngine(stor, false)
	engine.EnableMatching()

	_, err = engine.AddOrderBook(state.symbol,
		matching.NewUint(0),
		matching.StopPriceModeConfig{Market: true, Mark: true, Index: true},
	)
	require.NoError(t, err)

	engine.SetIndexPriceForOrderBook(state.symbol.ID(), state.startIndexPrice, false) //nolint:errcheck
	engine.SetMarkPriceForOrderBook(state.symbol.ID(), state.startMarkPrice, false)   //nolint:errcheck

	defer func() {
		// recover from panic if one occurred. Set err to nil otherwise.
		if recover() != nil {
			t.Logf("stacktrace from panic:\n%s\n", string(debug.Stack()))
			t.Log("engine set:\n")
			t.Log(stateAndChainToString(state, orders) + "\n")
			t.Fail()
		}
	}()

	t.Log(stateAndChainToString(state, orders))

	for _, oo := range orders {
		if len(oo.orders) == 1 {
			err = engine.AddOrder(oo.orders[0])
		}
		if len(oo.orders) == 2 {
			switch {
			case oo.orderType == orderTypeOCO:
				err = engine.AddOrdersPair(oo.orders[0], oo.orders[1])
			case oo.orderType == orderTypeTPSLLimit:
				err = engine.AddTPSL(oo.orders[0], oo.orders[1])
			case oo.orderType == orderTypeTPSLMarket:
				err = engine.AddTPSLMarket(oo.orders[0], oo.orders[1])
			}
		}

		if err != nil && !errors.Is(err, matching.ErrInvalidOrderPrice) &&
			!errors.Is(err, matching.ErrNotEnoughLockedAmount) &&
			!errors.Is(err, matching.ErrInvalidOrderQuantity) &&
			!errors.Is(err, matching.ErrInvalidOrderQuoteQuantity) &&
			!errors.Is(err, matching.ErrInvalidMarketSlippage) &&
			!errors.Is(err, matching.ErrInvalidOrderStopPrice) &&
			!errors.Is(err, matching.ErrBuyOCOStopPriceLessThanMarketPrice) &&
			!errors.Is(err, matching.ErrBuyOCOLimitPriceGreaterThanMarketPrice) &&
			!errors.Is(err, matching.ErrSellOCOStopPriceGreaterThanMarketPrice) &&
			!errors.Is(err, matching.ErrSellOCOLimitPriceLessThanMarketPrice) &&
			!errors.Is(err, matching.ErrSellSLStopPriceGreaterThanEnginePrice) &&
			!errors.Is(err, matching.ErrSellTPStopPriceLessThanEnginePrice) &&
			!errors.Is(err, matching.ErrBuySLStopPriceLessThanEnginePrice) &&
			!errors.Is(err, matching.ErrBuyTPStopPriceGreaterThanEnginePrice) {
			t.Logf("error: %s", err)
			t.FailNow()
		}
	}

	engine.SetIndexMarkPricesForOrderBook(state.symbol.ID(), state.endIndexPrice, state.endMarkPrice, true) //nolint:errcheck
}

func stateAndChainToString(state chainStateData, orders []sequenceItem) string {
	lines := []string{}
	lines = append(lines, fmt.Sprintf("symbol: price.min=%s, price.max=%s, price.step=%s, lot.min=%s, lot.max=%s, lot.step=%s",
		state.symbol.PriceLimits().Min.ToFloatString(),
		state.symbol.PriceLimits().Max.ToFloatString(),
		state.symbol.PriceLimits().Step.ToFloatString(),
		state.symbol.LotSizeLimits().Min.ToFloatString(),
		state.symbol.LotSizeLimits().Max.ToFloatString(),
		state.symbol.LotSizeLimits().Step.ToFloatString(),
	))
	lines = append(lines, fmt.Sprintf("prices: start.mark=%s, start.index=%s, end.mark=%s, end.index=%s",
		state.startMarkPrice.ToFloatString(),
		state.startIndexPrice.ToFloatString(),
		state.endMarkPrice.ToFloatString(),
		state.endIndexPrice.ToFloatString(),
	))
	for _, oo := range orders {
		if len(oo.orders) == 2 {
			lines = append(lines, "orders pair:")
		}
		for i := range oo.orders {
			lines = append(lines, fmt.Sprintf("id=%d type=%s side=%s, direction=%s, tif=%s, price=%s, stop price=%s, quantity=%s quoteQuant=%s availableQty=%s restQty=%s marketSlippage=%s",
				oo.orders[i].ID(),
				oo.orders[i].Type().String(),
				oo.orders[i].Side().String(),
				oo.orders[i].Direction().String(),
				oo.orders[i].TimeInForce().String(),
				oo.orders[i].Price().ToFloatString(),
				oo.orders[i].StopPrice().ToFloatString(),
				oo.orders[i].Quantity().ToFloatString(),
				oo.orders[i].QuoteQuantity().ToFloatString(),
				oo.orders[i].Available().ToFloatString(),
				oo.orders[i].RestQuantity().ToFloatString(),
				oo.orders[i].MarketSlippage().ToFloatString(),
			))
		}
	}

	return strings.Join(lines, "\n")
}

const symbolID = 1

type chainStateRaw struct {
	PriceMin        uint16
	PriceMax        uint16
	PriceStep       uint16
	LotMin          uint16
	LotMax          uint16
	LotStep         uint16
	StartMarkPrice  uint16
	StartIndexPrice uint16
	EndMarkPrice    uint16
	EndIndexPrice   uint16
	PopOrder        uint16 // pop orders from chains (1 - sell side, 0 -buy side)
}

const chainStateConfigSize = int(unsafe.Sizeof(chainStateRaw{}))

type chainStateData struct {
	symbol          matching.Symbol
	startMarkPrice  matching.Uint
	startIndexPrice matching.Uint
	endMarkPrice    matching.Uint
	endIndexPrice   matching.Uint
	popOrder        uint16
}

type chainOrderDataRaw struct {
	// Enums:
	// 4 bit for types (0)
	// 1 bit direction (4)
	// 2 bit TIF (5)
	// 1 bit modQuote (7)
	// 2 bit price mode (8)
	// 2 bit tpMode (10)
	// 2 bit slMode (12)
	// == 14 bit
	Enums      uint16
	Price      uint16
	Quantity   uint16
	StopPrice  uint16
	Slippage   uint16
	Visible    uint16
	RestLocked uint16
	// TPSL limit part
	TpStopPrice uint16
	TpPrice     uint16
	SlStopPrice uint16
	SlPrice     uint16
	TpQuantity  uint16
	SlQuantity  uint16
	TpSlippage  uint16
	SlSlippage  uint16
}

const chainOrderDataSize = int(unsafe.Sizeof(chainOrderDataRaw{}))

func (c chainOrderDataRaw) Type() matching.OrderType {
	return matching.OrderType(c.Enums&0b1111 + 1)
}

func (c chainOrderDataRaw) Direction() matching.OrderDirection {
	return matching.OrderDirection((c.Enums>>4)&0b1 + 1)
}

func (c chainOrderDataRaw) TIF() matching.OrderTimeInForce {
	return matching.OrderTimeInForce((c.Enums>>5)&0b11 + 1)
}

func (c chainOrderDataRaw) ModQQ() uint8 {
	return uint8((c.Enums >> 7) & 0b1)
}

func (c chainOrderDataRaw) PriceMode() matching.StopPriceMode {
	return matching.StopPriceMode((c.Enums>>8)&0b11 + 1)
}

func (c chainOrderDataRaw) TpPriceMode() matching.StopPriceMode {
	return matching.StopPriceMode((c.Enums>>10)&0b11 + 1)
}

func (c chainOrderDataRaw) SlPriceMode() matching.StopPriceMode {
	return matching.StopPriceMode((c.Enums>>12)&0b11 + 1)
}

func parseChainState(inp []byte) (chainStateData, error) {
	if len(inp) <= chainStateConfigSize {
		return chainStateData{}, errors.New("invalid input length")
	}

	buf := bytes.NewReader(inp)

	var cfg chainStateRaw
	err := binary.Read(buf, binary.BigEndian, &cfg)
	if err != nil {
		return chainStateData{}, err
	}

	var result chainStateData

	result.symbol = matching.NewSymbolWithLimits(
		symbolID,
		"a",
		matching.Limits{
			Min:  u16U(cfg.PriceMin),
			Max:  u16U(cfg.PriceMax),
			Step: u16U(cfg.PriceStep),
		},
		matching.Limits{
			Min:  u16U(cfg.LotMin),
			Max:  u16U(cfg.LotMax),
			Step: u16U(cfg.LotStep),
		},
	)
	if !result.symbol.Valid() {
		return chainStateData{}, matching.ErrInvalidSymbol
	}

	result.startIndexPrice = u16U(cfg.StartIndexPrice)
	result.startMarkPrice = u16U(cfg.StartMarkPrice)
	result.endIndexPrice = u16U(cfg.EndIndexPrice)
	result.endMarkPrice = u16U(cfg.EndMarkPrice)
	result.popOrder = cfg.PopOrder

	return result, nil
}

func parseOrderSequence(side matching.OrderSide, startID uint64, inp []byte) ([]sequenceItem, error) {
	if len(inp) <= chainOrderDataSize {
		return nil, errors.New("invalid input length")
	}
	if len(inp)%chainOrderDataSize != 0 {
		return nil, errors.New("invalid input length")
	}

	buf := bytes.NewReader(inp)
	var result []sequenceItem

	id := startID
	for j := 0; j < len(inp); j += chainOrderDataSize {
		id++
		var dt chainOrderDataRaw
		err := binary.Read(buf, binary.BigEndian, &dt)
		if err != nil {
			return nil, err
		}

		typ := dt.Type()
		dir := dt.Direction()
		tif := dt.TIF()
		priceMode := dt.PriceMode()
		tpMode := dt.TpPriceMode()
		slMode := dt.SlPriceMode()
		// for market orders: 0 - use quantity as quantity, 1 - use as quote quantity
		modQuote := dt.ModQQ()

		if !(typ == matching.OrderTypeLimit || typ == matching.OrderTypeStopLimit ||
			typ == matching.OrderTypeMarket || typ == matching.OrderTypeStop ||
			typ == orderTypeOCO || typ == orderTypeTPSLLimit || typ == orderTypeTPSLMarket) {
			return nil, matching.ErrInvalidOrderType
		}

		if !(dir == matching.OrderDirectionOpen || dir == matching.OrderDirectionClose) {
			return nil, matching.ErrInvalidOrderDirection
		}

		if !(priceMode == matching.StopPriceModeMarket || priceMode == matching.StopPriceModeIndex || priceMode == matching.StopPriceModeMark) {
			return nil, errors.New("invalid price mode")
		}

		if !(tpMode == matching.StopPriceModeMarket || tpMode == matching.StopPriceModeIndex || tpMode == matching.StopPriceModeMark) {
			return nil, errors.New("invalid price mode")
		}

		if !(slMode == matching.StopPriceModeMarket || slMode == matching.StopPriceModeIndex || slMode == matching.StopPriceModeMark) {
			return nil, errors.New("invalid price mode")
		}

		if !(tif == matching.OrderTimeInForceGTC || tif == matching.OrderTimeInForceIOC ||
			tif == matching.OrderTimeInForceFOK) {
			return nil, errors.New("invalid time-in-force")
		}

		price := u16U(dt.Price)
		quantity := u16U(dt.Quantity)
		stopPrice := u16U(dt.StopPrice)
		slippage := u16U(dt.Slippage)
		visible := u16U(dt.Visible)
		restLocked := u16U(dt.RestLocked)

		if quantity.IsZero() {
			return nil, matching.ErrInvalidOrderQuantity
		}
		if restLocked.IsZero() {
			return nil, matching.ErrNotEnoughLockedAmount
		}
		if (typ == matching.OrderTypeStopLimit || typ == orderTypeOCO) && price.Equals(stopPrice) {
			return nil, matching.ErrInvalidOrderStopPrice
		}

		// NOTE: this code is keep as documentation for restLocked calculation
		/*
			restLocked := quantity
			if dir == matching.OrderDirectionOpen {
				if typ == matching.OrderTypeLimit || typ == matching.OrderTypeStopLimit {
					restLocked = quantity.Mul(price).Div64(matching.UintPrecision)
				}
				if typ == orderTypeOCO {
					restLocked = matching.Max(quantity.Mul(price).Div64(matching.UintPrecision), quantity.Mul(stopPrice).Div64(matching.UintPrecision))
				}
				if typ == orderTypeTPSLLimit {
					tpPrice := u16U(dt.TpPrice)
					slPrice := u16U(dt.SlPrice)
					restLocked = matching.Max(quantity.Mul(tpPrice).Div64(matching.UintPrecision), quantity.Mul(slPrice).Div64(matching.UintPrecision))
				}
			}
		*/

		switch typ {
		case matching.OrderTypeLimit:
			result = append(result, sequenceItem{
				typ,
				[]matching.Order{matching.NewLimitOrder(
					symbolID, id, side, dir, tif, price, quantity, visible, restLocked,
				)}})
		case matching.OrderTypeStopLimit:
			result = append(result, sequenceItem{
				typ,
				[]matching.Order{matching.NewStopLimitOrder(
					symbolID, id, side, dir, tif, price, priceMode, stopPrice, quantity, visible, restLocked,
				)}})
		case matching.OrderTypeMarket:
			result = append(result, sequenceItem{
				typ,
				[]matching.Order{matching.NewMarketOrder(
					symbolID, id, side, dir, matching.OrderTimeInForceIOC, modQQ(modQuote, 0, quantity),
					modQQ(modQuote, 1, quantity), slippage, restLocked,
				)}})
		case matching.OrderTypeStop:
			result = append(result, sequenceItem{
				typ,
				[]matching.Order{matching.NewStopOrder(symbolID, id, side, dir, matching.OrderTimeInForceIOC,
					priceMode, stopPrice, modQQ(modQuote, 0, quantity),
					modQQ(modQuote, 1, quantity), slippage, restLocked,
				)}})
		case orderTypeOCO:
			id1 := id
			id++
			id2 := id
			result = append(result, sequenceItem{
				typ,
				[]matching.Order{
					matching.NewStopLimitOrder(symbolID, id1, side, dir, tif, price,
						priceMode, stopPrice,
						quantity, visible, matching.NewZeroUint(),
					),
					matching.NewLimitOrder(symbolID, id2, side, dir, tif, price,
						quantity, visible, restLocked,
					),
				}})
		case orderTypeTPSLLimit:
			tpPrice := u16U(dt.TpPrice)
			slPrice := u16U(dt.SlPrice)

			id1 := id
			id++
			id2 := id
			result = append(result, sequenceItem{
				typ,
				[]matching.Order{
					matching.NewStopLimitOrder(symbolID, id1, side, dir, tif, tpPrice,
						tpMode, u16U(dt.TpStopPrice),
						quantity, visible, restLocked,
					),
					matching.NewStopLimitOrder(symbolID, id2, side, dir, tif, slPrice,
						slMode, u16U(dt.SlStopPrice),
						quantity, visible, matching.NewZeroUint(),
					),
				}})
		case orderTypeTPSLMarket:
			restLocked = matching.Max(u16U(dt.TpQuantity), u16U(dt.SlQuantity))

			id1 := id
			id++
			id2 := id
			result = append(result, sequenceItem{
				typ,
				[]matching.Order{
					matching.NewStopOrder(symbolID, id1, side, dir, matching.OrderTimeInForceIOC,
						tpMode, u16U(dt.TpStopPrice),
						modQQ(modQuote, 0, u16U(dt.TpQuantity)),
						modQQ(modQuote, 1, u16U(dt.TpQuantity)),
						u16U(dt.TpSlippage), restLocked,
					),
					matching.NewStopOrder(symbolID, id2, side, dir, matching.OrderTimeInForceIOC,
						slMode, u16U(dt.SlStopPrice),
						modQQ(modQuote, 0, u16U(dt.SlQuantity)),
						modQQ(modQuote, 1, u16U(dt.SlQuantity)),
						u16U(dt.SlSlippage), matching.NewZeroUint(),
					),
				}})
		}
	}

	return result, nil
}

func mixSequences(mask uint16, seq1, seq2 []sequenceItem) []sequenceItem {
	bitPos := 0
	i1 := 0
	i2 := 0
	result := make([]sequenceItem, 0, len(seq1)+len(seq2))
	for i1 < len(seq1) || i2 < len(seq2) {
		side := (mask >> bitPos) & 1
		bitPos++
		if bitPos == 15 {
			bitPos = 0
		}
		switch {
		case side == 0 && i1 < len(seq1):
			result = append(result, seq1[i1])
			i1++
		case side == 1 && i2 < len(seq2):
			result = append(result, seq2[i2])
			i2++
		default:
			if i1 < len(seq1) {
				result = append(result, seq1[i1])
				i1++
			}
			if i2 < len(seq2) {
				result = append(result, seq2[i2])
				i2++
			}
		}
	}

	return result
}

func TestMix(t *testing.T) {
	seq1 := []sequenceItem{
		{orderType: matching.OrderTypeLimit},
		{orderType: matching.OrderTypeLimit},
		{orderType: matching.OrderTypeLimit},
	}
	seq2 := []sequenceItem{
		{orderType: matching.OrderTypeLimit},
	}
	masks := []uint16{0, 0xFFFF, 0xAAAA}
	for _, m := range masks {
		res := mixSequences(m, seq1, seq2)
		require.Len(t, res, 4)
		res = mixSequences(m, seq2, seq1)
		require.Len(t, res, 4)
	}
}
