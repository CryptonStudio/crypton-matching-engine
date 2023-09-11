package matching

const (
	// defaultOrderBookTaskQueueSize specifies size of queue of tasks which should be performed on single order book.
	defaultOrderBookTaskQueueSize = 256

	// defaultReservedOrderBookSlots specifies initial size of array storing order books by symbol id.
	defaultReservedOrderBookSlots = 1024

	// defaultReservedOrderSlots specifies initial size of hashmap array storing orders by order id separately for each order book.
	defaultReservedOrderSlots = 1024
)
