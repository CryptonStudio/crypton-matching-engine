package matching

import (
	"sync"

	"github.com/cryptonstudio/crypton-matching-engine/types/avl"
	"github.com/cryptonstudio/crypton-matching-engine/types/list"
)

// Allocator is an object encapsulating all used objects allocation using sync.Pool internally.
type Allocator struct {
	// If allocator is using pool (true is experimental)
	// TODO: debug unpredictable behavior with pool usage.
	usePool bool

	// Price levels
	priceLevels sync.Pool

	// Orders
	orders sync.Pool

	// Pools used by containers
	priceLevelNodes    sync.Pool // used by avl.Tree[Uint, *PriceLevelL3]
	orderQueueElements sync.Pool // used by list.List
}

// NewAllocator creates and returns new Allocator instance.
func NewAllocator(usePool bool) *Allocator {
	a := Allocator{
		usePool: usePool,
	}

	if !a.usePool {
		return &a
	}

	// Pool setup.

	// Price levels
	a.priceLevels = sync.Pool{New: func() any {
		return NewPriceLevelL3()
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

	return &a
}

////////////////////////////////////////////////////////////////
// Price levels
////////////////////////////////////////////////////////////////

// NewPriceLevelTree allocates PriceLevelTree instance in direct order.
func (a *Allocator) NewPriceLevelTree() avl.Tree[Uint, *PriceLevelL3] {
	if !a.usePool {
		return avl.NewTree[Uint, *PriceLevelL3](
			func(a, b Uint) int { return a.Cmp(b) },
		)
	}

	return avl.NewTreePooled[Uint, *PriceLevelL3](
		func(a, b Uint) int { return a.Cmp(b) },
		&a.priceLevelNodes,
	)
}

// NewPriceLevelReversedTree releases PriceLevelTree instance in reversed order.
func (a *Allocator) NewPriceLevelReversedTree() avl.Tree[Uint, *PriceLevelL3] {
	if !a.usePool {
		return avl.NewTree[Uint, *PriceLevelL3](
			func(a, b Uint) int { return -a.Cmp(b) },
		)
	}

	return avl.NewTreePooled[Uint, *PriceLevelL3](
		func(a, b Uint) int { return -a.Cmp(b) },
		&a.priceLevelNodes,
	)
}

// GetPriceLevel allocates PriceLevelL3 instance.
func (a *Allocator) GetPriceLevel() *PriceLevelL3 {
	if !a.usePool {
		return NewPriceLevelL3()
	}

	// Get from the pool
	return a.priceLevels.Get().(*PriceLevelL3)
}

// PutPriceLevel releases PriceLevelL3 instance.
func (a *Allocator) PutPriceLevel(priceLevel *PriceLevelL3) {
	if !a.usePool {
		return
	}

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
	if !a.usePool {
		return new(Order)
	}

	// Get from the pool
	return a.orders.Get().(*Order)
}

// PutOrder releases Order instance.
func (a *Allocator) PutOrder(order *Order) {
	if !a.usePool {
		return
	}

	// Clean up the instance before releasing
	order.Clean()
	// Put back to the pool
	a.orders.Put(order)
}
