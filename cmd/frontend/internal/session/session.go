// Package session implements a redis backed user sessions HTTP middleware.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/textproto"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/redispool"

	"github.com/inconshreveable/log15"

	"github.com/boj/redistore"
	"github.com/gorilla/sessions"
)

var sessionStore sessions.Store
var sessionCookieKey = env.Get("SRC_SESSION_COOKIE_KEY", "", "secret key used for securing the session cookies")

// defaultExpiryPeriod is the default session expiry period (if none is specified explicitly): 90 days.
const defaultExpiryPeriod = 90 * 24 * time.Hour

// cookieName is the name of the HTTP cookie that stores the session ID.
const cookieName = "sgs"

func init() {
	conf.ContributeValidator(func(c conf.Unified) (problems conf.Problems) {
		if c.AuthSessionExpiry == "" {
			return nil
		}

		d, err := time.ParseDuration(c.AuthSessionExpiry)
		if err != nil {
			return conf.NewSiteProblems("auth.sessionExpiry does not conform to the Go time.Duration format (https://golang.org/pkg/time/#ParseDuration). The default of 90 days will be used.")
		}
		if d == 0 {
			return conf.NewSiteProblems("auth.sessionExpiry should be greater than zero. The default of 90 days will be used.")
		}
		return nil
	})
}

// sessionInfo is the information we store in the session. The gorilla/sessions library doesn't appear to
// enforce the maxAge field in its session store implementations, so we include the expiry here.
type sessionInfo struct {
	Actor        *actor.Actor  `json:"actor"`
	LastActive   time.Time     `json:"lastActive"`
	ExpiryPeriod time.Duration `json:"expiryPeriod"`
}

// SetSessionStore sets the backing store used for storing sessions on the server. It should be called exactly once.
func SetSessionStore(s sessions.Store) {
	sessionStore = s
}

// sessionsStore wraps another sessions.Store to dynamically set the values
// of the session.Options.Secure and session.Options.SameSite fields to what
// is returned by the secure closure at invocation time.
type sessionsStore struct {
	sessions.Store
	secure func() bool
}

// Get returns a cached session, setting the secure cookie option dynamically.
func (st *sessionsStore) Get(r *http.Request, name string) (s *sessions.Session, err error) {
	defer st.setSecureOptions(s)
	return st.Store.Get(r, name)
}

// New creates and returns a new session with the secure cookie setting option set
// dynamically.
func (st *sessionsStore) New(r *http.Request, name string) (s *sessions.Session, err error) {
	defer st.setSecureOptions(s)
	return st.Store.New(r, name)
}

func (st *sessionsStore) setSecureOptions(s *sessions.Session) {
	if s != nil {
		if s.Options == nil {
			s.Options = new(sessions.Options)
		}

		setSessionSecureOptions(s.Options, st.secure())
	}
}

// NewRedisStore creates a new session store backed by Redis.
func NewRedisStore(secureCookie func() bool) sessions.Store {
	rstore, err := redistore.NewRediStoreWithPool(redispool.Store, []byte(sessionCookieKey))
	if err != nil {
		waitForRedis(rstore)
	}

	rstore.Options.Path = "/"
	rstore.Options.HttpOnly = true

	setSessionSecureOptions(rstore.Options, secureCookie())
	return &sessionsStore{
		Store:  rstore,
		secure: secureCookie,
	}
}

// setSessionSecureOptions set the values of the session.Options.Secure
// and session.Options.SameSite fields depending on the value of the
// secure field.
func setSessionSecureOptions(opts *sessions.Options, secure bool) {
	// if Sourcegraph is running via:
	//  * HTTP:  set "SameSite=Lax" in session cookie - users can sign in, but won't be able to use the
	// 			 browser extension. Note that users will be able to use the browser extension once they
	// 			 configure their instance to use HTTPS.
	// 	* HTTPS: set "SameSite=None" in session cookie - users can sign in, and will be able to use the
	// 			 browser extension.
	//
	// See https://github.com/sourcegraph/sourcegraph/issues/6167 for more information.
	opts.SameSite = http.SameSiteLaxMode
	if secure {
		opts.SameSite = http.SameSiteNoneMode
	}

	opts.Secure = secure
}

// Ping attempts to contact Redis and returns a non-nil error upon failure. It is intended to be
// used by health checks.
func Ping() error {
	if sessionStore == nil {
		return errors.New("redis session store is not available")
	}
	rstore, ok := sessionStore.(*redistore.RediStore)
	if !ok {
		// Only try to ping Redis session stores. If we add other types of session stores, add ways
		// to ping them here.
		return nil
	}
	return ping(rstore)
}

