package handlerutil

import (
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/gorilla/csrf"
)

// CSRFMiddleware is HTTP middleware that helps prevent cross-site request forgery. To make your
// forms compliant you will have to submit the token via the X-Csrf-Token header, which is made
// available in the client-side context.
func CSRFMiddleware(next http.Handler, isSecure func() bool) http.Handler {
	type handler struct {
		secure bool
		http.Handler
	}

	newHandler := func(secure bool) handler {

		// if Sourcegraph is running via:
		//  * HTTP:  set "SameSite=Lax" in csrf cookie - users can sign in, but won't be able to use the
		// 			 browser extension. Note that users will be able to use the browser extension once they
		// 			 configure their instance to use HTTPS.
		// 	* HTTPS: set "SameSite=None" in csrf cookie - users can sign in, and will be able to use the
		// 			 browser extension.
		//
		// See https://github.com/sourcegraph/sourcegraph/issues/6167 for more information.
		var sameSite = csrf.SameSite(csrf.SameSiteLaxMode)
		if secure {
			sameSite = csrf.SameSite(csrf.SameSiteNoneMode)
		}

		return handler{secure, csrf.Protect(
			[]byte("e953612ddddcdd5ec60d74e07d40218c"),
			// We do not use the name csrf_token since it is a common name. This
			// leads to conflicts between apps on localhost. See
			// https://github.com/sourcegraph/sourcegraph/issues/65
			csrf.CookieName("sg_csrf_token"),
			csrf.Path("/"),
			csrf.Secure(secure),
			sameSite,
		)(next)}
	}

	var v atomic.Value
	var mu sync.Mutex

	v.Store(newHandler(isSecure()))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h, secure := v.Load().(handler), isSecure()
		if secure != h.secure {
			mu.Lock()
			// Check if other go-routines didn't get there first.
			if h = v.Load().(handler); h.secure != secure {
				h = newHandler(secure)
				v.Store(h)
			}
			mu.Unlock()
		}
		h.ServeHTTP(w, r)
	})
}
