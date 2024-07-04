package matching

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOrderRestQuantities(t *testing.T) {
	testCases := []struct {
		name              string
		side              OrderSide
		available         Uint
		restQuantity      Uint
		price             Uint
		marketQuoteMode   bool
		restQuoteQuantity Uint
		executedQuote     Uint
		//
		expectRest      Uint
		expectRestQuote Uint
	}{
		// check for limits
		{
			name:              "limit,buy,available>rest",
			side:              OrderSideBuy,
			available:         NewUint(UintPrecision).Mul64(2), // 2.0
			restQuantity:      NewUint(UintPrecision),          // 1.0
			price:             NewUint(2).Mul64(UintPrecision), // 2.0
			marketQuoteMode:   false,
			restQuoteQuantity: NewZeroUint(),
			expectRest:        NewUint(UintPrecision),          // restQuantity = 1
			expectRestQuote:   NewUint(UintPrecision).Mul64(2), // restQuantity*price
		},
		{
			name:              "limit,sell,available>rest",
			side:              OrderSideSell,
			available:         NewUint(UintPrecision).Mul64(2), // 2.0
			restQuantity:      NewUint(UintPrecision),          // 1.0
			price:             NewUint(2).Mul64(UintPrecision), // 2.0
			marketQuoteMode:   false,
			restQuoteQuantity: NewZeroUint(),
			expectRest:        NewUint(UintPrecision),          // 1.0
			expectRestQuote:   NewUint(2).Mul64(UintPrecision), // 1.0 * price
		},
		// check for market by quote
		{
			name:              "market,buy-by-quote,available>rest",
			side:              OrderSideBuy,
			available:         NewUint(UintPrecision).Mul64(2), // 2.0
			restQuantity:      NewUint(UintPrecision),          // 1.0
			price:             NewUint(2).Mul64(UintPrecision), // 2.0
			marketQuoteMode:   true,
			restQuoteQuantity: NewUint(UintPrecision), // 1.0
			executedQuote:     NewUint(UintPrecision).Div64(2),
			expectRest:        NewUint(UintPrecision),          // == restQuantity
			expectRestQuote:   NewUint(UintPrecision).Div64(2), // quote - executedQuote
		},
		{
			name:              "market,sell-by-quote,available>rest",
			side:              OrderSideSell,
			available:         NewUint(UintPrecision).Mul64(2), // 2.0
			restQuantity:      NewUint(UintPrecision),          // 1.0
			price:             NewUint(2).Mul64(UintPrecision), // 2.0
			marketQuoteMode:   true,
			restQuoteQuantity: NewUint(UintPrecision), // 1.0
			executedQuote:     NewUint(UintPrecision).Div64(2),
			expectRest:        NewUint(UintPrecision),          // == restQuantity
			expectRestQuote:   NewUint(UintPrecision).Div64(2), // quote - executedQuote
		},
	}

	for _, tc := range testCases {
		order := Order{
			available:             tc.available,
			restQuantity:          tc.restQuantity,
			side:                  tc.side,
			marketQuoteMode:       tc.marketQuoteMode,
			quoteQuantity:         tc.restQuoteQuantity,
			executedQuoteQuantity: tc.executedQuote,
		}

		rest := order.RestQuantity()

		require.True(t, tc.expectRest.Equals(rest), fmt.Sprintf("%s, want: %s, have: %s", tc.name, tc.expectRest, rest))
	}
}
