package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/randstring"
	"golang.org/x/crypto/bcrypt"
)

func (u *users) IsPassword(ctx context.Context, id int32, password string) (bool, error) {
	var passwd sql.NullString
	if err := dbconn.Global.QueryRowContext(ctx, "SELECT passwd FROM users WHERE deleted_at IS NULL AND id=$1", id).Scan(&passwd); err != nil {
		return false, err
	}
	if !passwd.Valid {
		return false, nil
	}
	return validPassword(passwd.String, password), nil
}

var (
	passwordResetRateLimit    = "1 minute"
	ErrPasswordResetRateLimit = errors.New("password reset rate limit reached")
)

func (u *users) RenewPasswordResetCode(ctx context.Context, id int32) (string, error) {
	if _, err := u.GetByID(ctx, id); err != nil {
		return "", err
	}
	var b [40]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	code := base64.StdEncoding.EncodeToString(b[:])
	res, err := dbconn.Global.ExecContext(ctx, "UPDATE users SET passwd_reset_code=$1, passwd_reset_time=now() WHERE id=$2 AND (passwd_reset_time IS NULL OR passwd_reset_time + interval '"+passwordResetRateLimit+"' < now())", code, id)
	if err != nil {
		return "", err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return "", err
	}
	if affected == 0 {
		return "", ErrPasswordResetRateLimit
	}

	return code, nil
}

// SetPassword sets the user's password given a new password and a password reset code
func (u *users) SetPassword(ctx context.Context, id int32, resetCode, newPassword string) (bool, error) {
	// 🚨 SECURITY: no empty passwords
	if newPassword == "" {
		return false, errors.New("new password was empty")
	}
	// 🚨 SECURITY: check resetCode against what's in the DB and that it's not expired
	r := dbconn.Global.QueryRowContext(ctx, "SELECT count(*) FROM users WHERE id=$1 AND deleted_at IS NULL AND passwd_reset_code=$2 AND passwd_reset_time + interval '4 hours' > now()", id, resetCode)
	var ct int
	if err := r.Scan(&ct); err != nil {
		return false, err
	}
	if ct > 1 {
		return false, fmt.Errorf("illegal state: found more than one user matching ID %d", id)
	}
	if ct == 0 {
		return false, nil
	}
	passwd, err := hashPassword(newPassword)
	if err != nil {
		return false, err
	}
	// 🚨 SECURITY: set the new password and clear the reset code and expiry so the same code can't be reused.
	if _, err := dbconn.Global.ExecContext(ctx, "UPDATE users SET passwd_reset_code=NULL, passwd_reset_time=NULL, passwd=$1 WHERE id=$2", passwd, id); err != nil {
		return false, err
	}

	return true, nil
}

func (u *users) DeletePasswordResetCode(ctx context.Context, id int32) error {
	_, err := dbconn.Global.ExecContext(ctx, "UPDATE users SET passwd_reset_code=NULL, passwd_reset_time=NULL WHERE id=$1", id)
	return err
}

// UpdatePassword updates a user's password given the current password.
func (u *users) UpdatePassword(ctx context.Context, id int32, oldPassword, newPassword string) error {
	// 🚨 SECURITY: No empty passwords.
	if oldPassword == "" {
		return errors.New("old password was empty")
	}
	if newPassword == "" {
		return errors.New("new password was empty")
	}
	// 🚨 SECURITY: Make sure the caller provided the correct old password.
	if ok, err := u.IsPassword(ctx, id, oldPassword); err != nil {
		return err
	} else if !ok {
		return errors.New("wrong old password")
	}

	if err := CheckPasswordLength(newPassword); err != nil {
		return err
	}

	passwd, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	// 🚨 SECURITY: Set the new password
	if _, err := dbconn.Global.ExecContext(ctx, "UPDATE users SET passwd_reset_code=NULL, passwd_reset_time=NULL, passwd=$1 WHERE id=$2", passwd, id); err != nil {
		return err
	}

	return nil
}

// RandomizePasswordAndClearPasswordResetRateLimit overwrites a user's password with a hard-to-guess
// random password and clears the password reset rate limit. It is intended to be used by site admins,
// who can subsequently generate a new password reset code for the user (in case the user has locked
// themselves out, or in case the site admin wants to initiate a password reset).
//
// A randomized password is used (instead of an empty password) to avoid bugs where an empty password
// is considered to be no password. The random password is expected to be irretrievable.
func (u *users) RandomizePasswordAndClearPasswordResetRateLimit(ctx context.Context, id int32) error {
	passwd, err := hashPassword(randstring.NewLen(36))
	if err != nil {
		return err
	}
	// 🚨 SECURITY: Set the new random password and clear the reset code/expiry, so the old code
	// can't be reused, and so a new valid reset code can be generated afterward.
	_, err = dbconn.Global.ExecContext(ctx, "UPDATE users SET passwd_reset_code=NULL, passwd_reset_time=NULL, passwd=$1 WHERE id=$2", passwd, id)
	return err
}

func hashPassword(password string) (sql.NullString, error) {
	if dbtesting.MockHashPassword != nil {
		return dbtesting.MockHashPassword(password)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return sql.NullString{}, err
	}
	return sql.NullString{Valid: true, String: string(hash)}, nil
}

func validPassword(hash, password string) bool {
	if dbtesting.MockValidPassword != nil {
		return dbtesting.MockValidPassword(hash, password)
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
