// Package errcode maps Go errors to HTTP status codes as well as other useful
// functions for inspecting errors.
package errcode

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/gorilla/schema"

	"github.com/sourcegraph/sourcegraph/internal/vcs"
)

// HTTP returns the most appropriate HTTP status code that describes
// err. It contains a hard-coded list of error types and error values
// (such as mapping store.RepoNotFoundError to NotFound) and
// heuristics (such as mapping os.IsNotExist-satisfying errors to
// NotFound). All other errors are mapped to HTTP 500 Internal Server
// Error.
func HTTP(err error) int {
	if err == nil {
		return http.StatusOK
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return http.StatusRequestTimeout
	}

	if vcs.IsCloneInProgress(err) || strings.Contains(err.Error(), (&vcs.RepoNotExistError{CloneInProgress: true}).Error()) {
		return http.StatusAccepted
	} else if vcs.IsRepoNotExist(err) || strings.Contains(err.Error(), (&vcs.RepoNotExistError{}).Error()) {
		return http.StatusNotFound
	}

	var e interface{ HTTPStatusCode() int }
	if errors.As(err, &e) {
		return e.HTTPStatusCode()
	}

	if errors.HasType(err, schema.ConversionError{}) {
		return http.StatusBadRequest
	}
	if errors.HasType(err, schema.MultiError{}) {
		return http.StatusBadRequest
	}

	if errors.Is(err, os.ErrNotExist) {
		return http.StatusNotFound
	} else if errors.Is(err, os.ErrPermission) {
		return http.StatusForbidden
	} else if IsNotFound(err) {
		return http.StatusNotFound
	} else if IsBadRequest(err) {
		return http.StatusBadRequest
	}

	return http.StatusInternalServerError
}

type HTTPErr struct {
	Status int   // HTTP status code.
	Err    error // Optional reason for the HTTP error.
}

func (err *HTTPErr) Error() string {
	if err.Err != nil {
		return fmt.Sprintf("status %d, reason %s", err.Status, err.Err)
	}
	return fmt.Sprintf("Status %d", err.Status)
}

func (err *HTTPErr) HTTPStatusCode() int { return err.Status }

func IsHTTPErrorCode(err error, statusCode int) bool {
	return HTTP(err) == statusCode
}

// Mock is a convenience error which makes it easy to satisfy the optional
// interfaces errors implement.
type Mock struct {
	// Message is the return value for Error() string
	Message string

	// IsNotFound is the return value for NotFound
	IsNotFound bool
}

func (e *Mock) Error() string {
	return e.Message
}

func (e *Mock) NotFound() bool {
	return e.IsNotFound
}

// IsNotFound will check if err or one of its causes is a not found
// error. Note: This will not check os.IsNotExist, but rather is returned by
// methods like Repo.Get when the repo is not found. It will also *not* map
// HTTPStatusCode into not found.
func IsNotFound(err error) bool {
	var e interface{ NotFound() bool }
	return errors.As(err, &e) && e.NotFound()
}

// IsUnauthorized will check if err or one of its causes is an unauthorized
// error.
func IsUnauthorized(err error) bool {
	var e interface{ Unauthorized() bool }
	return errors.As(err, &e) && e.Unauthorized()
}

// IsForbidden will check if err or one of its causes is a forbidden error.
func IsForbidden(err error) bool {
	var e interface{ Forbidden() bool }
	return errors.As(err, &e) && e.Forbidden()
}

// IsAccountSuspended will check if err or one of its causes was due to the
// account being suspended
func IsAccountSuspended(err error) bool {
	var e interface{ AccountSuspended() bool }
	return errors.As(err, &e) && e.AccountSuspended()
}

// IsBadRequest will check if err or one of its causes is a bad request.
func IsBadRequest(err error) bool {
	var e interface{ BadRequest() bool }
	return errors.As(err, &e) && e.BadRequest()
}

// IsTemporary will check if err or one of its causes is temporary. A
// temporary error can be retried. Many errors in the go stdlib implement the
// temporary interface.
func IsTemporary(err error) bool {
	var e interface{ Temporary() bool }
	return errors.As(err, &e) && e.Temporary()
}

// IsBlocked will check if err or one of its causes is a blocked error.
func IsBlocked(err error) bool {
	var e interface{ Blocked() bool }
	return errors.As(err, &e) && e.Blocked()
}

// IsTimeout will check if err or one of its causes is a timeout. Many errors
// in the go stdlib implement the timeout interface.
func IsTimeout(err error) bool {
	var e interface{ Timeout() bool }
	return errors.As(err, &e) && e.Timeout()
}

// IsNonRetryable will check if err or one of its causes is a error that cannot be retried.
func IsNonRetryable(err error) bool {
	var e interface{ NonRetryable() bool }
	return errors.As(err, &e) && e.NonRetryable()
}

// MakeNonRetryable makes any error non-retryable.
func MakeNonRetryable(err error) error {
	return nonRetryableError{err}
}

type nonRetryableError struct{ error }

func (nonRetryableError) NonRetryable() bool { return true }
