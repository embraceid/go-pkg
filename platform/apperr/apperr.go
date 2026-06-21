package apperr

import (
	"errors"
	"fmt"
)

// Error is a typed application error that always carries a domain error code.
type Error struct {
	code  int
	cause error
}

// New creates an error with the given domain code and no wrapped cause.
func New(code int) *Error {
	return &Error{code: code}
}

// Wrap wraps cause with the given domain code.
func Wrap(code int, cause error) *Error {
	return &Error{code: code, cause: cause}
}

func (e *Error) Code() int     { return e.code }
func (e *Error) Unwrap() error { return e.cause }

// Is reports whether target is an *Error with the same code.
// This allows errors.Is(apperr.New(X), apperr.New(X)) to return true.
func (e *Error) Is(target error) bool {
	var t *Error
	if errors.As(target, &t) {
		return e.code == t.code
	}
	return false
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%d: %s", e.code, e.cause.Error())
	}
	return fmt.Sprintf("%d", e.code)
}

// CodeOf extracts the domain code from the first *Error in the chain.
func CodeOf(err error) (int, bool) {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.code, true
	}
	return 0, false
}

// HasCode reports whether err's chain contains an *Error with the given code.
func HasCode(err error, code int) bool {
	c, ok := CodeOf(err)
	return ok && c == code
}