func ping(s *redistore.RediStore) error {
	conn := s.Pool.Get()
	defer conn.Close()
	data, err := conn.Do("PING")
	if err != nil {
		return err
	}
	if data != "PONG" {
		return errors.New("no pong received")
	}
	return nil
}

// waitForRedis waits up to a certain timeout for Redis to become reachable, to reduce the
// likelihood of the HTTP handlers starting to serve requests while Redis (and therefore session
// data) is still unavailable. After the timeout has elapsed, if Redis is still unreachable, it
// continues anyway (because that's probably better than the site not coming up at all).
func waitForRedis(s *redistore.RediStore) {
	const timeout = 5 * time.Second
	deadline := time.Now().Add(timeout)
	var err error
	for {
		time.Sleep(150 * time.Millisecond)
		err = ping(s)
		if err == nil {
			return
		}
		if time.Now().After(deadline) {
			log15.Warn("Redis (used for session store) failed to become reachable. Will continue trying to establish connection in background.", "timeout", timeout, "error", err)
			return
		}
	}
}

// SetData sets the session data at the key. The session data is a map of keys to values. If no
// session exists, a new session is created.
//
// The value is JSON-encoded before being stored.
func SetData(w http.ResponseWriter, r *http.Request, key string, value interface{}) error {
	session, err := sessionStore.Get(r, cookieName)
	if err != nil {
		return errors.WithMessage(err, "getting session")
	}
	data, err := json.Marshal(value)
	if err != nil {
		return errors.WithMessage(err, fmt.Sprintf("encoding JSON session data for %q", key))
	}
	session.Values[key] = data
	if err := session.Save(r, w); err != nil {
		return errors.WithMessage(err, "saving session")
	}
	return nil
}

// GetData reads the session data at the key into the data structure addressed by value (which must
// be a pointer).
//
// The value is JSON-decoded from the raw bytes stored by the call to SetData.
func GetData(r *http.Request, key string, value interface{}) error {
	session, err := sessionStore.Get(r, cookieName)
	if err != nil {
		return errors.WithMessage(err, "getting session")
	}
	if data, ok := session.Values[key]; ok {
		if err := json.Unmarshal(data.([]byte), value); err != nil {
			return errors.WithMessage(err, fmt.Sprintf("decoding JSON session data for %q", key))
		}
	}
	return nil
}

// SetActor sets the actor in the session, or removes it if actor == nil. If no session exists, a
// new session is created.
//
// If expiryPeriod is 0, the default expiry period is used.
func SetActor(w http.ResponseWriter, r *http.Request, actor *actor.Actor, expiryPeriod time.Duration) error {
	var value *sessionInfo
	if actor != nil {
		if expiryPeriod == 0 {
			if cfgExpiry, err := time.ParseDuration(conf.Get().AuthSessionExpiry); err == nil {
				expiryPeriod = cfgExpiry
			} else { // if there is no valid session duration, fall back to the default one
				expiryPeriod = defaultExpiryPeriod
			}
		}
		value = &sessionInfo{Actor: actor, ExpiryPeriod: expiryPeriod, LastActive: time.Now()}
	}
	return SetData(w, r, "actor", value)
}

func hasSessionCookie(r *http.Request) bool {
	c, _ := r.Cookie(cookieName)
	return c != nil
}

// deleteSession deletes the current session. If an error occurs, it returns the error but does not
// write an HTTP error response.
//
// It should only be used when there is an unrecoverable, permanent error in the session data. To
// sign out the current user, use SetActor(r, nil).
func deleteSession(w http.ResponseWriter, r *http.Request) error {
	if !hasSessionCookie(r) {
		return nil // nothing to do
	}

	session, err := sessionStore.Get(r, cookieName)
	session.Options.MaxAge = -1 // expire immediately
	if err == nil {
		err = session.Save(r, w)
	}
	if err != nil && hasSessionCookie(r) {
		// Failsafe: delete the client's cookie even if the session store is unavailable.
		http.SetCookie(w, sessions.NewCookie(session.Name(), "", session.Options))
	}
	return errors.WithMessage(err, "deleting session")
}

// InvalidateSessionsCurrentUser invalidates all sessions for the current user
// If an error occurs, return the error
func InvalidateSessionCurrentUser(r *http.Request) error {
	a := actor.FromContext(r.Context())
	return db.Users.InvalidateSessionsByID(r.Context(), a.UID)
}

// InvalidateSessionsByID invalidates all sessions for a user
// If an error occurs, it returns the error
func InvalidateSessionsByID(ctx context.Context, id int32) error {
	// Get the user from the request context
	return db.Users.InvalidateSessionsByID(ctx, id)
}

