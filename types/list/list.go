package list

import (
	"sync"
)

// List represents a doubly linked list.
//
// A doubly linked list (DLL) is a special type of linked list in which each node contains
// a pointer to the previous node as well as the next node of the linked list.
type List[T any] struct {
	pool *sync.Pool // optional pool used to create/release list elements
	root Element[T] // sentinel list element, only &root, root.prev, and root.next are used
	len  int        // current list length excluding (this) sentinel element
}

// NewList creates new List instance.
func NewList[T any]() *List[T] {
	return NewListPooled[T](nil)
}

// NewListPooled creates new List instance.
// Pooled tree uses given pool for nodes creating/releasing.
func NewListPooled[T any](pool *sync.Pool) *List[T] {
	l := new(List[T])
	l.pool = pool
	l.root.next = &l.root
	l.root.prev = &l.root
	return l
}

// Front returns the first element of list l or nil if the list is empty.
func (l *List[T]) Front() *Element[T] {
	if l.len == 0 {
		return nil
	}
	return l.root.next
}

// Back returns the last element of list l or nil if the list is empty.
func (l *List[T]) Back() *Element[T] {
	if l.len == 0 {
		return nil
	}
	return l.root.prev
}

// Len returns the number of elements of list l.
func (l *List[T]) Len() int {
	return l.len
}

// PushFront inserts a new element e with value v at the front of list l and returns e.
func (l *List[T]) PushFront(v T) *Element[T] {
	l.lazyInit()
	return l.insertValue(v, &l.root)
}

// PushBack inserts a new element e with value v at the back of list l and returns e.
func (l *List[T]) PushBack(v T) *Element[T] {
	l.lazyInit()
	return l.insertValue(v, l.root.prev)
}

// PushBackList inserts a copy of another list at the back of list l.
// The lists l and other may be the same. They must not be nil.
func (l *List[T]) PushBackList(other *List[T]) {
	l.lazyInit()
	for i, e := other.Len(), other.Front(); i > 0; i, e = i-1, e.Next() {
		l.insertValue(e.Value, l.root.prev)
	}
}

// PushFrontList inserts a copy of another list at the front of list l.
// The lists l and other may be the same. They must not be nil.
func (l *List[T]) PushFrontList(other *List[T]) {
	l.lazyInit()
	for i, e := other.Len(), other.Back(); i > 0; i, e = i-1, e.Prev() {
		l.insertValue(e.Value, &l.root)
	}
}

// InsertBefore inserts a new element e with value v immediately before mark and returns e.
// If mark is not an element of l, the list is not modified.
// The mark must not be nil.
func (l *List[T]) InsertBefore(v T, mark *Element[T]) *Element[T] {
	if mark.list != l {
		return nil
	}
	// see comment in List.Remove about initialization of l
	return l.insertValue(v, mark.prev)
}

// InsertAfter inserts a new element e with value v immediately after mark and returns e.
// If mark is not an element of l, the list is not modified.
// The mark must not be nil.
func (l *List[T]) InsertAfter(v T, mark *Element[T]) *Element[T] {
	if mark.list != l {
		return nil
	}
	// see comment in List.Remove about initialization of l
	return l.insertValue(v, mark)
}

// MoveToFront moves element e to the front of list l.
// If e is not an element of l, the list is not modified.
// The element must not be nil.
func (l *List[T]) MoveToFront(e *Element[T]) {
	if e.list != l || l.root.next == e {
		return
	}
	// see comment in List.Remove about initialization of l
	l.move(e, &l.root)
}

// MoveToBack moves element e to the back of list l.
// If e is not an element of l, the list is not modified.
// The element must not be nil.
func (l *List[T]) MoveToBack(e *Element[T]) {
	if e.list != l || l.root.prev == e {
		return
	}
	// see comment in List.Remove about initialization of l
	l.move(e, l.root.prev)
}

// MoveBefore moves element e to its new position before mark.
// If e or mark is not an element of l, or e == mark, the list is not modified.
// The element and mark must not be nil.
func (l *List[T]) MoveBefore(e, mark *Element[T]) {
	if e.list != l || e == mark || mark.list != l {
		return
	}
	l.move(e, mark.prev)
}

// MoveAfter moves element e to its new position after mark.
// If e or mark is not an element of l, or e == mark, the list is not modified.
// The element and mark must not be nil.
func (l *List[T]) MoveAfter(e, mark *Element[T]) {
	if e.list != l || e == mark || mark.list != l {
		return
	}
	l.move(e, mark)
}

// Remove removes e from l if e is an element of list l.
func (l *List[T]) Remove(e *Element[T]) (v T, err error) {
	if e == nil {
		err = ErrorListElementIsNil
		return
	}
	if e.list != l {
		err = ErrorListElementIsNotInTheList
		return
	}
	v = e.Value
	l.remove(e)
	return
}

// Clean cleans list l by removing all existing elements.
func (l *List[T]) Clean() {
	// Release list entries if pool is used
	if l.pool != nil {
		for e := l.Front(); e != nil; {
			next := e.next
			// Clean up removed element to avoid memory leaks
			// don't need to clear e.Value, it always changed when adding.
			e.next, e.prev, e.list = nil, nil, nil
			l.pool.Put(e)
			e = next
		}
	}
	l.root.next = &l.root
	l.root.prev = &l.root

	l.len = 0
}

// lazyInit lazily initializes a zero List value.
func (l *List[T]) lazyInit() {
	if l.root.next == nil {
		l.root.next = &l.root
		l.root.prev = &l.root
	}
}

// insert inserts e after at, increments l.len, and returns e.
func (l *List[T]) insert(e, at *Element[T]) *Element[T] {
	e.prev = at
	e.next = at.next
	e.prev.next = e
	e.next.prev = e
	e.list = l
	l.len++
	return e
}

// insertValue is a convenience wrapper for insert(&Element{Value: v}, at).
func (l *List[T]) insertValue(v T, at *Element[T]) (e *Element[T]) {
	// Create list element
	if l.pool != nil {
		e = l.pool.Get().(*Element[T])
		e.Value = v
	} else {
		e = &Element[T]{Value: v}
	}
	return l.insert(e, at)
}

// move moves e to next to at.
func (l *List[T]) move(e, at *Element[T]) {
	if e == at {
		return
	}
	e.prev.next = e.next
	e.next.prev = e.prev

	e.prev = at
	e.next = at.next
	e.prev.next = e
	e.next.prev = e
}

// remove removes e from its list, decrements l.len
func (l *List[T]) remove(e *Element[T]) {
	e.prev.next = e.next
	e.next.prev = e.prev
	l.len--

	// Clean up removed element to avoid memory leaks
	// don't need to clear e.Value, it always changed when adding.
	e.next, e.prev, e.list = nil, nil, nil

	// Release list entry if pool is used
	if l.pool != nil {
		l.pool.Put(e)
	}
}
