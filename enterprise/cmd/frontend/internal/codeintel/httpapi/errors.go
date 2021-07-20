package httpapi

import "github.com/cockroachdb/errors"

type ClientError struct {
	err error
}

func (e *ClientError) Error() string {
	return e.err.Error()
}

func clientError(message string, vals ...interface{}) error {
	return &ClientError{err: errors.Errorf(message, vals...)}
}
