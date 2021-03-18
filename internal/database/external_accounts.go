package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/keegancsmith/sqlf"
	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"

	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/encryption"
	"github.com/sourcegraph/sourcegraph/internal/encryption/keyring"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/trace"
)

// userExternalAccountNotFoundError is the error that is returned when a user external account is not found.
type userExternalAccountNotFoundError struct {
	args []interface{}
}

func (err userExternalAccountNotFoundError) Error() string {
	return fmt.Sprintf("user external account not found: %v", err.args)
}

func (err userExternalAccountNotFoundError) NotFound() bool {
	return true
}

// UserExternalAccountsStore provides access to the `user_external_accounts` table.
type UserExternalAccountsStore struct {
	*basestore.Store
	once sync.Once

	key encryption.Key
}

// ExternalAccounts instantiates and returns a new UserExternalAccountsStore with prepared statements.
func ExternalAccounts(db dbutil.DB) *UserExternalAccountsStore {
	return &UserExternalAccountsStore{Store: basestore.NewWithDB(db, sql.TxOptions{})}
}

// ExternalAccountsWith instantiates and returns a new UserExternalAccountsStore using the other store handle.
func ExternalAccountsWith(other basestore.ShareableStore) *UserExternalAccountsStore {
	return &UserExternalAccountsStore{Store: basestore.NewWithHandle(other.Handle())}
}

func (s *UserExternalAccountsStore) With(other basestore.ShareableStore) *UserExternalAccountsStore {
	return &UserExternalAccountsStore{Store: s.Store.With(other), key: s.key}
}

func (s *UserExternalAccountsStore) WithEncryptionKey(key encryption.Key) *UserExternalAccountsStore {
	return &UserExternalAccountsStore{Store: s.Store, key: key}
}

func (s *UserExternalAccountsStore) Transact(ctx context.Context) (*UserExternalAccountsStore, error) {
	s.ensureStore()

	txBase, err := s.Store.Transact(ctx)
	return &UserExternalAccountsStore{Store: txBase, key: s.key}, err
}

// ensureStore instantiates a basestore.Store if necessary, using the dbconn.Global handle.
// This function ensures access to dbconn happens after the rest of the code or tests have
// initialized it.
func (s *UserExternalAccountsStore) ensureStore() {
	s.once.Do(func() {
		if s.Store == nil {
			s.Store = basestore.NewWithDB(dbconn.Global, sql.TxOptions{})
		}
	})
}

type encryptedAccountData struct {
	AuthData, Data string
}

func (s *UserExternalAccountsStore) getEncryptionKey() encryption.Key {
	if s.key != nil {
		return s.key
	}
	return keyring.Default().UserExternalAccountKey
}

// maybeEncryptAccountData encrypts and returns user auth data if an encryption.Key is configured
func (s *UserExternalAccountsStore) maybeEncryptAccountData(ctx context.Context, data extsvc.AccountData) (*encryptedAccountData, string, error) {
	var enc encryptedAccountData
	var keyID string
	var err error

	if data.AuthData != nil {
		enc.AuthData, keyID, err = MaybeEncrypt(ctx, s.getEncryptionKey(), string(*data.AuthData))
		if err != nil {
			return nil, "", err
		}
	}

	if data.Data != nil {
		enc.Data, keyID, err = MaybeEncrypt(ctx, s.getEncryptionKey(), string(*data.Data))
		if err != nil {
			return nil, "", err
		}
	}

	return &enc, keyID, nil
}

func (s *UserExternalAccountsStore) maybeDecryptAccountData(ctx context.Context, data *encryptedAccountData, keyIdent string) (extsvc.AccountData, error) {
	var acc extsvc.AccountData

	decryptedAuthData, err := MaybeDecrypt(ctx, s.getEncryptionKey(), data.AuthData, keyIdent)
	if err != nil {
		return extsvc.AccountData{}, err
	}
	decryptedData, err := MaybeDecrypt(ctx, s.getEncryptionKey(), data.Data, keyIdent)
	if err != nil {
		return extsvc.AccountData{}, err
	}

	if decryptedAuthData != "" {
		jAuthData := json.RawMessage(decryptedAuthData)
		acc.AuthData = &jAuthData
	}
	if decryptedData != "" {
		jData := json.RawMessage(decryptedData)
		acc.Data = &jData
	}

	return acc, nil
}

