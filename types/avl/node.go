package avl

type balanceFactor int8

const (
	balanceBalanced   balanceFactor = 0
	balanceRightHeavy balanceFactor = 1
	balanceLeftHeavy  balanceFactor = -1
)

type Node[K, V any] struct {
	key    K
	value  V
	parent *Node[K, V]
	left   *Node[K, V]
	right  *Node[K, V]
	height int
}

// Value returns key of the tree node.
func (n *Node[K, V]) Key() K {
	return n.key
}

// Value returns value of the tree node.
func (n *Node[K, V]) Value() V {
	return n.value
}

func (n *Node[K, V]) MostLeft() *Node[K, V] {
	if n.left == nil {
		// Found left most tree node
		return n
	}
	return n.left.MostLeft()
}

func (n *Node[K, V]) MostRight() *Node[K, V] {
	if n.right == nil {
		// Found right most tree node
		return n
	}
	return n.right.MostRight()
}

func (n *Node[K, V]) NextLeft() *Node[K, V] {
	if n.left == nil {
		// Found left most tree node, check parent
		parent := n.parent
		if parent != nil && n == parent.right {
			return parent
		}
		return nil
	}
	return n.left.MostRight()
}

func (n *Node[K, V]) NextRight() *Node[K, V] {
	if n.right == nil {
		// Found right most tree node, check parent
		parent := n.parent
		if parent != nil && n == parent.left {
			return parent
		}
		return nil
	}
	return n.right.MostLeft()
}

func (n *Node[K, V]) contains(key K, compare func(a, b K) int) bool {
	return n.find(key, compare) != nil
}

func (n *Node[K, V]) find(key K, compare func(a, b K) int) *Node[K, V] {
	current := n
	for {
		cmp := compare(key, current.key)
		switch {
		case cmp == 0:
			return current
		case current.left != nil && cmp < 0:
			current = current.left
		case current.right != nil:
			current = current.right
		default:
			return nil
		}
	}
}

func (n *Node[K, V]) add(node *Node[K, V], compare func(a, b K) int) (*Node[K, V], error) {
	cmp := compare(node.key, n.key)
	switch {
	case cmp < 0:
		if n.left == nil {
			n.left = node
			node.parent = n
		} else {
			newLeft, err := n.left.add(node, compare)
			if err != nil {
				return nil, err
			}
			n.left = newLeft
		}
	case cmp > 0:
		if n.right == nil {
			n.right = node
			node.parent = n
		} else {
			newRight, err := n.right.add(node, compare)
			if err != nil {
				return nil, err
			}
			n.right = newRight
		}
	default:
		return nil, ErrorTreeNodeDuplicate
	}
	return n.rebalance(), nil
}

func (n *Node[K, V]) remove(key K, compare func(a, b K) int) (*Node[K, V], *Node[K, V], error) {
	cmp := compare(key, n.key)
	if cmp == 0 {
		switch {
		case n.left == nil && n.right == nil:
			// Leaf Node. No special behavior needed
			return n, nil, nil
		case n.left == nil:
			// Single child: right
			return n, n.right, nil
		case n.right == nil:
			// Single child: left
			return n, n.left, nil
		default:
			// Two children
			newRight, mostLeft := n.right.popMostLeft()
			mostLeft.parent = n.parent
			mostLeft.left = n.left
			mostLeft.right = newRight
			mostLeft.height = mostLeft.calcHeight()
			return n, mostLeft.rebalance(), nil
		}
	} else if n.left != nil && cmp < 0 {
		removed, replacement, err := n.left.remove(key, compare)
		if err != nil {
			return nil, nil, err
		}
		n.left = replacement
		n.height = n.calcHeight()
		if replacement != nil {
			replacement.parent = n
		}
		return removed, n.rebalance(), nil
	} else if n.right != nil {
		removed, replacement, err := n.right.remove(key, compare)
		if err != nil {
			return nil, nil, err
		}
		n.right = replacement
		n.height = n.calcHeight()
		if replacement != nil {
			replacement.parent = n
		}
		return removed, n.rebalance(), nil
	}
	return nil, nil, ErrorTreeNodeNotFound
}

