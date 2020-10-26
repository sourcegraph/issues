package handlerutil

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/getsentry/raven-go"
	"github.com/gorilla/mux"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"

	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/internal/version"
)

var ravenClient *raven.Client

func init() {
	if dsn := env.Get("SENTRY_DSN_BACKEND", "", "Sentry/Raven DSN used for tracking of backend errors"); dsn != "" {
		var err error
		ravenClient, err = raven.New(dsn)
		if err != nil {
			log.Fatalf("error initializing Sentry error reporter: %s", err)
		}
		ravenClient.DropHandler = func(pkt *raven.Packet) {
			log.Println("WARNING: dropped error report because buffer is full:", pkt)
		}
		ravenClient.SetRelease(version.Version())
	}
}

// reportError reports an error to Sentry.
func reportError(r *http.Request, status int, err error, panicked bool) {
	if ravenClient == nil {
		return
	}
	if status > 0 && status < 500 {
		// Not a reportable error.
		return
	}

	// Catch panics here to be extra sure we don't disrupt the request handling.
	defer func() {
		if rv := recover(); rv != nil {
			log.Println("WARNING: panic in HTTP handler error reporter: (recovered)", rv)
		}
	}()

	var stacktrace *raven.Stacktrace
	if panicked {
		stacktrace = raven.NewStacktrace(4, 2, []string{"github.com/sourcegraph/"})
	}
	exception := raven.NewException(err, stacktrace)

	// The type of err can quite often be a wrapped type. We want the root
	// cause as the type.
	exception.Type = reflect.TypeOf(errors.Cause(err)).String()

	h := raven.NewHttp(r)
	h.Cookies = "" // Don't send session cookies (which have auth secrets).
	delete(h.Headers, "Cookie")
	delete(h.Headers, "Authorization")

	pkt := raven.NewPacket(err.Error(), exception, h)

	addTag := func(key, val string) {
		pkt.Tags = append(pkt.Tags, raven.Tag{Key: key, Value: val})
	}

	addTag("service", filepath.Base(os.Args[0]))

	// Add appdash span ID.
	if span := opentracing.SpanFromContext(r.Context()); span != nil {
		pkt.Extra["trace"] = trace.SpanURL(span)
	}

	// Add request context tags.
	if actor := actor.FromContext(r.Context()); actor.IsAuthenticated() {
		addTag("Authed", "yes")
		addTag("Authed UID", actor.UIDString())
	} else {
		addTag("Authed", "no")
	}
	if routeVars := mux.Vars(r); len(routeVars) > 0 {
		pkt.Extra["Route vars"] = routeVars
		for k, v := range routeVars {
			if v == "" {
				continue
			}
			addTag("Route "+k, v)

			// Allow filtering by repo owner.
			if k == "Repo" {
				parts := strings.Split(v, "/")
				if len(parts) == 3 {
					addTag("Route Repo Owner", parts[1])
					addTag("Route Repo Name", parts[2])
				}
			}
		}
	}

	// Add error information.
	pkt.Extra["Error value"] = fmt.Sprintf("%+v", err)

	ravenClient.Capture(pkt, nil)
}
