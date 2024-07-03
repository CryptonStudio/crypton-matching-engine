package matching

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCalcExecutingQuantities(t *testing.T) {
	// define steps
	lotStep, quoteLotStep := shifted("0.001"), shifted("0.00001")

	tests := []struct {
		name     string
		maker    Order
		taker    Order
		qty      Uint
		quoteQty Uint
		price    Uint
	}{
		{
			name: "sell taker by quote",
			maker: Order{
				side:                  OrderSideBuy,
				price:                 shifted("3"),
				quantity:              shifted("40"),
				quoteQuantity:         shifted("0"),
				available:             shifted("120"),
				restQuantity:          shifted("40"),
				executedQuantity:      shifted("0"),
				executedQuoteQuantity: shifted("0"),
				marketQuoteMode:       false,
			},
			taker: Order{
				side:                  OrderSideSell,
				price:                 shifted("1000000"),
				quantity:              shifted("10"),
				quoteQuantity:         shifted("100"),
				available:             NewMaxUint(),
				restQuantity:          shifted("0"),
				executedQuantity:      shifted("0"),
				executedQuoteQuantity: shifted("0"),
				marketQuoteMode:       true,
			},
			qty:      shifted("33.333"),
			quoteQty: shifted("100"),
			price:    shifted("3"),
		},
		{
			name: "buy available not enough",
			maker: Order{
				side:                  OrderSideBuy,
				price:                 shifted("20"),
				quantity:              shifted("15"),
				quoteQuantity:         shifted("0"),
				available:             shifted("10"),
				restQuantity:          shifted("15"),
				executedQuantity:      shifted("0"),
				executedQuoteQuantity: shifted("0"),
				marketQuoteMode:       false,
			},
			taker: Order{
				side:                  OrderSideSell,
				price:                 shifted("10"),
				quantity:              shifted("10"),
				quoteQuantity:         shifted("0"),
				available:             shifted("20"),
				restQuantity:          shifted("10"),
				executedQuantity:      shifted("0"),
				executedQuoteQuantity: shifted("0"),
				marketQuoteMode:       false,
			},
			qty:      shifted("0.5"),
			quoteQty: shifted("10"),
			price:    shifted("20"),
		},
		{
			name: "sell available not enough",
			maker: Order{
				side:                  OrderSideBuy,
				price:                 shifted("20"),
				quantity:              shifted("15"),
				quoteQuantity:         shifted("0"),
				available:             shifted("300"),
				restQuantity:          shifted("15"),
				executedQuantity:      shifted("0"),
				executedQuoteQuantity: shifted("0"),
				marketQuoteMode:       false,
			},
			taker: Order{
				side:                  OrderSideSell,
				price:                 shifted("10"),
				quantity:              shifted("10"),
				quoteQuantity:         shifted("0"),
				available:             shifted("8"),
				restQuantity:          shifted("10"),
				executedQuantity:      shifted("0"),
				executedQuoteQuantity: shifted("0"),
				marketQuoteMode:       false,
			},
			qty:      shifted("8"),
			quoteQty: shifted("160"),
			price:    shifted("20"),
		},
		{
			name: "both available enough",
			maker: Order{
				side:                  OrderSideBuy,
				price:                 shifted("20"),
				quantity:              shifted("15"),
				quoteQuantity:         shifted("0"),
				available:             shifted("300"),
				restQuantity:          shifted("15"),
				executedQuantity:      shifted("0"),
				executedQuoteQuantity: shifted("0"),
				marketQuoteMode:       false,
			},
			taker: Order{
				side:                  OrderSideSell,
				price:                 shifted("10"),
				quantity:              shifted("10"),
				quoteQuantity:         shifted("0"),
				available:             shifted("10"),
				restQuantity:          shifted("10"),
				executedQuantity:      shifted("0"),
				executedQuoteQuantity: shifted("0"),
				marketQuoteMode:       false,
			},
			qty:      shifted("10"),
			quoteQty: shifted("200"),
			price:    shifted("20"),
		},
	}

	for _, test := range tests {
		qty, quoteQty, price := calcExecutingQuantities(&test.maker, &test.taker, lotStep, quoteLotStep)
		require.True(t, price.Equals(test.price), fmt.Sprintf("%s, want: %s, got: %s", test.name, test.price.ToFloatString(), price.ToFloatString()))
		require.True(t, qty.Equals(test.qty), fmt.Sprintf("%s, want: %s, got: %s", test.name, test.qty.ToFloatString(), qty.ToFloatString()))
		require.True(t, quoteQty.Equals(test.quoteQty), fmt.Sprintf("%s, want: %s, got: %s", test.name, test.quoteQty.ToFloatString(), quoteQty.ToFloatString()))
	}
}

func shifted(val string) Uint {
	res, _ := NewUintFromFloatString(val)
	return res
}
