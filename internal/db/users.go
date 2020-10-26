package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/hashicorp/go-multierror"
	"github.com/inconshreveable/log15"
	"github.com/keegancsmith/sqlf"
	"github.com/lib/pq"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/db/globalstatedb"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/trace"
)

// users provides access to the `users` table.
//
// For a detailed overview of the schema, see schema.md.
type users struct {
	// PreCreateUser (if set) is a hook called before creating a new user in the DB by any means
	// (e.g., both directly via Users.Create or via ExternalAccounts.CreateUserAndSave).
	PreCreateUser func(context.Context) error
}

// userNotFoundErr is the error that is returned when a user is not found.
type userNotFoundErr struct {
	args []interface{}
}

func (err userNotFoundErr) Error() string {
	return fmt.Sprintf("user not found: %v", err.args)
}

func (err userNotFoundErr) NotFound() bool {
	return true
}

// NewUserNotFoundError returns a new error indicating that the user with the given user ID was not
// found.
func NewUserNotFoundError(userID int32) error {
	return userNotFoundErr{args: []interface{}{"userID", userID}}
}

// errCannotCreateUser is the error that is returned when
// a user cannot be added to the DB due to a constraint.
type errCannotCreateUser struct {
	code string
}

const (
	errorCodeUsernameExists = "err_username_exists"
	errorCodeEmailExists    = "err_email_exists"
)

func (err errCannotCreateUser) Error() string {
	return fmt.Sprintf("cannot create user: %v", err.code)
}

func (err errCannotCreateUser) Code() string {
	return err.code
}

// IsUsernameExists reports whether err is an error indicating that the intended username exists.
func IsUsernameExists(err error) bool {
	e, ok := err.(errCannotCreateUser)
	return ok && e.code == errorCodeUsernameExists
}

// IsEmailExists reports whether err is an error indicating that the intended email exists.
func IsEmailExists(err error) bool {
	e, ok := err.(errCannotCreateUser)
	return ok && e.code == errorCodeEmailExists
}

// NewUser describes a new to-be-created user.
type NewUser struct {
	Email       string
	Username    string
	DisplayName string
	Password    string
	AvatarURL   string // the new user's avatar URL, if known

	// EmailVerificationCode, if given, causes the new user's email address to be unverified until
	// they perform the email verification process and provied this code.
	EmailVerificationCode string `json:"-"` // forbid this field being set by JSON, just in case

	// EmailIsVerified is whether the email address should be considered already verified.
	//
	// 🚨 SECURITY: Only site admins are allowed to create users whose email addresses are initially
	// verified (i.e., with EmailVerificationCode == "").
	EmailIsVerified bool `json:"-"` // forbid this field being set by JSON, just in case

	// FailIfNotInitialUser causes the (users).Create call to return an error and not create the
	// user if at least one of the following is true: (1) the site has already been initialized or
	// (2) any other user account already exists.
	FailIfNotInitialUser bool `json:"-"` // forbid this field being set by JSON, just in case

	// EnforcePasswordLength is whether should enforce minimum and maximum password length requirement.
	// Users created by non-builtin auth providers do not have a password thus no need to check.
	EnforcePasswordLength bool `json:"-"` // forbid this field being set by JSON, just in case
}

// Create creates a new user in the database.
//
// If a password is given, then unauthenticated users can sign into the account using the
// username/email and password. If no password is given, a non-builtin auth provider must be used to
// sign into the account.
//
// CREATION OF SITE ADMINS
//
// The new user is made to be a site admin if the following are both true: (1) this user would be
// the first and only user on the server, and (2) the site has not yet been initialized. Otherwise,
// the user is created as a normal (non-site-admin) user. After the call, the site is marked as
// having been initialized (so that no subsequent (users).Create calls will yield a site
// admin). This is used to create the initial site admin user during site initialization.
//
// It's implemented as part of the (users).Create call instead of relying on the caller to do it in
// order to avoid a race condition where multiple initial site admins could be created or zero site
// admins could be created.
func (u *users) Create(ctx context.Context, info NewUser) (newUser *types.User, err error) {
	if Mocks.Users.Create != nil {
		return Mocks.Users.Create(ctx, info)
	}

	tx, err := dbconn.Global.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			rollErr := tx.Rollback()
			if rollErr != nil {
				err = multierror.Append(err, rollErr)
			}
			return
		}
		err = tx.Commit()
	}()

	return u.create(ctx, tx, info)
}

