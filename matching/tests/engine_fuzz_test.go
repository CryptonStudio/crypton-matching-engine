package matching_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"testing"
	"unsafe"

	matching "github.com/cryptonstudio/crypton-matching-engine/matching"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func FuzzAllOrders(f *testing.F) {

	f.Add([]byte{})
	f.Add([]byte("0x\x010100000\x01\x01\x01\x01\x01x00\x010000\x0100\x010000\x03\x02\x01\x02\x00000\x030000\x0100\x010000"))

	f.Fuzz(func(t *testing.T, a []byte) {
		testAllOrders(t, a)
	})
}

func TestFailedExample(t *testing.T) {
	testAllOrders(t, []byte("01\x010100000\x01\x01\x01\x01\x01000\x030000\x0100\x010000\xfe\x02\x01\x02\x00000\x030010\x0200\x030000"))
}

func testAllOrders(t *testing.T, a []byte) {
	data, err := parseBytesToData(a)
	if err != nil {
		return
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	stor := newFuzzStorage(t)
	for _, oo := range data.ordersSequence {
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

	_, err = engine.AddOrderBook(data.symbol,
		matching.NewUint(0),
		matching.StopPriceModeConfig{Market: true, Mark: true, Index: true},
	)
	require.NoError(t, err)

	engine.SetIndexPriceForOrderBook(data.symbol.ID(), data.startIndexPrice, false) //nolint:errcheck
	engine.SetMarkPriceForOrderBook(data.symbol.ID(), data.startMarkPrice, false)   //nolint:errcheck

	defer func() {
		// recover from panic if one occurred. Set err to nil otherwise.
		if recover() != nil {
			t.Logf("stacktrace from panic:\n%s\n", string(debug.Stack()))
			t.Log("engine set:\n")
			t.Log(data.String() + "\n")
			t.Fail()
		}
	}()

	t.Log(data.String())

	for _, oo := range data.ordersSequence {
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

	engine.SetIndexMarkPricesForOrderBook(data.symbol.ID(), data.endIndexPrice, data.endMarkPrice, true) //nolint:errcheck

	ob := engine.OrderBook(1)
	for _, oo := range data.ordersSequence {
		for _, o := range oo.orders {
			intOrder := ob.Order(o.ID())
			if intOrder == nil {
				continue
			}
			err := intOrder.CheckLocked(nil)
			if err != nil {
				t.Logf("error: %s", err)
				t.FailNow()
			}
		}
	}
}

// Data parsing:
// 2 bytes for 'float' numbers
// 1 byte for enums

const (
	orderTypeOCO        matching.OrderType = 7
	orderTypeTPSLLimit  matching.OrderType = 8
	orderTypeTPSLMarket matching.OrderType = 9
)

type allDataForFuzz struct {
	symbol          matching.Symbol
	startMarkPrice  matching.Uint
	startIndexPrice matching.Uint
	endMarkPrice    matching.Uint
	endIndexPrice   matching.Uint
	ordersSequence  []sequenceItem
}

type sequenceItem struct {
	orderType matching.OrderType
	orders    []matching.Order
}

func (a allDataForFuzz) validate() error {
	// check TPSL
	if len(a.ordersSequence) > 0 {
		if a.ordersSequence[0].orderType == orderTypeTPSLLimit ||
			a.ordersSequence[0].orderType == orderTypeTPSLMarket {
			return errors.New("TPSL order should be second record")
		}
	}
	// check sides
	// var sellCount, buyCount int
	// for i := range a.ordersSequence {
	// 	switch a.ordersSequence[i].orders[0].Side() {
	// 	case matching.OrderSideBuy:
	// 		buyCount++
	// 	case matching.OrderSideSell:
	// 		sellCount++
	// 	}
	// }
	// if sellCount == 0 || buyCount == 0 {
	// 	return errors.New("all order on one side")
	// }
	return nil
}

func (a allDataForFuzz) String() string {
	lines := []string{}
	lines = append(lines, fmt.Sprintf("symbol: price.min=%s, price.max=%s, price.step=%s, lot.min=%s, lot.max=%s, lot.step=%s",
		a.symbol.PriceLimits().Min.ToFloatString(),
		a.symbol.PriceLimits().Max.ToFloatString(),
		a.symbol.PriceLimits().Step.ToFloatString(),
		a.symbol.LotSizeLimits().Min.ToFloatString(),
		a.symbol.LotSizeLimits().Max.ToFloatString(),
		a.symbol.LotSizeLimits().Step.ToFloatString(),
	))
	lines = append(lines, fmt.Sprintf("prices: start.mark=%s, start.index=%s, end.mark=%s, end.index=%s",
		a.startMarkPrice.ToFloatString(),
		a.startIndexPrice.ToFloatString(),
		a.endMarkPrice.ToFloatString(),
		a.endIndexPrice.ToFloatString(),
	))
	for _, oo := range a.ordersSequence {
		if len(oo.orders) == 2 {
			lines = append(lines, "orders pair:")
		}
		for i := range oo.orders {
			lines = append(lines, fmt.Sprintf("id=%d type=%s side=%s, direction=%s, tif=%s, price=%s, stop price=%s, quantity=%s quoteQuant=%s availableQty=%s restQty=%s",
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
			))
		}
	}

	return strings.Join(lines, "\n")
}

func u8U(v uint8) matching.Uint {
	return matching.NewUint(uint64(v)).Mul64(matching.UintPrecision).Div64(100)
}

func u16U(v uint16) matching.Uint {
	return matching.NewUint(uint64(v)).Mul64(matching.UintPrecision).Div64(10000)
}

type stateConfig struct {
	PriceMin        uint8
	PriceMax        uint8
	PriceStep       uint8
	LotMin          uint8
	LotMax          uint8
	LotStep         uint8
	StartMarkPrice  uint8
	StartIndexPrice uint8
	EndMarkPrice    uint8
	EndIndexPrice   uint8
}

const stateConfigSize = int(unsafe.Sizeof(stateConfig{}))

type orderData struct {
	Type      matching.OrderType
	Side      matching.OrderSide
	Direction matching.OrderDirection
	TIF       matching.OrderTimeInForce
	ModQuote  uint8
	Price     uint8
	Quantity  uint8
	StopPrice uint8
	PriceMode matching.StopPriceMode
	Slippage  uint8
	Visible   uint8
	// TPSL limit part
	TpStopPrice uint8
	TpPrice     uint8
	TpMode      matching.StopPriceMode
	SlStopPrice uint8
	SlPrice     uint8
	SlMode      matching.StopPriceMode
	// TPSL market part
	// stop prices from above, mod quote from above
	TpQuantity uint8
	SlQuantity uint8
	TpSlippage uint8
	SlSlippage uint8
}

const orderDataSize = int(unsafe.Sizeof(orderData{}))

func parseBytesToData(inp []byte) (allDataForFuzz, error) {
	if len(inp) <= stateConfigSize {
		return allDataForFuzz{}, errors.New("invalid input length")
	}
	if (len(inp)-stateConfigSize)%orderDataSize != 0 {
		return allDataForFuzz{}, errors.New("invalid input length")
	}

	// 2 orders at least
	if (len(inp)-stateConfigSize)/orderDataSize < 2 {
		return allDataForFuzz{}, errors.New("need more orders")
	}

	const symbolID = 1

	buf := bytes.NewReader(inp)

	var cfg stateConfig
	err := binary.Read(buf, binary.BigEndian, &cfg)
	if err != nil {
		return allDataForFuzz{}, err
	}

	var result allDataForFuzz

	result.symbol = matching.NewSymbolWithLimits(
		symbolID,
		"a",
		matching.Limits{
			Min:  u8U(cfg.PriceMin),
			Max:  u8U(cfg.PriceMax),
			Step: u8U(cfg.PriceStep),
		},
		matching.Limits{
			Min:  u8U(cfg.LotMin),
			Max:  u8U(cfg.LotMax),
			Step: u8U(cfg.LotStep),
		},
	)
	if !result.symbol.Valid() {
		return allDataForFuzz{}, matching.ErrInvalidSymbol
	}

	result.startIndexPrice = u8U(cfg.StartIndexPrice)
	result.startMarkPrice = u8U(cfg.StartMarkPrice)
	result.endIndexPrice = u8U(cfg.EndIndexPrice)
	result.endMarkPrice = u8U(cfg.EndMarkPrice)

	id := uint64(0)
	for j := stateConfigSize; j < len(inp); j += orderDataSize {
		id++
		var dt orderData
		err := binary.Read(buf, binary.BigEndian, &dt)
		if err != nil {
			return allDataForFuzz{}, err
		}

		if !(dt.Type == matching.OrderTypeLimit || dt.Type == matching.OrderTypeStopLimit ||
			dt.Type == matching.OrderTypeMarket || dt.Type == matching.OrderTypeStop ||
			dt.Type == orderTypeOCO || dt.Type == orderTypeTPSLLimit || dt.Type == orderTypeTPSLMarket) {
			return allDataForFuzz{}, matching.ErrInvalidOrderType
		}

		if !(dt.Side == matching.OrderSideBuy || dt.Side == matching.OrderSideSell) {
			return allDataForFuzz{}, matching.ErrInvalidOrderSide
		}

		if !(dt.Direction == matching.OrderDirectionOpen || dt.Direction == matching.OrderDirectionClose) {
			return allDataForFuzz{}, matching.ErrInvalidOrderDirection
		}

		if !(dt.PriceMode == matching.StopPriceModeMarket || dt.PriceMode == matching.StopPriceModeIndex || dt.PriceMode == matching.StopPriceModeMark) {
			return allDataForFuzz{}, errors.New("invalid price mode")
		}

		if !(dt.TpMode == matching.StopPriceModeMarket || dt.TpMode == matching.StopPriceModeIndex || dt.TpMode == matching.StopPriceModeMark) {
			return allDataForFuzz{}, errors.New("invalid price mode")
		}

		if !(dt.SlMode == matching.StopPriceModeMarket || dt.SlMode == matching.StopPriceModeIndex || dt.SlMode == matching.StopPriceModeMark) {
			return allDataForFuzz{}, errors.New("invalid price mode")
		}

		if !(dt.TIF == matching.OrderTimeInForceGTC || dt.TIF == matching.OrderTimeInForceIOC ||
			dt.TIF == matching.OrderTimeInForceFOK) {
			return allDataForFuzz{}, errors.New("invalid time-in-force")
		}

		// for market orders: 0 - use quantity as quantity, 1 - use as quote quantity
		if !(dt.ModQuote == 0 || dt.ModQuote == 1) {
			return allDataForFuzz{}, errors.New("invalid mod quote")
		}

		price := u8U(dt.Price)
		quantity := u8U(dt.Quantity)
		stopPrice := u8U(dt.StopPrice)
		slippage := u8U(dt.Slippage)
		visible := u8U(dt.Visible)

		if quantity.IsZero() {
			return allDataForFuzz{}, matching.ErrInvalidOrderQuantity
		}
		if (dt.Type == matching.OrderTypeStopLimit || dt.Type == orderTypeOCO) && price.Equals(stopPrice) {
			return allDataForFuzz{}, matching.ErrInvalidOrderStopPrice
		}

		restLocked := quantity
		if dt.Direction == matching.OrderDirectionOpen {
			if dt.Type == matching.OrderTypeLimit || dt.Type == matching.OrderTypeStopLimit {
				restLocked = quantity.Mul(price).Div64(matching.UintPrecision)
			}
			if dt.Type == orderTypeOCO {
				restLocked = matching.Max(quantity.Mul(price).Div64(matching.UintPrecision), quantity.Mul(stopPrice).Div64(matching.UintPrecision))
			}
			if dt.Type == orderTypeTPSLLimit {
				tpPrice := u8U(dt.TpPrice)
				slPrice := u8U(dt.SlPrice)
				restLocked = matching.Max(quantity.Mul(tpPrice).Div64(matching.UintPrecision), quantity.Mul(slPrice).Div64(matching.UintPrecision))
			}
		}

		switch dt.Type {
		case matching.OrderTypeLimit:
			result.ordersSequence = append(result.ordersSequence, sequenceItem{
				dt.Type,
				[]matching.Order{matching.NewLimitOrder(
					symbolID, id, dt.Side, dt.Direction, dt.TIF, price, quantity, visible, restLocked,
				)}})
		case matching.OrderTypeStopLimit:
			result.ordersSequence = append(result.ordersSequence, sequenceItem{
				dt.Type,
				[]matching.Order{matching.NewStopLimitOrder(
					symbolID, id, dt.Side, dt.Direction, dt.TIF, price, dt.PriceMode, stopPrice, quantity, visible, restLocked,
				)}})
		case matching.OrderTypeMarket:
			result.ordersSequence = append(result.ordersSequence, sequenceItem{
				dt.Type,
				[]matching.Order{matching.NewMarketOrder(
					symbolID, id, dt.Side, dt.Direction, matching.OrderTimeInForceIOC, modQQ(dt.ModQuote, 0, quantity),
					modQQ(dt.ModQuote, 1, quantity), slippage, restLocked,
				)}})
		case matching.OrderTypeStop:
			result.ordersSequence = append(result.ordersSequence, sequenceItem{
				dt.Type,
				[]matching.Order{matching.NewStopOrder(symbolID, id, dt.Side, dt.Direction, matching.OrderTimeInForceIOC,
					dt.PriceMode, stopPrice, modQQ(dt.ModQuote, 0, quantity),
					modQQ(dt.ModQuote, 1, quantity), slippage, restLocked,
				)}})
		case orderTypeOCO:
			id1 := id
			id++
			id2 := id
			result.ordersSequence = append(result.ordersSequence, sequenceItem{
				dt.Type,
				[]matching.Order{
					matching.NewStopLimitOrder(symbolID, id1, dt.Side, dt.Direction, dt.TIF, price,
						dt.PriceMode, stopPrice,
						quantity, visible, matching.NewZeroUint(),
					),
					matching.NewLimitOrder(symbolID, id2, dt.Side, dt.Direction, dt.TIF, price,
						quantity, visible, restLocked,
					),
				}})
		case orderTypeTPSLLimit:
			tpPrice := u8U(dt.TpPrice)
			slPrice := u8U(dt.SlPrice)

			id1 := id
			id++
			id2 := id
			result.ordersSequence = append(result.ordersSequence, sequenceItem{
				dt.Type,
				[]matching.Order{
					matching.NewStopLimitOrder(symbolID, id1, dt.Side, dt.Direction, dt.TIF, tpPrice,
						dt.TpMode, u8U(dt.TpStopPrice),
						quantity, visible, restLocked,
					),
					matching.NewStopLimitOrder(symbolID, id2, dt.Side, dt.Direction, dt.TIF, slPrice,
						dt.SlMode, u8U(dt.SlStopPrice),
						quantity, visible, matching.NewZeroUint(),
					),
				}})
		case orderTypeTPSLMarket:
			restLocked = matching.Max(u8U(dt.TpQuantity), u8U(dt.SlQuantity))

			id1 := id
			id++
			id2 := id
			result.ordersSequence = append(result.ordersSequence, sequenceItem{
				dt.Type,
				[]matching.Order{
					matching.NewStopOrder(symbolID, id1, dt.Side, dt.Direction, matching.OrderTimeInForceIOC,
						dt.TpMode, u8U(dt.TpStopPrice),
						modQQ(dt.ModQuote, 0, u8U(dt.TpQuantity)),
						modQQ(dt.ModQuote, 1, u8U(dt.TpQuantity)),
						u8U(dt.TpSlippage), restLocked,
					),
					matching.NewStopOrder(symbolID, id2, dt.Side, dt.Direction, matching.OrderTimeInForceIOC,
						dt.SlMode, u8U(dt.SlStopPrice),
						modQQ(dt.ModQuote, 0, u8U(dt.SlQuantity)),
						modQQ(dt.ModQuote, 1, u8U(dt.SlQuantity)),
						u8U(dt.SlSlippage), matching.NewZeroUint(),
					),
				}})
		}
	}

	if err = result.validate(); err != nil {
		return allDataForFuzz{}, err
	}

	return result, nil
}

func modQQ(mode, req uint8, q matching.Uint) matching.Uint {
	if mode == req {
		return q
	}
	return matching.NewZeroUint()
}

// fuzzStorage implements Handler and it's need for check unlocking after matching.
type fuzzStorage struct {
	sync.Mutex
	orders map[uint64]*orderStorageData
	t      *testing.T
}

type orderStorageData struct {
	id            uint64
	locked        matching.Uint
	direction     matching.OrderDirection
	linkedOrderID uint64
}

func newFuzzStorage(t *testing.T) *fuzzStorage {
	return &fuzzStorage{
		orders: make(map[uint64]*orderStorageData),
		t:      t,
	}
}

func (fs *fuzzStorage) addOrder(order matching.Order, linkedOrderID uint64) {
	fs.orders[order.ID()] = &orderStorageData{
		id:            order.ID(),
		locked:        order.Available(),
		direction:     order.Direction(),
		linkedOrderID: linkedOrderID,
	}
}

func (fs *fuzzStorage) unlockAmount(_ *matching.OrderBook, upd matching.OrderUpdate) {
	fs.Lock()
	defer fs.Unlock()

	if upd.Quantity.IsZero() && upd.QuoteQuantity.IsZero() {
		fs.t.Fatalf("zero execution for order %d", upd.ID)
	}
	data, ok := fs.orders[upd.ID]
	if !ok {
		fs.t.Fatalf("can't found locked for order %d", upd.ID)
	}

	toUnlock := matching.NewZeroUint()

	switch data.direction {
	case matching.OrderDirectionClose:
		toUnlock = upd.Quantity
	case matching.OrderDirectionOpen:
		toUnlock = upd.QuoteQuantity
	}

	if toUnlock.IsZero() {
		fs.t.Fatalf("try to unlock zero amount for order %d", upd.ID)
	}

	// Linked orders.
	if data.locked.IsZero() && data.linkedOrderID != 0 {
		linkedData, ok := fs.orders[data.linkedOrderID]
		if !ok {
			fs.t.Fatalf("can't found locked for linked order %d", upd.ID)
		}

		data.locked = data.locked.Add(linkedData.locked)
		linkedData.locked = matching.NewZeroUint()
		fs.orders[upd.ID] = linkedData
	}

	if toUnlock.GreaterThan(data.locked) {
		fs.t.Fatalf("try to unlock more that locked for order %d: has %s, but try %s",
			upd.ID, data.locked.ToFloatString(), toUnlock.ToFloatString())
	}

	data.locked = data.locked.Sub(toUnlock)
	fs.orders[upd.ID] = data
}

func (fs *fuzzStorage) OnAddOrderBook(orderBook *matching.OrderBook)    {}
func (fs *fuzzStorage) OnUpdateOrderBook(orderBook *matching.OrderBook) {}
func (fs *fuzzStorage) OnDeleteOrderBook(orderBook *matching.OrderBook) {}

func (fs *fuzzStorage) OnAddPriceLevel(orderBook *matching.OrderBook, update matching.PriceLevelUpdate) {
}
func (fs *fuzzStorage) OnUpdatePriceLevel(orderBook *matching.OrderBook, update matching.PriceLevelUpdate) {
}
func (fs *fuzzStorage) OnDeletePriceLevel(orderBook *matching.OrderBook, update matching.PriceLevelUpdate) {
}

func (fs *fuzzStorage) OnAddOrder(orderBook *matching.OrderBook, order *matching.Order)    {}
func (fs *fuzzStorage) OnUpdateOrder(orderBook *matching.OrderBook, order *matching.Order) {}
func (fs *fuzzStorage) OnDeleteOrder(orderBook *matching.OrderBook, order *matching.Order) {}

func (fs *fuzzStorage) OnExecuteOrder(orderBook *matching.OrderBook, orderID uint64, price matching.Uint, quantity matching.Uint, quoteQuantity matching.Uint) {
}
func (fs *fuzzStorage) OnExecuteTrade(orderBook *matching.OrderBook, makerOrderUpdate matching.OrderUpdate,
	takerOrderUpdate matching.OrderUpdate, price matching.Uint, quantity matching.Uint, quoteQuantity matching.Uint) {
	fs.unlockAmount(orderBook, makerOrderUpdate)
	fs.unlockAmount(orderBook, takerOrderUpdate)
}

func (fs *fuzzStorage) OnError(orderBook *matching.OrderBook, err error) {}
