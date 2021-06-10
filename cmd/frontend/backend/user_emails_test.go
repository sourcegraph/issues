package backend

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/txemail"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/schema"
)

func TestCheckEmailAbuse(t *testing.T) {
	ctx := testContext()

	cfg := conf.Get()
	cfg.EmailSmtp = &schema.SMTPServerConfig{}
	conf.Mock(cfg)
	defer func() {
		cfg.EmailSmtp = nil
		conf.Mock(cfg)
	}()

	envvar.MockSourcegraphDotComMode(true)
	defer envvar.MockSourcegraphDotComMode(false)

	now := time.Now()

	tests := []struct {
		name       string
		mockEmails []*database.UserEmail
		hasQuote   bool
		expAbused  bool
		expReason  string
		expErr     error
	}{
		{
			name: "no verified email address",
			mockEmails: []*database.UserEmail{
				{
					Email: "alice@example.com",
				},
			},
			hasQuote:  false,
			expAbused: true,
			expReason: "a verified email is required before you can add additional email addressed to your account",
			expErr:    nil,
		},
		{
			name: "reached maximum number of unverified email addresses",
			mockEmails: []*database.UserEmail{
				{
					Email:      "alice@example.com",
					VerifiedAt: &now,
				},
				{
					Email: "alice2@example.com",
				},
				{
					Email: "alice3@example.com",
				},
				{
					Email: "alice4@example.com",
				},
			},
			hasQuote:  false,
			expAbused: true,
			expReason: "too many existing unverified email addresses",
			expErr:    nil,
		},
		{
			name: "no quota",
			mockEmails: []*database.UserEmail{
				{
					Email:      "alice@example.com",
					VerifiedAt: &now,
				},
			},
			hasQuote:  false,
			expAbused: true,
			expReason: "email address quota exceeded (contact support to increase the quota)",
			expErr:    nil,
		},

		{
			name: "no abuse",
			mockEmails: []*database.UserEmail{
				{
					Email:      "alice@example.com",
					VerifiedAt: &now,
				},
			},
			hasQuote:  true,
			expAbused: false,
			expReason: "",
			expErr:    nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database.Mocks.Users.CheckAndDecrementInviteQuota = func(context.Context, int32) (bool, error) {
				return test.hasQuote, nil
			}
			database.Mocks.UserEmails.ListByUser = func(context.Context, database.UserEmailsListOptions) ([]*database.UserEmail, error) {
				return test.mockEmails, nil
			}
			defer func() {
				database.Mocks.Users.CheckAndDecrementInviteQuota = nil
				database.Mocks.UserEmails.ListByUser = nil
			}()

			abused, reason, err := checkEmailAbuse(ctx, &dbtesting.MockDB{}, 1)
			if test.expErr != err {
				t.Fatalf("err: want %v but got %v", test.expErr, err)
			} else if test.expAbused != abused {
				t.Fatalf("abused: want %v but got %v", test.expAbused, abused)
			} else if test.expReason != reason {
				t.Fatalf("reason: want %q but got %q", test.expReason, reason)
			}
		})
	}
}

func TestSendUserEmailVerificationEmail(t *testing.T) {
	var sent *txemail.Message
	txemail.MockSend = func(ctx context.Context, message txemail.Message) error {
		sent = &message
		return nil
	}
	defer func() { txemail.MockSend = nil }()

	if err := SendUserEmailVerificationEmail(context.Background(), "Alan Johnson", "a@example.com", "c"); err != nil {
		t.Fatal(err)
	}
	if sent == nil {
		t.Fatal("want sent != nil")
	}
	if want := (txemail.Message{
		FromName: "",
		To:       []string{"a@example.com"},
		Template: verifyEmailTemplates,
		Data: struct {
			Username string
			URL      string
			Host     string
		}{
			Username: "Alan Johnson",
			URL:      "http://example.com/-/verify-email?code=c&email=a%40example.com",
			Host:     "example.com",
		},
	}); !reflect.DeepEqual(*sent, want) {
		t.Errorf("got %+v, want %+v", *sent, want)
	}
}

func TestSendUserEmailOnFieldUpdate(t *testing.T) {
	var sent *txemail.Message
	txemail.MockSend = func(ctx context.Context, message txemail.Message) error {
		sent = &message
		return nil
	}
	database.Mocks.UserEmails.GetPrimaryEmail = func(ctx context.Context, id int32) (emailCanonicalCase string, verified bool, err error) {
		return "a@example.com", true, nil
	}
	database.Mocks.Users.GetByID = func(ctx context.Context, id int32) (*types.User, error) {
		return &types.User{Username: "Foo"}, nil
	}
	defer func() {
		txemail.MockSend = nil
		database.Mocks.UserEmails.GetPrimaryEmail = nil
		database.Mocks.Users.GetByID = nil
	}()

	if err := UserEmails.SendUserEmailOnFieldUpdate(context.Background(), &dbtesting.MockDB{}, 123, "updated password"); err != nil {
		t.Fatal(err)
	}
	if sent == nil {
		t.Fatal("want sent != nil")
	}
	if want := (txemail.Message{
		FromName: "",
		To:       []string{"a@example.com"},
		Template: updateAccountEmailTemplate,
		Data: struct {
			Email    string
			Change   string
			Username string
			Host     string
		}{
			Email:    "a@example.com",
			Change:   "updated password",
			Username: "Foo",
			Host:     "example.com",
		},
	}); !reflect.DeepEqual(*sent, want) {
		t.Errorf("got %+v, want %+v", *sent, want)
	}
}