// maxPasswordRunes is the maximum number of UTF-8 runes that a password can contain.
// This safety limit is to protect us from a DDOS attack caused by hashing very large passwords on Sourcegraph.com.
const maxPasswordRunes = 256

// CheckPasswordLength returns an error if the length of the password is not in the required range.
func CheckPasswordLength(pw string) error {
	pwLen := utf8.RuneCountInString(pw)
	minPasswordRunes := conf.AuthMinPasswordLength()
	if pwLen < minPasswordRunes ||
		pwLen > maxPasswordRunes {
		return errcode.NewPresentationError(fmt.Sprintf("Password may not be less than %d or be more than %d characters.", minPasswordRunes, maxPasswordRunes))
	}
	return nil
}

// create is like Create, except it uses the provided DB transaction. It must execute in a
// transaction because the post-user-creation hooks must run atomically with the user creation.
func (u *users) create(ctx context.Context, tx *sql.Tx, info NewUser) (newUser *types.User, err error) {
	if Mocks.Users.Create != nil {
		return Mocks.Users.Create(ctx, info)
	}

	if info.EnforcePasswordLength {
		if err := CheckPasswordLength(info.Password); err != nil {
			return nil, err
		}
	}

	if info.Email != "" && info.EmailVerificationCode == "" && !info.EmailIsVerified {
		return nil, errors.New("no email verification code provided for new user with unverified email")
	}

	createdAt := time.Now()
	updatedAt := createdAt
	invalidatedSessionsAt := createdAt
	var id int32

	var passwd sql.NullString
	if info.Password == "" {
		passwd = sql.NullString{Valid: false}
	} else {
		// Compute hash of password
		passwd, err = hashPassword(info.Password)
		if err != nil {
			return nil, err
		}
	}

	var avatarURL *string
	if info.AvatarURL != "" {
		avatarURL = &info.AvatarURL
	}

	dbEmailCode := sql.NullString{String: info.EmailVerificationCode}
	dbEmailCode.Valid = info.EmailVerificationCode != ""

	// Creating the initial site admin user is equivalent to initializing the
	// site. ensureInitialized runs in the transaction, so we are guaranteed that the user account
	// creation and site initialization operations occur atomically (to guarantee to the legitimate
	// site admin that if they successfully initialize the server, then no attacker's account could
	// have been created as a site admin).
	alreadyInitialized, err := globalstatedb.EnsureInitialized(ctx, tx)
	if err != nil {
		return nil, err
	}
	if alreadyInitialized && info.FailIfNotInitialUser {
		return nil, errCannotCreateUser{"site_already_initialized"}
	}

	// Run PreCreateUser hook.
	if u.PreCreateUser != nil {
		if err := u.PreCreateUser(ctx); err != nil {
			return nil, err
		}
	}

	var siteAdmin bool
	err = tx.QueryRowContext(
		ctx,
		"INSERT INTO users(username, display_name, avatar_url, created_at, updated_at, passwd, invalidated_sessions_at, site_admin) VALUES($1, $2, $3, $4, $5, $6, $7, $8 AND NOT EXISTS(SELECT * FROM users)) RETURNING id, site_admin",
		info.Username, info.DisplayName, avatarURL, createdAt, updatedAt, passwd, invalidatedSessionsAt, !alreadyInitialized).Scan(&id, &siteAdmin)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Constraint {
			case "users_username":
				return nil, errCannotCreateUser{errorCodeUsernameExists}
			case "users_username_max_length", "users_username_valid_chars", "users_display_name_max_length":
				return nil, errCannotCreateUser{pqErr.Constraint}
			}
		}
		return nil, err
	}
	if info.FailIfNotInitialUser && !siteAdmin {
		// Refuse to make the user the initial site admin if there are other existing users.
		return nil, errCannotCreateUser{"initial_site_admin_must_be_first_user"}
	}

	// Reserve username in shared users+orgs namespace.
	if _, err := tx.ExecContext(ctx, "INSERT INTO names(name, user_id) VALUES($1, $2)", info.Username, id); err != nil {
		return nil, errCannotCreateUser{errorCodeUsernameExists}
	}

	if info.Email != "" {
		var err error
		if info.EmailIsVerified {
			_, err = tx.ExecContext(ctx, "INSERT INTO user_emails(user_id, email, verified_at) VALUES ($1, $2, now())", id, info.Email)
		} else {
			_, err = tx.ExecContext(ctx, "INSERT INTO user_emails(user_id, email, verification_code) VALUES ($1, $2, $3)", id, info.Email, info.EmailVerificationCode)
		}
		if err != nil {
			if pqErr, ok := err.(*pq.Error); ok {
				switch pqErr.Constraint {
				case "user_emails_unique_verified_email":
					return nil, errCannotCreateUser{errorCodeEmailExists}
				}
			}
			return nil, err
		}
	}

	{
		// Run hooks.
		//
		// NOTE: If we need more hooks in the future, we should do something better than just
		// adding random calls here.

		// Ensure the user (all users, actually) is joined to the orgs specified in auth.userOrgMap.
		orgs, errs := orgsForAllUsersToJoin(conf.Get().AuthUserOrgMap)
		for _, err := range errs {
			log15.Warn(err.Error())
		}
		if err := OrgMembers.CreateMembershipInOrgsForAllUsers(ctx, tx, orgs); err != nil {
			return nil, err
		}
	}

	return &types.User{
		ID:                    id,
		Username:              info.Username,
		DisplayName:           info.DisplayName,
		AvatarURL:             info.AvatarURL,
		CreatedAt:             createdAt,
		UpdatedAt:             updatedAt,
		SiteAdmin:             siteAdmin,
		BuiltinAuth:           info.Password != "",
		InvalidatedSessionsAt: invalidatedSessionsAt,
	}, nil
}

