package args

import (
	"errors"
	"fmt"
)

type errUnexpectedArg struct {
	params []*param
	arg    string
}

func (e errUnexpectedArg) Error() string {
	return fmt.Sprintf("unexpected argument: %q", e.arg)
}

func (e errUnexpectedArg) Choices() []*param {
	return e.params
}

var ErrHelped = errors.New("help flagged")
