package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/authz"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
)

func TestAccessTokenAuthMiddleware(t *testing.T) {
	handler := AccessTokenAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor := actor.FromContext(r.Context())
		if actor.IsAuthenticated() {
			fmt.Fprintf(w, "user %v", actor.UID)
		} else {
			fmt.Fprint(w, "no user")
		}
	}))
	checkHTTPResponse := func(t *testing.T, req *http.Request, wantStatusCode int, wantBody string) {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != wantStatusCode {
			t.Errorf("got response status %d, want %d", rr.Code, wantStatusCode)
		}
		if got := rr.Body.String(); got != wantBody {
			t.Errorf("got response body %q, want %q", got, wantBody)
		}
	}

	t.Run("no header", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/", nil)
		checkHTTPResponse(t, req, http.StatusOK, "no user")
	})

	// Test that the absence of an Authorization header doesn't unset the actor provided by a prior
	// auth middleware.
	t.Run("no header, actor present", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/", nil)
		req = req.WithContext(actor.WithActor(context.Background(), &actor.Actor{UID: 123}))
		checkHTTPResponse(t, req, http.StatusOK, "user 123")
	})

	for _, unrecognizedHeaderValue := range []string{"x", "x y", "Basic abcd"} {
		t.Run("unrecognized header "+unrecognizedHeaderValue, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			req.Header.Set("Authorization", unrecognizedHeaderValue)
			checkHTTPResponse(t, req, http.StatusOK, "no user")
		})
	}

	for _, invalidHeaderValue := range []string{"token-sudo abc", `token-sudo token=""`, "token "} {
		t.Run("invalid header "+invalidHeaderValue, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			req.Header.Set("Authorization", invalidHeaderValue)
			checkHTTPResponse(t, req, http.StatusUnauthorized, "Invalid Authorization header.\n")
		})
	}

	t.Run("valid header with invalid token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "token badbad")
		var calledAccessTokensLookup bool
		db.Mocks.AccessTokens.Lookup = func(tokenHexEncoded, requiredScope string) (subjectUserID int32, err error) {
			calledAccessTokensLookup = true
			return 0, errors.New("x")
		}
		defer func() { db.Mocks = db.MockStores{} }()
		checkHTTPResponse(t, req, http.StatusUnauthorized, "Invalid access token.\n")
		if !calledAccessTokensLookup {
			t.Error("!calledAccessTokensLookup")
		}
	})

	for _, headerValue := range []string{"token abcdef", `token token="abcdef"`} {
		t.Run("valid non-sudo token: "+headerValue, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			req.Header.Set("Authorization", headerValue)
			var calledAccessTokensLookup bool
			db.Mocks.AccessTokens.Lookup = func(tokenHexEncoded, requiredScope string) (subjectUserID int32, err error) {
				calledAccessTokensLookup = true
				if want := "abcdef"; tokenHexEncoded != want {
					t.Errorf("got %q, want %q", tokenHexEncoded, want)
				}
				if want := authz.ScopeUserAll; requiredScope != want {
					t.Errorf("got %q, want %q", requiredScope, want)
				}
				return 123, nil
			}
			defer func() { db.Mocks = db.MockStores{} }()
			checkHTTPResponse(t, req, http.StatusOK, "user 123")
			if !calledAccessTokensLookup {
				t.Error("!calledAccessTokensLookup")
			}
		})
	}

	// Test that an access token overwrites the actor set by a prior auth middleware.
	t.Run("actor present, valid non-sudo token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "token abcdef")
		req = req.WithContext(actor.WithActor(context.Background(), &actor.Actor{UID: 456}))
		var calledAccessTokensLookup bool
		db.Mocks.AccessTokens.Lookup = func(tokenHexEncoded, requiredScope string) (subjectUserID int32, err error) {
			calledAccessTokensLookup = true
			if want := "abcdef"; tokenHexEncoded != want {
				t.Errorf("got %q, want %q", tokenHexEncoded, want)
			}
			if want := authz.ScopeUserAll; requiredScope != want {
				t.Errorf("got %q, want %q", requiredScope, want)
			}
			return 123, nil
		}
		defer func() { db.Mocks = db.MockStores{} }()
		checkHTTPResponse(t, req, http.StatusOK, "user 123")
		if !calledAccessTokensLookup {
			t.Error("!calledAccessTokensLookup")
		}
	})

	// Test that an access token overwrites the actor set by a prior auth middleware.
	const (
		sourceQueryParam = "query-param"
		sourceBasicAuth  = "basic-auth"
	)
	for _, source := range []string{sourceQueryParam, sourceBasicAuth} {
		t.Run("actor present, valid non-sudo token in "+source, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			if source == sourceQueryParam {
				q := url.Values{}
				q.Add("token", "abcdef")
				req.URL.RawQuery = q.Encode()
			} else {
				req.SetBasicAuth("abcdef", "")
			}
			req = req.WithContext(actor.WithActor(context.Background(), &actor.Actor{UID: 456}))
			var calledAccessTokensLookup bool
			db.Mocks.AccessTokens.Lookup = func(tokenHexEncoded, requiredScope string) (subjectUserID int32, err error) {
				calledAccessTokensLookup = true
				if want := "abcdef"; tokenHexEncoded != want {
					t.Errorf("got %q, want %q", tokenHexEncoded, want)
				}
				if want := authz.ScopeUserAll; requiredScope != want {
					t.Errorf("got %q, want %q", requiredScope, want)
				}
				return 123, nil
			}
			defer func() { db.Mocks = db.MockStores{} }()
			checkHTTPResponse(t, req, http.StatusOK, "user 123")
			if !calledAccessTokensLookup {
				t.Error("!calledAccessTokensLookup")
			}
		})
	}

	t.Run("valid sudo token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", `token-sudo token="abcdef",user="alice"`)
		var calledAccessTokensLookup bool
		db.Mocks.AccessTokens.Lookup = func(tokenHexEncoded, requiredScope string) (subjectUserID int32, err error) {
			calledAccessTokensLookup = true
			if want := "abcdef"; tokenHexEncoded != want {
				t.Errorf("got %q, want %q", tokenHexEncoded, want)
			}
			if want := authz.ScopeSiteAdminSudo; requiredScope != want {
				t.Errorf("got %q, want %q", requiredScope, want)
			}
			return 123, nil
		}
		var calledUsersGetByID bool
		db.Mocks.Users.GetByID = func(ctx context.Context, userID int32) (*types.User, error) {
			calledUsersGetByID = true
			if want := int32(123); userID != want {
				t.Errorf("got %d, want %d", userID, want)
			}
			return &types.User{ID: userID, SiteAdmin: true}, nil
		}
		var calledUsersGetByUsername bool
		db.Mocks.Users.GetByUsername = func(ctx context.Context, username string) (*types.User, error) {
			calledUsersGetByUsername = true
			if want := "alice"; username != want {
				t.Errorf("got %q, want %q", username, want)
			}
			return &types.User{ID: 456, SiteAdmin: true}, nil
		}
		defer func() { db.Mocks = db.MockStores{} }()
		checkHTTPResponse(t, req, http.StatusOK, "user 456")
		if !calledAccessTokensLookup {
			t.Error("!calledAccessTokensLookup")
		}
		if !calledUsersGetByID {
			t.Error("!calledUsersGetByID")
		}
		if !calledUsersGetByUsername {
			t.Error("!calledUsersGetByUsername")
		}
	})

	// Test that if a sudo token's subject user is not a site admin (which means they were demoted
	// from site admin AFTER the token was created), then the sudo token is invalid.
	t.Run("valid sudo token, subject is not site admin", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", `token-sudo token="abcdef",user="alice"`)
		var calledAccessTokensLookup bool
		db.Mocks.AccessTokens.Lookup = func(tokenHexEncoded, requiredScope string) (subjectUserID int32, err error) {
			calledAccessTokensLookup = true
			if want := "abcdef"; tokenHexEncoded != want {
				t.Errorf("got %q, want %q", tokenHexEncoded, want)
			}
			if want := authz.ScopeSiteAdminSudo; requiredScope != want {
				t.Errorf("got %q, want %q", requiredScope, want)
			}
			return 123, nil
		}
		var calledUsersGetByID bool
		db.Mocks.Users.GetByID = func(ctx context.Context, userID int32) (*types.User, error) {
			calledUsersGetByID = true
			if want := int32(123); userID != want {
				t.Errorf("got %d, want %d", userID, want)
			}
			return &types.User{ID: userID, SiteAdmin: false}, nil
		}
		defer func() { db.Mocks = db.MockStores{} }()
		checkHTTPResponse(t, req, http.StatusForbidden, "The subject user of a sudo access token must be a site admin.\n")
		if !calledAccessTokensLookup {
			t.Error("!calledAccessTokensLookup")
		}
		if !calledUsersGetByID {
			t.Error("!calledUsersGetByID")
		}
	})

	t.Run("valid sudo token, invalid sudo user", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", `token-sudo token="abcdef",user="doesntexist"`)
		var calledAccessTokensLookup bool
		db.Mocks.AccessTokens.Lookup = func(tokenHexEncoded, requiredScope string) (subjectUserID int32, err error) {
			calledAccessTokensLookup = true
			if want := "abcdef"; tokenHexEncoded != want {
				t.Errorf("got %q, want %q", tokenHexEncoded, want)
			}
			if want := authz.ScopeSiteAdminSudo; requiredScope != want {
				t.Errorf("got %q, want %q", requiredScope, want)
			}
			return 123, nil
		}
		var calledUsersGetByID bool
		db.Mocks.Users.GetByID = func(ctx context.Context, userID int32) (*types.User, error) {
			calledUsersGetByID = true
			if want := int32(123); userID != want {
				t.Errorf("got %d, want %d", userID, want)
			}
			return &types.User{ID: userID, SiteAdmin: true}, nil
		}
		var calledUsersGetByUsername bool
		db.Mocks.Users.GetByUsername = func(ctx context.Context, username string) (*types.User, error) {
			calledUsersGetByUsername = true
			if want := "doesntexist"; username != want {
				t.Errorf("got %q, want %q", username, want)
			}
			return nil, &errcode.Mock{IsNotFound: true}
		}
		defer func() { db.Mocks = db.MockStores{} }()
		checkHTTPResponse(t, req, http.StatusForbidden, "Unable to sudo to nonexistent user.\n")
		if !calledAccessTokensLookup {
			t.Error("!calledAccessTokensLookup")
		}
		if !calledUsersGetByID {
			t.Error("!calledUsersGetByID")
		}
		if !calledUsersGetByUsername {
			t.Error("!calledUsersGetByUsername")
		}
	})
}