// orgsForAllUsersToJoin returns the list of org names that all users should be joined to. The second return value
// is a list of errors encountered while generating this list. Note that even if errors are returned, the first
// return value is still valid.
func orgsForAllUsersToJoin(userOrgMap map[string][]string) ([]string, []error) {
	var errors []error
	for userPattern, orgs := range userOrgMap {
		if userPattern != "*" {
			errors = append(errors, fmt.Errorf("unsupported auth.userOrgMap user pattern %q (only \"*\" is supported)", userPattern))
			continue
		}
		return orgs, errors
	}
	return nil, errors
}

// UserUpdate describes user fields to update.
type UserUpdate struct {
	Username string // update the Username to this value (if non-zero)

	// For the following fields:
	//
	// - If nil, the value in the DB is unchanged.
	// - If pointer to "" (empty string), the value in the DB is set to null.
	// - If pointer to a non-empty string, the value in the DB is set to the string.
	DisplayName, AvatarURL *string
}

// Update updates a user's profile information.
func (u *users) Update(ctx context.Context, id int32, update UserUpdate) error {
	if Mocks.Users.Update != nil {
		return Mocks.Users.Update(id, update)
	}

	tx, err := dbconn.Global.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			rollErr := tx.Rollback()
			if rollErr != nil {
				err = multierror.Append(err, rollErr)
			}
			return
		}
		err = tx.Commit()
	}()

	fieldUpdates := []*sqlf.Query{
		sqlf.Sprintf("updated_at=now()"), // always update updated_at timestamp
	}
	if update.Username != "" {
		fieldUpdates = append(fieldUpdates, sqlf.Sprintf("username=%s", update.Username))

		// Ensure new username is available in shared users+orgs namespace.
		if _, err := tx.ExecContext(ctx, "UPDATE names SET name=$1 WHERE user_id=$2", update.Username, id); err != nil {
			return err
		}
	}
	strOrNil := func(s string) *string {
		if s == "" {
			return nil
		}
		return &s
	}
	if update.DisplayName != nil {
		fieldUpdates = append(fieldUpdates, sqlf.Sprintf("display_name=%s", strOrNil(*update.DisplayName)))
	}
	if update.AvatarURL != nil {
		fieldUpdates = append(fieldUpdates, sqlf.Sprintf("avatar_url=%s", strOrNil(*update.AvatarURL)))
	}
	query := sqlf.Sprintf("UPDATE users SET %s WHERE id=%d", sqlf.Join(fieldUpdates, ", "), id)
	res, err := tx.ExecContext(ctx, query.Query(sqlf.PostgresBindVar), query.Args()...)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Constraint == "users_username" {
			return errCannotCreateUser{errorCodeUsernameExists}
		}
		return err
	}
	nrows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if nrows == 0 {
		return userNotFoundErr{args: []interface{}{id}}
	}
	return nil
}

