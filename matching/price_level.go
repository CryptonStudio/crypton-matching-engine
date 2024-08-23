package matching

import (
	"github.com/cryptonstudio/crypton-matching-engine/types/list"
)

// PriceLevelL2 contains price and visible volume in order bool.
type PriceLevelL2 struct {
	Price  uint64
	Volume uint64
}

// PriceLevelL3 contains price and total/visible volume in order bool and encapsulates order queue management.
// Total volume is separated to visible and hidden parts (only total and visible values are stored for optimization).
// Internal queue structure is either pre-created circular queue in struct or circular queue in heap.
// By default pre-created queue in struct is used until orders amount increase it's capacity.
// NOTE: Not thread-safe.
type PriceLevelL3 struct {
	price   Uint
	volume  Uint // total volume of entire order queue
	visible Uint // visible volume of entire order queue
	queue   *list.List[*Order]
}

// NewPriceLevelL3 creates and returns new PriceLevelL3 instance.
func NewPriceLevelL3() *PriceLevelL3 {
	return &PriceLevelL3{
		queue: list.NewList[*Order](),
	}
}

////////////////////////////////////////////////////////////////
// Getters
////////////////////////////////////////////////////////////////

// Price returns price level of the queue.
func (pl *PriceLevelL3) Price() Uint {
	return pl.price
}

// Volume returns total orders volume.
func (pl *PriceLevelL3) Volume() Uint {
	return pl.volume
}

// Visible returns visible orders volume.
func (pl *PriceLevelL3) Visible() Uint {
	return pl.visible
}

// Orders returns amount of orders in the queue.
func (pl *PriceLevelL3) Orders() int {
	return pl.queue.Len()
}

// Queue returns the order queue.
func (pl *PriceLevelL3) Queue() *list.List[*Order] {
	return pl.queue
}

func (pl *PriceLevelL3) Iterator() list.Iterator[*Order] {
	return pl.queue.Iterator()
}

// Clean cleans the price level by removing all queued orders.
func (pl *PriceLevelL3) Clean() {
	pl.price = NewZeroUint()
	pl.volume = NewZeroUint()
	pl.visible = NewZeroUint()
	pl.queue.Clean()
}
