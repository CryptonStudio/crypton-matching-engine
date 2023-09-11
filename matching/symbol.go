package matching

// Symbol contains basic info about a trading symbol.
type Symbol struct {
	id   uint32
	name string
}

// NewSymbol creates new symbol with specified ID and name.
func NewSymbol(id uint32, name string) Symbol {
	return Symbol{
		id:   id,
		name: name,
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
