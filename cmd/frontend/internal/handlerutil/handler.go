package handlerutil

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"

	"github.com/inconshreveable/log15"

	"github.com/pkg/errors"

	"github.com/neelance/parallel"

	"github.com/sourcegraph/sourcegraph/internal/errcode"
)

// HandlerWithErrorReturn wraps a http.HandlerFunc-like func that also
// returns an error.  If the error is nil, this wrapper is a no-op. If
// the error is non-nil, it attempts to determine the HTTP status code
// equivalent of the returned error (if non-nil) and set that as the
// HTTP status.
//
// Error must never panic. If it has to execute something that may panic
// (for example, call out into an external code), then it must use recover
// to catch potential panics. If Error panics, the panic will propagate upstream.
type HandlerWithErrorReturn struct {
	Handler func(http.ResponseWriter, *http.Request) error       // the underlying handler
	Error   func(http.ResponseWriter, *http.Request, int, error) // called to send an error response (e.g., an error page), it must not panic

	PretendPanic bool
}

func (h HandlerWithErrorReturn) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Handle when h.Handler panics.
	defer func() {
		if e := recover(); e != nil {
			log15.Error("panic in HandlerWithErrorReturn.Handler", "error", e)
			stack := make([]byte, 1024*1024)
			n := runtime.Stack(stack, false)
			stack = stack[:n]
			_, _ = io.WriteString(os.Stderr, "\nstack trace:\n")
			_, _ = os.Stderr.Write(stack)

			err := fmt.Errorf("panic: %v\n\nstack trace:\n%s", e, stack)
			status := http.StatusInternalServerError
			reportError(r, status, err, true)
			h.Error(w, r, status, err) // No need to handle a possible panic in h.Error because it's required not to panic.
		}
	}()

	err := collapseMultipleErrors(h.Handler(w, r))
	if err != nil {
		status := httpErrCode(r, err)
		reportError(r, status, err, false)
		h.Error(w, r, status, err)
	}
}

// httpErrCode maps an error to a status code. If the client canceled the
// request we return the non-standard "499 Client Closed Request" used by
// nginx.
func httpErrCode(r *http.Request, err error) int {
	// If we failed due to ErrCanceled, it may be due to the client closing
	// the connection. If that is the case, return 499. We do not just check
	// if the client closed the connection, in case we failed due to another
	// reason leading to the client closing the connection.
	if errors.Cause(err) == context.Canceled && r.Context().Err() == context.Canceled {
		return 499
	}
	return errcode.HTTP(err)
}

// collapseMultipleErrors returns the first err if err is a
// parallel.Errors list of length 1. Otherwise it returns err
// unchanged. This lets us return the proper HTTP status code for
// single errors.
func collapseMultipleErrors(err error) error {
	if errs, ok := err.(parallel.Errors); ok && len(errs) == 1 {
		return errs[0]
	}
	return err
}
