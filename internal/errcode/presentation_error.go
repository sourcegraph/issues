package errcode

import "github.com/cockroachdb/errors"

// A PresentationError is an error with a message (returned by the PresentationError method) that is
// suitable for presentation to the user.
type PresentationError interface {
	error

	// PresentationError returns the message suitable for presentation to the user. The message
	// should be written in full sentences and must not contain any information that the user is not
	// authorized to see.
	PresentationError() string
}

// WithPresentationMessage annotates err with a new message suitable for presentation to the
// user. If err is nil, WithPresentationMessage returns nil. Otherwise, the return value implements
// PresentationError.
//
// The message should be written in full sentences and must not contain any information that the
// user is not authorized to see.
func WithPresentationMessage(err error, message string) error {
	if err == nil {
		return nil
	}
	return &presentationError{cause: err, msg: message}
}

// NewPresentationError returns a new error with a message suitable for presentation to the user.
// The message should be written in full sentences and must not contain any information that the
// user is not authorized to see.
//
// If there is an underlying error associated with this message, use WithPresentationMessage
// instead.
func NewPresentationError(message string) error {
	return &presentationError{cause: nil, msg: message}
}

// presentationError implements PresentationError.
type presentationError struct {
	cause error
	msg   string
}

func (e *presentationError) Error() string {
	if e.cause != nil {
		return e.cause.Error()
	}
	return e.msg
}

func (e *presentationError) PresentationError() string { return e.msg }

// PresentationMessage returns the message, if any, suitable for presentation to the user for err or
// one of its causes. An error provides a presentation message by implementing the PresentationError
// interface (e.g., by using WithPresentationMessage). If no presentation message exists for err,
// the empty string is returned.
func PresentationMessage(err error) string {
	var e PresentationError
	if errors.As(err, &e) {
		return e.PresentationError()
	}

	return ""
}
