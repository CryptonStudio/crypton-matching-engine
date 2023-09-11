package avl

import (
	"testing"
	"unicode/utf8"
)

type intNode = Node[int, int]

func TestAVLNodeRotateRight(t *testing.T) {
	/*
		    4
		   /
		  2
		 / \
		1   3
	*/
	gotNode1 := &intNode{
		key:   1,
		value: 1,
	}
	gotNode3 := &intNode{
		key:   3,
		value: 3,
	}
	gotNode2 := &intNode{
		key:    2,
		value:  2,
		height: 1,
		left:   gotNode1,
		right:  gotNode3,
	}
	gotNode4 := &intNode{
		key:    4,
		value:  4,
		height: 2,
		left:   gotNode2,
	}
	gotNode1.parent = gotNode2
	gotNode3.parent = gotNode2
	gotNode2.parent = gotNode4
	tree := gotNode4

	/*
		  2
		 / \
		1   4
		   /
		  3
	*/
	wantNode3 := &intNode{
		key:   3,
		value: 3,
	}
	wantNode1 := &intNode{
		key:   1,
		value: 1,
	}
	wantNode4 := &intNode{
		key:    4,
		value:  4,
		height: 1,
		left:   wantNode3,
	}
	wantNode2 := &intNode{
		key:    2,
		value:  2,
		height: 2,
		left:   wantNode1,
		right:  wantNode4,
	}
	wantNode3.parent = wantNode4
	wantNode1.parent = wantNode2
	wantNode4.parent = wantNode2
	want := wantNode2
	got := tree.rotateRight()
	assertAVLNode(t, want, got)
}

func TestAVLNodeRotateRightLeft(t *testing.T) {
	/*
		  3
		 /
		1
		 \
		  2
	*/
	gotNode2 := &intNode{
		key:   2,
		value: 2,
	}
	gotNode1 := &intNode{
		key:    1,
		value:  1,
		height: 1,
		right:  gotNode2,
	}
	gotNode3 := &intNode{
		key:    3,
		value:  3,
		height: 2,
		left:   gotNode1,
	}
	gotNode2.parent = gotNode1
	gotNode1.parent = gotNode3
	tree := gotNode3

	/*
		  2
		 / \
		1   3
	*/
	wantNode1 := &intNode{
		key:   1,
		value: 1,
	}
	wantNode3 := &intNode{
		key:   3,
		value: 3,
	}
	wantNode2 := &intNode{
		key:    2,
		value:  2,
		height: 1,
		left:   wantNode1,
		right:  wantNode3,
	}
	wantNode1.parent = wantNode2
	wantNode3.parent = wantNode2
	want := wantNode2
	got := tree.rotateRightLeft()
	assertAVLNode(t, want, got)
}

func TestAVLNodeRotateLeft(t *testing.T) {
	/*
		1
		 \
		  3
		 / \
		2   4
	*/
	gotNode2 := &intNode{
		key:   2,
		value: 2,
	}
	gotNode4 := &intNode{
		key:   4,
		value: 4,
	}
	gotNode3 := &intNode{
		key:    3,
		value:  3,
		height: 1,
		left:   gotNode2,
		right:  gotNode4,
	}
	gotNode1 := &intNode{
		key:    1,
		value:  1,
		height: 2,
		right:  gotNode3,
	}
	gotNode2.parent = gotNode3
	gotNode4.parent = gotNode3
	gotNode3.parent = gotNode1
	tree := gotNode1

	/*
		  3
		 / \
		1   4
		 \
		  2
	*/
	wantNode2 := &intNode{
		key:   2,
		value: 2,
	}
	wantNode1 := &intNode{
		key:    1,
		value:  1,
		height: 1,
		right:  wantNode2,
	}
	wantNode4 := &intNode{
		key:   4,
		value: 4,
	}
	wantNode3 := &intNode{
		key:    3,
		value:  3,
		height: 2,
		left:   wantNode1,
		right:  wantNode4,
	}
	wantNode2.parent = wantNode1
	wantNode1.parent = wantNode3
	wantNode4.parent = wantNode3
	want := wantNode3
	got := tree.rotateLeft()
	assertAVLNode(t, want, got)
}