// Delete performs a soft-delete of the user and all resources associated with this user.
func (u *users) Delete(ctx context.Context, id int32) error {
	if Mocks.Users.Delete != nil {
		return Mocks.Users.Delete(ctx, id)
	}

	// Wrap in transaction because we delete from multiple tables.
	tx, err := dbconn.Global.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			rollErr := tx.Rollback()
			if rollErr != nil {
				err = multierror.Append(err, rollErr)
			}
			return
		}
		err = tx.Commit()
	}()

	res, err := tx.ExecContext(ctx, "UPDATE users SET deleted_at=now() WHERE id=$1 AND deleted_at IS NULL", id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return userNotFoundErr{args: []interface{}{id}}
	}

	// Release the username so it can be used by another user or org.
	if _, err := tx.ExecContext(ctx, "DELETE FROM names WHERE user_id=$1", id); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, "UPDATE access_tokens SET deleted_at=now() WHERE subject_user_id=$1 OR creator_user_id=$1", id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM user_emails WHERE user_id=$1", id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE user_external_accounts SET deleted_at=now() WHERE user_id=$1 AND deleted_at IS NULL", id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE org_invitations SET deleted_at=now() WHERE deleted_at IS NULL AND (sender_user_id=$1 OR recipient_user_id=$1)", id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE registry_extensions SET deleted_at=now() WHERE deleted_at IS NULL AND publisher_user_id=$1", id); err != nil {
		return err
	}

	return nil
}

// HardDelete removes the user and all resources associated with this user.
func (u *users) HardDelete(ctx context.Context, id int32) error {
	if Mocks.Users.HardDelete != nil {
		return Mocks.Users.HardDelete(ctx, id)
	}

	// Wrap in transaction because we delete from multiple tables.
	tx, err := dbconn.Global.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			rollErr := tx.Rollback()
			if rollErr != nil {
				err = multierror.Append(err, rollErr)
			}
			return
		}
		err = tx.Commit()
	}()

	if _, err := tx.ExecContext(ctx, "DELETE FROM names WHERE user_id=$1", id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM access_tokens WHERE subject_user_id=$1 OR creator_user_id=$1", id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM user_emails WHERE user_id=$1", id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM user_external_accounts WHERE user_id=$1", id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM survey_responses WHERE user_id=$1", id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM registry_extension_releases WHERE registry_extension_id IN (SELECT id FROM registry_extensions WHERE publisher_user_id=$1)", id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM registry_extensions WHERE publisher_user_id=$1", id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM org_invitations WHERE sender_user_id=$1 OR recipient_user_id=$1", id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM org_members WHERE user_id=$1", id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM settings WHERE user_id=$1", id); err != nil {
		return err
	}

	// Settings that were merely authored by this user should not be deleted. They may be global or
	// org settings that apply to other users, too. There is currently no way to hard-delete
	// settings for an org or globally, but we can handle those rare cases manually.
	if _, err := tx.ExecContext(ctx, "UPDATE settings SET author_user_id=NULL WHERE author_user_id=$1", id); err != nil {
		return err
	}

	res, err := tx.ExecContext(ctx, "DELETE FROM users WHERE id=$1", id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return userNotFoundErr{args: []interface{}{id}}
	}
	return nil
}

func (u *users) SetIsSiteAdmin(ctx context.Context, id int32, isSiteAdmin bool) error {
	if Mocks.Users.SetIsSiteAdmin != nil {
		return Mocks.Users.SetIsSiteAdmin(id, isSiteAdmin)
	}
	_, err := dbconn.Global.ExecContext(ctx, "UPDATE users SET site_admin=$1 WHERE id=$2", isSiteAdmin, id)
	return err
}

