package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/RoaringBitmap/roaring"
	"github.com/keegancsmith/sqlf"
	"github.com/lib/pq"
	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/authz"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/db/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/trace"
)

var ErrPermsUpdatedAtNotSet = errors.New("permissions UpdatedAt timestamp must be set")

// PermsStore is the unified interface for managing permissions explicitly in the database.
// It is concurrency-safe and maintains data consistency over the 'user_permissions',
// 'repo_permissions', 'user_pending_permissions', and 'repo_pending_permissions' tables.
type PermsStore struct {
	db    dbutil.DB
	clock func() time.Time
}

// NewPermsStore returns a new PermsStore with given parameters.
func NewPermsStore(db dbutil.DB, clock func() time.Time) *PermsStore {
	return &PermsStore{
		db:    db,
		clock: clock,
	}
}

// LoadUserPermissions loads stored user permissions into p. An ErrPermsNotFound is returned
// when there are no valid permissions available.
func (s *PermsStore) LoadUserPermissions(ctx context.Context, p *authz.UserPermissions) (err error) {
	if Mocks.Perms.LoadUserPermissions != nil {
		return Mocks.Perms.LoadUserPermissions(ctx, p)
	}

	ctx, save := s.observe(ctx, "LoadUserPermissions", "")
	defer func() { save(&err, p.TracingFields()...) }()

	vals, err := s.load(ctx, loadUserPermissionsQuery(p, ""))
	if err != nil {
		return err
	}
	p.IDs = vals.ids
	p.UpdatedAt = vals.updatedAt
	p.SyncedAt = vals.syncedAt
	return nil
}

func loadUserPermissionsQuery(p *authz.UserPermissions, lock string) *sqlf.Query {
	const format = `
-- source: enterprise/internal/db/perms_store.go:loadUserPermissionsQuery
SELECT user_id, object_ids, updated_at, synced_at
FROM user_permissions
WHERE user_id = %s
AND permission = %s
AND object_type = %s
`

	return sqlf.Sprintf(
		format+lock,
		p.UserID,
		p.Perm.String(),
		p.Type,
	)
}

// LoadRepoPermissions loads stored repository permissions into p. An ErrPermsNotFound is
// returned when there are no valid permissions available.
func (s *PermsStore) LoadRepoPermissions(ctx context.Context, p *authz.RepoPermissions) (err error) {
	if Mocks.Perms.LoadRepoPermissions != nil {
		return Mocks.Perms.LoadRepoPermissions(ctx, p)
	}

	ctx, save := s.observe(ctx, "LoadRepoPermissions", "")
	defer func() { save(&err, p.TracingFields()...) }()

	vals, err := s.load(ctx, loadRepoPermissionsQuery(p, ""))
	if err != nil {
		return err
	}
	p.UserIDs = vals.ids
	p.UpdatedAt = vals.updatedAt
	p.SyncedAt = vals.syncedAt
	return nil
}

func loadRepoPermissionsQuery(p *authz.RepoPermissions, lock string) *sqlf.Query {
	const format = `
-- source: enterprise/internal/db/perms_store.go:loadRepoPermissionsQuery
SELECT repo_id, user_ids, updated_at, synced_at
FROM repo_permissions
WHERE repo_id = %s
AND permission = %s
`

	return sqlf.Sprintf(
		format+lock,
		p.RepoID,
		p.Perm.String(),
	)
}

// SetUserPermissions performs a full update for p, new object IDs found in p will be upserted
// and object IDs no longer in p will be removed. This method updates both `user_permissions`
// and `repo_permissions` tables.
//
// Example input:
// &UserPermissions{
//     UserID: 1,
//     Perm: authz.Read,
//     Type: authz.PermRepos,
//     IDs: bitmap{1, 2},
// }
//
// Table states for input:
// 	"user_permissions":
//   user_id | permission | object_type |  object_ids   | updated_at | synced_at
//  ---------+------------+-------------+---------------+------------+------------
//         1 |       read |       repos |        {1, 2} | <DateTime> | <DateTime>
//
//  "repo_permissions":
//   repo_id | permission | user_ids  | updated_at |  synced_at
//  ---------+------------+-----------+------------+-------------
//         1 |       read |       {1} | <DateTime> | <Unchanged>
//         2 |       read |       {1} | <DateTime> | <Unchanged>
func (s *PermsStore) SetUserPermissions(ctx context.Context, p *authz.UserPermissions) (err error) {
	if Mocks.Perms.SetUserPermissions != nil {
		return Mocks.Perms.SetUserPermissions(ctx, p)
	}

	ctx, save := s.observe(ctx, "SetUserPermissions", "")
	defer func() { save(&err, p.TracingFields()...) }()

	if p.IDs == nil {
		p.IDs = roaring.New()
	}

	q := sqlf.Sprintf(`
-- source: enterprise/cmd/frontend/db/perms_store.go:PermsStore.SetUserPermissions
WITH args AS (
	SELECT
		%s::INT AS user_id,
		%s::TEXT AS permission,
		%s::TEXT AS object_type,
		%s::INT[] AS new_ids
),

old AS (
	SELECT object_ids, synced_at FROM user_permissions
	JOIN args ON
		user_permissions.user_id = args.user_id AND
		user_permissions.permission = args.permission AND
		user_permissions.object_type = args.object_type
	UNION
	SELECT '{}' AS object_ids, NOW() AS synced_at
	ORDER BY synced_at ASC NULLS FIRST
	LIMIT 1
	-- Relying on the fact the last synced_at is before current time
),
new AS (SELECT new_ids AS ids FROM args),

perms AS (
	SELECT
		old.object_ids - new.ids AS removed,
		new.ids - old.object_ids AS added
	FROM old, new
),
repo_perms AS (
    INSERT INTO repo_permissions (repo_id, permission, user_ids, updated_at)
    SELECT UNNEST(perms.added | perms.removed), args.permission, INTSET(args.user_id), NOW()
    FROM perms, args
    ON CONFLICT ON CONSTRAINT repo_permissions_perm_unique
    DO UPDATE SET
		updated_at = excluded.updated_at,
		user_ids = (
			SELECT CASE WHEN perms.added::INT[] @> INTSET(repo_permissions.repo_id)
				THEN repo_permissions.user_ids | excluded.user_ids
				ELSE repo_permissions.user_ids - excluded.user_ids
			END
		FROM perms)
)

INSERT INTO user_permissions (user_id, permission, object_type, object_ids, updated_at, synced_at)
SELECT args.user_id, args.permission, args.object_type, new.ids, NOW(), NOW()
FROM new, args
ON CONFLICT ON CONSTRAINT user_permissions_perm_object_unique
DO UPDATE SET
	updated_at = excluded.updated_at,
	synced_at = excluded.synced_at,
	object_ids = excluded.object_ids
`, p.UserID, p.Perm.String(), p.Type, pq.Array(p.IDs.ToArray()))
	return s.execute(ctx, q)
}

