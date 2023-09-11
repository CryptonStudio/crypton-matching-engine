package matching

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOrderRestAvailableQuantities(t *testing.T) {
	testCases := []struct {
		name              string
		side              OrderSide
		available         Uint
		restQuantity      Uint
		price             Uint
		marketQuoteMode   bool
		restQuoteQuantity Uint
		//
		expectRest      Uint
		expectRestQuote Uint
	}{
		// check for limits
		{
			name:              "limit,buy,available<rest",
			side:              OrderSideBuy,
			available:         NewUint(UintPrecision).Div64(2), // 0.5
			restQuantity:      NewUint(UintPrecision),          // 1.0
			price:             NewUint(2).Mul64(UintPrecision), // 2.0
			marketQuoteMode:   false,
			restQuoteQuantity: NewZeroUint(),
			expectRest:        NewUint(UintPrecision).Div64(4), // available/price = 0.25
			expectRestQuote:   NewUint(UintPrecision).Div64(2), // available
		},
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
			name:              "limit,sell,available<rest",
			side:              OrderSideSell,
			available:         NewUint(UintPrecision).Div64(2), // 0.5
			restQuantity:      NewUint(UintPrecision),          // 1.0
			price:             NewUint(2).Mul64(UintPrecision), // 2.0
			marketQuoteMode:   false,
			restQuoteQuantity: NewZeroUint(),
			expectRest:        NewUint(UintPrecision).Div64(2), // 0.5
			expectRestQuote:   NewUint(UintPrecision),
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
			restQuoteQuantity: NewUint(UintPrecision),          // 1.0
			expectRest:        NewUint(UintPrecision).Div64(2), // 0.5
			expectRestQuote:   NewUint(UintPrecision),          // 1.0 / price
		},
		{
			name:              "market,sell-by-quote,available>rest",
			side:              OrderSideSell,
			available:         NewUint(UintPrecision).Mul64(2), // 2.0
			restQuantity:      NewUint(UintPrecision),          // 1.0
			price:             NewUint(2).Mul64(UintPrecision), // 2.0
			marketQuoteMode:   true,
			restQuoteQuantity: NewUint(UintPrecision),          // 1.0
			expectRest:        NewUint(UintPrecision).Div64(2), // 0.5
			expectRestQuote:   NewUint(UintPrecision),          // 1.0 / price
		},
	}

	for _, tc := range testCases {
		order := Order{
			available:       tc.available,
			restQuantity:    tc.restQuantity,
			side:            tc.side,
			marketQuoteMode: tc.marketQuoteMode,
			quoteQuantity:   tc.restQuoteQuantity,
		}

		rest, restQuote := order.RestAvailableQuantities(tc.price)

		require.True(t, tc.expectRest.Equals(rest), tc.name)
		require.True(t, tc.expectRestQuote.Equals(restQuote), tc.name)
	}
}
