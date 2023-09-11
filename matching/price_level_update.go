package matching

// PriceLevelType is an enumeration of possible price level types (ask/bid).
type PriceLevelType uint8

const (
	// PriceLevelTypeBid represents bid price level type.
	PriceLevelTypeBid PriceLevelType = iota + 1
	// PriceLevelTypeAsk represents ask price level type.
	PriceLevelTypeAsk
)

////////////////////////////////////////////////////////////////

// PriceLevelUpdateKind is an enumeration of possible price level update kinds (add, update, delete).
type PriceLevelUpdateKind uint8

const (
	// PriceLevelUpdateKindAdd represents add price level update kind.
	PriceLevelUpdateKindAdd PriceLevelUpdateKind = iota + 1
	// PriceLevelUpdateKindUpdate represents update price level update kind.
	PriceLevelUpdateKindUpdate
	// PriceLevelUpdateKindDelete represents delete price level update kind.
	PriceLevelUpdateKindDelete
)

////////////////////////////////////////////////////////////////

// PriceLevelUpdate contains complete info about a price level update.
type PriceLevelUpdate struct {
	ID      uint64
	Kind    PriceLevelUpdateKind
	Side    OrderSide
	Price   Uint // price of the price level
	Volume  Uint // total volume of the price level
	Visible Uint // visible volume of the price level
	Orders  int  // amount of orders queued in the price level
	Top     bool // top of the order book flag
}
