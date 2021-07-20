package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/jackc/pgconn"
	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

// OrgNotFoundError occurs when an organization is not found.
type OrgNotFoundError struct {
	Message string
}

func (e *OrgNotFoundError) Error() string {
	return fmt.Sprintf("org not found: %s", e.Message)
}

func (e *OrgNotFoundError) NotFound() bool {
	return true
}

var errOrgNameAlreadyExists = errors.New("organization name is already taken (by a user or another organization)")

type OrgStore struct {
	*basestore.Store
}

// Orgs instantiates and returns a new OrgStore with prepared statements.
func Orgs(db dbutil.DB) *OrgStore {
	return &OrgStore{Store: basestore.NewWithDB(db, sql.TxOptions{})}
}

// NewOrgStoreWithDB instantiates and returns a new OrgStore using the other store handle.
func OrgsWith(other basestore.ShareableStore) *OrgStore {
	return &OrgStore{Store: basestore.NewWithHandle(other.Handle())}
}

func (o *OrgStore) With(other basestore.ShareableStore) *OrgStore {
	return &OrgStore{Store: o.Store.With(other)}
}

func (o *OrgStore) Transact(ctx context.Context) (*OrgStore, error) {
	txBase, err := o.Store.Transact(ctx)
	return &OrgStore{Store: txBase}, err
}

