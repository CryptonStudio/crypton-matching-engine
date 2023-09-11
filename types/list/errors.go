package list

import (
	"errors"
)

var (
	ErrorListElementIsNil          = errors.New("list element is nil")
	ErrorListElementIsNotInTheList = errors.New("list element is not in the list")
)
