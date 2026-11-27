package matching

// OrderOp is an enumeration of order book task operations reported in OrderError.
type OrderOp uint8

const (
	// OrderOpAdd is a task of AddOrder.
	OrderOpAdd OrderOp = iota + 1
	// OrderOpAddPair is a task of AddOrdersPair.
	OrderOpAddPair
	// OrderOpDelete is a task of DeleteOrder.
	OrderOpDelete
)

func (op OrderOp) String() string {
	switch op {
	case OrderOpAdd:
		return "add"
	case OrderOpAddPair:
		return "add_pair"
	case OrderOpDelete:
		return "delete"
	default:
		return "unknown"
	}
}
