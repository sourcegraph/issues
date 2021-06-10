package userpasswd

import (
	"context"
	"testing"
	"time"

	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtesting"
)

func TestCheckEmailAbuse(t *testing.T) {
	ctx := context.Background()

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	farFuture := now.AddDate(100, 0, 0)

	tests := []struct {
		name      string
		mockEmail *database.UserEmail
		mockErr   error
		expAbused bool
		expReason string
		expErr    error
	}{
		{
			name:      "no emails found",
			mockEmail: nil,
			mockErr:   database.MockUserEmailNotFoundErr,
			expAbused: false,
			expReason: "",
			expErr:    nil,
		},
		{
			name: "needs cool down",
			mockEmail: &database.UserEmail{
				LastVerificationSentAt: &farFuture,
			},
			mockErr:   nil,
			expAbused: true,
			expReason: "too frequent attempt since last verification email sent",
			expErr:    nil,
		},

		{
			name: "no abuse",
			mockEmail: &database.UserEmail{
				LastVerificationSentAt: &yesterday,
			},
			mockErr:   nil,
			expAbused: false,
			expReason: "",
			expErr:    nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database.Mocks.UserEmails.GetLatestVerificationSentEmail = func(context.Context, string) (*database.UserEmail, error) {
				return test.mockEmail, test.mockErr
			}
			defer func() {
				database.Mocks.UserEmails.GetLatestVerificationSentEmail = nil
			}()

			abused, reason, err := checkEmailAbuse(ctx, &dbtesting.MockDB{}, "fake@localhost")
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
