package main

import (
	"flag"
	"fmt"
	"math"
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/cryptonstudio/crypton-matching-engine/matching"
)

// nolint
func main() {
	var symCount, ordersCount int
	var norm, heavy bool
	flag.IntVar(&symCount, "s", 3, "Symbols count")
	flag.IntVar(&ordersCount, "i", 5_000_000, "Input orders count")
	flag.BoolVar(&norm, "n", false, "Use normal distribution for price and quantity")
	flag.BoolVar(&heavy, "heavy", false, "Generate heavy sides for orderbook")
	flag.Parse()

	symbols := []uint32{}
	handler := &Matcher{}
	engine := matching.NewEngine(handler, true)
	for i := range symCount {
		sym := uint32(i + 1)
		symbols = append(symbols, sym)
		engine.AddOrderBook(
			matching.NewSymbolWithLimits(sym, strconv.FormatUint(uint64(sym), 10),
				matching.Limits{
					Min:  matching.NewUint(1).Mul64(matching.UintPrecision),
					Max:  matching.NewUint(100).Mul64(matching.UintPrecision),
					Step: matching.NewUint(1).Mul64(matching.UintPrecision).Div64(100),
				},
				matching.Limits{
					Min:  matching.NewUint(1).Mul64(matching.UintPrecision),
					Max:  matching.NewUint(100).Mul64(matching.UintPrecision),
					Step: matching.NewUint(1).Mul64(matching.UintPrecision).Div64(100),
				},
			),
			matching.NewZeroUint(),
			matching.StopPriceModeConfig{
				Market: true,
			},
		)
	}

	fmt.Println("prepare input")

	inp := generateInput(ordersCount, norm, heavy, symbols)

	fmt.Println("start execution")
	engine.EnableMatching()
	engine.Start()

	s := time.Now()
	for o := range inp {
		engine.AddOrder(o)
	}
	engine.Stop(false)
	e := time.Now()

	handler.PrintStatistics()

	rps := float64(ordersCount) * float64(time.Second) / float64(e.Sub(s))

	fmt.Printf("RPS: %.5f\n", rps)
}

func randomFloat(down, up float64, prec int, norm bool) float64 {
	var raw float64
	switch norm {
	case false:
		raw = rand.Float64()*(up-down) + down
	case true:
		std := (up - down) / (2.0 * 5) // range = [-5*std; +5*std]
		mean := (up + down) / 2.0
		raw = rand.NormFloat64()*std + mean
		// cut edges
		if raw < down {
			raw = down
		}
		if raw > up {
			raw = up
		}
	}
	pow := math.Pow10(prec)
	return math.Round(raw*pow) / pow
}

func randomChoice[T any](list []T) T {
	var empty T
	if len(list) == 0 {
		return empty
	}

	return list[rand.IntN(len(list))]
}

func generateInput(ordersCount int, norm, heavy bool, symbols []uint32) chan matching.Order {
	inp := make(chan matching.Order, ordersCount)
	for i := range ordersCount {
		var price matching.Uint
		var side matching.OrderSide
		switch heavy {
		case false:
			price, _ = matching.NewUintFromFloatString(strconv.FormatFloat(randomFloat(1, 100, 2, norm), 'f', 2, 64))
			side = randomChoice([]matching.OrderSide{matching.OrderSideBuy, matching.OrderSideSell})
		case true:
			if i < ordersCount/2 {
				side = randomChoice([]matching.OrderSide{matching.OrderSideBuy, matching.OrderSideSell})
				switch side {
				case matching.OrderSideBuy:
					price, _ = matching.NewUintFromFloatString(strconv.FormatFloat(randomFloat(1, 50, 2, norm), 'f', 2, 64))
				case matching.OrderSideSell:
					price, _ = matching.NewUintFromFloatString(strconv.FormatFloat(randomFloat(51, 100, 2, norm), 'f', 2, 64))
				}
			} else {
				price, _ = matching.NewUintFromFloatString(strconv.FormatFloat(randomFloat(1, 100, 2, norm), 'f', 2, 64))
				side = randomChoice([]matching.OrderSide{matching.OrderSideBuy, matching.OrderSideSell})
			}
		}
		stopPrice, _ := matching.NewUintFromFloatString(strconv.FormatFloat(randomFloat(1, 100, 2, norm), 'f', 2, 64))
		quant, _ := matching.NewUintFromFloatString(strconv.FormatFloat(randomFloat(1, 100, 2, norm), 'f', 2, 64))
		slippage, _ := matching.NewUintFromFloatString(strconv.FormatFloat(randomFloat(1, 5, 2, false), 'f', 2, 64))
		var o matching.Order

		switch randomChoice([]matching.OrderType{matching.OrderTypeLimit, matching.OrderTypeMarket}) {
		case matching.OrderTypeLimit:
			o = matching.NewLimitOrder(
				randomChoice(symbols),
				uint64(i+1),
				side,
				matching.OrderDirectionClose,
				matching.OrderTimeInForceGTC,
				price,
				quant,
				matching.NewMaxUint(),
				quant,
			)
		case matching.OrderTypeMarket:
			o = matching.NewMarketOrder(
				randomChoice(symbols),
				uint64(i+1),
				side,
				matching.OrderDirectionClose,
				matching.OrderTimeInForceIOC,
				quant,
				matching.NewZeroUint(),
				slippage,
				quant,
			)
		case matching.OrderTypeStopLimit:
			o = matching.NewStopLimitOrder(
				randomChoice(symbols),
				uint64(i+1),
				side,
				matching.OrderDirectionClose,
				matching.OrderTimeInForceGTC,
				price,
				matching.StopPriceModeMarket,
				stopPrice,
				quant,
				matching.NewMaxUint(),
				quant,
			)
		}

		inp <- o
	}

	close(inp)

	return inp
}
