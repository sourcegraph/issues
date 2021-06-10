package app

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

func TestLatestPingHandler(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	db := dbtest.NewDB(t, "")

	t.Run("non-admins can't access the ping data", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/site-admin/pings/latest", nil)
		rec := httptest.NewRecorder()
		latestPingHandler(new(dbtesting.MockDB))(rec, req)

		if have, want := rec.Code, http.StatusUnauthorized; have != want {
			t.Errorf("status code: have %d, want %d", have, want)
		}
	})

	tests := []struct {
		desc     string
		pingFn   func(ctx context.Context) (*types.Event, error)
		wantBody string
	}{
		{
			desc: "with no ping events recorded",
			pingFn: func(ctx context.Context) (*types.Event, error) {
				return &types.Event{Argument: `{}`}, nil
			},
			wantBody: `{}`,
		},
		{
			desc: "with ping events recorded",
			pingFn: func(ctx context.Context) (*types.Event, error) {
				return &types.Event{Argument: `{"key": "value"}`}, nil
			},
			wantBody: `{"key": "value"}`,
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			database.Mocks.EventLogs.LatestPing = test.pingFn
			defer func() { database.Mocks.EventLogs.LatestPing = nil }()

			req, _ := http.NewRequest("GET", "/site-admin/pings/latest", nil)
			rec := httptest.NewRecorder()
			latestPingHandler(db)(rec, req.WithContext(backend.WithAuthzBypass(context.Background())))

			resp := rec.Result()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if have, want := resp.StatusCode, http.StatusOK; have != want {
				t.Errorf("Status: have %d, want %d", have, want)
			}
			if have, want := string(body), test.wantBody; have != want {
				t.Errorf("Body: have %q, want %q", have, want)
			}
		})
	}
}
