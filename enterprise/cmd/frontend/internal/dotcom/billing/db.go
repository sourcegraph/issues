package billing

import (
	"context"
	"database/sql"

	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
)

// dbBilling provides billing-related database operations.
type dbBilling struct{}

// getUserBillingCustomerID gets the billing customer ID (if any) for a user.
//
// If a transaction tx is provided, the query is executed using the transaction. If tx is nil, the
// global DB handle is used.
func (dbBilling) getUserBillingCustomerID(ctx context.Context, tx *sql.Tx, userID int32) (billingCustomerID *string, err error) {
	var dbh dbh
	if tx != nil {
		dbh = tx
	} else {
		dbh = dbconn.Global
	}

	query := sqlf.Sprintf("SELECT billing_customer_id FROM users WHERE id=%d AND deleted_at IS NULL", userID)
	err = dbh.QueryRowContext(ctx, query.Query(sqlf.PostgresBindVar), query.Args()...).Scan(&billingCustomerID)
	if err == sql.ErrNoRows {
		return nil, db.NewUserNotFoundError(userID)
	}
	return billingCustomerID, err
}

// setUserBillingCustomerID sets or unsets the billing customer ID for a user.
//
// If a transaction tx is provided, the query is executed using the transaction. If tx is nil, the
// global DB handle is used.
func (dbBilling) setUserBillingCustomerID(ctx context.Context, tx *sql.Tx, userID int32, billingCustomerID *string) error {
	var dbh dbh
	if tx != nil {
		dbh = tx
	} else {
		dbh = dbconn.Global
	}

	query := sqlf.Sprintf("UPDATE users SET billing_customer_id=%s WHERE id=%d AND deleted_at IS NULL", billingCustomerID, userID)
	res, err := dbh.ExecContext(ctx, query.Query(sqlf.PostgresBindVar), query.Args()...)
	if err != nil {
		return err
	}
	nrows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if nrows == 0 {
		return db.NewUserNotFoundError(userID)
	}
	return nil
}

type dbh interface {
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
}
