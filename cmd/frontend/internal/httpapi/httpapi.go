package httpapi

import (
	"log"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/graph-gophers/graphql-go"
	"github.com/inconshreveable/log15"
	"github.com/opentracing/opentracing-go"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/enterprise"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/app/updatecheck"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/handlerutil"
	apirouter "github.com/sourcegraph/sourcegraph/cmd/frontend/internal/httpapi/router"
	frontendsearch "github.com/sourcegraph/sourcegraph/cmd/frontend/internal/search"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/registry"
	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/search"
	"github.com/sourcegraph/sourcegraph/internal/trace"
)

// NewHandler returns a new API handler that uses the provided API
// router, which must have been created by httpapi/router.New, or
// creates a new one if nil.
//
// 🚨 SECURITY: The caller MUST wrap the returned handler in middleware that checks authentication
// and sets the actor in the request context.
func NewHandler(m *mux.Router, schema *graphql.Schema, githubWebhook, gitlabWebhook, bitbucketServerWebhook http.Handler, newCodeIntelUploadHandler enterprise.NewCodeIntelUploadHandler) http.Handler {
	if m == nil {
		m = apirouter.New(nil)
	}
	m.StrictSlash(true)

	handler := jsonMiddleware(&errorHandler{
		// Only display error message to admins when in debug mode, since it
		// may contain sensitive info (like API keys in net/http error
		// messages).
		WriteErrBody: env.InsecureDev,
	})

	// Set handlers for the installed routes.
	m.Get(apirouter.RepoShield).Handler(trace.TraceRoute(handler(serveRepoShield)))

	m.Get(apirouter.RepoRefresh).Handler(trace.TraceRoute(handler(serveRepoRefresh)))

	m.Get(apirouter.GitHubWebhooks).Handler(trace.TraceRoute(githubWebhook))
	m.Get(apirouter.GitLabWebhooks).Handler(trace.TraceRoute(gitlabWebhook))
	m.Get(apirouter.BitbucketServerWebhooks).Handler(trace.TraceRoute(bitbucketServerWebhook))
	m.Get(apirouter.LSIFUpload).Handler(trace.TraceRoute(newCodeIntelUploadHandler(false)))

	if envvar.SourcegraphDotComMode() {
		m.Path("/updates").Methods("GET", "POST").Name("updatecheck").Handler(trace.TraceRoute(http.HandlerFunc(updatecheck.Handler)))
	}

	m.Get(apirouter.GraphQL).Handler(trace.TraceRoute(handler(serveGraphQL(schema))))

	m.Get(apirouter.SearchStream).Handler(trace.TraceRoute(http.HandlerFunc(frontendsearch.ServeStream)))

	// Return the minimum src-cli version that's compatible with this instance
	m.Get(apirouter.SrcCliVersion).Handler(trace.TraceRoute(handler(srcCliVersionServe)))
	m.Get(apirouter.SrcCliDownload).Handler(trace.TraceRoute(handler(srcCliDownloadServe)))

	m.Get(apirouter.Registry).Handler(trace.TraceRoute(handler(registry.HandleRegistry)))

	m.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("API no route: %s %s from %s", r.Method, r.URL, r.Referer())
		http.Error(w, "no route", http.StatusNotFound)
	})

	return m
}

