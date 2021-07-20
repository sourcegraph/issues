package database

import (
	"context"
	"testing"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/go-cmp/cmp"

	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
)

func TestUserEmail_NeedsVerificationCoolDown(t *testing.T) {
	timePtr := func(t time.Time) *time.Time {
		return &t
	}

	tests := []struct {
		name                   string
		lastVerificationSentAt *time.Time
		needsCoolDown          bool
	}{
		{
			name:                   "nil",
			lastVerificationSentAt: nil,
			needsCoolDown:          false,
		},
		{
			name:                   "needs cool down",
			lastVerificationSentAt: timePtr(time.Now().Add(time.Minute)),
			needsCoolDown:          true,
		},
		{
			name:                   "does not need cool down",
			lastVerificationSentAt: timePtr(time.Now().Add(-1 * time.Minute)),
			needsCoolDown:          false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			email := &UserEmail{
				LastVerificationSentAt: test.lastVerificationSentAt,
			}
			needsCoolDown := email.NeedsVerificationCoolDown()
			if test.needsCoolDown != needsCoolDown {
				t.Fatalf("needsCoolDown: want %v but got %v", test.needsCoolDown, needsCoolDown)
			}
		})
	}
}

func TestUserEmails_Get(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()
	db := dbtest.NewDB(t, "")
	ctx := context.Background()

	user, err := Users(db).Create(ctx, NewUser{
		Email:                 "a@example.com",
		Username:              "u2",
		Password:              "pw",
		EmailVerificationCode: "c",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := UserEmails(db).Add(ctx, user.ID, "b@example.com", nil); err != nil {
		t.Fatal(err)
	}

	emailA, verifiedA, err := UserEmails(db).Get(ctx, user.ID, "A@EXAMPLE.com")
	if err != nil {
		t.Fatal(err)
	}
	if want := "a@example.com"; emailA != want {
		t.Errorf("got email %q, want %q", emailA, want)
	}
	if verifiedA {
		t.Error("want verified == false")
	}

	emailB, verifiedB, err := UserEmails(db).Get(ctx, user.ID, "B@EXAMPLE.com")
	if err != nil {
		t.Fatal(err)
	}
	if want := "b@example.com"; emailB != want {
		t.Errorf("got email %q, want %q", emailB, want)
	}
	if verifiedB {
		t.Error("want verified == false")
	}

	if _, _, err := UserEmails(db).Get(ctx, user.ID, "doesntexist@example.com"); !errcode.IsNotFound(err) {
		t.Errorf("got %v, want IsNotFound", err)
	}
}

func TestUserEmails_GetPrimary(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()
	db := dbtest.NewDB(t, "")
	ctx := context.Background()

	user, err := Users(db).Create(ctx, NewUser{
		Email:                 "a@example.com",
		Username:              "u2",
		Password:              "pw",
		EmailVerificationCode: "c",
	})
	if err != nil {
		t.Fatal(err)
	}

	checkPrimaryEmail := func(t *testing.T, wantEmail string, wantVerified bool) {
		t.Helper()
		email, verified, err := UserEmails(db).GetPrimaryEmail(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}
		if email != wantEmail {
			t.Errorf("got email %q, want %q", email, wantEmail)
		}
		if verified != wantVerified {
			t.Errorf("got verified %v, want %v", verified, wantVerified)
		}
	}

	// Initial address should be primary
	checkPrimaryEmail(t, "a@example.com", false)
	// Add a second address
	if err := UserEmails(db).Add(ctx, user.ID, "b1@example.com", nil); err != nil {
		t.Fatal(err)
	}
	// Primary should still be the first one
	checkPrimaryEmail(t, "a@example.com", false)
	// Verify second
	if err := UserEmails(db).SetVerified(ctx, user.ID, "b1@example.com", true); err != nil {
		t.Fatal(err)
	}
	// Set as primary
	if err := UserEmails(db).SetPrimaryEmail(ctx, user.ID, "b1@example.com"); err != nil {
		t.Fatal(err)
	}
	// Confirm it is now the primary
	checkPrimaryEmail(t, "b1@example.com", true)
}

func TestUserEmails_SetPrimary(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()
	db := dbtest.NewDB(t, "")
	ctx := context.Background()

	user, err := Users(db).Create(ctx, NewUser{
		Email:                 "a@example.com",
		Username:              "u2",
		Password:              "pw",
		EmailVerificationCode: "c",
	})
	if err != nil {
		t.Fatal(err)
	}

	checkPrimaryEmail := func(t *testing.T, wantEmail string, wantVerified bool) {
		t.Helper()
		email, verified, err := UserEmails(db).GetPrimaryEmail(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}
		if email != wantEmail {
			t.Errorf("got email %q, want %q", email, wantEmail)
		}
		if verified != wantVerified {
			t.Errorf("got verified %v, want %v", verified, wantVerified)
		}
	}

	// Initial address should be primary
	checkPrimaryEmail(t, "a@example.com", false)
	// Add a another address
	if err := UserEmails(db).Add(ctx, user.ID, "b1@example.com", nil); err != nil {
		t.Fatal(err)
	}
	// Setting it as primary should fail since it is not verified
	if err := UserEmails(db).SetPrimaryEmail(ctx, user.ID, "b1@example.com"); err == nil {
		t.Fatal("Expected an error as address is not verified")
	}
}

func TestUserEmails_ListByUser(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()
	db := dbtest.NewDB(t, "")
	ctx := context.Background()

	user, err := Users(db).Create(ctx, NewUser{
		Email:                 "a@example.com",
		Username:              "u2",
		Password:              "pw",
		EmailVerificationCode: "c",
	})
	if err != nil {
		t.Fatal(err)
	}

	testTime := time.Now().Round(time.Second).UTC()
	if _, err := db.ExecContext(ctx,
		`INSERT INTO user_emails(user_id, email, verification_code, verified_at) VALUES($1, $2, $3, $4)`,
		user.ID, "b@example.com", "c2", testTime); err != nil {
		t.Fatal(err)
	}

	t.Run("list all emails", func(t *testing.T) {
		userEmails, err := UserEmails(db).ListByUser(ctx, UserEmailsListOptions{
			UserID: user.ID,
		})
		if err != nil {
			t.Fatal(err)
		}
		normalizeUserEmails(userEmails)
		want := []*UserEmail{
			{UserID: user.ID, Email: "a@example.com", VerificationCode: strptr("c"), Primary: true},
			{UserID: user.ID, Email: "b@example.com", VerificationCode: strptr("c2"), VerifiedAt: &testTime},
		}
		if diff := cmp.Diff(want, userEmails); diff != "" {
			t.Fatalf("userEmails: %s", diff)
		}
	})

	t.Run("list only verified emails", func(t *testing.T) {
		userEmails, err := UserEmails(db).ListByUser(ctx, UserEmailsListOptions{
			UserID:       user.ID,
			OnlyVerified: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		normalizeUserEmails(userEmails)
		want := []*UserEmail{
			{UserID: user.ID, Email: "b@example.com", VerificationCode: strptr("c2"), VerifiedAt: &testTime},
		}
		if diff := cmp.Diff(want, userEmails); diff != "" {
			t.Fatalf("userEmails: %s", diff)
		}
	})
}

func normalizeUserEmails(userEmails []*UserEmail) {
	for _, v := range userEmails {
		v.CreatedAt = time.Time{}
		if v.VerifiedAt != nil {
			tmp := v.VerifiedAt.Round(time.Second).UTC()
			v.VerifiedAt = &tmp
		}
	}
}

func TestUserEmails_Add_Remove(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()
	db := dbtest.NewDB(t, "")
	ctx := context.Background()

	const emailA = "a@example.com"
	const emailB = "b@example.com"
	user, err := Users(db).Create(ctx, NewUser{
		Email:                 emailA,
		Username:              "u2",
		Password:              "pw",
		EmailVerificationCode: "c",
	})
	if err != nil {
		t.Fatal(err)
	}

	if primary, err := isUserEmailPrimary(ctx, db, user.ID, emailA); err != nil {
		t.Fatal(err)
	} else if want := true; primary != want {
		t.Fatalf("got primary %v, want %v", primary, want)
	}

	if err := UserEmails(db).Add(ctx, user.ID, emailB, nil); err != nil {
		t.Fatal(err)
	}
	if verified, err := isUserEmailVerified(ctx, db, user.ID, emailB); err != nil {
		t.Fatal(err)
	} else if want := false; verified != want {
		t.Fatalf("got verified %v, want %v", verified, want)
	}

	if emails, err := UserEmails(db).ListByUser(ctx, UserEmailsListOptions{
		UserID: user.ID,
	}); err != nil {
		t.Fatal(err)
	} else if want := 2; len(emails) != want {
		t.Errorf("got %d emails, want %d", len(emails), want)
	}

	if err := UserEmails(db).Add(ctx, user.ID, emailB, nil); err == nil {
		t.Fatal("got err == nil for Add on existing email")
	}
	if err := UserEmails(db).Add(ctx, 12345 /* bad user ID */, "foo@example.com", nil); err == nil {
		t.Fatal("got err == nil for Add on bad user ID")
	}
	if emails, err := UserEmails(db).ListByUser(ctx, UserEmailsListOptions{
		UserID: user.ID,
	}); err != nil {
		t.Fatal(err)
	} else if want := 2; len(emails) != want {
		t.Errorf("got %d emails, want %d", len(emails), want)
	}

	// Attempt to remove primary
	if err := UserEmails(db).Remove(ctx, user.ID, emailA); err == nil {
		t.Fatal("expected error, can't delete primary email")
	}
	// Remove non primary
	if err := UserEmails(db).Remove(ctx, user.ID, emailB); err != nil {
		t.Fatal(err)
	}
	if emails, err := UserEmails(db).ListByUser(ctx, UserEmailsListOptions{
		UserID: user.ID,
	}); err != nil {
		t.Fatal(err)
	} else if want := 1; len(emails) != want {
		t.Errorf("got %d emails (after removing), want %d", len(emails), want)
	}

	if err := UserEmails(db).Remove(ctx, user.ID, "foo@example.com"); err == nil {
		t.Fatal("got err == nil for Remove on nonexistent email")
	}
	if err := UserEmails(db).Remove(ctx, 12345 /* bad user ID */, "foo@example.com"); err == nil {
		t.Fatal("got err == nil for Remove on bad user ID")
	}
}

func TestUserEmails_SetVerified(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()
	db := dbtest.NewDB(t, "")
	ctx := context.Background()

	const email = "a@example.com"
	user, err := Users(db).Create(ctx, NewUser{
		Email:                 email,
		Username:              "u2",
		Password:              "pw",
		EmailVerificationCode: "c",
	})
	if err != nil {
		t.Fatal(err)
	}

	if verified, err := isUserEmailVerified(ctx, db, user.ID, email); err != nil {
		t.Fatal(err)
	} else if want := false; verified != want {
		t.Fatalf("before SetVerified, got verified %v, want %v", verified, want)
	}

	if err := UserEmails(db).SetVerified(ctx, user.ID, email, true); err != nil {
		t.Fatal(err)
	}
	if verified, err := isUserEmailVerified(ctx, db, user.ID, email); err != nil {
		t.Fatal(err)
	} else if want := true; verified != want {
		t.Fatalf("after SetVerified true, got verified %v, want %v", verified, want)
	}

	if err := UserEmails(db).SetVerified(ctx, user.ID, email, false); err != nil {
		t.Fatal(err)
	}
	if verified, err := isUserEmailVerified(ctx, db, user.ID, email); err != nil {
		t.Fatal(err)
	} else if want := false; verified != want {
		t.Fatalf("after SetVerified false, got verified %v, want %v", verified, want)
	}

	if err := UserEmails(db).SetVerified(ctx, user.ID, "otheremail@example.com", false); err == nil {
		t.Fatal("got err == nil for SetVerified on bad email")
	}
}

func isUserEmailVerified(ctx context.Context, db dbutil.DB, userID int32, email string) (bool, error) {
	userEmails, err := UserEmails(db).ListByUser(ctx, UserEmailsListOptions{
		UserID: userID,
	})
	if err != nil {
		return false, err
	}
	for _, v := range userEmails {
		if v.Email == email {
			return v.VerifiedAt != nil, nil
		}
	}
	return false, errors.Errorf("email not found: %s", email)
}

func isUserEmailPrimary(ctx context.Context, db dbutil.DB, userID int32, email string) (bool, error) {
	userEmails, err := UserEmails(db).ListByUser(ctx, UserEmailsListOptions{
		UserID: userID,
	})
	if err != nil {
		return false, err
	}
	for _, v := range userEmails {
		if v.Email == email {
			return v.Primary, nil
		}
	}
	return false, errors.Errorf("email not found: %s", email)
}

func TestUserEmails_SetLastVerificationSentAt(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()
	db := dbtest.NewDB(t, "")
	ctx := context.Background()

	const addr = "alice@example.com"
	user, err := Users(db).Create(ctx, NewUser{
		Email:                 addr,
		Username:              "alice",
		Password:              "pw",
		EmailVerificationCode: "c",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify "last_verification_sent_at" column is NULL
	emails, err := UserEmails(db).ListByUser(ctx, UserEmailsListOptions{
		UserID: user.ID,
	})
	if err != nil {
		t.Fatal(err)
	} else if len(emails) != 1 {
		t.Fatalf("want 1 email but got %d emails: %v", len(emails), emails)
	} else if emails[0].LastVerificationSentAt != nil {
		t.Fatalf("lastVerificationSentAt: want nil but got %v", emails[0].LastVerificationSentAt)
	}

	if err = UserEmails(db).SetLastVerification(ctx, user.ID, addr, "c"); err != nil {
		t.Fatal(err)
	}

	// Verify "last_verification_sent_at" column is not NULL
	emails, err = UserEmails(db).ListByUser(ctx, UserEmailsListOptions{
		UserID: user.ID,
	})
	if err != nil {
		t.Fatal(err)
	} else if len(emails) != 1 {
		t.Fatalf("want 1 email but got %d emails: %v", len(emails), emails)
	} else if emails[0].LastVerificationSentAt == nil {
		t.Fatalf("lastVerificationSentAt: want non-nil but got nil")
	}
}

func TestUserEmails_GetLatestVerificationSentEmail(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()
	db := dbtest.NewDB(t, "")
	ctx := context.Background()

	const addr = "alice@example.com"
	user, err := Users(db).Create(ctx, NewUser{
		Email:                 addr,
		Username:              "alice",
		Password:              "pw",
		EmailVerificationCode: "c",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Should return "not found" because "last_verification_sent_at" column is NULL
	_, err = UserEmails(db).GetLatestVerificationSentEmail(ctx, addr)
	if err == nil || !errcode.IsNotFound(err) {
		t.Fatalf("err: want a not found error but got %v", err)
	} else if err = UserEmails(db).SetLastVerification(ctx, user.ID, addr, "c"); err != nil {
		t.Fatal(err)
	}

	// Should return an email because "last_verification_sent_at" column is not NULL
	email, err := UserEmails(db).GetLatestVerificationSentEmail(ctx, addr)
	if err != nil {
		t.Fatal(err)
	} else if email.Email != addr {
		t.Fatalf("Email: want %s but got %q", addr, email.Email)
	}

	// Create another user with same email address and set "last_verification_sent_at" column
	user2, err := Users(db).Create(ctx, NewUser{
		Email:                 addr,
		Username:              "bob",
		Password:              "pw",
		EmailVerificationCode: "c",
	})
	if err != nil {
		t.Fatal(err)
	} else if err = UserEmails(db).SetLastVerification(ctx, user2.ID, addr, "c"); err != nil {
		t.Fatal(err)
	}

	// Should return the email for the second user
	email, err = UserEmails(db).GetLatestVerificationSentEmail(ctx, addr)
	if err != nil {
		t.Fatal(err)
	} else if email.Email != addr {
		t.Fatalf("Email: want %s but got %q", addr, email.Email)
	} else if email.UserID != user2.ID {
		t.Fatalf("UserID: want %d but got %d", user2.ID, email.UserID)
	}
}

func TestUserEmails_GetVerifiedEmails(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()
	db := dbtest.NewDB(t, "")
	ctx := context.Background()

	newUsers := []NewUser{
		{
			Email:           "alice@example.com",
			Username:        "alice",
			EmailIsVerified: true,
		},
		{
			Email:                 "bob@example.com",
			Username:              "bob",
			EmailVerificationCode: "c",
		},
	}

	for _, newUser := range newUsers {
		_, err := Users(db).Create(ctx, newUser)
		if err != nil {
			t.Fatal(err)
		}
	}

	emails, err := UserEmails(db).GetVerifiedEmails(ctx, "alice@example.com", "bob@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(emails) != 1 {
		t.Fatalf("got %d emails, but want 1", len(emails))
	}
	if emails[0].Email != "alice@example.com" {
		t.Errorf("got %s, but want %q", emails[0].Email, "alice@example.com")
	}
}
