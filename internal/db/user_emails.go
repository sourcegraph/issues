package db

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/keegancsmith/sqlf"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/globalstatedb"
)

// UserEmail represents a row in the `user_emails` table.
type UserEmail struct {
	UserID                 int32
	Email                  string
	CreatedAt              time.Time
	VerificationCode       *string
	VerifiedAt             *time.Time
	LastVerificationSentAt *time.Time
}

// NeedsVerificationCoolDown returns true if the verification cooled down time is behind current time.
func (email *UserEmail) NeedsVerificationCoolDown() bool {
	const defaultDur = 30 * time.Second
	return email.LastVerificationSentAt != nil &&
		time.Now().UTC().Before(email.LastVerificationSentAt.Add(defaultDur))
}

// userEmailNotFoundError is the error that is returned when a user email is not found.
type userEmailNotFoundError struct {
	args []interface{}
}

func (err userEmailNotFoundError) Error() string {
	return fmt.Sprintf("user email not found: %v", err.args)
}

func (err userEmailNotFoundError) NotFound() bool {
	return true
}

// userEmails provides access to the `user_emails` table.
type userEmails struct{}

// GetInitialSiteAdminEmail returns a best guess of the email of the initial Sourcegraph installer/site admin.
// Because the initial site admin's email isn't marked, this returns the email of the active site admin with
// the lowest user ID.
//
// If the site has not yet been initialized, returns an empty string.
func (*userEmails) GetInitialSiteAdminEmail(ctx context.Context) (email string, err error) {
	if init, err := globalstatedb.SiteInitialized(ctx); err != nil || !init {
		return "", err
	}
	if err := dbconn.Global.QueryRowContext(ctx, "SELECT email FROM user_emails JOIN users ON user_emails.user_id=users.id WHERE users.site_admin AND users.deleted_at IS NULL ORDER BY users.id ASC LIMIT 1").Scan(&email); err != nil {
		return "", errors.New("initial site admin email not found")
	}
	return email, nil
}

// GetPrimaryEmail gets the oldest email associated with the user, preferring a verified email to an
// unverified email.
func (*userEmails) GetPrimaryEmail(ctx context.Context, id int32) (email string, verified bool, err error) {
	if Mocks.UserEmails.GetPrimaryEmail != nil {
		return Mocks.UserEmails.GetPrimaryEmail(ctx, id)
	}

	if err := dbconn.Global.QueryRowContext(ctx, "SELECT email, verified_at IS NOT NULL AS verified FROM user_emails WHERE user_id=$1 ORDER BY (verified_at IS NOT NULL) DESC, created_at ASC, email ASC LIMIT 1",
		id,
	).Scan(&email, &verified); err != nil {
		return "", false, userEmailNotFoundError{[]interface{}{fmt.Sprintf("id %d", id)}}
	}
	return email, verified, nil
}

// Get gets information about the user's associated email address.
func (*userEmails) Get(ctx context.Context, userID int32, email string) (emailCanonicalCase string, verified bool, err error) {
	if Mocks.UserEmails.Get != nil {
		return Mocks.UserEmails.Get(userID, email)
	}

	if err := dbconn.Global.QueryRowContext(ctx, "SELECT email, verified_at IS NOT NULL AS verified FROM user_emails WHERE user_id=$1 AND email=$2",
		userID, email,
	).Scan(&emailCanonicalCase, &verified); err != nil {
		return "", false, userEmailNotFoundError{[]interface{}{fmt.Sprintf("userID %d email %q", userID, email)}}
	}
	return emailCanonicalCase, verified, nil
}

// Add adds new user email. When added, it is always unverified.
func (*userEmails) Add(ctx context.Context, userID int32, email string, verificationCode *string) error {
	_, err := dbconn.Global.ExecContext(ctx, "INSERT INTO user_emails(user_id, email, verification_code) VALUES($1, $2, $3)", userID, email, verificationCode)
	return err
}

// Remove removes a user email. It returns an error if there is no such email associated with the user.
func (*userEmails) Remove(ctx context.Context, userID int32, email string) error {
	res, err := dbconn.Global.ExecContext(ctx, "DELETE FROM user_emails WHERE user_id=$1 AND email=$2", userID, email)
	if err != nil {
		return err
	}
	nrows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if nrows == 0 {
		return errors.New("user email not found")
	}
	return nil
}

// Verify verifies the user's email address given the email verification code. If the code is not
// correct (not the one originally used when creating the user or adding the user email), then it
// returns false.
func (*userEmails) Verify(ctx context.Context, userID int32, email, code string) (bool, error) {
	var dbCode sql.NullString
	if err := dbconn.Global.QueryRowContext(ctx, "SELECT verification_code FROM user_emails WHERE user_id=$1 AND email=$2", userID, email).Scan(&dbCode); err != nil {
		return false, err
	}
	if !dbCode.Valid {
		return false, errors.New("email already verified")
	}
	// 🚨 SECURITY: Use constant-time comparisons to avoid leaking the verification code via timing attack. It is not important to avoid leaking the *length* of the code, because the length of verification codes is constant.
	if len(dbCode.String) != len(code) || subtle.ConstantTimeCompare([]byte(dbCode.String), []byte(code)) != 1 {
		return false, nil
	}
	if _, err := dbconn.Global.ExecContext(ctx, "UPDATE user_emails SET verification_code=null, verified_at=now() WHERE user_id=$1 AND email=$2", userID, email); err != nil {
		return false, err
	}

	return true, nil
}

