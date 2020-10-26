package userpasswd

import (
	"context"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/txemail"
)

func TestHandleSetPasswordEmail(t *testing.T) {
	ctx := context.Background()
	ctx = actor.WithActor(ctx, &actor.Actor{UID: 1})

	var sent *txemail.Message
	txemail.MockSend = func(ctx context.Context, message txemail.Message) error {
		sent = &message
		return nil
	}
	defer func() { txemail.MockSend = nil }()
	defer func() { backend.MockMakePasswordResetURL = nil }()
	defer func() { db.Mocks.UserEmails.GetPrimaryEmail = nil }()
	defer func() { db.Mocks.Users.GetByID = nil }()

	backend.MockMakePasswordResetURL = func(context.Context, int32) (*url.URL, error) {
		query := url.Values{}
		query.Set("userID", "1")
		query.Set("code", "foo")
		return &url.URL{Path: "/password-reset", RawQuery: query.Encode()}, nil
	}

	db.Mocks.UserEmails.GetPrimaryEmail = func(context.Context, int32) (string, bool, error) {
		return "a@example.com", true, nil
	}

	db.Mocks.Users.GetByID = func(context.Context, int32) (*types.User, error) {
		return &types.User{ID: 1, Username: "test"}, nil
	}

	tests := []struct {
		name    string
		id      int32
		ctx     context.Context
		wantURL string
		wantErr bool
		email   string
	}{
		{
			name:    "Valid ID",
			id:      1,
			ctx:     ctx,
			wantURL: "http://example.com/password-reset?code=foo&userID=1",
			wantErr: false,
			email:   "a@example.com",
		},
	}

	for _, tst := range tests {
		t.Run(tst.name, func(t *testing.T) {
			got, err := HandleSetPasswordEmail(tst.ctx, tst.id)
			if diff := cmp.Diff(tst.wantURL, got); diff != "" {
				t.Errorf("Message mismatch (-want +got):\n%s", diff)
			}
			if (err != nil) != tst.wantErr {
				if tst.wantErr {
					t.Fatalf("input %d error expected", tst.id)
				} else {
					t.Fatalf("input %d got unexpected error %q", tst.id, err.Error())
				}
			}

			if sent == nil {
				t.Fatal("want sent != nil")
			}

			gotEmail := &txemail.Message{
				To:       []string{tst.email},
				Template: setPasswordEmailTemplates,
				Data: struct {
					Username string
					URL      string
				}{
					Username: "test",
					URL:      got,
				},
			}

			want := &txemail.Message{
				To:       []string{tst.email},
				Template: setPasswordEmailTemplates,
				Data: struct {
					Username string
					URL      string
				}{
					Username: "test",
					URL:      tst.wantURL,
				},
			}

			if diff := cmp.Diff(want, gotEmail); diff != "" {
				t.Errorf("Message mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
