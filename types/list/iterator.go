package list

// Iterator with ability to validate himself when current element is removed from list.
type Iterator[T any] struct {
	list    *List[T]
	prev    *Element[T]
	current *Element[T]
	next    *Element[T]
}

// Creates iterator. Iterator is not valid until Next() call.
func NewIterator[T any](list *List[T]) Iterator[T] {
	return Iterator[T]{
		list:    list,
		prev:    &list.root,
		current: nil,
		next:    nil,
	}
}

func (it *Iterator[T]) Current() *Element[T] {
	return it.current
}

func (it *Iterator[T]) Next() bool {
	// 1. start iteration
	if it.prev == &it.list.root && it.current == nil {
		it.current = it.list.Front()
	} else // 2. check first element is removed
	if it.prev == &it.list.root && it.current != it.list.Front() {
		it.current = it.list.Front()
	} else // 3. check middle element is removed
	if it.prev != &it.list.root && it.prev.Next() != it.current {
		it.current = it.prev.Next()
	} else { // 4. no changes in list
		it.prev = it.current
		it.current = it.next
	}

	if it.current == nil {
		return false
	}
	it.next = it.current.Next()
	return true
}

func (it *Iterator[T]) Valid() bool {
	return it.current != nil
}