func (n *Node[K, V]) popMostLeft() (child, mostLeft *Node[K, V]) {
	if n.left == nil {
		// Found left most tree node
		return n.right, n
	}
	newLeft, popped := n.left.popMostLeft()
	if newLeft != nil {
		newLeft.parent = n
	}
	n.left = newLeft
	n.height = n.calcHeight()
	return n, popped
}

func (n *Node[K, V]) iteratePreOrder(f func(v *Node[K, V]) bool) {
	if f(n) {
		return
	}
	if n.left != nil {
		n.left.iteratePreOrder(f)
	}
	if n.right != nil {
		n.right.iteratePreOrder(f)
	}
}

func (n *Node[K, V]) iterateInOrder(f func(v *Node[K, V]) bool) {
	if n.left != nil {
		n.left.iterateInOrder(f)
	}
	if f(n) {
		return
	}
	if n.right != nil {
		n.right.iterateInOrder(f)
	}
}

func (n *Node[K, V]) iteratePostOrder(f func(v *Node[K, V]) bool) {
	if n.left != nil {
		n.left.iteratePostOrder(f)
	}
	if n.right != nil {
		n.right.iteratePostOrder(f)
	}
	if f(n) {
		return
	}
}

func (n *Node[K, V]) rebalance() *Node[K, V] {
	switch n.calcBalanceFactor() {
	case balanceRightHeavy:
		if n.right != nil && n.right.calcBalanceFactor() == balanceLeftHeavy {
			return n.rotateLeftRight()
		}
		return n.rotateLeft()
	case balanceLeftHeavy:
		if n.left != nil && n.left.calcBalanceFactor() == balanceRightHeavy {
			return n.rotateRightLeft()
		}
		return n.rotateRight()
	}
	return n
}

func (n *Node[K, V]) calcBalanceFactor() balanceFactor {
	leftHeight, rightHeight := n.leftHeight(), n.rightHeight()
	if leftHeight-rightHeight > 1 {
		return balanceLeftHeavy
	}
	if rightHeight-leftHeight > 1 {
		return balanceRightHeavy
	}
	return balanceBalanced
}

func (n *Node[K, V]) leftHeight() int {
	if n.left == nil {
		return 0
	}
	return n.left.height
}

func (n *Node[K, V]) rightHeight() int {
	if n.right == nil {
		return 0
	}
	return n.right.height
}

func (n *Node[K, V]) calcHeight() int {
	switch {
	case n.left == nil && n.right == nil:
		return 0
	case n.left == nil:
		return 1 + n.rightHeight()
	case n.right == nil:
		return 1 + n.leftHeight()
	default:
		leftHeight, rightHeight := n.leftHeight(), n.rightHeight()
		maxHeight := leftHeight
		if maxHeight < rightHeight {
			maxHeight = rightHeight
		}
		return 1 + maxHeight
	}
}

func (n *Node[K, V]) rotateLeft() *Node[K, V] {
	prevRoot := n
	newRoot := prevRoot.right
	prevRoot.parent = newRoot
	prevRoot.right = newRoot.left
	if prevRoot.right != nil {
		prevRoot.right.parent = prevRoot
		prevRoot.right.height = prevRoot.right.calcHeight()
	}
	prevRoot.height = prevRoot.calcHeight()
	newRoot.parent = nil
	newRoot.left = prevRoot
	newRoot.height = newRoot.calcHeight()
	return newRoot
}

func (n *Node[K, V]) rotateRight() *Node[K, V] {
	prevRoot := n
	newRoot := prevRoot.left
	prevRoot.parent = newRoot
	prevRoot.left = newRoot.right
	if prevRoot.left != nil {
		prevRoot.left.parent = prevRoot
		prevRoot.left.height = prevRoot.left.calcHeight()
	}
	prevRoot.height = prevRoot.calcHeight()
	newRoot.parent = nil
	newRoot.right = prevRoot
	newRoot.height = newRoot.calcHeight()
	return newRoot
}

func (n *Node[K, V]) rotateLeftRight() *Node[K, V] {
	n.right = n.right.rotateRight()
	return n.rotateLeft()
}

func (n *Node[K, V]) rotateRightLeft() *Node[K, V] {
	n.left = n.left.rotateLeft()
	return n.rotateRight()
}