// Get gets information about the user external account.
func (s *UserExternalAccountsStore) Get(ctx context.Context, id int32) (*extsvc.Account, error) {
	if Mocks.ExternalAccounts.Get != nil {
		return Mocks.ExternalAccounts.Get(id)
	}
	return s.getBySQL(ctx, sqlf.Sprintf("WHERE id=%d AND deleted_at IS NULL LIMIT 1", id))
}

// LookupUserAndSave is used for authenticating a user (when both their Sourcegraph account and the
// association with the external account already exist).
//
// It looks up the existing user associated with the external account's extsvc.AccountSpec. If
// found, it updates the account's data and returns the user. It NEVER creates a user; you must call
// CreateUserAndSave for that.
func (s *UserExternalAccountsStore) LookupUserAndSave(ctx context.Context, spec extsvc.AccountSpec, data extsvc.AccountData) (userID int32, err error) {
	if Mocks.ExternalAccounts.LookupUserAndSave != nil {
		return Mocks.ExternalAccounts.LookupUserAndSave(spec, data)
	}
	s.ensureStore()

	encAuthdata, keyIdent, err := s.maybeEncryptAccountData(ctx, data)
	if err != nil {
		return 0, err
	}

	err = s.Handle().DB().QueryRowContext(ctx, `
-- source: internal/database/external_accounts.go:UserExternalAccountsStore.LookupUserAndSave
UPDATE user_external_accounts
SET
	auth_data = $5,
	account_data = $6,
	encryption_key_id = $7,
	updated_at = now(),
	expired_at = NULL
WHERE
	service_type = $1
AND service_id = $2
AND client_id = $3
AND account_id = $4
AND deleted_at IS NULL
RETURNING user_id
`, spec.ServiceType, spec.ServiceID, spec.ClientID, spec.AccountID, encAuthdata.AuthData, encAuthdata.Data, keyIdent).Scan(&userID)
	if err == sql.ErrNoRows {
		err = userExternalAccountNotFoundError{[]interface{}{spec}}
	}
	return userID, err
}