// SetRepoPermissions performs a full update for p, new user IDs found in p will be upserted
// and user IDs no longer in p will be removed. This method updates both `user_permissions`
// and `repo_permissions` tables.
//
// This method starts its own transaction for update consistency if the caller hasn't started one already.
//
// Example input:
// &RepoPermissions{
//     RepoID: 1,
//     Perm: authz.Read,
//     UserIDs: bitmap{1, 2},
// }
//
// Table states for input:
// 	"user_permissions":
//   user_id | permission | object_type | object_ids | updated_at |  synced_at
//  ---------+------------+-------------+------------+------------+-------------
//         1 |       read |       repos |        {1} | <DateTime> | <Unchanged>
//         2 |       read |       repos |        {1} | <DateTime> | <Unchanged>
//
//  "repo_permissions":
//   repo_id | permission |   user_ids   | updated_at | synced_at
//  ---------+------------+--------------+------------+------------
//         1 |       read |       {1, 2} | <DateTime> | <DateTime>
func (s *PermsStore) SetRepoPermissions(ctx context.Context, p *authz.RepoPermissions) (err error) {
	if Mocks.Perms.SetRepoPermissions != nil {
		return Mocks.Perms.SetRepoPermissions(ctx, p)
	}

	ctx, save := s.observe(ctx, "SetRepoPermissions", "")
	defer func() { save(&err, p.TracingFields()...) }()

	var txs *PermsStore
	if s.inTx() {
		txs = s
	} else {
		txs, err = s.Transact(ctx)
		if err != nil {
			return err
		}
		defer txs.Done(&err)
	}

	if p.UserIDs == nil {
		p.UserIDs = roaring.NewBitmap()
	}

	q := sqlf.Sprintf(`
-- source: enterprise/cmd/frontend/db/perms_store.go:PermsStore.SetRepoPermissions
WITH args AS (
	SELECT
		%s::INT AS repo_id,
		%s::TEXT AS permission,
		%s::INT[] AS new_ids
),

old AS (
	SELECT user_ids, synced_at FROM repo_permissions
	JOIN args ON
		repo_permissions.repo_id = args.repo_id AND
		repo_permissions.permission = args.permission
	UNION
	SELECT '{}' AS user_ids, NOW() AS synced_at
	ORDER BY synced_at ASC NULLS FIRST
	LIMIT 1
	-- Relying on the fact the last synced_at is before current time
),
new AS (SELECT new_ids AS ids FROM args),

perms AS (
	SELECT
		old.user_ids - new.ids AS removed,
		new.ids - old.user_ids AS added
	FROM old, new
),
user_perms AS (
    INSERT INTO user_permissions (user_id, permission, object_type, object_ids, updated_at)
    SELECT UNNEST(perms.added | perms.removed), args.permission, 'repos', INTSET(args.repo_id), NOW()
    FROM perms, args
    ON CONFLICT ON CONSTRAINT user_permissions_perm_object_unique
    DO UPDATE SET
		updated_at = excluded.updated_at,
		object_ids = (
			SELECT CASE WHEN perms.added::INT[] @> INTSET(user_permissions.user_id)
				THEN user_permissions.object_ids | excluded.object_ids
				ELSE user_permissions.object_ids - excluded.object_ids
			END
		FROM perms)
)

INSERT INTO repo_permissions (repo_id, permission, user_ids, updated_at, synced_at)
SELECT args.repo_id, args.permission, new.ids, NOW(), NOW()
FROM new, args
ON CONFLICT ON CONSTRAINT repo_permissions_perm_unique
DO UPDATE SET
	updated_at = excluded.updated_at,
	synced_at = excluded.synced_at,
	user_ids = excluded.user_ids
`, p.RepoID, p.Perm.String(), pq.Array(p.UserIDs.ToArray()))
	return txs.execute(ctx, q)
}

func upsertUserPermissionsBatchQuery(userIDs []uint32, perm authz.Perms, permType authz.PermType, objectIDs []int32, updatedAt time.Time) (*sqlf.Query, error) {
	const format = `
-- source: enterprise/internal/db/perms_store.go:upsertUserPermissionsBatchQuery
INSERT INTO user_permissions
  (user_id, permission, object_type, object_ids, updated_at)
VALUES
  %s
ON CONFLICT ON CONSTRAINT
  user_permissions_perm_object_unique
DO UPDATE SET
  object_ids = user_permissions.object_ids | excluded.object_ids,
  updated_at = excluded.updated_at
`

	if updatedAt.IsZero() {
		return nil, ErrPermsUpdatedAtNotSet
	}

	items := make([]*sqlf.Query, 0, len(userIDs))
	for _, userID := range userIDs {
		items = append(items, sqlf.Sprintf("(%s, %s, %s, %s, %s)",
			userID,
			perm.String(),
			permType,
			pq.Array(objectIDs),
			updatedAt.UTC(),
		))
	}

	return sqlf.Sprintf(
		format,
		sqlf.Join(items, ","),
	), nil
}

