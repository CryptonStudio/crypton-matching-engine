package matching

// Symbol contains basic info about a trading symbol.
type Symbol struct {
	id            uint32
	name          string
	priceLimits   Limits
	lotSizeLimits Limits
}

// Limits contains just 3 numbers (min, max and step) and used for price and lot size limitations.
type Limits struct {
	Min  Uint
	Max  Uint
	Step Uint
}

// NewSymbol creates new symbol with specified ID and name.
func NewSymbol(id uint32, name string) Symbol {
	return Symbol{
		id:   id,
		name: name,
		priceLimits: Limits{
			Min:  NewUint(1),
			Max:  NewMaxUint(),
			Step: NewUint(1),
		},
		lotSizeLimits: Limits{
			Min:  NewUint(1),
			Max:  NewMaxUint(),
			Step: NewUint(1),
		},
	}
}

// NewSymbolWithLimits creates new symbol with specified ID, name, price and lot size limits.
func NewSymbolWithLimits(id uint32, name string, priceLimits Limits, lotSizeLimits Limits) Symbol {
	return Symbol{
		id:            id,
		name:          name,
		priceLimits:   priceLimits,
		lotSizeLimits: lotSizeLimits,
	}
}

// ID returns the symbol ID.
func (s Symbol) ID() uint32 {
	return s.id
}

// Name returns the symbol name.
func (s Symbol) Name() string {
	return s.name
}

// PriceLimits returns price limitations of the symbol.
func (s Symbol) PriceLimits() Limits {
	return s.priceLimits
}

// LotSizeLimits returns lot size limitations of the symbol.
func (s Symbol) LotSizeLimits() Limits {
	return s.lotSizeLimits
}