// CheckAndDecrementInviteQuota should be called before the user (identified
// by userID) is allowed to invite any other user. If ok is false, then the
// user is not allowed to invite any other user (either because they've
// invited too many users, or some other error occurred). If the user has
// quota remaining, their quota is decremented and ok is true.
func (u *users) CheckAndDecrementInviteQuota(ctx context.Context, userID int32) (ok bool, err error) {
	if Mocks.Users.CheckAndDecrementInviteQuota != nil {
		return Mocks.Users.CheckAndDecrementInviteQuota(ctx, userID)
	}

	var quotaRemaining int32
	sqlQuery := `
	UPDATE users SET invite_quota=(invite_quota - 1)
	WHERE users.id=$1 AND invite_quota>0 AND deleted_at IS NULL
	RETURNING invite_quota`
	row := dbconn.Global.QueryRowContext(ctx, sqlQuery, userID)
	if err := row.Scan(&quotaRemaining); err == sql.ErrNoRows {
		// It's possible that some other problem occurred, such as the user being deleted,
		// but treat that as a quota exceeded error, too.
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil // the user has remaining quota to send invites
}

func (u *users) GetByID(ctx context.Context, id int32) (*types.User, error) {
	if Mocks.Users.GetByID != nil {
		return Mocks.Users.GetByID(ctx, id)
	}
	return u.getOneBySQL(ctx, "WHERE id=$1 AND deleted_at IS NULL LIMIT 1", id)
}

// GetByVerifiedEmail returns the user (if any) with the specified verified email address. If a user
// has a matching *unverified* email address, they will not be returned by this method. At most one
// user may have any given verified email address.
func (u *users) GetByVerifiedEmail(ctx context.Context, email string) (*types.User, error) {
	if Mocks.Users.GetByVerifiedEmail != nil {
		return Mocks.Users.GetByVerifiedEmail(ctx, email)
	}
	return u.getOneBySQL(ctx, "WHERE id=(SELECT user_id FROM user_emails WHERE email=$1 AND verified_at IS NOT NULL) AND deleted_at IS NULL LIMIT 1", email)
}

func (u *users) GetByUsername(ctx context.Context, username string) (*types.User, error) {
	if Mocks.Users.GetByUsername != nil {
		return Mocks.Users.GetByUsername(ctx, username)
	}

	return u.getOneBySQL(ctx, "WHERE username=$1 AND deleted_at IS NULL LIMIT 1", username)
}

// GetByUsernames returns a list of users by given usernames. The number of results list could be less
// than the candidate list due to no user is associated with some usernames.
func (u *users) GetByUsernames(ctx context.Context, usernames ...string) ([]*types.User, error) {
	if Mocks.Users.GetByUsernames != nil {
		return Mocks.Users.GetByUsernames(ctx, usernames...)
	}

	if len(usernames) == 0 {
		return []*types.User{}, nil
	}

	items := make([]*sqlf.Query, len(usernames))
	for i := range usernames {
		items[i] = sqlf.Sprintf("%s", usernames[i])
	}
	q := sqlf.Sprintf("WHERE username IN (%s) AND deleted_at IS NULL ORDER BY id ASC", sqlf.Join(items, ","))
	return u.getBySQL(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...)
}

var ErrNoCurrentUser = errors.New("no current user")

func (u *users) GetByCurrentAuthUser(ctx context.Context) (*types.User, error) {
	if Mocks.Users.GetByCurrentAuthUser != nil {
		return Mocks.Users.GetByCurrentAuthUser(ctx)
	}

	actor := actor.FromContext(ctx)
	if !actor.IsAuthenticated() {
		return nil, ErrNoCurrentUser
	}
	if dbconn.Global == nil {
		return nil, ErrNoCurrentUser
	}

	return u.getOneBySQL(ctx, "WHERE id=$1 AND deleted_at IS NULL LIMIT 1", actor.UID)
}

func (u *users) InvalidateSessionsByID(ctx context.Context, id int32) error {
	if Mocks.Users.InvalidateSessionsByID != nil {
		return Mocks.Users.InvalidateSessionsByID(ctx, id)
	}

	tx, err := dbconn.Global.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			rollErr := tx.Rollback()
			if rollErr != nil {
				err = multierror.Append(err, rollErr)
			}
			return
		}
		err = tx.Commit()
	}()

	query := sqlf.Sprintf(`
		UPDATE users
		SET
			updated_at=now(),
			invalidated_sessions_at=now()
		WHERE id=%d
		`, id)
	res, err := tx.ExecContext(ctx, query.Query(sqlf.PostgresBindVar), query.Args()...)
	if err != nil {
		return err
	}
	nrows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if nrows == 0 {
		return userNotFoundErr{args: []interface{}{id}}
	}
	return nil
}

