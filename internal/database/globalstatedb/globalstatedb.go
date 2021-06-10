package globalstatedb

import (
	"context"
	"database/sql"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgconn"
	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
)

type State struct {
	SiteID      string
	Initialized bool // whether the initial site admin account has been created
}

func Get(ctx context.Context, db dbutil.DB) (*State, error) {
	if Mock.Get != nil {
		return Mock.Get(ctx)
	}

	configuration, err := getConfiguration(ctx, db)
	if err == nil {
		return configuration, nil
	}

	if err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "getConfiguration")
	}

	err = tryInsertNew(ctx, db)
	if err != nil {
		return nil, err
	}
	return getConfiguration(ctx, db)
}

func SiteInitialized(ctx context.Context, db dbutil.DB) (alreadyInitialized bool, err error) {
	if err := db.QueryRowContext(ctx, `SELECT initialized FROM global_state LIMIT 1`).Scan(&alreadyInitialized); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return alreadyInitialized, err
}

// EnsureInitialized ensures the site is marked as having been initialized. If the site was already
// initialized, it does nothing. It returns whether the site was already initialized prior to the
// call.
//
// 🚨 SECURITY: Initialization is an important security measure. If a new account is created on a
// site that is not initialized, and no other accounts exist, it is granted site admin
// privileges. If the site *has* been initialized, then a new account is not granted site admin
// privileges (even if all other users are deleted). This reduces the risk of (1) a site admin
// accidentally deleting all user accounts and opening up their site to any attacker becoming a site
// admin and (2) a bug in user account creation code letting attackers create site admin accounts.
func EnsureInitialized(ctx context.Context, db dbutil.DB) (alreadyInitialized bool, err error) {
	dbh := basestore.NewHandleWithDB(db, sql.TxOptions{})
	tx, err := dbh.Transact(ctx)
	if err != nil {
		return false, err
	}
	defer func() {
		err = tx.Done(err)
	}()
	if err := tryInsertNew(ctx, tx.DB()); err != nil {
		return false, err
	}

	// The "SELECT ... FOR UPDATE" prevents a race condition where two calls, each in their own transaction,
	// would see this initialized value as false and then set it to true below.
	q := sqlf.Sprintf(`SELECT initialized FROM global_state FOR UPDATE LIMIT 1`)
	if err := tx.DB().QueryRowContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...).Scan(&alreadyInitialized); err != nil {
		return false, err
	}

	if !alreadyInitialized {
		_, err = tx.DB().ExecContext(ctx, sqlf.Sprintf("UPDATE global_state SET initialized=true").Query(sqlf.PostgresBindVar))
	}

	return alreadyInitialized, err
}

func getConfiguration(ctx context.Context, db dbutil.DB) (*State, error) {
	configuration := &State{}
	err := db.QueryRowContext(ctx, "SELECT site_id, initialized FROM global_state LIMIT 1").Scan(
		&configuration.SiteID,
		&configuration.Initialized,
	)
	return configuration, err
}

func tryInsertNew(ctx context.Context, db dbutil.DB) error {
	siteID, err := uuid.NewRandom()
	if err != nil {
		return err
	}
	// In the normal case (when no users exist yet because the instance is brand new), create the row
	// with initialized=false.
	//
	// If any users exist, then set the site as initialized so that the init screen doesn't show
	// up. (It would not let the visitor initialize the site anyway, because other users exist.) The
	// most likely reason the instance would get into this state (uninitialized but has users) is
	// because previously global state had a siteID and now we ignore that (or someone ran `DELETE
	// FROM global_state;` in the PostgreSQL database). In either case, it's safe to generate a new
	// site ID and set the site as initialized.
	q := sqlf.Sprintf(`
	INSERT INTO global_state(
		site_id,
		initialized
	) values(
		%s,
		EXISTS (SELECT 1 FROM users WHERE deleted_at IS NULL)
	);`, siteID)
	_, err = db.ExecContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.ConstraintName == "global_state_pkey" {
				// The row we were trying to insert already exists.
				// Don't treat this as an error.
				err = nil
			}
		}
	}
	return err
}