// SetVerified bypasses the normal email verification code process and manually sets the verified
// status for an email.
func (*userEmails) SetVerified(ctx context.Context, userID int32, email string, verified bool) error {
	if Mocks.UserEmails.SetVerified != nil {
		return Mocks.UserEmails.SetVerified(ctx, userID, email, verified)
	}

	var res sql.Result
	var err error
	if verified {
		// Mark as verified.
		res, err = dbconn.Global.ExecContext(ctx, "UPDATE user_emails SET verification_code=null, verified_at=now() WHERE user_id=$1 AND email=$2", userID, email)
	} else {
		// Mark as unverified.
		res, err = dbconn.Global.ExecContext(ctx, "UPDATE user_emails SET verification_code=null, verified_at=null WHERE user_id=$1 AND email=$2", userID, email)
	}
	if err != nil {
		return err
	}
	nrows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if nrows == 0 {
		return errors.New("user email not found")
	}
	return nil
}

// SetLastVerificationSentAt sets the "last_verification_sent_at" column to now() for given email of the user.
func (*userEmails) SetLastVerificationSentAt(ctx context.Context, userID int32, email string) error {
	res, err := dbconn.Global.ExecContext(ctx, "UPDATE user_emails SET last_verification_sent_at=now() WHERE user_id=$1 AND email=$2", userID, email)
	if err != nil {
		return err
	}
	nrows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if nrows == 0 {
		return errors.New("user email not found")
	}
	return nil
}

// GetLatestVerificationSentEmail returns the email with the lastest time of "last_verification_sent_at" column,
// it excludes rows with "last_verification_sent_at IS NULL".
func (*userEmails) GetLatestVerificationSentEmail(ctx context.Context, email string) (*UserEmail, error) {
	if Mocks.UserEmails.GetLatestVerificationSentEmail != nil {
		return Mocks.UserEmails.GetLatestVerificationSentEmail(ctx, email)
	}

	q := sqlf.Sprintf(`
WHERE email=%s AND last_verification_sent_at IS NOT NULL
ORDER BY last_verification_sent_at DESC
LIMIT 1
`, email)
	emails, err := (&userEmails{}).getBySQL(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...)
	if err != nil {
		return nil, err
	} else if len(emails) < 1 {
		return nil, userEmailNotFoundError{[]interface{}{fmt.Sprintf("email %q", email)}}
	}
	return emails[0], nil
}

// GetVerifiedEmails returns a list of verified emails from the candidate list. Some emails are excluded
// from the results list because of unverified or simply don't exist.
func (*userEmails) GetVerifiedEmails(ctx context.Context, emails ...string) ([]*UserEmail, error) {
	if Mocks.UserEmails.GetVerifiedEmails != nil {
		return Mocks.UserEmails.GetVerifiedEmails(ctx, emails...)
	}

	if len(emails) == 0 {
		return []*UserEmail{}, nil
	}

	items := make([]*sqlf.Query, len(emails))
	for i := range emails {
		items[i] = sqlf.Sprintf("%s", emails[i])
	}
	q := sqlf.Sprintf("WHERE email IN (%s) AND verified_at IS NOT NULL", sqlf.Join(items, ","))
	return (&userEmails{}).getBySQL(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...)
}

// getBySQL returns user emails matching the SQL query, if any exist.
func (*userEmails) getBySQL(ctx context.Context, query string, args ...interface{}) ([]*UserEmail, error) {
	rows, err := dbconn.Global.QueryContext(ctx,
		`SELECT user_emails.user_id, user_emails.email, user_emails.created_at, user_emails.verification_code,
				user_emails.verified_at, user_emails.last_verification_sent_at FROM user_emails `+query, args...)
	if err != nil {
		return nil, err
	}

	var userEmails []*UserEmail
	defer rows.Close()
	for rows.Next() {
		var v UserEmail
		err := rows.Scan(&v.UserID, &v.Email, &v.CreatedAt, &v.VerificationCode, &v.VerifiedAt, &v.LastVerificationSentAt)
		if err != nil {
			return nil, err
		}
		userEmails = append(userEmails, &v)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return userEmails, nil
}

// UserEmailsListOptions specifies the options for listing user emails.
type UserEmailsListOptions struct {
	// UserID specifies the id of the user for listing emails.
	UserID int32
	// OnlyVerified excludes unverified emails from the list.
	OnlyVerified bool
}

// ListByUser returns a list of emails that are associated to the given user.
func (*userEmails) ListByUser(ctx context.Context, opt UserEmailsListOptions) ([]*UserEmail, error) {
	if Mocks.UserEmails.ListByUser != nil {
		return Mocks.UserEmails.ListByUser(ctx, opt)
	}

	conds := []*sqlf.Query{
		sqlf.Sprintf("user_id=%s", opt.UserID),
	}
	if opt.OnlyVerified {
		conds = append(conds, sqlf.Sprintf("verified_at IS NOT NULL"))
	}

	q := sqlf.Sprintf("WHERE %s ORDER BY created_at ASC, email ASC", sqlf.Join(conds, "AND"))
	return (&userEmails{}).getBySQL(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...)
}
