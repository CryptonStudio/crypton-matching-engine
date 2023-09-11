package matching

import (
	"encoding/json"
	"fmt"
	"strings"

	"lukechampine.com/uint128"
)

const (
	// UintPrecision is precision of decimal places for Uint.
	UintPrecision = 1_000_000_000_000
	// UintComma is the amount of zeros in UintPrecision.
	UintComma = 12
)

// uintMaxValue is the max possible value of the Uint.
var uintMaxValue = Uint{uint128.Max}

type Uint struct {
	v uint128.Uint128
}

func NewZeroUint() Uint {
	return Uint{}
}

func NewMaxUint() Uint {
	return Uint{uint128.Max}
}

func NewUint(u uint64) Uint {
	return Uint{v: uint128.From64(u)}
}

func NewUintFromUint128(u uint128.Uint128) Uint {
	return Uint{v: u}
}

func NewUintFromStr(v string) (Uint, error) {
	if v == "" {
		return NewZeroUint(), nil
	}

	u, err := uint128.FromString(v)
	if err != nil {
		return Uint{}, err
	}

	return Uint{
		v: u,
	}, nil
}

func NewUintFromFloatString(number string) (Uint, error) {
	integer, decimal := takePartsFromFloat(number)
	resultUint := NewZeroUint()

	// number is integer return its uint form with
	if decimal == "" {
		integer = fmt.Sprintf("%s%s", integer, strings.Repeat("0", UintComma))

		return NewUintFromStr(integer)
	}

	// add integer part of number
	if integer != "0" {
		uintFromStr, err := NewUintFromStr(fmt.Sprintf("%s%s", integer, strings.Repeat("0", UintComma)))
		if err != nil {
			return Uint{}, err
		}

		resultUint = resultUint.Add(uintFromStr)
	}

	// if number len after "." more than comma, truncate it
	if len(decimal) > UintComma {
		decimal = decimal[:UintComma]
	}

	// if number len after "." less than comma, supplement it
	if len(decimal) < UintComma {
		decimal = fmt.Sprintf("%s%s", decimal, strings.Repeat("0", UintComma-len(decimal)))
	}

	// add decimal part of number
	uintFromStr, err := NewUintFromStr(removeLeadingZerosFromStr(decimal))
	if err != nil {
		return Uint{}, err
	}

	resultUint = resultUint.Add(uintFromStr)

	return resultUint, nil
}

func (u Uint) ToUint128() uint128.Uint128 {
	return u.v
}

func (u Uint) ToFloatString() string {
	integerPart, remainderPart := u.QuoRem(NewUint(UintPrecision))

	resultStr := integerPart.String()

	if !remainderPart.IsZero() {
		remainderStr := remainderPart.String()

		if len(remainderStr) < UintComma {
			remainderStr = addLeadingZerosToStr(remainderStr)
		}

		resultStr = removeTrailingZerosFromFloatStr(fmt.Sprintf("%s.%s", resultStr, remainderStr))
	}

	return resultStr
}

func (u Uint) Add(v Uint) Uint {
	u.v = u.v.Add(v.v)
	return u
}

func (u Uint) Add64(v uint64) Uint {
	u.v = u.v.Add64(v)
	return u
}

func (u Uint) Sub(v Uint) Uint {
	u.v = u.v.Sub(v.v)
	return u
}

func (u Uint) Mul(v Uint) Uint {
	u.v = u.v.Mul(v.v)
	return u
}

func (u Uint) Mul64(v uint64) Uint {
	u.v = u.v.Mul64(v)
	return u
}

func (u Uint) QuoRem(v Uint) (Uint, Uint) {
	var remainder uint128.Uint128
	u.v, remainder = u.v.QuoRem(v.v)
	return u, Uint{v: remainder}
}

func (u Uint) Div64(v uint64) Uint {
	u.v = u.v.Div64(v)
	return u
}

func (u Uint) Cmp(v Uint) int {
	return u.v.Cmp(v.v)
}

func (u Uint) IsZero() bool {
	return u.v.IsZero()
}

func (u Uint) IsMax() bool {
	return u.Equals(uintMaxValue)
}

func (u Uint) Equals(v Uint) bool {
	return u.v.Equals(v.v)
}

func (u Uint) Equals64(v uint64) bool {
	return u.v.Equals64(v)
}

func (u Uint) LessThan(v Uint) bool {
	return u.v.Cmp(v.v) < 0
}

func (u Uint) LessThanOrEqualTo(v Uint) bool {
	return u.v.Cmp(v.v) <= 0
}

func (u Uint) GreaterThan(v Uint) bool {
	return u.v.Cmp(v.v) > 0
}

func (u Uint) GreaterThanOrEqualTo(v Uint) bool {
	return u.v.Cmp(v.v) >= 0
}

func (u Uint) String() string {
	return u.v.String()
}

// ---------------------JSON---------------------

var (
	_ json.Marshaler   = Uint{}
	_ json.Unmarshaler = &Uint{}
)

func (u Uint) MarshalJSON() ([]byte, error) {
	return []byte(u.String()), nil
}

func (u *Uint) UnmarshalJSON(data []byte) error {
	u128, err := uint128.FromString(string(data))
	if err != nil {
		return err
	}

	u.v = u128

	return nil
}

// -------------------GOGO PROTO-------------------

func (u Uint) Marshal() ([]byte, error) {
	return []byte(u.String()), nil
}

func (u Uint) MarshalTo(data []byte) (int, error) {
	copy(data, u.String())

	return len(data), nil
}

func (u *Uint) Unmarshal(data []byte) error {
	u128, err := uint128.FromString(string(data))
	if err != nil {
		return err
	}

	u.v = u128

	return nil
}

func (u Uint) Size() int {
	return len(u.String())
}

func Min(a Uint, b Uint) Uint {
	if a.Cmp(b) <= 0 {
		return a
	}
	return b
}

func Max(a Uint, b Uint) Uint {
	if a.Cmp(b) >= 0 {
		return a
	}
	return b
}

func removeTrailingZerosFromFloatStr(number string) string {
	number = strings.TrimRight(number, "0")
	if number[len(number)-1:] == "." {
		return number[:len(number)-1]
	}

	return number
}

func removeLeadingZerosFromStr(number string) string {
	return strings.TrimLeft(number, "0")
}

func addLeadingZerosToStr(number string) string {
	if len(number) > UintComma {
		panic("number len more than comma value")
	}

	for len(number) < UintComma {
		number = fmt.Sprintf("0%s", number)
	}

	return number
}

func takePartsFromFloat(number string) (string, string) {
	numberParts := strings.Split(number, ".")

	if len(numberParts) == 1 {
		return numberParts[0], ""
	}

	return numberParts[0], numberParts[1]
}
