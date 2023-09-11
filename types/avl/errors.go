package avl

import (
	"errors"
)

var (
	ErrorTreeNodeDuplicate = errors.New("tree node is duplicated")
	ErrorTreeNodeNotFound  = errors.New("tree node is not found")
)
