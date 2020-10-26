package db

import (
	"context"
	"testing"

	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
)

func TestUsers_BuiltinAuth(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	if _, err := Users.Create(ctx, NewUser{
		Email:       "foo@bar.com",
		Username:    "foo",
		DisplayName: "foo",
		Password:    "asdfasdf",
	}); err == nil {
		t.Fatal("user created without email verification code or admin-verified status")
	}

	usr, err := Users.Create(ctx, NewUser{
		Email:                 "foo@bar.com",
		Username:              "foo",
		DisplayName:           "foo",
		Password:              "right-password",
		EmailVerificationCode: "email-code",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, verified, err := UserEmails.GetPrimaryEmail(ctx, usr.ID)
	if err != nil {
		t.Fatal(err)
	}
	if verified {
		t.Fatal("new user should not be verified")
	}
	if isValid, err := UserEmails.Verify(ctx, usr.ID, "foo@bar.com", "wrong_email-code"); err == nil && isValid {
		t.Fatal("should not validate email with wrong code")
	}
	if isValid, err := UserEmails.Verify(ctx, usr.ID, "foo@bar.com", "email-code"); err != nil || !isValid {
		t.Fatal("couldn't vaidate email")
	}
	usr, err = Users.GetByID(ctx, usr.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, verified, err := UserEmails.GetPrimaryEmail(ctx, usr.ID); err != nil {
		t.Fatal(err)
	} else if !verified {
		t.Fatal("user should not be verified")
	}
	if isPassword, err := Users.IsPassword(ctx, usr.ID, "right-password"); err != nil || !isPassword {
		t.Fatal("didn't accept correct password")
	}
	if isPassword, err := Users.IsPassword(ctx, usr.ID, "wrong-password"); err == nil && isPassword {
		t.Fatal("accepted wrong password")
	}
	if _, err := Users.RenewPasswordResetCode(ctx, 193092309); err == nil {
		t.Fatal("no error renewing password reset for non-existent users")
	}
	resetCode, err := Users.RenewPasswordResetCode(ctx, usr.ID)
	if err != nil {
		t.Fatal(err)
	}
	if success, err := Users.SetPassword(ctx, usr.ID, "wrong-code", "new-password"); err == nil && success {
		t.Fatal("password updated without right reset code")
	}
	if success, err := Users.SetPassword(ctx, usr.ID, "", "new-password"); err == nil && success {
		t.Fatal("password updated without reset code")
	}
	if isPassword, err := Users.IsPassword(ctx, usr.ID, "right-password"); err != nil || !isPassword {
		t.Fatal("password changed")
	}
	if success, err := Users.SetPassword(ctx, usr.ID, resetCode, "new-password"); err != nil || !success {
		t.Fatalf("failed to update user password with code: %s", err)
	}
	if isPassword, err := Users.IsPassword(ctx, usr.ID, "new-password"); err != nil || !isPassword {
		t.Fatalf("new password doesn't work: %s", err)
	}
	if isPassword, err := Users.IsPassword(ctx, usr.ID, "right-password"); err == nil && isPassword {
		t.Fatal("old password still works")
	}
}

func TestUsers_BuiltinAuth_VerifiedEmail(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	user, err := Users.Create(ctx, NewUser{
		Email:           "foo@bar.com",
		Username:        "foo",
		Password:        "asdf",
		EmailIsVerified: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, verified, err := UserEmails.GetPrimaryEmail(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !verified {
		t.Error("!verified")
	}
}

func TestUsers_BuiltinAuthPasswordResetRateLimit(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	oldPasswordResetRateLimit := passwordResetRateLimit
	defer func() {
		passwordResetRateLimit = oldPasswordResetRateLimit
	}()

	passwordResetRateLimit = "24 hours"
	usr, err := Users.Create(ctx, NewUser{
		Email:                 "foo@bar.com",
		Username:              "foo",
		DisplayName:           "foo",
		Password:              "right-password",
		EmailVerificationCode: "email-code",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Users.RenewPasswordResetCode(ctx, usr.ID); err != nil {
		t.Fatalf("unexpected password reset error: %s", err)
	}
	if _, err := Users.RenewPasswordResetCode(ctx, usr.ID); err != ErrPasswordResetRateLimit {
		t.Fatal("expected to hit rate limit")
	}

	passwordResetRateLimit = "0 hours"
	if _, err := Users.RenewPasswordResetCode(ctx, usr.ID); err != nil {
		t.Fatalf("unexpected password reset error: %s", err)
	}
	if _, err := Users.RenewPasswordResetCode(ctx, usr.ID); err != nil {
		t.Fatalf("unexpected password reset error: %s", err)
	}
}

func TestUsers_UpdatePassword(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	usr, err := Users.Create(ctx, NewUser{
		Email:                 "foo@bar.com",
		Username:              "foo",
		Password:              "right-password",
		EmailVerificationCode: "c",
	})
	if err != nil {
		t.Fatal(err)
	}

	if isPassword, err := Users.IsPassword(ctx, usr.ID, "right-password"); err != nil || !isPassword {
		t.Fatal("didn't accept correct password")
	}
	if isPassword, err := Users.IsPassword(ctx, usr.ID, "wrong-password"); err == nil && isPassword {
		t.Fatal("accepted wrong password")
	}
	if err := Users.UpdatePassword(ctx, usr.ID, "wrong-password", "new-password"); err == nil {
		t.Fatal("accepted wrong old password")
	}
	if isPassword, err := Users.IsPassword(ctx, usr.ID, "right-password"); err != nil || !isPassword {
		t.Fatal("didn't accept correct password")
	}
	if isPassword, err := Users.IsPassword(ctx, usr.ID, "wrong-password"); err == nil && isPassword {
		t.Fatal("accepted wrong password")
	}

	if err := Users.UpdatePassword(ctx, usr.ID, "right-password", "new-password"); err != nil {
		t.Fatal(err)
	}
	if isPassword, err := Users.IsPassword(ctx, usr.ID, "new-password"); err != nil || !isPassword {
		t.Fatal("didn't accept correct password")
	}
	if isPassword, err := Users.IsPassword(ctx, usr.ID, "wrong-password"); err == nil && isPassword {
		t.Fatal("accepted wrong password")
	}
	if isPassword, err := Users.IsPassword(ctx, usr.ID, "right-password"); err == nil && isPassword {
		t.Fatal("accepted wrong (old) password")
	}
}
