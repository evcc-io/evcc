package api

import (
	"errors"
	"fmt"
)

// ErrNotAvailable indicates that a feature is not available
var ErrNotAvailable = errors.New("not available")

type Error struct {
	code int
	error
}

func HttpError(code int, msg string) Error {
	return Error{
		code:  code,
		error: errors.New(msg),
	}
}

func HttpErrorf(code int, msg string, a ...interface{}) Error {
	return Error{
		code:  code,
		error: fmt.Errorf(msg, a...),
	}
}

func (e Error) Code() int {
	return e.code
}