// LoadUserPendingPermissions returns pending permissions found by given parameters.
// An ErrPermsNotFound is returned when there are no pending permissions available.
func (s *PermsStore) LoadUserPendingPermissions(ctx context.Context, p *authz.UserPendingPermissions) (err error) {
	if Mocks.Perms.LoadUserPendingPermissions != nil {
		return Mocks.Perms.LoadUserPendingPermissions(ctx, p)
	}

	ctx, save := s.observe(ctx, "LoadUserPendingPermissions", "")
	defer func() { save(&err, p.TracingFields()...) }()

	vals, err := s.load(ctx, loadUserPendingPermissionsQuery(p, ""))
	if err != nil {
		return err
	}
	p.ID = vals.id
	p.IDs = vals.ids
	p.UpdatedAt = vals.updatedAt
	return nil
}

func loadUserPendingPermissionsQuery(p *authz.UserPendingPermissions, lock string) *sqlf.Query {
	const format = `
-- source: enterprise/internal/db/perms_store.go:loadUserPendingPermissionsQuery
SELECT id, object_ids, updated_at, NULL
FROM user_pending_permissions
WHERE service_type = %s
AND service_id = %s
AND permission = %s
AND object_type = %s
AND bind_id = %s
`
	return sqlf.Sprintf(
		format+lock,
		p.ServiceType,
		p.ServiceID,
		p.Perm.String(),
		p.Type,
		p.BindID,
	)
}

// SetRepoPendingPermissions performs a full update for p with given accounts, new account IDs
// found will be upserted and account IDs no longer in AccountIDs will be removed.
//
// This method updates both `user_pending_permissions` and `repo_pending_permissions` tables.
//
// This method starts its own transaction for update consistency if the caller hasn't started one already.
//
// Example input:
//  &extsvc.Accounts{
//      ServiceType: "sourcegraph",
//      ServiceID:   "https://sourcegraph.com/",
//      AccountIDs:  []string{"alice", "bob"},
//  }
//  &authz.RepoPermissions{
//      RepoID: 1,
//      Perm: authz.Read,
//  }
//
// Table states for input:
// 	"user_pending_permissions":
//   id | service_type |        service_id        | bind_id | permission | object_type | object_ids | updated_at
//  ----+--------------+--------------------------+---------+------------+-------------+------------+-----------
//    1 | sourcegraph  | https://sourcegraph.com/ |   alice |       read |       repos |        {1} | <DateTime>
//    2 | sourcegraph  | https://sourcegraph.com/ |     bob |       read |       repos |        {1} | <DateTime>
//
//  "repo_pending_permissions":
//   repo_id | permission |   user_ids   | updated_at
//  ---------+------------+--------------+------------
//         1 |       read |       {1, 2} | <DateTime>
func (s *PermsStore) SetRepoPendingPermissions(ctx context.Context, accounts *extsvc.Accounts, p *authz.RepoPermissions) (err error) {
	if Mocks.Perms.SetRepoPendingPermissions != nil {
		return Mocks.Perms.SetRepoPendingPermissions(ctx, accounts, p)
	}

	ctx, save := s.observe(ctx, "SetRepoPendingPermissions", "")
	defer func() { save(&err, append(p.TracingFields(), accounts.TracingFields()...)...) }()

	var txs *PermsStore
	if s.inTx() {
		txs = s
	} else {
		txs, err = s.Transact(ctx)
		if err != nil {
			return err
		}
		defer txs.Done(&err)
	}

	var q *sqlf.Query

	p.UserIDs = roaring.NewBitmap()

	// Insert rows for AccountIDs without one in the "user_pending_permissions" table.
	// The insert does not store any permissions data but uses auto-increment key to generate unique ID.
	// This help guarantees rows of all AccountIDs exist when getting user IDs in next load query.
	updatedAt := txs.clock()
	p.UpdatedAt = updatedAt
	idToBindIDs := make(map[uint32]string)
	if len(accounts.AccountIDs) > 0 {
		// NOTE: Row-level locking is not needed here because we're creating stub rows and not modifying permissions.
		q, err = insertUserPendingPermissionsBatchQuery(accounts, p)
		if err != nil {
			return err
		}

		idToBindIDs, err = txs.loadUserPendingPermissionsIDs(ctx, q)
		if err != nil {
			return errors.Wrap(err, "load user pending permissions IDs")
		}

		// Make up p.UserIDs from the result set.
		for id := range idToBindIDs {
			p.UserIDs.Add(id)
		}
	}

	// Retrieve currently stored user IDs of this repository.
	vals, err := txs.load(ctx, loadRepoPendingPermissionsQuery(p, "FOR UPDATE"))
	if err != nil && err != authz.ErrPermsNotFound {
		return errors.Wrap(err, "load repo pending permissions")
	}
	oldIDs := roaring.NewBitmap()
	if vals != nil && vals.ids != nil {
		oldIDs = vals.ids
	}

	// Compute differences between the old and new sets.
	added := roaring.AndNot(p.UserIDs, oldIDs)
	removed := roaring.AndNot(oldIDs, p.UserIDs)

	// In case nothing added or removed.
	if added.GetCardinality() == 0 && removed.GetCardinality() == 0 {
		return nil
	}

	if added.GetCardinality() > 0 {
		bindIDs := make([]string, 0, added.GetCardinality())
		for _, id := range added.ToArray() {
			bindIDs = append(bindIDs, idToBindIDs[id])
		}

		if q, err := upsertUserPendingPermissionsBatchQuery(upsertUserPendingPermissionsBatchQueryArgs{
			ServiceType: accounts.ServiceType,
			ServiceID:   accounts.ServiceID,
			BindIDs:     bindIDs,
			Perm:        p.Perm,
			Type:        authz.PermRepos,
			ObjectIDs:   []int32{p.RepoID},
			UpdatedAt:   updatedAt,
		}); err != nil {
			return err
		} else if err = txs.execute(ctx, q); err != nil {
			return errors.Wrap(err, "execute upsert user pending permissions batch query")
		}
	}

	if removed.GetCardinality() > 0 {
		q := sqlf.Sprintf(`
UPDATE user_pending_permissions SET object_ids = object_ids - %s::INT, updated_at = %s WHERE id = ANY (%s);
`, p.RepoID, updatedAt.UTC(), pq.Array(removed.ToArray()))
		if err = txs.execute(ctx, q); err != nil {
			return errors.Wrap(err, "execute update removed user pending permissions query")
		}
	}

	if q, err = upsertRepoPendingPermissionsBatchQuery(p); err != nil {
		return err
	} else if err = txs.execute(ctx, q); err != nil {
		return errors.Wrap(err, "execute upsert repo pending permissions batch query")
	}

	return nil
}

