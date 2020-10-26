package auth

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/mux"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/app/router"
	uirouter "github.com/sourcegraph/sourcegraph/cmd/frontend/internal/app/ui/router"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/conf"
)

// RequireAuthMiddleware is a middleware that requires authentication for all HTTP requests, except
// those allowed by allowAnonymousRequest. It's used when auth.public == false.
//
// It is enabled for all auth providers, but an auth provider may reject or redirect the user to its
// own auth flow before the request reaches here.
//
// 🚨 SECURITY: Any change to this function could introduce security exploits.
var RequireAuthMiddleware = &Middleware{
	API: func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If an anonymous user tries to access an API endpoint that requires authentication,
			// prevent access.
			if !actor.FromContext(r.Context()).IsAuthenticated() && !AllowAnonymousRequest(r) {
				// Report HTTP 401 Unauthorized for API requests.
				code := anonymousStatusCode(r, http.StatusUnauthorized)
				http.Error(w, "Private mode requires authentication.", code)
				return
			}

			// The client is authenticated, or the request is accessible to anonymous clients.
			next.ServeHTTP(w, r)
		})
	},
	App: func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If an anonymous user tries to access an app endpoint that requires authentication,
			// prevent access and redirect them to the login page.
			if !actor.FromContext(r.Context()).IsAuthenticated() && !AllowAnonymousRequest(r) {
				// Redirect 302 Found for web page requests.
				code := anonymousStatusCode(r, http.StatusFound)
				q := url.Values{}
				q.Set("returnTo", r.URL.String())
				http.Redirect(w, r, "/sign-in?"+q.Encode(), code)
				return
			}

			// The client is authenticated, or the request is accessible to anonymous clients.
			next.ServeHTTP(w, r)
		})
	},
}

var (
	// 🚨 SECURITY: These maps define route names that anonymous users can access. They MUST NOT leak any sensitive
	// data or allow unprivileged users to perform undesired actions.
	anonymousAccessibleAPIRoutes = map[string]struct{}{
		router.RobotsTxt:          {},
		router.Favicon:            {},
		router.Logout:             {},
		router.SignUp:             {},
		router.SiteInit:           {},
		router.SignIn:             {},
		router.SignOut:            {},
		router.ResetPasswordInit:  {},
		router.ResetPasswordCode:  {},
		router.CheckUsernameTaken: {},
	}
	anonymousAccessibleUIRoutes = map[string]struct{}{
		uirouter.RouteSignIn:        {},
		uirouter.RouteSignUp:        {},
		uirouter.RoutePasswordReset: {},
	}
	// Some routes return non-standard HTTP responses when a user is not
	// signed in.
	anonymousUIStatusCode = map[string]int{
		// This route lives in the app, but should act like the API since most
		// clients are extensions.
		uirouter.RouteRaw: http.StatusUnauthorized,
	}
)

func matchedRouteName(req *http.Request, router *mux.Router) string {
	var m mux.RouteMatch
	if !router.Match(req, &m) || m.Route == nil {
		return ""
	}
	return m.Route.GetName()
}

// AllowAnonymousRequest reports whether handling of the HTTP request (which is from an anonymous
// user) should proceed. The eventual handler for the request may still perform other authn/authz
// checks.
//
// 🚨 SECURITY: This func MUST return false if handling req would leak any sensitive data or allow unprivileged
// users to perform undesired actions.
func AllowAnonymousRequest(req *http.Request) bool {
	if conf.AuthPublic() {
		return true
	}

	if strings.HasPrefix(req.URL.Path, "/.assets/") {
		return true
	}

	// Permission is checked by github token
	if strings.HasPrefix(req.URL.Path, "/.api/lsif/upload") {
		return true
	}

	// This is just a redirect to a public download
	if strings.HasPrefix(req.URL.Path, "/.api/src-cli") {
		return true
	}

	// Authentication is performed in the webhook handler itself.
	for _, prefix := range []string{
		"/.api/github-webhooks",
		"/.api/gitlab-webhooks",
		"/.api/bitbucket-server-webhooks",
	} {
		if strings.HasPrefix(req.URL.Path, prefix) {
			return true
		}
	}

	// Permission is checked by a shared token
	if strings.HasPrefix(req.URL.Path, "/.internal-code-intel") {
		return true
	}

	apiRouteName := matchedRouteName(req, router.Router())
	if apiRouteName == router.UI {
		// Test against UI router. (Some of its handlers inject private data into the title or meta tags.)
		uiRouteName := matchedRouteName(req, uirouter.Router)
		_, ok := anonymousAccessibleUIRoutes[uiRouteName]
		return ok
	}
	_, ok := anonymousAccessibleAPIRoutes[apiRouteName]
	return ok
}

func anonymousStatusCode(req *http.Request, defaultCode int) int {
	name := matchedRouteName(req, router.Router())
	if name != router.UI {
		return defaultCode
	}

	name = matchedRouteName(req, uirouter.Router)
	if code, ok := anonymousUIStatusCode[name]; ok {
		return code
	}

	return defaultCode
}