func (u *users) Count(ctx context.Context, opt *UsersListOptions) (int, error) {
	if Mocks.Users.Count != nil {
		return Mocks.Users.Count(ctx, opt)
	}

	if opt == nil {
		opt = &UsersListOptions{}
	}
	conds := u.listSQL(*opt)
	q := sqlf.Sprintf("SELECT COUNT(*) FROM users u WHERE %s", sqlf.Join(conds, "AND"))

	var count int
	if err := dbconn.Global.QueryRowContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// UsersListOptions specifies the options for listing users.
type UsersListOptions struct {
	// Query specifies a search query for users.
	Query string
	// UserIDs specifies a list of user IDs to include.
	UserIDs []int32

	Tag string // only include users with this tag

	*LimitOffset
}

func (u *users) List(ctx context.Context, opt *UsersListOptions) (_ []*types.User, err error) {
	if Mocks.Users.List != nil {
		return Mocks.Users.List(ctx, opt)
	}

	tr, ctx := trace.New(ctx, "db.Users.List", fmt.Sprintf("%+v", opt))
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()

	if opt == nil {
		opt = &UsersListOptions{}
	}
	conds := u.listSQL(*opt)

	q := sqlf.Sprintf("WHERE %s ORDER BY id ASC %s", sqlf.Join(conds, "AND"), opt.LimitOffset.SQL())
	return u.getBySQL(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...)
}

// ListDates lists all user's created and deleted dates, used by usage stats.
func (*users) ListDates(ctx context.Context) (dates []types.UserDates, _ error) {
	rows, err := dbconn.Global.QueryContext(ctx, listDatesQuery)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var d types.UserDates

		err := rows.Scan(&d.UserID, &d.CreatedAt, &dbutil.NullTime{Time: &d.DeletedAt})
		if err != nil {
			return nil, err
		}

		dates = append(dates, d)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return dates, nil
}

const listDatesQuery = `
-- source: internal/db/users.go:ListDates
SELECT id, created_at, deleted_at
FROM users
ORDER BY id ASC
`

func (*users) listSQL(opt UsersListOptions) (conds []*sqlf.Query) {
	conds = []*sqlf.Query{sqlf.Sprintf("TRUE")}
	conds = append(conds, sqlf.Sprintf("deleted_at IS NULL"))
	if opt.Query != "" {
		query := "%" + opt.Query + "%"
		conds = append(conds, sqlf.Sprintf("(username ILIKE %s OR display_name ILIKE %s)", query, query))
	}
	if opt.UserIDs != nil {
		if len(opt.UserIDs) == 0 {
			// Must return empty result set.
			conds = append(conds, sqlf.Sprintf("FALSE"))
		} else {
			items := []*sqlf.Query{}
			for _, id := range opt.UserIDs {
				items = append(items, sqlf.Sprintf("%d", id))
			}
			conds = append(conds, sqlf.Sprintf("u.id IN (%s)", sqlf.Join(items, ",")))
		}
	}
	if opt.Tag != "" {
		conds = append(conds, sqlf.Sprintf("%s::text = ANY(u.tags)", opt.Tag))
	}
	return conds
}

func (u *users) getOneBySQL(ctx context.Context, query string, args ...interface{}) (*types.User, error) {
	users, err := u.getBySQL(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	if len(users) != 1 {
		return nil, userNotFoundErr{args}
	}
	return users[0], nil
}

// getBySQL returns users matching the SQL query, if any exist.
func (*users) getBySQL(ctx context.Context, query string, args ...interface{}) ([]*types.User, error) {
	rows, err := dbconn.Global.QueryContext(ctx, "SELECT u.id, u.username, u.display_name, u.avatar_url, u.created_at, u.updated_at, u.site_admin, u.passwd IS NOT NULL, u.tags, u.invalidated_sessions_at FROM users u "+query, args...)
	if err != nil {
		return nil, err
	}

	users := []*types.User{}
	defer rows.Close()
	for rows.Next() {
		var u types.User
		var displayName, avatarURL sql.NullString
		err := rows.Scan(&u.ID, &u.Username, &displayName, &avatarURL, &u.CreatedAt, &u.UpdatedAt, &u.SiteAdmin, &u.BuiltinAuth, pq.Array(&u.Tags), &u.InvalidatedSessionsAt)
		if err != nil {
			return nil, err
		}
		u.DisplayName = displayName.String
		u.AvatarURL = avatarURL.String
		users = append(users, &u)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