func (s *PermsStore) loadUserPendingPermissionsIDs(ctx context.Context, q *sqlf.Query) (idToBindIDs map[uint32]string, err error) {
	ctx, save := s.observe(ctx, "loadUserPendingPermissionsIDs", "")
	defer func() {
		save(&err,
			otlog.String("Query.Query", q.Query(sqlf.PostgresBindVar)),
			otlog.Object("Query.Args", q.Args()),
		)
	}()

	rows, err := s.db.QueryContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	idToBindIDs = make(map[uint32]string)
	for rows.Next() {
		var id uint32
		var bindID string
		if err = rows.Scan(&id, &bindID); err != nil {
			return nil, err
		}
		idToBindIDs[id] = bindID
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return idToBindIDs, nil
}

func insertUserPendingPermissionsBatchQuery(
	accounts *extsvc.Accounts,
	p *authz.RepoPermissions,
) (*sqlf.Query, error) {
	const format = `
-- source: enterprise/internal/db/perms_store.go:insertUserPendingPermissionsBatchQuery
INSERT INTO user_pending_permissions
  (service_type, service_id, bind_id, permission, object_type, object_ids, updated_at)
VALUES
  %s
ON CONFLICT ON CONSTRAINT
  user_pending_permissions_service_perm_object_unique
DO UPDATE SET
  updated_at = excluded.updated_at
RETURNING id, bind_id
`

	if p.UpdatedAt.IsZero() {
		return nil, ErrPermsUpdatedAtNotSet
	}

	items := make([]*sqlf.Query, len(accounts.AccountIDs))
	for i := range accounts.AccountIDs {
		items[i] = sqlf.Sprintf("(%s, %s, %s, %s, %s, %s, %s)",
			accounts.ServiceType,
			accounts.ServiceID,
			accounts.AccountIDs[i],
			p.Perm.String(),
			authz.PermRepos,
			pq.Array([]uint32{}),
			p.UpdatedAt.UTC(),
		)
	}

	return sqlf.Sprintf(
		format,
		sqlf.Join(items, ","),
	), nil
}

func loadRepoPendingPermissionsQuery(p *authz.RepoPermissions, lock string) *sqlf.Query {
	const format = `
-- source: enterprise/internal/db/perms_store.go:loadRepoPendingPermissionsQuery
SELECT repo_id, user_ids, updated_at, NULL
FROM repo_pending_permissions
WHERE repo_id = %s
AND permission = %s
`
	return sqlf.Sprintf(
		format+lock,
		p.RepoID,
		p.Perm.String(),
	)
}

type upsertUserPendingPermissionsBatchQueryArgs struct {
	ServiceType string
	ServiceID   string
	BindIDs     []string
	Perm        authz.Perms
	Type        authz.PermType
	ObjectIDs   []int32
	UpdatedAt   time.Time
}

func upsertUserPendingPermissionsBatchQuery(args upsertUserPendingPermissionsBatchQueryArgs) (*sqlf.Query, error) {
	const format = `
-- source: enterprise/internal/db/perms_store.go:upsertUserPendingPermissionsBatchQuery
INSERT INTO user_pending_permissions
  (service_type, service_id, bind_id, permission, object_type, object_ids, updated_at)
VALUES
  %s
ON CONFLICT ON CONSTRAINT
  user_pending_permissions_service_perm_object_unique
DO UPDATE SET
  object_ids = user_pending_permissions.object_ids | excluded.object_ids,
  updated_at = excluded.updated_at
`
	if args.UpdatedAt.IsZero() {
		return nil, ErrPermsUpdatedAtNotSet
	}

	items := make([]*sqlf.Query, 0, len(args.BindIDs))
	for _, bindID := range args.BindIDs {
		items = append(items, sqlf.Sprintf("(%s, %s, %s, %s, %s, %s, %s)",
			args.ServiceType,
			args.ServiceID,
			bindID,
			args.Perm.String(),
			args.Type,
			pq.Array(args.ObjectIDs),
			args.UpdatedAt.UTC(),
		))
	}

	return sqlf.Sprintf(
		format,
		sqlf.Join(items, ","),
	), nil
}

func upsertRepoPendingPermissionsBatchQuery(ps ...*authz.RepoPermissions) (*sqlf.Query, error) {
	const format = `
-- source: enterprise/internal/db/perms_store.go:upsertRepoPendingPermissionsBatchQuery
INSERT INTO repo_pending_permissions
  (repo_id, permission, user_ids, updated_at)
VALUES
  %s
ON CONFLICT ON CONSTRAINT
  repo_pending_permissions_perm_unique
DO UPDATE SET
  user_ids = excluded.user_ids,
  updated_at = excluded.updated_at
`

	items := make([]*sqlf.Query, len(ps))
	for i := range ps {
		if ps[i].UpdatedAt.IsZero() {
			return nil, ErrPermsUpdatedAtNotSet
		}

		ps[i].UserIDs.RunOptimize()
		items[i] = sqlf.Sprintf("(%s, %s, %s, %s)",
			ps[i].RepoID,
			ps[i].Perm.String(),
			pq.Array(ps[i].UserIDs.ToArray()),
			ps[i].UpdatedAt.UTC(),
		)
	}

	return sqlf.Sprintf(
		format,
		sqlf.Join(items, ","),
	), nil
}

// GrantPendingPermissions is used to grant pending permissions when the associated "ServiceType",
// "ServiceID" and "BindID" found in p becomes effective for a given user, e.g. username as bind ID when
// a user is created, email as bind ID when the email address is verified.
//
// Because there could be multiple external services and bind IDs that are associated with a single user
// (e.g. same user on different code hosts, multiple email addresses), it merges data from "repo_pending_permissions"
// and "user_pending_permissions" tables to "repo_permissions" and "user_permissions" tables for the user.
// Therefore, permissions are unioned not replaced, which is one of the main differences from SetRepoPermissions
// and SetRepoPendingPermissions methods. Another main difference is that multiple calls to this method
// are not idempotent as it conceptually does nothing when there is no data in the pending permissions
// tables for the user.
//
// This method starts its own transaction for update consistency if the caller hasn't started one already.
//
// 🚨 SECURITY: This method takes arbitrary string as a valid bind ID and does not interpret the meaning
// of the value it represents. Therefore, it is caller's responsibility to ensure the legitimate relation
// between the given user ID and the bind ID found in p.
func (s *PermsStore) GrantPendingPermissions(ctx context.Context, userID int32, p *authz.UserPendingPermissions) (err error) {
	ctx, save := s.observe(ctx, "GrantPendingPermissions", "")
	defer func() { save(&err, append(p.TracingFields(), otlog.Int32("userID", userID))...) }()

	var txs *PermsStore
	if s.inTx() {
		txs = s
	} else {
		txs, err = s.Transact(ctx)
		if err != nil {
			return err
		}
		defer txs.Done(&err)
	}

	vals, err := txs.load(ctx, loadUserPendingPermissionsQuery(p, "FOR UPDATE"))
	if err != nil {
		// Skip the whole grant process if the user has no pending permissions.
		if err == authz.ErrPermsNotFound {
			return nil
		}
		return errors.Wrap(err, "load user pending permissions")
	}
	p.ID = vals.id
	p.IDs = vals.ids

	// NOTE: We currently only have "repos" type, so avoid unnecessary type checking for now.
	ids := p.IDs.ToArray()
	if len(ids) == 0 {
		return nil
	}

	updatedAt := txs.clock()

	if q, err := upsertRepoPermissionsBatchQuery(p.IDs.ToArray(), p.Perm, []int32{userID}, updatedAt); err != nil {
		return err
	} else if err = txs.execute(ctx, q); err != nil {
		return errors.Wrap(err, "execute upsert repo permissions batch query")
	}

	// Load existing user permissions to be merged if any. Since we're doing union of permissions,
	// whatever we have already in the "user_permissions" table is all valid thus we don't
	// need to do any clean up.
	up := &authz.UserPermissions{
		UserID: userID,
		Perm:   p.Perm,
		Type:   p.Type,
	}
	var oldIDs *roaring.Bitmap
	vals, err = txs.load(ctx, loadUserPermissionsQuery(up, "FOR UPDATE"))
	if err != nil {
		if err != authz.ErrPermsNotFound {
			return errors.Wrap(err, "load user permissions")
		}
		oldIDs = roaring.NewBitmap()
	} else {
		oldIDs = vals.ids
	}
	up.IDs = roaring.Or(oldIDs, p.IDs)

	objectIDs := make([]int32, 0, up.IDs.GetCardinality())
	for _, id := range up.IDs.ToArray() {
		objectIDs = append(objectIDs, int32(id))
	}
	if q, err := upsertUserPermissionsBatchQuery([]uint32{uint32(userID)}, p.Perm, p.Type, objectIDs, updatedAt); err != nil {
		return err
	} else if err = txs.execute(ctx, q); err != nil {
		return errors.Wrap(err, "execute upsert user permissions query")
	}

	// NOTE: Practically, we don't need to clean up "repo_pending_permissions" table because the value of "id" column
	// that is associated with this user will be invalidated automatically by deleting this row. Thus, we are able to
	// avoid database deadlocks with other methods (e.g. SetRepoPermissions, SetRepoPendingPermissions).
	if err = txs.execute(ctx, deleteUserPendingPermissionsQuery(p)); err != nil {
		return errors.Wrap(err, "execute delete user pending permissions query")
	}

	return nil
}

func upsertRepoPermissionsBatchQuery(repoIDs []uint32, perm authz.Perms, userIDs []int32, updatedAt time.Time) (*sqlf.Query, error) {
	const format = `
-- source: enterprise/internal/db/perms_store.go:upsertRepoPermissionsBatchQuery
INSERT INTO repo_permissions
  (repo_id, permission, user_ids, updated_at)
VALUES
  %s
ON CONFLICT ON CONSTRAINT
  repo_permissions_perm_unique
DO UPDATE SET
  user_ids = repo_permissions.user_ids | excluded.user_ids,
  updated_at = excluded.updated_at
`

	if updatedAt.IsZero() {
		return nil, ErrPermsUpdatedAtNotSet
	}

	items := make([]*sqlf.Query, 0, len(repoIDs))
	for _, repoID := range repoIDs {
		items = append(items, sqlf.Sprintf("(%s, %s, %s, %s)",
			repoID,
			perm.String(),
			pq.Array(userIDs),
			updatedAt.UTC(),
		))
	}

	return sqlf.Sprintf(
		format,
		sqlf.Join(items, ","),
	), nil
}

func deleteUserPendingPermissionsQuery(p *authz.UserPendingPermissions) *sqlf.Query {
	const format = `
-- source: enterprise/internal/db/perms_store.go:deleteUserPendingPermissionsQuery
DELETE FROM user_pending_permissions
WHERE service_type = %s
AND service_id = %s
AND permission = %s
AND object_type = %s
AND bind_id = %s
`

	return sqlf.Sprintf(
		format,
		p.ServiceType,
		p.ServiceID,
		p.Perm.String(),
		p.Type,
		p.BindID,
	)
}

// ListPendingUsers returns a list of bind IDs who have pending permissions by given
// service type and ID.
func (s *PermsStore) ListPendingUsers(ctx context.Context, serviceType, serviceID string) (bindIDs []string, err error) {
	if Mocks.Perms.ListPendingUsers != nil {
		return Mocks.Perms.ListPendingUsers(ctx)
	}

	ctx, save := s.observe(ctx, "ListPendingUsers", "")
	defer save(&err)

	q := sqlf.Sprintf(`
SELECT bind_id, object_ids
FROM user_pending_permissions
WHERE service_type = %s
AND service_id = %s
`, serviceType, serviceID)

	var rows *sql.Rows
	rows, err = s.db.QueryContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var bindID string
		var ids []int64
		if err = rows.Scan(&bindID, pq.Array(&ids)); err != nil {
			return nil, err
		}

		if len(ids) == 0 {
			continue
		}

		bm := roaring.NewBitmap()
		for _, id := range ids {
			bm.Add(uint32(id))
		}
		bindIDs = append(bindIDs, bindID)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return bindIDs, nil
}

// DeleteAllUserPermissions deletes all rows with given user ID from the "user_permissions" table,
// which effectively removes access to all repositories for the user.
func (s *PermsStore) DeleteAllUserPermissions(ctx context.Context, userID int32) (err error) {
	ctx, save := s.observe(ctx, "DeleteAllUserPermissions", "")
	defer func() { save(&err, otlog.Int32("userID", userID)) }()

	// NOTE: Practically, we don't need to clean up "repo_permissions" table because the value of "id" column
	// that is associated with this user will be invalidated automatically by deleting this row.
	if err = s.execute(ctx, sqlf.Sprintf(`DELETE FROM user_permissions WHERE user_id = %s`, userID)); err != nil {
		return errors.Wrap(err, "execute delete user permissions query")
	}

	return nil
}

// DeleteAllUserPendingPermissions deletes all rows with given bind IDs from the "user_pending_permissions" table.
// It accepts list of bind IDs because a user has multiple bind IDs, e.g. username and email addresses.
func (s *PermsStore) DeleteAllUserPendingPermissions(ctx context.Context, accounts *extsvc.Accounts) (err error) {
	ctx, save := s.observe(ctx, "DeleteAllUserPendingPermissions", "")
	defer func() { save(&err, accounts.TracingFields()...) }()

	// NOTE: Practically, we don't need to clean up "repo_pending_permissions" table because the value of "id" column
	// that is associated with this user will be invalidated automatically by deleting this row.
	items := make([]*sqlf.Query, len(accounts.AccountIDs))
	for i := range accounts.AccountIDs {
		items[i] = sqlf.Sprintf("%s", accounts.AccountIDs[i])
	}
	q := sqlf.Sprintf(`
-- source: enterprise/internal/db/perms_store.go:PermsStore.DeleteAllUserPendingPermissions
DELETE FROM user_pending_permissions
WHERE service_type = %s
AND service_id = %s
AND bind_id IN (%s)`,
		accounts.ServiceType, accounts.ServiceID, sqlf.Join(items, ","))
	if err = s.execute(ctx, q); err != nil {
		return errors.Wrap(err, "execute delete user pending permissions query")
	}

	return nil
}

func (s *PermsStore) execute(ctx context.Context, q *sqlf.Query, vs ...interface{}) (err error) {
	ctx, save := s.observe(ctx, "execute", "")
	defer func() { save(&err, otlog.Object("q", q)) }()

	var rows *sql.Rows
	rows, err = s.db.QueryContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...)
	if err != nil {
		return err
	}

	if len(vs) > 0 {
		if !rows.Next() {
			// One row is expected, return ErrPermsNotFound if no other errors occurred.
			err = rows.Err()
			if err == nil {
				err = authz.ErrPermsNotFound
			}
			return err
		}

		if err = rows.Scan(vs...); err != nil {
			return err
		}
	}

	return rows.Close()
}

