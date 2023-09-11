package matching

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewUintFromFloatString(t *testing.T) {
	tc := []struct {
		number   string
		expected string
	}{
		{
			number:   "10",
			expected: "10000000000000",
		},
		{
			number:   "0.000000000001",
			expected: "1",
		},
		{
			number:   "1.000000000000",
			expected: "1000000000000",
		},
		{
			number:   "0.000000000100",
			expected: "100",
		},
		{
			number:   "1.0000000001",
			expected: "1000000000100",
		},
		{
			number:   "0.999999999999",
			expected: "999999999999",
		},
		{
			number:   "0.9999999999990000000000000",
			expected: "999999999999",
		},
		{
			number:   "0.",
			expected: "0",
		},
		{
			number:   "0.0",
			expected: "0",
		},
	}

	for _, v := range tc {
		expected, err := NewUintFromStr(v.expected)
		require.NoError(t, err, v.expected)
		result, err := NewUintFromFloatString(v.number)
		require.NoError(t, err, v.number)

		require.Equal(t, expected.String(), result.String())
	}
}

func TestRemoveTrailingZeros(t *testing.T) {
	tc := []struct {
		number   string
		expected string
	}{
		{
			number:   "123.123000",
			expected: "123.123",
		},
		{
			number:   "123.000",
			expected: "123",
		},
		{
			number:   "123.00100",
			expected: "123.001",
		},
		{
			number:   "123.0",
			expected: "123",
		},
		{
			number:   "123.",
			expected: "123",
		},
	}

	for _, v := range tc {
		result := removeTrailingZerosFromFloatStr(v.number)
		require.Equal(t, v.expected, result)
	}
}

func TestUintToFloatString(t *testing.T) {
	tc := []struct {
		number   string
		expected string
	}{
		{
			number:   "1000000000000",
			expected: "1",
		},
		{
			number:   "100000000000",
			expected: "0.1",
		},
		{
			number:   "10000000000000",
			expected: "10",
		},
		{
			number:   "10000000000100",
			expected: "10.0000000001",
		},
		{
			number:   "999999999999",
			expected: "0.999999999999",
		},
		{
			number:   "10",
			expected: "0.00000000001",
		},
		{
			number:   "0",
			expected: "0",
		},
	}

	for _, v := range tc {
		uintForm, err := NewUintFromStr(v.number)
		require.NoError(t, err)

		floatForm := uintForm.ToFloatString()
		require.Equal(t, v.expected, floatForm)
	}
}

func TestAddLeadingZeros(t *testing.T) {
	tc := []struct {
		number   string
		expected string
		panics   bool
	}{
		{
			number:   "1",
			expected: "000000000001",
			panics:   false,
		},
		{
			number:   "000000010",
			expected: "000000000010",
			panics:   false,
		},
		{
			number:   "101010101010",
			expected: "101010101010",
			panics:   false,
		},
		{
			number:   "1010101010101",
			expected: "",
			panics:   true,
		},
	}

	for _, v := range tc {
		if !v.panics {
			floatForm := addLeadingZerosToStr(v.number)
			require.Equal(t, v.expected, floatForm)
		} else {
			require.Panics(t, func() {
				addLeadingZerosToStr(v.number)
			})
		}
	}
}

func TestUintQuoRem(t *testing.T) {
	tc := []struct {
		number            Uint
		div               uint64
		expectedInteger   string
		expectedRemainder string
	}{
		{
			number:            NewUint(10000),
			div:               100,
			expectedInteger:   "100",
			expectedRemainder: "0",
		},
		{
			number:            NewUint(10001),
			div:               100,
			expectedInteger:   "100",
			expectedRemainder: "1",
		},
		{
			number:            NewUint(10099),
			div:               100,
			expectedInteger:   "100",
			expectedRemainder: "99",
		},
	}

	for _, v := range tc {
		integer, remainder := v.number.QuoRem(NewUint(v.div))

		require.Equal(t, v.expectedInteger, integer.String())
		require.Equal(t, v.expectedRemainder, remainder.String())
	}
}

func BenchmarkRemoveTrailingZeros(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = removeTrailingZerosFromFloatStr("123.00100")
	}
}

func BenchmarkTakeParts(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = takePartsFromFloat("123.00100")
	}
}

func BenchmarkNewUintFromFloatString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = NewUintFromFloatString("123.00100")
	}
}

func TestFloatEdges(t *testing.T) {
	const postfix = "999999999999"
	testCases := []struct {
		number   string
		expected string
	}{
		{number: "1.13" + postfix, expected: "1.139999999999"},
		{number: "1.", expected: "1"},
	}

	for _, tc := range testCases {
		v, err := NewUintFromFloatString(tc.number)
		require.NoError(t, err)
		require.Equal(t, tc.expected, v.ToFloatString())
	}
}

/*
func TestNewUintFromFloatString2(t *testing.T) {
	r, _ := NewUintFromFloatString("0.01")
	t.Fatalf("r=%s", r.String())
}
*/
