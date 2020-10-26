// Package errorutil exports a HTTP Middleware for HTTP handlers which return
// errors.
package errorutil

import (
	"fmt"
	"net/http"

	"github.com/inconshreveable/log15"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/handlerutil"
	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/internal/trace"
)

// Handler is a wrapper func for app HTTP handlers that enables app
// error pages.
func Handler(h func(http.ResponseWriter, *http.Request) error) http.Handler {
	return handlerutil.HandlerWithErrorReturn{
		Handler: h,
		Error: func(w http.ResponseWriter, req *http.Request, status int, err error) {
			if status < 200 || status >= 400 {
				var spanURL string
				if span := opentracing.SpanFromContext(req.Context()); span != nil {
					ext.Error.Set(span, true)
					span.SetTag("err", err)
					spanURL = trace.SpanURL(span)
				}
				log15.Error("App HTTP handler error response", "method", req.Method, "request_uri", req.URL.RequestURI(), "status_code", status, "error", err, "trace", spanURL)
			}

			trace.SetRequestErrorCause(req.Context(), err)

			w.Header().Set("cache-control", "no-cache")

			var body string
			if env.InsecureDev {
				body = fmt.Sprintf("Error: HTTP %d %s\n\nError: %s", status, http.StatusText(status), err.Error())
			} else {
				body = fmt.Sprintf("Error: HTTP %d: %s", status, http.StatusText(status))
			}
			http.Error(w, body, status)
		},
	}
}