// permsLoadValues contains return values of (*PermsStore).load method.
type permsLoadValues struct {
	id        int32           // An integer ID
	ids       *roaring.Bitmap // Bitmap of unmarshalled IDs
	updatedAt time.Time       // Last updated time of the row
	syncedAt  time.Time       // Last synced time of the row
}

// load is a generic method that scans three values from one database table row, these values must have
// types and be scanned in the order of int32 (id), []byte (ids), time.Time (updatedAt) and nullable
// time.Time (syncedAt). In addition, it unmarshalles the []byte into a *roaring.Bitmap.
func (s *PermsStore) load(ctx context.Context, q *sqlf.Query) (*permsLoadValues, error) {
	var err error
	ctx, save := s.observe(ctx, "load", "")
	defer func() {
		save(&err,
			otlog.String("Query.Query", q.Query(sqlf.PostgresBindVar)),
			otlog.Object("Query.Args", q.Args()),
		)
	}()

	var rows *sql.Rows
	rows, err = s.db.QueryContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...)
	if err != nil {
		return nil, err
	}

	if !rows.Next() {
		// One row is expected, return ErrPermsNotFound if no other errors occurred.
		err = rows.Err()
		if err == nil {
			err = authz.ErrPermsNotFound
		}
		return nil, err
	}

	var id int32
	var ids []int64
	var updatedAt time.Time
	var syncedAt time.Time
	if err = rows.Scan(&id, pq.Array(&ids), &updatedAt, &dbutil.NullTime{Time: &syncedAt}); err != nil {
		return nil, err
	}

	if err = rows.Close(); err != nil {
		return nil, err
	}

	vals := &permsLoadValues{
		id:        id,
		ids:       roaring.NewBitmap(),
		syncedAt:  syncedAt,
		updatedAt: updatedAt,
	}
	for _, id := range ids {
		vals.ids.Add(uint32(id))
	}
	return vals, nil
}

