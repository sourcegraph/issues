package session

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
)

func TestSetActorDeleteSession(t *testing.T) {
	cleanup := ResetMockSessionStore(t)
	defer cleanup()

	db.Mocks.Users.GetByID = func(ctx context.Context, id int32) (*types.User, error) {
		return &types.User{ID: id}, nil
	}
	defer func() { db.Mocks = db.MockStores{} }()

	// Start new session
	w := httptest.NewRecorder()
	actr := &actor.Actor{UID: 123, FromSessionCookie: true}
	if err := SetActor(w, httptest.NewRequest("GET", "/", nil), actr, 24*time.Hour); err != nil {
		t.Fatal(err)
	}
	var authCookies []*http.Cookie
	for _, cookie := range w.Result().Cookies() {
		if cookie.Expires.After(time.Now()) || cookie.MaxAge > 0 {
			authCookies = append(authCookies, cookie)
		}
	}

	// Create authed request with session cookie
	authedReq := httptest.NewRequest("GET", "/", nil)
	for _, cookie := range authCookies {
		authedReq.AddCookie(cookie)
	}
	if len(authedReq.Cookies()) != 1 {
		t.Fatal("expected exactly 1 authed cookie")
	}

	// Check that session cookie was created
	setCookie, err := authedReq.Cookie(cookieName)
	if err != nil {
		t.Fatalf("cookie was not created, error: %s", err)
	}
	if setCookie.Path != "" {
		t.Fatalf("expected cookie path to be \"\", was %q", setCookie.Path)
	}
	if setCookie.Value != sessionCookie(authedReq) {
		t.Errorf("sessionCookie value did not match actual session cookie value: %v != %v", setCookie.Value, sessionCookie(authedReq))
	}

	// Check that actor exists in the session
	session, err := sessionStore.Get(authedReq, cookieName)
	if err != nil {
		t.Fatalf("didn't find session: %v", err)
	}
	if session == nil {
		t.Fatal("session was nil")
	}
	authedActor := actor.FromContext(authenticateByCookie(authedReq, httptest.NewRecorder()))
	if !reflect.DeepEqual(actr, authedActor) {
		t.Fatalf("session was not created: %+v != %+v", authedActor, actr)
	}

	// Delete session
	authedReq2 := httptest.NewRequest("GET", "/", nil)
	for _, cookie := range authCookies {
		authedReq2.AddCookie(cookie)
	}
	w = httptest.NewRecorder()
	if err := deleteSession(w, authedReq2); err != nil {
		t.Fatal(err)
	}
	// Check that the session cookie was deleted
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Unexpected status code: %d", resp.StatusCode)
	}
	checkCookieDeleted(t, resp)

	// Check that the actor no longer exists in the session, even when we have the original cookie
	authedReq3 := httptest.NewRequest("GET", "/", nil)
	for _, cookie := range authCookies {
		authedReq3.AddCookie(cookie)
	}
	actor3 := actor.FromContext(authenticateByCookie(authedReq3, httptest.NewRecorder()))
	if !reflect.DeepEqual(actor3, &actor.Actor{}) {
		t.Fatalf("underlying session was not deleted: %+v != %+v", actor3, &actor.Actor{})
	}

	// Check that the cookie is deleted on the client when we call deleteSession even if
	// getting/saving the session failed.
	authedReq4 := httptest.NewRequest("GET", "/", nil)
	for _, cookie := range authCookies {
		authedReq4.AddCookie(cookie)
	}
	w = httptest.NewRecorder()
	if err := deleteSession(w, authedReq2); err == nil {
		t.Fatal("got no error from deleteSession, want error (because we already deleted the session)")
	}
	checkCookieDeleted(t, w.Result())
}

func checkCookieDeleted(t *testing.T, resp *http.Response) {
	t.Helper()

	if len(resp.Cookies()) != 1 {
		t.Fatalf("expected exactly 1 Set-Cookie, got %+v", resp.Cookies())
	}

	deleteCookie := resp.Cookies()[0]
	if deleteCookie.Name != cookieName {
		t.Fatalf("did not delete cookie (cookie name was not %q)", cookieName)
	}
	if deleteCookie.MaxAge >= 0 {
		t.Fatal("did not delete cookie (max-age was not less than 0)")
	}
	if deleteCookie.Expires.After(time.Now()) {
		t.Fatal("did not delete cookie (cookie not expired)")
	}
}

