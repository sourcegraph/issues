package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbconn"
)

// DBPublisher is a publisher of extensions to the registry.
type DBPublisher struct {
	UserID, OrgID int32 // exactly 1 is nonzero

	// NonCanonicalExtensionID is the publisher's username (for a user) or name (for an org) as of
	// when the query executed. Do not persist this.
	NonCanonicalName string
}

// IsZero reports whether p is the zero value.
func (p DBPublisher) IsZero() bool { return p == DBPublisher{} }

// publisherNotFoundError occurs when a registry extension publisher is not found.
type publisherNotFoundError struct {
	args []interface{}
}

// NotFound implements errcode.NotFounder.
func (err publisherNotFoundError) NotFound() bool { return true }

func (err publisherNotFoundError) Error() string {
	return fmt.Sprintf("registry extension publisher not found: %v (there is no user or organization with the specified name)", err.args)
}

// DBPublishersListOptions contains options for listing publishers of extensions in the
// registry.
type DBPublishersListOptions struct {
	*database.LimitOffset
}

// ListPublishers lists all publishers of extensions to the registry.
//
// 🚨 SECURITY: The caller must ensure that the actor is permitted to list with the specified
// options.
func (s *DBExtensions) ListPublishers(ctx context.Context, opt DBPublishersListOptions) ([]*DBPublisher, error) {
	return s.listPublishers(ctx, nil, opt.LimitOffset)
}

func (s *DBExtensions) publishersSQLCTE() *sqlf.Query {
	return sqlf.Sprintf(`WITH publishers AS (
  (SELECT DISTINCT ON (publisher_user_id) publisher_user_id AS user_id, NULL AS org_id FROM registry_extensions WHERE publisher_user_id IS NOT NULL AND deleted_at IS NULL)
  UNION
  (SELECT DISTINCT ON (publisher_org_id) NULL AS user_id, publisher_org_id AS org_id FROM registry_extensions WHERE publisher_org_id IS NOT NULL AND deleted_at IS NULL)
) `)
}

func (s *DBExtensions) listPublishers(ctx context.Context, conds []*sqlf.Query, limitOffset *database.LimitOffset) ([]*DBPublisher, error) {
	conds = append(conds, sqlf.Sprintf("TRUE"))
	q := sqlf.Sprintf(`%s
SELECT user_id, org_id, COALESCE(users.username, orgs.name) AS non_canonical_name FROM publishers
LEFT JOIN users ON users.id=user_id
LEFT JOIN orgs ON orgs.id=org_id
WHERE (%s)
ORDER BY org_id ASC NULLS LAST, user_id ASC NULLS LAST
%s`,
		s.publishersSQLCTE(),
		sqlf.Join(conds, ") AND ("),
		limitOffset.SQL(),
	)

	rows, err := dbconn.Global.QueryContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*DBPublisher
	for rows.Next() {
		var t DBPublisher
		var userID, orgID sql.NullInt64
		if err := rows.Scan(&userID, &orgID, &t.NonCanonicalName); err != nil {
			return nil, err
		}
		t.UserID = int32(userID.Int64)
		t.OrgID = int32(orgID.Int64)
		results = append(results, &t)
	}
	return results, nil
}

// CountPublishers counts all registry publishers that satisfy the options (ignoring limit and offset).
//
// 🚨 SECURITY: The caller must ensure that the actor is permitted to count the results.
func (s *DBExtensions) CountPublishers(ctx context.Context, opt DBPublishersListOptions) (int, error) {
	q := sqlf.Sprintf(`%s SELECT COUNT(*) FROM publishers`, s.publishersSQLCTE())
	var count int
	if err := s.QueryRow(ctx, q).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// GePublisher gets the registry publisher with the given name.
func (s *DBExtensions) GetPublisher(ctx context.Context, name string) (*DBPublisher, error) {
	var userID, orgID sql.NullInt64
	var p DBPublisher
	q := sqlf.Sprintf(getPublisherQueryFmtstr, name, name)
	err := s.QueryRow(ctx, q).Scan(&userID, &orgID, &p.NonCanonicalName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, &publisherNotFoundError{[]interface{}{"name", name}}
		}
		return nil, err
	}
	p.UserID = int32(userID.Int64)
	p.OrgID = int32(orgID.Int64)
	return &p, nil
}

const getPublisherQueryFmtstr = `
-- source:enterprise/cmd/frontend/internal/registry/publishers_db.go:GetPublisher
WITH publishers AS (
	(SELECT id AS user_id, NULL AS org_id, username AS non_canonical_name FROM users WHERE username = %s AND deleted_at IS NULL)
	UNION
	(SELECT NULL AS user_id, id AS org_id, name AS non_canonical_name FROM orgs WHERE name = %s AND deleted_at IS NULL)
)
SELECT
	user_id,
	org_id,
	non_canonical_name
FROM publishers
ORDER BY user_id NULLS LAST
LIMIT 1
`
