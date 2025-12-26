package morm

import (
	"errors"
	"fmt"
)

var ErrValIsNotExpectedType = errors.New("val is not of the expected type")

// err_wrap wraps an error with a message
func err_wrap(e error, msg string) error {
	e = fmt.Errorf("%s => %s", e.Error(), msg)
	return e
}