func TestSessionExpiry(t *testing.T) {
	cleanup := ResetMockSessionStore(t)
	defer cleanup()

	db.Mocks.Users.GetByID = func(ctx context.Context, id int32) (*types.User, error) {
		return &types.User{ID: id}, nil
	}
	defer func() { db.Mocks = db.MockStores{} }()

	// Start new session
	w := httptest.NewRecorder()
	actr := &actor.Actor{UID: 123, FromSessionCookie: true}
	if err := SetActor(w, httptest.NewRequest("GET", "/", nil), actr, time.Second); err != nil {
		t.Fatal(err)
	}
	var authCookies []*http.Cookie
	for _, cookie := range w.Result().Cookies() {
		if cookie.Expires.After(time.Now()) || cookie.MaxAge > 0 {
			authCookies = append(authCookies, cookie)
		}
	}

	// Create authed request with session cookie
	authedReq := httptest.NewRequest("GET", "/", nil)
	for _, cookie := range authCookies {
		authedReq.AddCookie(cookie)
	}
	if len(authedReq.Cookies()) != 1 {
		t.Fatal("expected exactly 1 authed cookie")
	}

	if gotActor := actor.FromContext(authenticateByCookie(authedReq, httptest.NewRecorder())); !reflect.DeepEqual(gotActor, actr) {
		t.Errorf("didn't find actor %v != %v", gotActor, actr)
	}
	time.Sleep(1100 * time.Millisecond)
	if gotActor := actor.FromContext(authenticateByCookie(authedReq, httptest.NewRecorder())); !reflect.DeepEqual(gotActor, &actor.Actor{}) {
		t.Errorf("session didn't expire, found actor %+v", gotActor)
	}
}

func TestManualSessionExpiry(t *testing.T) {
	cleanup := ResetMockSessionStore(t)
	defer cleanup()

	user := &types.User{ID: 123, InvalidatedSessionsAt: time.Now()}
	db.Mocks.Users.GetByID = func(ctx context.Context, id int32) (*types.User, error) {
		user.ID = id
		return user, nil
	}
	defer func() { db.Mocks = db.MockStores{} }()

	// Start new session
	w := httptest.NewRecorder()
	actr := &actor.Actor{UID: 123, FromSessionCookie: true}
	if err := SetActor(w, httptest.NewRequest("GET", "/", nil), actr, time.Hour); err != nil {
		t.Fatal(err)
	}
	var authCookies []*http.Cookie
	for _, cookie := range w.Result().Cookies() {
		if cookie.Expires.After(time.Now()) || cookie.MaxAge > 0 {
			authCookies = append(authCookies, cookie)
		}
	}
	user.InvalidatedSessionsAt = time.Now().Add(6 * time.Minute)

	// Create authed request with session cookie
	authedReq := httptest.NewRequest("GET", "/", nil)
	for _, cookie := range authCookies {
		authedReq.AddCookie(cookie)
	}
	if len(authedReq.Cookies()) != 1 {
		t.Fatal("expected exactly 1 authed cookie")
	}

	if gotActor := actor.FromContext(authenticateByCookie(authedReq, httptest.NewRecorder())); reflect.DeepEqual(gotActor, actr) {
		t.Errorf("Actor should have been deleted, got %v", gotActor)
	}
}

func TestCookieMiddleware(t *testing.T) {
	cleanup := ResetMockSessionStore(t)
	defer cleanup()

	actors := []*actor.Actor{{UID: 123, FromSessionCookie: true}, {UID: 456}, {UID: 789}}

	db.Mocks.Users.GetByID = func(ctx context.Context, id int32) (*types.User, error) {
		if id == actors[0].UID {
			return &types.User{ID: id}, nil
		}
		if id == actors[1].UID {
			return nil, &errcode.Mock{IsNotFound: true}
		}
		return nil, errors.New("x") // other error
	}
	defer func() { db.Mocks = db.MockStores{} }()

	// Start new sessions for all actors
	authedReqs := make([]*http.Request, len(actors))
	for i, actr := range actors {
		w := httptest.NewRecorder()
		if err := SetActor(w, httptest.NewRequest("GET", "/", nil), actr, time.Hour); err != nil {
			t.Fatal(err)
		}

		// Test cases for when session exists
		authedReqs[i] = httptest.NewRequest("GET", "/", nil)
		for _, cookie := range w.Result().Cookies() {
			if cookie.Expires.After(time.Now()) || cookie.MaxAge > 0 {
				authedReqs[i].AddCookie(cookie)
			}
		}
	}

	testcases := []struct {
		req      *http.Request
		expActor *actor.Actor
		deleted  bool // whether the session was deleted
	}{
		{
			req:      httptest.NewRequest("GET", "/", nil),
			expActor: &actor.Actor{},
		}, {
			req:      authedReqs[0],
			expActor: actors[0],
		}, {
			req:      authedReqs[1],
			expActor: &actor.Actor{},
			deleted:  true,
		},
		{
			req:      authedReqs[2],
			expActor: &actor.Actor{},
		},
	}
	for _, testcase := range testcases {
		rr := httptest.NewRecorder()
		CookieMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotActor := actor.FromContext(r.Context())
			if !reflect.DeepEqual(testcase.expActor, gotActor) {
				t.Errorf("on authenticated request, got actor %+v, expected %+v", gotActor, testcase.expActor)
			}
		})).ServeHTTP(rr, testcase.req)
		if deleted := strings.Contains(rr.Header().Get("Set-Cookie"), cookieName+"=;"); deleted != testcase.deleted {
			t.Errorf("got deleted %v, want %v", deleted, testcase.deleted)
		}
	}
}

