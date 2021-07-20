package executor

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"

	"github.com/cockroachdb/errors"
	"github.com/gorilla/mux"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	"github.com/sourcegraph/sourcegraph/internal/env"
)

var sharedUsername = env.Get("EXECUTOR_FRONTEND_USERNAME", "", "The username used to securely communicate between the executor service and the internal API provided by this proxy.")
var sharedPassword = env.Get("EXECUTOR_FRONTEND_PASSWORD", "", "The password used to securely communicate between the executor service and the internal API provided by this proxy.")

func newInternalProxyHandler(uploadHandler http.Handler) (func() http.Handler, error) {
	if queueURL == "" {
		factory := func() http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			})
		}

		return factory, nil
	}

	if sharedUsername == "" {
		return nil, errors.Errorf("invalid value for EXECUTOR_FRONTEND_USERNAME: no value supplied")
	}
	if sharedPassword == "" {
		return nil, errors.Errorf("invalid value for EXECUTOR_FRONTEND_PASSWORD: no value supplied")
	}

	host, port, err := net.SplitHostPort(envvar.HTTPAddrInternal)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to parse internal API address %q", envvar.HTTPAddrInternal))
	}

	frontendOrigin, err := url.Parse(fmt.Sprintf("http://%s:%s/.internal/git", host, port))
	if err != nil {
		return nil, errors.Wrap(err, "failed to construct the origin for the internal frontend")
	}

	queueOrigin, err := url.Parse(queueURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to construct the origin for the executor-queue")
	}

	factory := func() http.Handler {
		// 🚨 SECURITY: These routes are secured by checking a token shared between services.
		base := mux.NewRouter().PathPrefix("/.executors/").Subrouter()
		base.StrictSlash(true)

		// Proxy only info/refs and git-upload-pack for gitservice (git clone/fetch)
		base.Path("/git/{rest:.*/(?:info/refs|git-upload-pack)}").Handler(reverseProxy(frontendOrigin))

		// Proxy only the known routes in the executor queue API
		base.Path("/queue/{rest:heartbeat|.*/(?:dequeue|addExecutionLogEntry|markComplete|markErrored|markFailed)}").Handler(reverseProxy(queueOrigin))

		// Upload LSIF indexes without a sudo access token or github tokens
		base.Path("/lsif/upload").Methods("POST").Handler(uploadHandler)

		return basicAuthMiddleware(base)
	}

	return factory, nil
}

// basicAuthMiddleware rejects requests that do not have a basic auth username and password matching
// the expected username and password. This should only be used for internal _services_, not users,
// in which a shared key exchange can be done so safely.
func basicAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			// This header is required to be present with 401 responses in order to prompt the client
			// to retry the request with basic auth credentials. If we do not send this header, the
			// git fetch/clone flow will break against the internal gitservice with a permanent 401.
			w.Header().Add("WWW-Authenticate", `Basic realm="Sourcegraph"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if username != sharedUsername || password != sharedPassword {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// TODO(efritz) - add tracing, metrics
var client = http.DefaultClient

// reverseProxy creates an HTTP handler that will proxy requests to the given target URL. See
// makeProxyRequest for details on how the request URI is constructed.
//
// Note that we do not use httputil.ReverseProxy. We need to ensure that there are no redirect
// requests to unreachable (internal) routes sent back to the client, and the built-in reverse
// proxy does not give sufficient control over redirects.
func reverseProxy(target *url.URL) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r, err := makeProxyRequest(r, target)
		if err != nil {
			log15.Error("Failed to construct proxy request", "err", err)
			http.Error(w, fmt.Sprintf("failed to construct proxy request: %s", err), http.StatusInternalServerError)
			return
		}

		resp, err := client.Do(r)
		if err != nil {
			log15.Error("Failed to perform proxy request", "err", err)
			http.Error(w, fmt.Sprintf("failed to perform proxy request: %s", err), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		copyHeader(w.Header(), resp.Header)
		w.WriteHeader(resp.StatusCode)

		if _, err := io.Copy(w, resp.Body); err != nil {
			log15.Error("Failed to write payload to client", "err", err)
		}
	})
}

// getRest returns the "rest" segment of the request's URL. This is a function variable so
// we can swap it out easily during testing. The gorilla/mux does have a testing function to
// set variables on a request context, but the context gets lost somewhere between construction
// of the request and the default client's handling of the request.
var getRest = func(r *http.Request) string {
	return mux.Vars(r)["rest"]
}

// makeProxyRequest returns a new HTTP request object with the given HTTP request's headers
// and body. The resulting request's URL is a URL constructed with the given target URL as
// a base, and the text matching the "{rest:.*}" portion of the given request's route as the
// additional path segment. The resulting request's GetBody field is populated so that a
// 307 Temporary Redirect can be followed when making POST requests. This is necessary to
// properly proxy git service operations without being redirected to an inaccessible API.
func makeProxyRequest(r *http.Request, target *url.URL) (*http.Request, error) {
	content, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	u := r.URL
	u.Scheme = target.Scheme
	u.Host = target.Host
	u.Path = path.Join("/", target.Path, getRest(r))

	req, err := http.NewRequest(r.Method, u.String(), bytes.NewReader(content))
	if err != nil {
		return nil, err
	}

	copyHeader(req.Header, r.Header)
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(content)), nil }
	return req, nil
}

// copyHeader adds the header values from src to dst.
func copyHeader(dst, src http.Header) {
	for k, vs := range src {
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
}
