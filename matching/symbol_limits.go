package matching

import (
	"math"
)

// Limits contains just 3 numbers (min, max and step) and used for price and lot size limitations.
type Limits struct {
	Min  Uint
	Max  Uint
	Step Uint
}

func (l Limits) Valid() bool {
	if l.Min.GreaterThanOrEqualTo(l.Max) {
		return false
	}

	if l.Step.IsZero() {
		return false
	}

	if l.Min.LessThan(l.Step) {
		return false
	}

	return true
}

func QuoteLotSizeLimits(priceLimits Limits, lotSizeLimits Limits) Limits {
	limits := Limits{
		Min:  priceLimits.Min.Mul(lotSizeLimits.Min).Div64(UintPrecision),
		Max:  priceLimits.Max.Div64(UintPrecision).Mul(lotSizeLimits.Max),
		Step: priceLimits.Step.Mul(lotSizeLimits.Step).Div64(UintPrecision),
	}

	return limits
}

func ApplyLimits(v Uint, limits Limits) Uint {
	if v.GreaterThan(limits.Max) {
		return limits.Max
	}

	if v.LessThan(limits.Min) {
		return NewZeroUint()
	}

	return ApplySteps(v, limits.Step)
}

func ApplySteps(v Uint, step Uint) Uint {
	steps, _ := v.QuoRem(step)
	v = steps.Mul(step)

	return v
}

type symbolLimits struct {
	priceLimits   Limits
	lotSizeLimits Limits
}

func GetSoftLimits() Limits {
	min := NewUint(uint64(math.Sqrt(UintPrecision)))
	step := min
	max := ApplySteps(NewUint(UintPrecision).Mul64(UintPrecision), step)

	return Limits{
		Min:  min,
		Max:  max,
		Step: step,
	}
}