// sessionCookie returns the session cookie from the header of the given request.
func sessionCookie(r *http.Request) string {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return ""
	}
	return c.Value
}

func TestRecoverFromInvalidCookieValue(t *testing.T) {
	cleanup := ResetMockSessionStore(t)
	defer cleanup()

	// An actual cookie value that is an encoded JWT set by our old github.com/crewjam/saml-based
	// SAML impl.
	const signedToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJodHRwOi8vbG9jYWxob3N0OjMwODAvLmF1dGgvc2FtbC9tZXRhZGF0YSIsImV4cCI6MTUzNDk5MTcwNiwiaWF0IjoxNTI3MjE1NzA2LCJuYmYiOjE1MjcyMTU3MDYsInN1YiI6IkctNDU0ZTBlYWEtYjcxOC00ZWUxLTk2NDctYWU5ZDExM2NlOTUzIiwiYXR0ciI6eyJSb2xlIjpbInVtYV9hdXRob3JpemF0aW9uIiwidmlldy1wcm9maWxlIiwiYWRtaW4iLCJtYW5hZ2UtaWRlbnRpdHktcHJvdmlkZXJzIiwiY3JlYXRlLWNsaWVudCIsInZpZXctcmVhbG0iLCJ2aWV3LWV2ZW50cyIsIm1hbmFnZS11c2VycyIsInZpZXctaWRlbnRpdHktcHJvdmlkZXJzIiwidmlldy1jbGllbnRzIiwidmlldy11c2VycyIsIm1hbmFnZS1yZWFsbSIsInF1ZXJ5LWNsaWVudHMiLCJtYW5hZ2UtY2xpZW50cyIsImNyZWF0ZS1yZWFsbSIsIm1hbmFnZS1ldmVudHMiLCJtYW5hZ2UtYXV0aG9yaXphdGlvbiIsInF1ZXJ5LXJlYWxtcyIsInZpZXctYXV0aG9yaXphdGlvbiIsInF1ZXJ5LWdyb3VwcyIsInF1ZXJ5LXVzZXJzIiwiaW1wZXJzb25hdGlvbiIsIm1hbmFnZS1hY2NvdW50IiwibWFuYWdlLWFjY291bnQtbGlua3MiXSwiZW1haWwiOlsiYWxpY2VAZXhhbXBsZS5jb20iXSwiZ2l2ZW5OYW1lIjpbIkFsaWNlIl0sInN1cm5hbWUiOlsiWmhhbyJdfX0.Pgoqfs6KI7hU10tn9eqW7N3JOUXNPqAJGaQtxiz-jxs"

	// Issue a request with a cookie that resembles the cookies set by our old
	// github.com/crewjam/saml-based SAML impl (which used JWTs in cookies).
	//
	// Attempting to decode this cookie will fail.
	req, _ := http.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{
		Name:     cookieName,
		Value:    signedToken,
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
	})
	w := httptest.NewRecorder()

	CookieMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(w, req)

	// Want the request to succeed and clear the bad cookie.
	resp := w.Result()
	if want := http.StatusOK; resp.StatusCode != want {
		t.Errorf("got HTTP %d, want %d", resp.StatusCode, want)
	}
	cookies := resp.Cookies()
	if want := []*http.Cookie{{
		Name:       cookieName,
		Path:       "/",
		RawExpires: "Thu, 01 Jan 1970 00:00:01 GMT",
		MaxAge:     -1,
		Expires:    time.Date(1970, time.January, 1, 0, 0, 1, 0, time.UTC),
		Raw:        cookieName + "=; Path=/; Expires=Thu, 01 Jan 1970 00:00:01 GMT; Max-Age=0",
	}}; !reflect.DeepEqual(cookies, want) {
		t.Errorf("got cookies %+v, want %+v", cookies, want)
	}
}
