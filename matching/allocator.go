package matching

import (
	"sync"

	"github.com/cryptonstudio/crypton-matching-engine/types/avl"
	"github.com/cryptonstudio/crypton-matching-engine/types/list"
)

// Allocator is an object encapsulating all used objects allocation using sync.Pool internally.
type Allocator struct {

	// Price levels
	priceLevels sync.Pool

	// Orders
	orders sync.Pool

	// Pools used by containers
	priceLevelNodes    sync.Pool // used by avl.Tree[Uint, *PriceLevelL3]
	orderQueueElements sync.Pool // used by list.List
}

// NewAllocator creates and returns new Allocator instance.
func NewAllocator() *Allocator {
	a := new(Allocator)
	// Price levels
	a.priceLevels = sync.Pool{New: func() any {
		return NewPriceLevelL3(a)
	}}
	// Orders
	a.orders = sync.Pool{New: func() any {
		return new(Order)
	}}
	// Pools used by containers
	a.priceLevelNodes = sync.Pool{New: func() any {
		return new(avl.Node[Uint, *PriceLevelL3])
	}}
	a.orderQueueElements = sync.Pool{New: func() any {
		return new(list.Element[*Order])
	}}
	return a
}

////////////////////////////////////////////////////////////////
// Price levels
////////////////////////////////////////////////////////////////

// GetPriceLevel allocates PriceLevelL3 instance.
func (a *Allocator) GetPriceLevel() *PriceLevelL3 {
	priceLevel := a.priceLevels.Get().(*PriceLevelL3)
	// Get from the pool
	return priceLevel
}

// PutPriceLevel releases PriceLevelL3 instance.
func (a *Allocator) PutPriceLevel(priceLevel *PriceLevelL3) {
	// Clean up the instance before releasing
	priceLevel.Clean()
	// Put back to the pool
	a.priceLevels.Put(priceLevel)
}

////////////////////////////////////////////////////////////////
// Orders
////////////////////////////////////////////////////////////////

// GetOrder allocates Order instance.
func (a *Allocator) GetOrder() *Order {
	order := a.orders.Get().(*Order)
	// Get from the pool
	return order
}

// PutOrder releases Order instance.
func (a *Allocator) PutOrder(order *Order) {
	// Clean up the instance before releasing
	order.Clean()
	// Put back to the pool
	a.orders.Put(order)
}