// CookieMiddleware is an http.Handler middleware that authenticates
// future HTTP request via cookie.
func CookieMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Cookie")
		next.ServeHTTP(w, r.WithContext(authenticateByCookie(r, w)))
	})
}

// CookieMiddlewareWithCSRFSafety is a middleware that authenticates HTTP requests using the
// provided cookie (if any), *only if* the request is a non-simple CORS request (see
// https://www.w3.org/TR/cors/#cross-origin-request-with-preflight-0). This relies on the client's
// CORS checks to guarantee that one of the following is true, thereby protecting against CSRF
// attacks:
//
// - The request originates from the same origin. -OR-
//
// - The request is cross-origin but passed the CORS preflight check (because otherwise the
//   preflight OPTIONS response from secureHeadersMiddleware would have caused the browser to refuse
//   to send this HTTP request).
//
// To determine if it's a non-simple CORS request, it checks for the presence of either
// "Content-Type: application/json; charset=utf-8" or a non-empty HTTP request header whose name is
// given in corsAllowHeader.
//
// If the request is a simple CORS request, or if neither of these is true, then the cookie is not
// used to authenticate the request. The request is still allowed to proceed (but will be
// unauthenticated unless some other authentication is provided, such as an access token).
func CookieMiddlewareWithCSRFSafety(next http.Handler, corsAllowHeader string, isTrustedOrigin func(*http.Request) bool) http.Handler {
	corsAllowHeader = textproto.CanonicalMIMEHeaderKey(corsAllowHeader)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Cookie, Authorization, "+corsAllowHeader)

		_, isTrusted := r.Header[corsAllowHeader]
		if !isTrusted {
			isTrusted = isTrustedOrigin(r)
		}
		if !isTrusted {
			contentType := r.Header.Get("Content-Type")
			isTrusted = contentType == "application/json" || contentType == "application/json; charset=utf-8"
		}
		if isTrusted {
			r = r.WithContext(authenticateByCookie(r, w))
		}

		next.ServeHTTP(w, r)
	})
}

func authenticateByCookie(r *http.Request, w http.ResponseWriter) context.Context {
	// If the request is already authenticated from a cookie (and not a token), then do not clobber the request's existing
	// authenticated actor with the actor (if any) derived from the session cookie.
	if a := actor.FromContext(r.Context()); a.IsAuthenticated() && a.FromSessionCookie {
		if hasSessionCookie(r) {
			// Delete the session cookie to avoid confusion. (This occurs most often when switching
			// the auth provider to http-header; in that case, we want to rely on the http-header
			// auth provider for auth, not the user's old session.
			_ = deleteSession(w, r)
		}
		return r.Context() // unchanged
	}

	var info *sessionInfo
	if err := GetData(r, "actor", &info); err != nil {
		if !strings.Contains(err.Error(), "illegal base64 data at input byte 36") {
			// Skip log if the error message indicates the cookie value was a JWT (which almost
			// certainly means that the cookie was a pre-2.8 SAML cookie, so this error will only
			// occur once and the user will be automatically redirected to the SAML auth flow).
			log15.Warn("Error reading session actor. The session cookie was invalid and will be cleared. This error can be safely ignored unless it persists.", "err", err)
		}
		_ = deleteSession(w, r) // clear the bad value
		return r.Context()
	}
	if info != nil {
		// Check expiry
		if info.LastActive.Add(info.ExpiryPeriod).Before(time.Now()) {
			_ = deleteSession(w, r) // clear the bad value
			return actor.WithActor(r.Context(), &actor.Actor{})
		}

		// Check that user still exists.
		usr, err := db.Users.GetByID(r.Context(), info.Actor.UID)
		if err != nil {
			if errcode.IsNotFound(err) {
				_ = deleteSession(w, r) // clear the bad value
			} else {
				// Don't delete session, since the error might be an ephemeral DB error, and we don't
				// want that to cause all active users to be signed out.
				log15.Error("Error looking up user for session.", "uid", info.Actor.UID, "error", err)
			}
			return r.Context() // not authenticated
		}

		// Check that the session is still valid
		if info.LastActive.Before(usr.InvalidatedSessionsAt) {
			_ = deleteSession(w, r) // Delete the now invalid session
			return r.Context()
		}

		// Renew session
		if time.Since(info.LastActive) > 5*time.Minute {
			info.LastActive = time.Now()
			if err := SetData(w, r, "actor", info); err != nil {
				log15.Error("error renewing session", "error", err)
				return r.Context()
			}
		}

		info.Actor.FromSessionCookie = true
		return actor.WithActor(r.Context(), info.Actor)
	}

	return r.Context()
}