// ListExternalAccounts returns all external accounts that are associated with given user.
func (s *PermsStore) ListExternalAccounts(ctx context.Context, userID int32) (accounts []*extsvc.Account, err error) {
	if Mocks.Perms.ListExternalAccounts != nil {
		return Mocks.Perms.ListExternalAccounts(ctx, userID)
	}

	ctx, save := s.observe(ctx, "ListExternalAccounts", "")
	defer func() { save(&err, otlog.Int32("userID", userID)) }()

	q := sqlf.Sprintf(`
-- source: enterprise/internal/db/perms_store.go:PermsStore.ListExternalAccounts
SELECT id, user_id,
       service_type, service_id, client_id, account_id,
       auth_data, account_data,
       created_at, updated_at
FROM user_external_accounts
WHERE user_id = %d
ORDER BY id ASC
`, userID)
	rows, err := s.db.QueryContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var acct extsvc.Account
		if err := rows.Scan(
			&acct.ID, &acct.UserID,
			&acct.ServiceType, &acct.ServiceID, &acct.ClientID, &acct.AccountID,
			&acct.AuthData, &acct.Data,
			&acct.CreatedAt, &acct.UpdatedAt,
		); err != nil {
			return nil, err
		}
		accounts = append(accounts, &acct)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return accounts, nil
}