// NewInternalHandler returns a new API handler for internal endpoints that uses
// the provided API router, which must have been created by httpapi/router.NewInternal.
//
// 🚨 SECURITY: This handler should not be served on a publicly exposed port. 🚨
// This handler is not guaranteed to provide the same authorization checks as
// public API handlers.
func NewInternalHandler(m *mux.Router, schema *graphql.Schema, newCodeIntelUploadHandler enterprise.NewCodeIntelUploadHandler) http.Handler {
	if m == nil {
		m = apirouter.New(nil)
	}
	m.StrictSlash(true)

	handler := jsonMiddleware(&errorHandler{
		// Internal endpoints can expose sensitive errors
		WriteErrBody: true,
	})

	m.Get(apirouter.ExternalServiceConfigs).Handler(trace.TraceRoute(handler(serveExternalServiceConfigs)))
	m.Get(apirouter.ExternalServicesList).Handler(trace.TraceRoute(handler(serveExternalServicesList)))
	m.Get(apirouter.PhabricatorRepoCreate).Handler(trace.TraceRoute(handler(servePhabricatorRepoCreate)))
	reposList := &reposListServer{
		SourcegraphDotComMode: envvar.SourcegraphDotComMode(),
		Repos:                 backend.Repos,
		Indexers:              search.Indexers(),
	}
	m.Get(apirouter.ReposIndex).Handler(trace.TraceRoute(handler(reposList.serveIndex)))
	m.Get(apirouter.ReposListEnabled).Handler(trace.TraceRoute(handler(serveReposListEnabled)))
	m.Get(apirouter.ReposGetByName).Handler(trace.TraceRoute(handler(serveReposGetByName)))
	m.Get(apirouter.SettingsGetForSubject).Handler(trace.TraceRoute(handler(serveSettingsGetForSubject)))
	m.Get(apirouter.SavedQueriesListAll).Handler(trace.TraceRoute(handler(serveSavedQueriesListAll)))
	m.Get(apirouter.SavedQueriesGetInfo).Handler(trace.TraceRoute(handler(serveSavedQueriesGetInfo)))
	m.Get(apirouter.SavedQueriesSetInfo).Handler(trace.TraceRoute(handler(serveSavedQueriesSetInfo)))
	m.Get(apirouter.SavedQueriesDeleteInfo).Handler(trace.TraceRoute(handler(serveSavedQueriesDeleteInfo)))
	m.Get(apirouter.OrgsListUsers).Handler(trace.TraceRoute(handler(serveOrgsListUsers)))
	m.Get(apirouter.OrgsGetByName).Handler(trace.TraceRoute(handler(serveOrgsGetByName)))
	m.Get(apirouter.UsersGetByUsername).Handler(trace.TraceRoute(handler(serveUsersGetByUsername)))
	m.Get(apirouter.UserEmailsGetEmail).Handler(trace.TraceRoute(handler(serveUserEmailsGetEmail)))
	m.Get(apirouter.ExternalURL).Handler(trace.TraceRoute(handler(serveExternalURL)))
	m.Get(apirouter.CanSendEmail).Handler(trace.TraceRoute(handler(serveCanSendEmail)))
	m.Get(apirouter.SendEmail).Handler(trace.TraceRoute(handler(serveSendEmail)))
	m.Get(apirouter.GitExec).Handler(trace.TraceRoute(handler(serveGitExec)))
	m.Get(apirouter.GitResolveRevision).Handler(trace.TraceRoute(handler(serveGitResolveRevision)))
	m.Get(apirouter.GitTar).Handler(trace.TraceRoute(handler(serveGitTar)))
	gitService := &gitServiceHandler{
		Gitserver: gitserver.DefaultClient,
	}
	m.Get(apirouter.GitInfoRefs).Handler(trace.TraceRoute(http.HandlerFunc(gitService.serveInfoRefs)))
	m.Get(apirouter.GitUploadPack).Handler(trace.TraceRoute(http.HandlerFunc(gitService.serveGitUploadPack)))
	m.Get(apirouter.Telemetry).Handler(trace.TraceRoute(telemetryHandler))
	m.Get(apirouter.GraphQL).Handler(trace.TraceRoute(handler(serveGraphQL(schema))))
	m.Get(apirouter.Configuration).Handler(trace.TraceRoute(handler(serveConfiguration)))
	m.Get(apirouter.SearchConfiguration).Handler(trace.TraceRoute(handler(serveSearchConfiguration)))
	m.Path("/ping").Methods("GET").Name("ping").HandlerFunc(handlePing)

	m.Get(apirouter.LSIFUpload).Handler(trace.TraceRoute(newCodeIntelUploadHandler(true)))

	m.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("API no route: %s %s from %s", r.Method, r.URL, r.Referer())
		http.Error(w, "no route", http.StatusNotFound)
	})

	return m
}

var schemaDecoder = schema.NewDecoder()

func init() {
	schemaDecoder.IgnoreUnknownKeys(true)

	// Register a converter for unix timestamp strings -> time.Time values
	// (needed for Appdash PageLoadEvent type).
	schemaDecoder.RegisterConverter(time.Time{}, func(s string) reflect.Value {
		ms, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return reflect.ValueOf(err)
		}
		return reflect.ValueOf(time.Unix(0, ms*int64(time.Millisecond)))
	})
}

type errorHandler struct {
	WriteErrBody bool
}

func (h *errorHandler) Handle(w http.ResponseWriter, r *http.Request, status int, err error) {
	trace.SetRequestErrorCause(r.Context(), err)

	// Handle custom errors
	if ee, ok := err.(*handlerutil.URLMovedError); ok {
		err := handlerutil.RedirectToNewRepoName(w, r, ee.NewRepo)
		if err != nil {
			log15.Error("error redirecting to new URI", "err", err, "new_url", ee.NewRepo)
		}
		return
	}

	// Never cache error responses.
	w.Header().Set("cache-control", "no-cache, max-age=0")

	errBody := err.Error()

	var displayErrBody string
	if h.WriteErrBody {
		displayErrBody = string(errBody)
	}
	http.Error(w, displayErrBody, status)
	traceSpan := opentracing.SpanFromContext(r.Context())
	var spanURL string
	if traceSpan != nil {
		spanURL = trace.SpanURL(traceSpan)
	}
	if status < 200 || status >= 500 {
		log15.Error("API HTTP handler error response", "method", r.Method, "request_uri", r.URL.RequestURI(), "status_code", status, "error", err, "trace", spanURL)
	}
}

func jsonMiddleware(errorHandler *errorHandler) func(func(http.ResponseWriter, *http.Request) error) http.Handler {
	return func(h func(http.ResponseWriter, *http.Request) error) http.Handler {
		return handlerutil.HandlerWithErrorReturn{
			Handler: func(w http.ResponseWriter, r *http.Request) error {
				w.Header().Set("Content-Type", "application/json")
				return h(w, r)
			},
			Error: errorHandler.Handle,
		}
	}
}