// GetByUserID returns a list of all organizations for the user. An empty slice is
// returned if the user is not authenticated or is not a member of any org.
func (o *OrgStore) GetByUserID(ctx context.Context, userID int32) ([]*types.Org, error) {
	rows, err := o.Handle().DB().QueryContext(ctx, "SELECT orgs.id, orgs.name, orgs.display_name,  orgs.created_at, orgs.updated_at FROM org_members LEFT OUTER JOIN orgs ON org_members.org_id = orgs.id WHERE user_id=$1 AND orgs.deleted_at IS NULL", userID)
	if err != nil {
		return []*types.Org{}, err
	}

	orgs := []*types.Org{}
	defer rows.Close()
	for rows.Next() {
		org := types.Org{}
		err := rows.Scan(&org.ID, &org.Name, &org.DisplayName, &org.CreatedAt, &org.UpdatedAt)
		if err != nil {
			return nil, err
		}

		orgs = append(orgs, &org)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return orgs, nil
}

func (o *OrgStore) GetByID(ctx context.Context, orgID int32) (*types.Org, error) {
	if Mocks.Orgs.GetByID != nil {
		return Mocks.Orgs.GetByID(ctx, orgID)
	}
	orgs, err := o.getBySQL(ctx, "WHERE deleted_at IS NULL AND id=$1 LIMIT 1", orgID)
	if err != nil {
		return nil, err
	}
	if len(orgs) == 0 {
		return nil, &OrgNotFoundError{fmt.Sprintf("id %d", orgID)}
	}
	return orgs[0], nil
}

func (o *OrgStore) GetByName(ctx context.Context, name string) (*types.Org, error) {
	if Mocks.Orgs.GetByName != nil {
		return Mocks.Orgs.GetByName(ctx, name)
	}
	orgs, err := o.getBySQL(ctx, "WHERE deleted_at IS NULL AND name=$1 LIMIT 1", name)
	if err != nil {
		return nil, err
	}
	if len(orgs) == 0 {
		return nil, &OrgNotFoundError{fmt.Sprintf("name %s", name)}
	}
	return orgs[0], nil
}

func (o *OrgStore) Count(ctx context.Context, opt OrgsListOptions) (int, error) {
	if Mocks.Orgs.Count != nil {
		return Mocks.Orgs.Count(ctx, opt)
	}

	q := sqlf.Sprintf("SELECT COUNT(*) FROM orgs WHERE %s", o.listSQL(opt))

	var count int
	if err := o.QueryRow(ctx, q).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// OrgsListOptions specifies the options for listing organizations.
type OrgsListOptions struct {
	// Query specifies a search query for organizations.
	Query string

	*LimitOffset
}

func (o *OrgStore) List(ctx context.Context, opt *OrgsListOptions) ([]*types.Org, error) {
	if Mocks.Orgs.List != nil {
		return Mocks.Orgs.List(ctx, opt)
	}

	if opt == nil {
		opt = &OrgsListOptions{}
	}
	q := sqlf.Sprintf("WHERE %s ORDER BY id ASC %s", o.listSQL(*opt), opt.LimitOffset.SQL())
	return o.getBySQL(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...)
}

func (*OrgStore) listSQL(opt OrgsListOptions) *sqlf.Query {
	conds := []*sqlf.Query{sqlf.Sprintf("deleted_at IS NULL")}
	if opt.Query != "" {
		query := "%" + opt.Query + "%"
		conds = append(conds, sqlf.Sprintf("name ILIKE %s OR display_name ILIKE %s", query, query))
	}
	return sqlf.Sprintf("(%s)", sqlf.Join(conds, ") AND ("))
}

func (o *OrgStore) getBySQL(ctx context.Context, query string, args ...interface{}) ([]*types.Org, error) {
	rows, err := o.Handle().DB().QueryContext(ctx, "SELECT id, name, display_name, created_at, updated_at FROM orgs "+query, args...)
	if err != nil {
		return nil, err
	}

	orgs := []*types.Org{}
	defer rows.Close()
	for rows.Next() {
		org := types.Org{}
		err := rows.Scan(&org.ID, &org.Name, &org.DisplayName, &org.CreatedAt, &org.UpdatedAt)
		if err != nil {
			return nil, err
		}

		orgs = append(orgs, &org)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return orgs, nil
}

func (o *OrgStore) Create(ctx context.Context, name string, displayName *string) (newOrg *types.Org, err error) {
	tx, err := o.Transact(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = tx.Done(err)
	}()

	newOrg = &types.Org{
		Name:        name,
		DisplayName: displayName,
	}
	newOrg.CreatedAt = time.Now()
	newOrg.UpdatedAt = newOrg.CreatedAt
	err = tx.Handle().DB().QueryRowContext(
		ctx,
		"INSERT INTO orgs(name, display_name, created_at, updated_at) VALUES($1, $2, $3, $4) RETURNING id",
		newOrg.Name, newOrg.DisplayName, newOrg.CreatedAt, newOrg.UpdatedAt).Scan(&newOrg.ID)
	if err != nil {
		var e *pgconn.PgError
		if errors.As(err, &e) {
			switch e.ConstraintName {
			case "orgs_name":
				return nil, errOrgNameAlreadyExists
			case "orgs_name_max_length", "orgs_name_valid_chars":
				return nil, errors.Errorf("org name invalid: %s", e.ConstraintName)
			case "orgs_display_name_max_length":
				return nil, errors.Errorf("org display name invalid: %s", e.ConstraintName)
			}
		}

		return nil, err
	}

	// Reserve organization name in shared users+orgs namespace.
	if _, err := tx.Handle().DB().ExecContext(ctx, "INSERT INTO names(name, org_id) VALUES($1, $2)", newOrg.Name, newOrg.ID); err != nil {
		return nil, errOrgNameAlreadyExists
	}

	return newOrg, nil
}

func (o *OrgStore) Update(ctx context.Context, id int32, displayName *string) (*types.Org, error) {
	org, err := o.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// NOTE: It is not possible to update an organization's name. If it becomes possible, we need to
	// also update the `names` table to ensure the new name is available in the shared users+orgs
	// namespace.

	if displayName != nil {
		org.DisplayName = displayName
		if _, err := o.Handle().DB().ExecContext(ctx, "UPDATE orgs SET display_name=$1 WHERE id=$2 AND deleted_at IS NULL", org.DisplayName, id); err != nil {
			return nil, err
		}
	}
	org.UpdatedAt = time.Now()
	if _, err := o.Handle().DB().ExecContext(ctx, "UPDATE orgs SET updated_at=$1 WHERE id=$2 AND deleted_at IS NULL", org.UpdatedAt, id); err != nil {
		return nil, err
	}

	return org, nil
}

func (o *OrgStore) Delete(ctx context.Context, id int32) (err error) {
	// Wrap in transaction because we delete from multiple tables.
	tx, err := o.Transact(ctx)
	if err != nil {
		return err
	}
	defer func() {
		err = tx.Done(err)
	}()

	res, err := tx.Handle().DB().ExecContext(ctx, "UPDATE orgs SET deleted_at=now() WHERE id=$1 AND deleted_at IS NULL", id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return &OrgNotFoundError{fmt.Sprintf("id %d", id)}
	}

	// Release the organization name so it can be used by another user or org.
	if _, err := tx.Handle().DB().ExecContext(ctx, "DELETE FROM names WHERE org_id=$1", id); err != nil {
		return err
	}

	if _, err := tx.Handle().DB().ExecContext(ctx, "UPDATE org_invitations SET deleted_at=now() WHERE deleted_at IS NULL AND org_id=$1", id); err != nil {
		return err
	}
	if _, err := tx.Handle().DB().ExecContext(ctx, "UPDATE registry_extensions SET deleted_at=now() WHERE deleted_at IS NULL AND publisher_org_id=$1", id); err != nil {
		return err
	}

	return nil
}