// GetUserIDsByExternalAccounts returns all user IDs matched by given external account specs.
// The returned set has mapping relation as "account ID -> user ID". The number of results
// could be less than the candidate list due to some users are not associated with any external
// account.
func (s *PermsStore) GetUserIDsByExternalAccounts(ctx context.Context, accounts *extsvc.Accounts) (_ map[string]int32, err error) {
	if Mocks.Perms.GetUserIDsByExternalAccounts != nil {
		return Mocks.Perms.GetUserIDsByExternalAccounts(ctx, accounts)
	}

	ctx, save := s.observe(ctx, "ListUsersByExternalAccounts", "")
	defer func() { save(&err, accounts.TracingFields()...) }()

	items := make([]*sqlf.Query, len(accounts.AccountIDs))
	for i := range accounts.AccountIDs {
		items[i] = sqlf.Sprintf("%s", accounts.AccountIDs[i])
	}

	q := sqlf.Sprintf(`
-- source: enterprise/internal/db/perms_store.go:PermsStore.GetUserIDsByExternalAccounts
SELECT user_id, account_id
FROM user_external_accounts
WHERE service_type = %s
AND service_id = %s
AND account_id IN (%s)
`, accounts.ServiceType, accounts.ServiceID, sqlf.Join(items, ","))
	rows, err := s.db.QueryContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	userIDs := make(map[string]int32)
	for rows.Next() {
		var userID int32
		var accountID string
		if err := rows.Scan(&userID, &accountID); err != nil {
			return nil, err
		}
		userIDs[accountID] = userID
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return userIDs, nil
}

// UserIDsWithNoPerms returns a list of user IDs with no permissions found in
// the database.
func (s *PermsStore) UserIDsWithNoPerms(ctx context.Context) ([]int32, error) {
	q := sqlf.Sprintf(`
-- source: enterprise/internal/db/perms_store.go:PermsStore.UserIDsWithNoPerms
SELECT users.id, NULL FROM users
WHERE users.site_admin = FALSE
AND users.deleted_at IS NULL
AND users.id NOT IN
	(SELECT perms.user_id FROM user_permissions AS perms)
`)
	results, err := s.loadIDsWithTime(ctx, q)
	if err != nil {
		return nil, err
	}

	ids := make([]int32, 0, len(results))
	for id := range results {
		ids = append(ids, id)
	}
	return ids, nil
}

// UserIDsWithNoPerms returns a list of private repository IDs with no permissions
// found in the database.
func (s *PermsStore) RepoIDsWithNoPerms(ctx context.Context) ([]api.RepoID, error) {
	q := sqlf.Sprintf(`
-- source: enterprise/internal/db/perms_store.go:PermsStore.RepoIDsWithNoPerms
SELECT repo.id, NULL FROM repo
WHERE repo.deleted_at IS NULL
AND repo.private = TRUE
AND repo.id NOT IN
	(SELECT perms.repo_id FROM repo_permissions AS perms
	 UNION
	 SELECT pending.repo_id FROM repo_pending_permissions AS pending)
`)

	results, err := s.loadIDsWithTime(ctx, q)
	if err != nil {
		return nil, err
	}

	ids := make([]api.RepoID, 0, len(results))
	for id := range results {
		ids = append(ids, api.RepoID(id))
	}
	return ids, nil
}

// UserIDsWithOldestPerms returns a list of user ID and last updated pairs for users who
// have the least recent synced permissions in the database and capped results by the limit.
func (s *PermsStore) UserIDsWithOldestPerms(ctx context.Context, limit int) (map[int32]time.Time, error) {
	q := sqlf.Sprintf(`
-- source: enterprise/internal/db/perms_store.go:PermsStore.UserIDsWithOldestPerms
SELECT perms.user_id, perms.synced_at FROM user_permissions AS perms
WHERE perms.user_id IN
	(SELECT users.id FROM users
	 WHERE users.deleted_at IS NULL)
ORDER BY perms.synced_at ASC NULLS FIRST
LIMIT %s
`, limit)
	return s.loadIDsWithTime(ctx, q)
}

// ReposIDsWithOldestPerms returns a list of repository ID and last updated pairs for
// repositories that have the least recent synced permissions in the database and caps
// results by the limit.
func (s *PermsStore) ReposIDsWithOldestPerms(ctx context.Context, limit int) (map[api.RepoID]time.Time, error) {
	q := sqlf.Sprintf(`
-- source: enterprise/internal/db/perms_store.go:PermsStore.ReposIDsWithOldestPerms
SELECT perms.repo_id, perms.synced_at FROM repo_permissions AS perms
WHERE perms.repo_id IN
	(SELECT repo.id FROM repo
	 WHERE repo.deleted_at IS NULL)
ORDER BY perms.synced_at ASC NULLS FIRST
LIMIT %s
`, limit)

	pairs, err := s.loadIDsWithTime(ctx, q)
	if err != nil {
		return nil, err
	}

	results := make(map[api.RepoID]time.Time, len(pairs))
	for id, t := range pairs {
		results[api.RepoID(id)] = t
	}
	return results, nil
}

// loadIDsWithTime runs the query and returns a list of ID and nullable time pairs.
func (s *PermsStore) loadIDsWithTime(ctx context.Context, q *sqlf.Query) (map[int32]time.Time, error) {
	rows, err := s.db.QueryContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make(map[int32]time.Time)
	for rows.Next() {
		var id int32
		var t time.Time
		if err = rows.Scan(&id, &dbutil.NullTime{Time: &t}); err != nil {
			return nil, err
		}

		results[id] = t
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// PermsMetrics contains metrics values calculated by querying the database.
type PermsMetrics struct {
	// The number of users with stale permissions.
	UsersWithStalePerms int64
	// The seconds between users with oldest and the most up-to-date permissions.
	UsersPermsGapSeconds float64
	// The number of repositories with stale permissions.
	ReposWithStalePerms int64
	// The seconds between repositories with oldest and the most up-to-date permissions.
	ReposPermsGapSeconds float64
}

// Metrics returns calculated metrics values by querying the database. The "staleDur"
// argument indicates how long ago was the last update to be considered as stale.
func (s *PermsStore) Metrics(ctx context.Context, staleDur time.Duration) (*PermsMetrics, error) {
	m := &PermsMetrics{}

	stale := s.clock().Add(-1 * staleDur)
	q := sqlf.Sprintf(`
SELECT COUNT(*) FROM user_permissions
WHERE updated_at <= %s
`, stale)
	if err := s.execute(ctx, q, &m.UsersWithStalePerms); err != nil {
		return nil, errors.Wrap(err, "users with stale perms")
	}

	var seconds sql.NullFloat64
	q = sqlf.Sprintf(`
SELECT EXTRACT(EPOCH FROM (MAX(updated_at) - MIN(updated_at)))
FROM user_permissions
`)
	if err := s.execute(ctx, q, &seconds); err != nil {
		return nil, errors.Wrap(err, "users perms gap seconds")
	}
	m.UsersPermsGapSeconds = seconds.Float64

	q = sqlf.Sprintf(`
SELECT COUNT(*) FROM repo_permissions AS perms
WHERE perms.repo_id NOT IN
	(SELECT repo.id FROM repo
	 WHERE repo.deleted_at IS NOT NULL)
AND perms.updated_at <= %s
`, stale)
	if err := s.execute(ctx, q, &m.ReposWithStalePerms); err != nil {
		return nil, errors.Wrap(err, "repos with stale perms")
	}

	q = sqlf.Sprintf(`
SELECT EXTRACT(EPOCH FROM (MAX(perms.updated_at) - MIN(perms.updated_at)))
FROM repo_permissions AS perms
WHERE perms.repo_id NOT IN
	(SELECT repo.id FROM repo
	 WHERE repo.deleted_at IS NOT NULL)
`)
	if err := s.execute(ctx, q, &seconds); err != nil {
		return nil, errors.Wrap(err, "repos perms gap seconds")
	}
	m.ReposPermsGapSeconds = seconds.Float64

	return m, nil
}

// tx begins a new transaction.
func (s *PermsStore) tx(ctx context.Context) (*sql.Tx, error) {
	switch t := s.db.(type) {
	case *sql.Tx:
		return t, nil
	case *sql.DB:
		tx, err := t.BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		return tx, nil
	default:
		panic(fmt.Sprintf("can't open transaction with unknown implementation of dbutil.DB: %T", t))
	}
}

// Transact begins a new transaction and make a new PermsStore over it.
func (s *PermsStore) Transact(ctx context.Context) (*PermsStore, error) {
	if Mocks.Perms.Transact != nil {
		return Mocks.Perms.Transact(ctx)
	}

	tx, err := s.tx(ctx)
	if err != nil {
		return nil, err
	}
	return NewPermsStore(tx, s.clock), nil
}

// inTx returns true if the current PermsStore wraps an underlying transaction.
func (s *PermsStore) inTx() bool {
	_, ok := s.db.(*sql.Tx)
	return ok
}

// Done commits the transaction if error is nil. Otherwise, rolls back the transaction.
func (s *PermsStore) Done(err *error) {
	if !s.inTx() {
		return
	}

	tx := s.db.(*sql.Tx)
	if err == nil || *err == nil {
		_ = tx.Commit()
	} else {
		_ = tx.Rollback()
	}
}

func (s *PermsStore) observe(ctx context.Context, family, title string) (context.Context, func(*error, ...otlog.Field)) {
	began := s.clock()
	tr, ctx := trace.New(ctx, "db.PermsStore."+family, title)

	return ctx, func(err *error, fs ...otlog.Field) {
		now := s.clock()
		took := now.Sub(began)

		fs = append(fs, otlog.String("Duration", took.String()))

		tr.LogFields(fs...)

		success := err == nil || *err == nil
		if !success {
			tr.SetError(*err)
		}

		tr.Finish()
	}
}