func TestAVLNodeRotateLeftRight(t *testing.T) {
	/*
		1
		 \
		  3
		 /
		2
	*/
	gotNode2 := &intNode{
		key:   2,
		value: 2,
	}
	gotNode3 := &intNode{
		key:    3,
		value:  3,
		height: 1,
		left:   gotNode2,
	}
	gotNode1 := &intNode{
		key:    1,
		value:  1,
		height: 2,
		right:  gotNode3,
	}
	gotNode2.parent = gotNode3
	gotNode3.parent = gotNode1
	tree := gotNode1

	/*
		  2
		 / \
		1   3
	*/
	wantNode1 := &intNode{
		key:   1,
		value: 1,
	}
	wantNode3 := &intNode{
		key:   3,
		value: 3,
	}
	wantNode2 := &intNode{
		key:    2,
		value:  2,
		height: 1,
		left:   wantNode1,
		right:  wantNode3,
	}
	wantNode1.parent = wantNode2
	wantNode3.parent = wantNode2
	want := wantNode2
	got := tree.rotateLeftRight()
	assertAVLNode(t, want, got)
}

func FuzzOrderedTree_AddRemove(f *testing.F) {
	testcases := []string{
		"abcdefg",
		"a",
	}
	for _, tc := range testcases {
		f.Add(tc)
	}
	f.Fuzz(func(t *testing.T, str string) {
		tree := NewOrderedTree[rune, rune]()
		t.Logf("using runes: %q", str)
		strLen := utf8.RuneCountInString(str)
		for _, r := range str {
			tree.Add(r, r)
			if !tree.Contains(r) {
				t.Errorf("just added, but contains(%q) == false", string(r))
			}
		}
		if tree.Size() != strLen {
			t.Errorf("want len=%d, got len=%d", strLen, tree.Size())
		}
		for _, r := range str {
			lenBefore := tree.Size()
			if _, err := tree.Remove(r); err != nil {
				t.Errorf("failed to remove value %d", r)
			}
			if lenBefore-1 != tree.Size() {
				t.Errorf("len did not shrink by 1: want %d, got %d", lenBefore-1, tree.Size())
			}
		}
		if tree.Size() != 0 {
			t.Errorf("want empty, got len=%d", tree.Size())
		}
	})
}

func assertAVLNode[K, V comparable](t *testing.T, want, got *Node[K, V]) {
	assertAVLNodeRec(t, want, got, "root")
}

func assertAVLNodeRec[K, V comparable](t *testing.T, want, got *Node[K, V], path string) {
	if got.key != want.key {
		t.Errorf("want %[1]s.key==%[2]v, got %[1]s.key==%[3]v", path, want.key, got.key)
	}
	if got.value != want.value {
		t.Errorf("want %[1]s.value==%[2]v, got %[1]s.value==%[3]v", path, want.value, got.value)
	}
	if got.height != want.height {
		t.Errorf("want %[1]s.height==%[2]v, got %[1]s.height==%[3]v", path, want.height, got.height)
	}
	if got.parent == nil && want.parent != nil {
		t.Errorf("want %[1]s.parent!=nil, got %[1]s.parent==nil", path)
	} else if got.parent != nil && want.parent == nil {
		t.Errorf("want %[1]s.parent==nil, got %[1]s.parent!=nil", path)
	} else if got.parent != nil && want.parent != nil && got.parent.key != want.parent.key {
		t.Errorf("want %[1]s.parent.key==%[2]v, got %[1]s.parent.key==%[3]v", path, want.parent.key, got.parent.key)
	}
	if got.left == nil && want.left != nil {
		t.Errorf("want %[1]s.left!=nil, got %[1]s.left==nil", path)
	} else if got.left != nil && want.left == nil {
		t.Errorf("want %[1]s.left==nil, got %[1]s.left!=nil", path)
	} else if got.left != nil && want.left != nil {
		assertAVLNodeRec(t, want.left, got.left, path+".left")
	}
	if got.right == nil && want.right != nil {
		t.Errorf("want %[1]s.right!=nil, got %[1]s.right==nil", path)
	} else if got.right != nil && want.right == nil {
		t.Errorf("want %[1]s.right==nil, got %[1]s.right!=nil", path)
	} else if got.right != nil && want.right != nil {
		assertAVLNodeRec(t, want.right, got.right, path+".right")
	}
}
