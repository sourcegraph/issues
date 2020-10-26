package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"

	"github.com/keegancsmith/sqlf"

	"github.com/lib/pq"
)

type orgMembers struct{}

func (*orgMembers) Create(ctx context.Context, orgID, userID int32) (*types.OrgMembership, error) {
	m := types.OrgMembership{
		OrgID:  orgID,
		UserID: userID,
	}
	err := dbconn.Global.QueryRowContext(
		ctx,
		"INSERT INTO org_members(org_id, user_id) VALUES($1, $2) RETURNING id, created_at, updated_at",
		m.OrgID, m.UserID).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Constraint == "org_members_org_id_user_id_key" {
				return nil, errors.New("user is already a member of the organization")
			}
		}
		return nil, err
	}
	return &m, nil
}

func (m *orgMembers) GetByUserID(ctx context.Context, userID int32) ([]*types.OrgMembership, error) {
	return m.getBySQL(ctx, "INNER JOIN users ON org_members.user_id=users.id WHERE org_members.user_id=$1 AND users.deleted_at IS NULL", userID)
}

func (m *orgMembers) GetByOrgIDAndUserID(ctx context.Context, orgID, userID int32) (*types.OrgMembership, error) {
	if Mocks.OrgMembers.GetByOrgIDAndUserID != nil {
		return Mocks.OrgMembers.GetByOrgIDAndUserID(ctx, orgID, userID)
	}
	return m.getOneBySQL(ctx, "INNER JOIN users ON org_members.user_id=users.id WHERE org_id=$1 AND user_id=$2 AND users.deleted_at IS NULL LIMIT 1", orgID, userID)
}

func (*orgMembers) Remove(ctx context.Context, orgID, userID int32) error {
	_, err := dbconn.Global.ExecContext(ctx, "DELETE FROM org_members WHERE (org_id=$1 AND user_id=$2)", orgID, userID)
	return err
}

// GetByOrgID returns a list of all members of a given organization.
func (*orgMembers) GetByOrgID(ctx context.Context, orgID int32) ([]*types.OrgMembership, error) {
	org, err := Orgs.GetByID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return OrgMembers.getBySQL(ctx, "INNER JOIN users ON org_members.user_id = users.id WHERE org_id=$1 AND users.deleted_at IS NULL ORDER BY upper(users.display_name), users.id", org.ID)
}

// ErrOrgMemberNotFound is the error that is returned when
// a user is not in an org.
type ErrOrgMemberNotFound struct {
	args []interface{}
}

func (err *ErrOrgMemberNotFound) Error() string {
	return fmt.Sprintf("org member not found: %v", err.args)
}

func (ErrOrgMemberNotFound) NotFound() bool { return true }

func (m *orgMembers) getOneBySQL(ctx context.Context, query string, args ...interface{}) (*types.OrgMembership, error) {
	members, err := m.getBySQL(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	if len(members) != 1 {
		return nil, &ErrOrgMemberNotFound{args}
	}
	return members[0], nil
}

func (*orgMembers) getBySQL(ctx context.Context, query string, args ...interface{}) ([]*types.OrgMembership, error) {
	rows, err := dbconn.Global.QueryContext(ctx, "SELECT org_members.id, org_members.org_id, org_members.user_id, org_members.created_at, org_members.updated_at FROM org_members "+query, args...)
	if err != nil {
		return nil, err
	}

	members := []*types.OrgMembership{}
	defer rows.Close()
	for rows.Next() {
		m := types.OrgMembership{}
		err := rows.Scan(&m.ID, &m.OrgID, &m.UserID, &m.CreatedAt, &m.UpdatedAt)
		if err != nil {
			return nil, err
		}
		members = append(members, &m)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return members, nil
}

// CreateMembershipInOrgsForAllUsers causes *ALL* users to become members of every org in the
// orgNames list.
//
// The provided dbh is used as the DB handle to execute the query. It may be either a global
// DB handle or a transaction. If nil, the global DB handle is used.
func (*orgMembers) CreateMembershipInOrgsForAllUsers(ctx context.Context, dbh interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}, orgNames []string) error {
	if len(orgNames) == 0 {
		return nil
	}

	orgNameVars := []*sqlf.Query{}
	for _, orgName := range orgNames {
		orgNameVars = append(orgNameVars, sqlf.Sprintf("%s", orgName))
	}

	sqlQuery := sqlf.Sprintf(`
			WITH org_ids AS (SELECT id FROM orgs WHERE name IN (%s)),
				 user_ids AS (SELECT id FROM users WHERE deleted_at IS NULL),
				 to_join AS (SELECT org_ids.id AS org_id, user_ids.id AS user_id
						  FROM org_ids join user_ids ON true
						  LEFT JOIN org_members ON org_members.org_id=org_ids.id AND
									org_members.user_id=user_ids.id
						  WHERE org_members.id is null)
			INSERT INTO org_members(org_id,user_id) SELECT to_join.org_id, to_join.user_id FROM to_join;`,
		sqlf.Join(orgNameVars, ","))

	if dbh == nil {
		dbh = dbconn.Global
	}
	_, err := dbh.ExecContext(ctx, sqlQuery.Query(sqlf.PostgresBindVar), sqlQuery.Args()...)
	return err
}