// AssociateUserAndSave is used for linking a new, additional external account with an existing
// Sourcegraph account.
//
// It creates a user external account and associates it with the specified user. If the external
// account already exists and is associated with:
//
// - the same user: it updates the data and returns a nil error; or
// - a different user: it performs no update and returns a non-nil error
func (s *UserExternalAccountsStore) AssociateUserAndSave(ctx context.Context, userID int32, spec extsvc.AccountSpec, data extsvc.AccountData) (err error) {
	if Mocks.ExternalAccounts.AssociateUserAndSave != nil {
		return Mocks.ExternalAccounts.AssociateUserAndSave(userID, spec, data)
	}
	s.ensureStore()

	// This "upsert" may cause us to return an ephemeral failure due to a race condition, but it
	// won't result in inconsistent data.  Wrap in transaction.

	tx, err := s.Transact(ctx)
	if err != nil {
		return err
	}
	defer func() { err = tx.Done(err) }()

	// Find whether the account exists and, if so, which user ID the account is associated with.
	var exists bool
	var existingID, associatedUserID int32
	err = tx.QueryRow(ctx, sqlf.Sprintf(`
-- source: internal/database/external_accounts.go:UserExternalAccountsStore.AssociateUserAndSave
SELECT id, user_id
FROM user_external_accounts
WHERE
	service_type = %s
AND service_id = %s
AND client_id = %s
AND account_id = %s
AND deleted_at IS NULL
`, spec.ServiceType, spec.ServiceID, spec.ClientID, spec.AccountID)).Scan(&existingID, &associatedUserID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	exists = err != sql.ErrNoRows
	err = nil

	if exists && associatedUserID != userID {
		// The account already exists and is associated with another user.
		return fmt.Errorf("unable to change association of external account from user %d to user %d (delete the external account and then try again)", associatedUserID, userID)
	}

	if !exists {
		// Create the external account (it doesn't yet exist).
		return tx.insert(ctx, userID, spec, data)
	}

	encAuthdata, keyIdent, err := s.maybeEncryptAccountData(ctx, data)
	if err != nil {
		return err
	}

	// Update the external account (it exists).
	res, err := tx.ExecResult(ctx, sqlf.Sprintf(`
-- source: internal/database/external_accounts.go:UserExternalAccountsStore.AssociateUserAndSave
UPDATE user_external_accounts
SET
	auth_data = %s,
	account_data = %s,
	encryption_key_id = %s,
	updated_at = now(),
	expired_at = NULL
WHERE
	service_type = %s
AND service_id = %s
AND client_id = %s
AND account_id = %s
AND user_id = %s
AND deleted_at IS NULL
`, encAuthdata.AuthData, encAuthdata.Data, keyIdent, spec.ServiceType, spec.ServiceID, spec.ClientID, spec.AccountID, userID))
	if err != nil {
		return err
	}
	nrows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if nrows == 0 {
		return userExternalAccountNotFoundError{[]interface{}{existingID}}
	}
	return nil
}

// CreateUserAndSave is used to create a new Sourcegraph user account from an external account
// (e.g., "signup from SAML").
//
// It creates a new user and associates it with the specified external account. If the user to
// create already exists, it returns an error.
func (s *UserExternalAccountsStore) CreateUserAndSave(ctx context.Context, newUser NewUser, spec extsvc.AccountSpec, data extsvc.AccountData) (createdUserID int32, err error) {
	if Mocks.ExternalAccounts.CreateUserAndSave != nil {
		return Mocks.ExternalAccounts.CreateUserAndSave(newUser, spec, data)
	}

	tx, err := s.Transact(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { err = tx.Done(err) }()

	createdUser, err := UsersWith(tx).create(ctx, newUser)
	if err != nil {
		return 0, err
	}

	err = tx.insert(ctx, createdUser.ID, spec, data)
	return createdUser.ID, err
}

func (s *UserExternalAccountsStore) insert(ctx context.Context, userID int32, spec extsvc.AccountSpec, data extsvc.AccountData) error {
	encAuthdata, keyIdent, err := s.maybeEncryptAccountData(ctx, data)
	if err != nil {
		return err
	}

	return s.Exec(ctx, sqlf.Sprintf(`
-- source: internal/database/external_accounts.go:UserExternalAccountsStore.insert
INSERT INTO user_external_accounts (user_id, service_type, service_id, client_id, account_id, auth_data, account_data, encryption_key_id)
VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
`, userID, spec.ServiceType, spec.ServiceID, spec.ClientID, spec.AccountID, encAuthdata.AuthData, encAuthdata.Data, keyIdent))
}

// TouchExpired sets the given user external account to be expired now.
func (s *UserExternalAccountsStore) TouchExpired(ctx context.Context, id int32) error {
	if Mocks.ExternalAccounts.TouchExpired != nil {
		return Mocks.ExternalAccounts.TouchExpired(ctx, id)
	}
	s.ensureStore()

	_, err := s.Handle().DB().ExecContext(ctx, `
-- source: internal/database/external_accounts.go:UserExternalAccountsStore.TouchExpired
UPDATE user_external_accounts
SET expired_at = now()
WHERE id = $1
`, id)
	return err
}

// TouchLastValid sets last valid time of the given user external account to be now.
func (s *UserExternalAccountsStore) TouchLastValid(ctx context.Context, id int32) error {
	if Mocks.ExternalAccounts.TouchLastValid != nil {
		return Mocks.ExternalAccounts.TouchLastValid(ctx, id)
	}
	s.ensureStore()

	_, err := s.Handle().DB().ExecContext(ctx, `
-- source: internal/database/external_accounts.go:UserExternalAccountsStore.TouchLastValid
UPDATE user_external_accounts
SET
	expired_at = NULL,
	last_valid_at = now()
WHERE id = $1
`, id)
	return err
}

// Delete deletes a user external account.
func (s *UserExternalAccountsStore) Delete(ctx context.Context, id int32) error {
	if Mocks.ExternalAccounts.Delete != nil {
		return Mocks.ExternalAccounts.Delete(id)
	}
	s.ensureStore()

	res, err := s.Handle().DB().ExecContext(ctx, "UPDATE user_external_accounts SET deleted_at=now() WHERE id=$1 AND deleted_at IS NULL", id)
	if err != nil {
		return err
	}
	nrows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if nrows == 0 {
		return userExternalAccountNotFoundError{[]interface{}{id}}
	}
	return nil
}

// ExternalAccountsListOptions specifies the options for listing user external accounts.
type ExternalAccountsListOptions struct {
	UserID                           int32
	ServiceType, ServiceID, ClientID string
	AccountID                        int64
	ExcludeExpired                   bool
	*LimitOffset
}

func (s *UserExternalAccountsStore) List(ctx context.Context, opt ExternalAccountsListOptions) (acct []*extsvc.Account, err error) {
	if Mocks.ExternalAccounts.List != nil {
		return Mocks.ExternalAccounts.List(opt)
	}
	s.ensureStore()

	tr, ctx := trace.New(ctx, "UserExternalAccountsStore.List", "")
	defer func() {
		if err != nil {
			tr.SetError(err)
		}

		tr.LogFields(
			otlog.Object("opt", opt),
			otlog.Int("accounts.count", len(acct)),
		)

		tr.Finish()
	}()

	conds := s.listSQL(opt)
	return s.listBySQL(ctx, sqlf.Sprintf("WHERE %s ORDER BY id ASC %s", sqlf.Join(conds, "AND"), opt.LimitOffset.SQL()))
}

func (s *UserExternalAccountsStore) Count(ctx context.Context, opt ExternalAccountsListOptions) (int, error) {
	if Mocks.ExternalAccounts.Count != nil {
		return Mocks.ExternalAccounts.Count(opt)
	}
	s.ensureStore()

	conds := s.listSQL(opt)
	q := sqlf.Sprintf("SELECT COUNT(*) FROM user_external_accounts WHERE %s", sqlf.Join(conds, "AND"))
	var count int
	err := s.QueryRow(ctx, q).Scan(&count)
	return count, err
}

func (s *UserExternalAccountsStore) getBySQL(ctx context.Context, querySuffix *sqlf.Query) (*extsvc.Account, error) {
	s.ensureStore()
	results, err := s.listBySQL(ctx, querySuffix)
	if err != nil {
		return nil, err
	}
	if len(results) != 1 {
		return nil, userExternalAccountNotFoundError{querySuffix.Args()}
	}
	return results[0], nil
}

func (s *UserExternalAccountsStore) listBySQL(ctx context.Context, querySuffix *sqlf.Query) ([]*extsvc.Account, error) {
	s.ensureStore()
	q := sqlf.Sprintf(`SELECT t.id, t.user_id, t.service_type, t.service_id, t.client_id, t.account_id, t.auth_data, t.account_data, t.created_at, t.updated_at, t.encryption_key_id FROM user_external_accounts t %s`, querySuffix)
	rows, err := s.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*extsvc.Account
	for rows.Next() {
		var acct extsvc.Account
		var keyIdent string
		var authData, data sql.NullString
		if err := rows.Scan(
			&acct.ID, &acct.UserID,
			&acct.ServiceType, &acct.ServiceID, &acct.ClientID, &acct.AccountID,
			&authData, &data,
			&acct.CreatedAt, &acct.UpdatedAt,
			&keyIdent,
		); err != nil {
			return nil, err
		}

		var encAuthdata encryptedAccountData
		if authData.Valid {
			encAuthdata.AuthData = authData.String
		}
		if data.Valid {
			encAuthdata.Data = data.String
		}

		acct.AccountData, err = s.maybeDecryptAccountData(ctx, &encAuthdata, keyIdent)
		if err != nil {
			return nil, err
		}

		results = append(results, &acct)
	}
	return results, rows.Err()
}

func (s *UserExternalAccountsStore) listSQL(opt ExternalAccountsListOptions) (conds []*sqlf.Query) {
	conds = []*sqlf.Query{sqlf.Sprintf("deleted_at IS NULL")}

	if opt.UserID != 0 {
		conds = append(conds, sqlf.Sprintf("user_id=%d", opt.UserID))
	}
	if opt.ServiceType != "" || opt.ServiceID != "" || opt.ClientID != "" {
		conds = append(conds, sqlf.Sprintf("(service_type=%s AND service_id=%s AND client_id=%s)", opt.ServiceType, opt.ServiceID, opt.ClientID))
	}
	if opt.AccountID != 0 {
		conds = append(conds, sqlf.Sprintf("account_id=%d", strconv.Itoa(int(opt.AccountID))))
	}
	if opt.ExcludeExpired {
		conds = append(conds, sqlf.Sprintf("expired_at IS NULL"))
	}

	return conds
}

// MockExternalAccounts mocks the Stores.ExternalAccounts DB store.
type MockExternalAccounts struct {
	Get                  func(id int32) (*extsvc.Account, error)
	LookupUserAndSave    func(extsvc.AccountSpec, extsvc.AccountData) (userID int32, err error)
	AssociateUserAndSave func(userID int32, spec extsvc.AccountSpec, data extsvc.AccountData) error
	CreateUserAndSave    func(NewUser, extsvc.AccountSpec, extsvc.AccountData) (createdUserID int32, err error)
	Delete               func(id int32) error
	List                 func(ExternalAccountsListOptions) ([]*extsvc.Account, error)
	Count                func(ExternalAccountsListOptions) (int, error)
	TouchExpired         func(ctx context.Context, id int32) error
	TouchLastValid       func(ctx context.Context, id int32) error
}

// MaybeEncrypt encrypts data with the given key returns the id of the key. If the key is nil, it returns the data unchanged.
func MaybeEncrypt(ctx context.Context, key encryption.Key, data string) (maybeEncryptedData, keyID string, err error) {
	var keyIdent string

	if key != nil {
		encrypted, err := key.Encrypt(ctx, []byte(data))
		if err != nil {
			return "", "", err
		}
		data = string(encrypted)
		keyIdent, err = key.ID(ctx)
		if err != nil {
			return "", "", err
		}
	}

	return data, keyIdent, nil
}

// MaybeDecrypt decrypts data with the given key if keyIdent is not empty.
func MaybeDecrypt(ctx context.Context, key encryption.Key, data, keyIdent string) (string, error) {
	if keyIdent == "" {
		// data is not encrypted, return plaintext
		return data, nil
	}
	if key == nil {
		return data, fmt.Errorf("couldn't decrypt encrypted data, key is nil")
	}
	decrypted, err := key.Decrypt(ctx, []byte(data))
	if err != nil {
		return data, err
	}

	return decrypted.Secret(), nil
}

// ExternalServiceConfigMigrator is a background job that encrypts
// external services config on startup.
// It periodically waits until a keyring is configured to determine
// how many services it must migrate.
// Scheduling and progress report is deleguated to the out of band
// migration package.
// The migration is non destructive and can be reverted.
type ExternalAccountsMigrator struct {
	store     *basestore.Store
	BatchSize int
}

func NewExternalAccountsMigrator(store *basestore.Store) *ExternalAccountsMigrator {
	// not locking too many external accounts at a time to prevent congestion
	return &ExternalAccountsMigrator{store: store, BatchSize: 50}
}

func NewExternalAccountsMigratorWithDB(db dbutil.DB) *ExternalAccountsMigrator {
	return NewExternalAccountsMigrator(basestore.NewWithDB(db, sql.TxOptions{}))
}

// ID of the migration row in the out_of_band_migrations table.
func (m *ExternalAccountsMigrator) ID() int {
	return 4
}

// Progress returns a value from 0 to 1 representing the percentage of configuration already migrated.
func (m *ExternalAccountsMigrator) Progress(ctx context.Context) (float64, error) {
	progress, _, err := basestore.ScanFirstFloat(m.store.Query(ctx, sqlf.Sprintf(`
		SELECT
			CASE c2.count WHEN 0 THEN 1 ELSE
				CAST(c1.count AS float) / CAST(c2.count AS float)
			END
		FROM
			(SELECT COUNT(*) AS count FROM user_external_accounts WHERE encryption_key_id != '') c1,
			(SELECT COUNT(*) AS count FROM user_external_accounts) c2
	`)))
	return progress, err
}

// Up loads BatchSize external accounts, locks them, and encrypts their config using the
// key returned by keyring.Default().
// If there is no ring, it will periodically try again until the key is setup in the config.
// Up ensures the configuration can be decrypted with the same key before overwitting it.
// The key id is stored alongside the encrypted configuration.
func (m *ExternalAccountsMigrator) Up(ctx context.Context) (err error) {
	key := keyring.Default().UserExternalAccountKey
	if key == nil {
		return nil
	}

	keyIdent, err := key.ID(ctx)
	if err != nil {
		return err
	}

	tx, err := m.store.Transact(ctx)
	if err != nil {
		return err
	}
	defer func() { err = tx.Done(err) }()

	store := ExternalAccountsWith(tx)
	accounts, err := store.listBySQL(ctx, sqlf.Sprintf("WHERE encryption_key_id = '' ORDER BY id ASC LIMIT %s FOR UPDATE SKIP LOCKED", m.BatchSize))
	if err != nil {
		return err
	}

	for _, acc := range accounts {
		var encAuthData, encData []byte
		if acc.AuthData != nil {
			encAuthData, err = key.Encrypt(ctx, []byte(*acc.AuthData))
			if err != nil {
				return err
			}

			// ensure encryption round-trip is valid
			decrypted, err := key.Decrypt(ctx, encAuthData)
			if err != nil {
				return err
			}
			if decrypted.Secret() != string(*acc.AuthData) {
				return errors.New("invalid encryption round-trip")
			}
		}

		if acc.Data != nil {
			encData, err = key.Encrypt(ctx, []byte(*acc.Data))
			if err != nil {
				return err
			}

			// ensure encryption round-trip is valid
			decrypted, err := key.Decrypt(ctx, encData)
			if err != nil {
				return err
			}
			if decrypted.Secret() != string(*acc.Data) {
				return errors.New("invalid encryption round-trip")
			}
		}

		if err := tx.Exec(ctx, sqlf.Sprintf(
			"UPDATE user_external_accounts SET auth_data = %s, account_data = %s, encryption_key_id = %s WHERE id = %d",
			string(encAuthData),
			string(encData),
			keyIdent,
			acc.ID,
		)); err != nil {
			return err
		}
	}

	return nil
}

func (m *ExternalAccountsMigrator) Down(ctx context.Context) (err error) {
	key := keyring.Default().UserExternalAccountKey
	if key == nil {
		return nil
	}

	// For records that were encrypted, we need to decrypt the configuration,
	// store it in plain text and remove the encryption_key_id.
	tx, err := m.store.Transact(ctx)
	if err != nil {
		return err
	}
	defer func() { err = tx.Done(err) }()

	store := ExternalAccountsWith(tx)
	accounts, err := store.listBySQL(ctx, sqlf.Sprintf("WHERE encryption_key_id != '' ORDER BY id ASC LIMIT %s FOR UPDATE SKIP LOCKED", m.BatchSize))
	if err != nil {
		return err
	}

	for _, acc := range accounts {
		if err := tx.Exec(ctx, sqlf.Sprintf(
			"UPDATE user_external_accounts SET auth_data = %s, account_data = %s, encryption_key_id = '' WHERE id = %s",
			acc.AuthData,
			acc.Data,
			acc.ID,
		)); err != nil {
			return err
		}
	}

	return nil
}
