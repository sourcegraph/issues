package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/auth"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/session"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/db"
)

func TestOverrideAuthMiddleware(t *testing.T) {
	cleanup := session.ResetMockSessionStore(t)
	defer cleanup()

	handler := OverrideAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor := actor.FromContext(r.Context())
		if actor.IsAuthenticated() {
			fmt.Fprintf(w, "user %v", actor.UID)
		} else {
			fmt.Fprint(w, "no user")
		}
	}))

	const overrideSecret = "s"

	t.Run("disabled, not sent", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/", nil)
		handler.ServeHTTP(rr, req)
		if got, want := rr.Body.String(), "no user"; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("not sent", func(t *testing.T) {
		envOverrideAuthSecret = overrideSecret
		defer func() { envOverrideAuthSecret = "" }()
		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/", nil)
		handler.ServeHTTP(rr, req)
		if got, want := rr.Body.String(), "no user"; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("not sent, actor present", func(t *testing.T) {
		envOverrideAuthSecret = overrideSecret
		defer func() { envOverrideAuthSecret = "" }()
		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/", nil)
		req = req.WithContext(actor.WithActor(context.Background(), &actor.Actor{UID: 2}))
		handler.ServeHTTP(rr, req)
		if got, want := rr.Body.String(), "user 2"; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("sent, actor not set", func(t *testing.T) {
		envOverrideAuthSecret = overrideSecret
		defer func() { envOverrideAuthSecret = "" }()
		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Set(overrideSecretHeader, overrideSecret)
		var calledMock bool
		auth.MockGetAndSaveUser = func(ctx context.Context, op auth.GetAndSaveUserOp) (userID int32, safeErrMsg string, err error) {
			calledMock = true
			if want := defaultUsername; op.UserProps.Username != want {
				t.Errorf("got %q, want %q", op.UserProps.Username, want)
			}
			return 1, "", nil
		}
		defer func() { auth.MockGetAndSaveUser = nil }()
		db.Mocks.Users.SetIsSiteAdmin = func(int32, bool) error { return nil }
		defer func() { db.Mocks = db.MockStores{} }()
		handler.ServeHTTP(rr, req)
		if got, want := rr.Body.String(), "user 1"; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
		if !calledMock {
			t.Error("!calledMock")
		}
	})

	t.Run("sent, actor already set", func(t *testing.T) {
		envOverrideAuthSecret = overrideSecret
		defer func() { envOverrideAuthSecret = "" }()
		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Set(overrideSecretHeader, overrideSecret)
		req = req.WithContext(actor.WithActor(context.Background(), &actor.Actor{UID: 123}))
		var calledMock bool
		auth.MockGetAndSaveUser = func(ctx context.Context, op auth.GetAndSaveUserOp) (userID int32, safeErrMsg string, err error) {
			calledMock = true
			if op.ExternalAccount.ServiceType == "override" && op.ExternalAccount.AccountID == defaultUsername {
				return 1, "", nil
			}
			return 0, "safeErr", errors.New("x")
		}
		defer func() { auth.MockGetAndSaveUser = nil }()
		db.Mocks.Users.SetIsSiteAdmin = func(int32, bool) error { return nil }
		defer func() { db.Mocks = db.MockStores{} }()
		handler.ServeHTTP(rr, req)
		if got, want := rr.Body.String(), "user 1"; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
		if !calledMock {
			t.Error("!calledMock")
		}
	})

	t.Run("sent, wrong secret", func(t *testing.T) {
		envOverrideAuthSecret = overrideSecret
		defer func() { envOverrideAuthSecret = "" }()
		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Set(overrideSecretHeader, "bad")
		handler.ServeHTTP(rr, req)
		if got, want := rr.Body.String(), "no user"; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}
