package avl

import (
	"sync"

	"gopkg.in/typ.v4"
)

// Tree is a binary search tree (BST) for ordered Go types (numbers & strings),
// implemented as an AVL tree (Adelson-Velsky and Landis tree), a type of self-balancing BST.
// This guarantees O(log t) operations on insertion, searching, and deletion.
type Tree[K, V any] struct {
	compare   func(a, b K) int
	pool      *sync.Pool
	root      *Node[K, V]
	mostLeft  *Node[K, V]
	mostRight *Node[K, V]
	size      int
}

////////////////////////////////////////////////////////////////

// NewOrderedTree creates a new AVL tree using a default comparator function
// for any ordered type (ints, uints, floats, strings).
func NewOrderedTree[K typ.Ordered, V any]() Tree[K, V] {
	return NewTree[K, V](typ.Compare[K])
}

// NewTree creates a new AVL tree using a comparator function that is
// expected to return 0 if a == b, -1 if a < b, and +1 if a > b.
func NewTree[K, V any](compare func(a, b K) int) Tree[K, V] {
	return Tree[K, V]{
		compare: compare,
	}
}

// NewTreePooled creates a new AVL tree using a comparator function that is
// expected to return 0 if a == b, -1 if a < b, and +1 if a > b.
// Pooled tree uses given pool for nodes creating/releasing.
func NewTreePooled[K, V any](compare func(a, b K) int, pool *sync.Pool) Tree[K, V] {
	return Tree[K, V]{
		compare: compare,
		pool:    pool,
	}
}

////////////////////////////////////////////////////////////////

// Size returns the amount of nodes in the tree.
func (t *Tree[K, V]) Size() int {
	return t.size
}

// Contains checks if node with given key exists in the tree by iterating the binary search tree.
func (t *Tree[K, V]) Contains(key K) bool {
	if t.root == nil {
		return false
	}
	return t.root.contains(key, t.compare)
}

// Find finds the node with given key in the tree by iterating the binary search tree.
func (t *Tree[K, V]) Find(key K) *Node[K, V] {
	if t.root == nil {
		return nil
	}
	return t.root.find(key, t.compare)
}

// Add inserts a node with given key and value to the tree.
// Duplicate keys are not allowed so error will be returned on duplicate.
func (t *Tree[K, V]) Add(key K, value V) (node *Node[K, V], err error) {
	// Create tree node
	if t.pool != nil {
		node = t.pool.Get().(*Node[K, V])
		node.key = key
		node.value = value
	} else {
		node = &Node[K, V]{
			key:   key,
			value: value,
		}
	}
	// Add the node to the tree
	if t.root == nil {
		t.root = node
	} else {
		newRoot, err := t.root.add(node, t.compare)
		if err != nil {
			return nil, err
		}
		t.root = newRoot
	}
	t.size++
	// Update most left/right nodes
	if t.mostLeft == nil || t.compare(node.key, t.mostLeft.key) < 0 {
		t.mostLeft = node
	}
	if t.mostRight == nil || t.compare(node.key, t.mostRight.key) > 0 {
		t.mostRight = node
	}
	return
}

// Remove removes a node with given value from the tree.
func (t *Tree[K, V]) Remove(key K) (value V, err error) {
	if t.root == nil {
		err = ErrorTreeNodeNotFound
		return
	}
	var node, newRoot *Node[K, V]
	node, newRoot, err = t.root.remove(key, t.compare)
	if err != nil {
		return
	}
	t.root = newRoot
	value = node.value
	// Release tree node if pool is used
	if t.pool != nil {
		*node = Node[K, V]{}
		t.pool.Put(node)
	}
	t.size--
	// Update most left/right nodes
	if t.mostLeft == node {
		// TODO: Optimize somehow to safely use nextLeft instead (using node.nextLeft does not look safe yet)
		if t.root != nil {
			t.mostLeft = t.root.MostLeft()
		} else {
			t.mostLeft = nil
		}
	}
	if t.mostRight == node {
		// TODO: Optimize somehow to safely use nextRight instead (using node.nextRight does not look safe yet)
		if t.root != nil {
			t.mostRight = t.root.MostRight()
		} else {
			t.mostRight = nil
		}
	}
	return
}

// MostLeft returns most left node.
func (t *Tree[K, V]) MostLeft() *Node[K, V] {
	return t.mostLeft
}

// MostRight returns most right node.
func (t *Tree[K, V]) MostRight() *Node[K, V] {
	return t.mostRight
}

// Clear will reset this tree to an empty tree.
func (t *Tree[K, V]) Clear() {
	if t.root != nil {
		t.root.iteratePostOrder(func(node *Node[K, V]) bool {
			if t.pool != nil {
				*node = Node[K, V]{}
				t.pool.Put(node)
			}

			return false
		})
	}
	t.root = nil
	t.size = 0
}

// IteratePreOrder will iterate all values in this tree by first visiting each
// node's value, followed by the its left branch, and then its right branch.
//
// This is useful when copying binary search trees, as inserting back in this
// order will guarantee the clone will have the exact same layout.
func (t *Tree[K, V]) IteratePreOrder(f func(value V) bool) {
	if t.root == nil {
		return
	}
	t.root.iteratePreOrder(func(v *Node[K, V]) bool {
		return f(v.value)
	})
}

// IterateInOrder will iterate all values in this tree by first visiting each
// node's left branch, followed by the its own value, and then its right branch.
//
// This is useful when reading a tree's values in order, as this guarantees
// iterating them in a sorted order.
func (t *Tree[K, V]) IterateInOrder(f func(value V) bool) {
	if t.root == nil {
		return
	}
	t.root.iterateInOrder(func(v *Node[K, V]) bool {
		return f(v.value)
	})
}

// IteratePostOrder will iterate all values in this tree by first visiting each
// node's left branch, followed by the its right branch, and then its own value.
//
// This is useful when deleting values from a tree, as this guarantees to always
// delete leaf nodes.
func (t *Tree[K, V]) IteratePostOrder(f func(value V) bool) {
	if t.root == nil {
		return
	}
	t.root.iteratePostOrder(func(v *Node[K, V]) bool {
		return f(v.value)
	})
}
