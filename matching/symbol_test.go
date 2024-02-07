package matching

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSymbol(t *testing.T) {
	testCases := []struct {
		name  string
		sym   Symbol
		valid bool
	}{
		{
			name: "valid",
			sym: Symbol{id: 1, name: "",
				priceLimits:   Limits{Min: NewUint(1), Max: NewUint(10), Step: NewUint(1)},
				lotSizeLimits: Limits{Min: NewUint(1), Max: NewUint(10), Step: NewUint(1)},
			},
			valid: true,
		},
		{
			name: "min-max price",
			sym: Symbol{id: 1, name: "",
				priceLimits:   Limits{Min: NewUint(10), Max: NewUint(10), Step: NewUint(1)},
				lotSizeLimits: Limits{Min: NewUint(1), Max: NewUint(10), Step: NewUint(1)},
			},
			valid: false,
		},
		{
			name: "min-max lot",
			sym: Symbol{id: 1, name: "",
				priceLimits:   Limits{Min: NewUint(1), Max: NewUint(10), Step: NewUint(1)},
				lotSizeLimits: Limits{Min: NewUint(10), Max: NewUint(10), Step: NewUint(1)},
			},
			valid: false,
		},
		{
			name: "price big step",
			sym: Symbol{id: 1, name: "",
				priceLimits:   Limits{Min: NewUint(1), Max: NewUint(10), Step: NewUint(10)},
				lotSizeLimits: Limits{Min: NewUint(1), Max: NewUint(10), Step: NewUint(1)},
			},
			valid: false,
		},
		{
			name: "lot big step",
			sym: Symbol{id: 1, name: "",
				priceLimits:   Limits{Min: NewUint(1), Max: NewUint(10), Step: NewUint(1)},
				lotSizeLimits: Limits{Min: NewUint(1), Max: NewUint(10), Step: NewUint(10)},
			},
			valid: false,
		},
		{
			name: "price zero step",
			sym: Symbol{id: 1, name: "",
				priceLimits:   Limits{Min: NewUint(1), Max: NewUint(10), Step: NewUint(0)},
				lotSizeLimits: Limits{Min: NewUint(1), Max: NewUint(10), Step: NewUint(1)},
			},
			valid: false,
		},
		{
			name: "lot zero step",
			sym: Symbol{id: 1, name: "",
				priceLimits:   Limits{Min: NewUint(1), Max: NewUint(10), Step: NewUint(1)},
				lotSizeLimits: Limits{Min: NewUint(1), Max: NewUint(10), Step: NewUint(0)},
			},
			valid: false,
		},
	}
	for _, tc := range testCases {
		require.Equal(t, tc.valid, tc.sym.Valid(), tc.name)
	}
}
