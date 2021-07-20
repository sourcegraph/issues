package codemonitors

import (
	"context"
	"database/sql"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/internal/actor"
)

type Monitor struct {
	ID              int64
	CreatedBy       int32
	CreatedAt       time.Time
	ChangedBy       int32
	ChangedAt       time.Time
	Description     string
	Enabled         bool
	NamespaceUserID *int32
	NamespaceOrgID  *int32
}

var monitorColumns = []*sqlf.Query{
	sqlf.Sprintf("cm_monitors.id"),
	sqlf.Sprintf("cm_monitors.created_by"),
	sqlf.Sprintf("cm_monitors.created_at"),
	sqlf.Sprintf("cm_monitors.changed_by"),
	sqlf.Sprintf("cm_monitors.changed_at"),
	sqlf.Sprintf("cm_monitors.description"),
	sqlf.Sprintf("cm_monitors.enabled"),
	sqlf.Sprintf("cm_monitors.namespace_user_id"),
	sqlf.Sprintf("cm_monitors.namespace_org_id"),
}

func (s *Store) CreateMonitor(ctx context.Context, args *graphqlbackend.CreateMonitorArgs) (m *Monitor, err error) {
	var q *sqlf.Query
	q, err = s.createCodeMonitorQuery(ctx, args)
	if err != nil {
		return nil, err
	}
	return s.runMonitorQuery(ctx, q)
}

func (s *Store) UpdateMonitor(ctx context.Context, args *graphqlbackend.UpdateCodeMonitorArgs) (m *Monitor, err error) {
	var q *sqlf.Query
	q, err = s.updateCodeMonitorQuery(ctx, args)
	if err != nil {
		return nil, err
	}
	return s.runMonitorQuery(ctx, q)
}

func (s *Store) ToggleMonitor(ctx context.Context, args *graphqlbackend.ToggleCodeMonitorArgs) (m *Monitor, err error) {
	var q *sqlf.Query
	q, err = s.toggleCodeMonitorQuery(ctx, args)
	if err != nil {
		return nil, err
	}
	return s.runMonitorQuery(ctx, q)
}

func (s *Store) DeleteMonitor(ctx context.Context, args *graphqlbackend.DeleteCodeMonitorArgs) (err error) {
	var q *sqlf.Query
	q, err = s.deleteMonitorQuery(ctx, args)
	if err != nil {
		return err
	}
	return s.Exec(ctx, q)
}

func (s *Store) Monitors(ctx context.Context, userID int32, args *graphqlbackend.ListMonitorsArgs) ([]*Monitor, error) {
	q, err := monitorsQuery(userID, args)
	if err != nil {
		return nil, err
	}
	rows, err := s.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMonitors(rows)
}

const monitorByIDFmtStr = `
SELECT id, created_by, created_at, changed_by, changed_at, description, enabled, namespace_user_id, namespace_org_id
FROM cm_monitors
WHERE id = %s
`

func (s *Store) MonitorByIDInt64(ctx context.Context, monitorID int64) (m *Monitor, err error) {
	return s.runMonitorQuery(ctx, sqlf.Sprintf(monitorByIDFmtStr, monitorID))
}

const totalCountMonitorsFmtStr = `
SELECT COUNT(*)
FROM cm_monitors
WHERE namespace_user_id = %s;
`

func (s *Store) TotalCountMonitors(ctx context.Context, userID int32) (count int32, err error) {
	err = s.QueryRow(ctx, sqlf.Sprintf(totalCountMonitorsFmtStr, userID)).Scan(&count)
	return count, err
}

const monitorsFmtStr = `
SELECT id, created_by, created_at, changed_by, changed_at, description, enabled, namespace_user_id, namespace_org_id
FROM cm_monitors
WHERE namespace_user_id = %s
AND id > %s
ORDER BY id ASC
LIMIT %s
`

func monitorsQuery(userID int32, args *graphqlbackend.ListMonitorsArgs) (*sqlf.Query, error) {
	after, err := unmarshalAfter(args.After)
	if err != nil {
		return nil, err
	}
	query := sqlf.Sprintf(
		monitorsFmtStr,
		userID,
		after,
		args.First,
	)
	return query, nil
}

const toggleCodeMonitorFmtStr = `
UPDATE cm_monitors
SET enabled = %s,
	changed_by = %s,
	changed_at = %s
WHERE id = %s
RETURNING %s
`

func (s *Store) toggleCodeMonitorQuery(ctx context.Context, args *graphqlbackend.ToggleCodeMonitorArgs) (*sqlf.Query, error) {
	var monitorID int64
	err := relay.UnmarshalSpec(args.Id, &monitorID)
	if err != nil {
		return nil, err
	}
	actorUID := actor.FromContext(ctx).UID
	query := sqlf.Sprintf(
		toggleCodeMonitorFmtStr,
		args.Enabled,
		actorUID,
		s.Now(),
		monitorID,
		sqlf.Join(monitorColumns, ", "),
	)
	return query, nil
}

const insertCodeMonitorFmtStr = `
INSERT INTO cm_monitors
(created_at, created_by, changed_at, changed_by, description, enabled, namespace_user_id, namespace_org_id)
VALUES (%s,%s,%s,%s,%s,%s,%s,%s)
RETURNING %s;
`

func (s *Store) createCodeMonitorQuery(ctx context.Context, args *graphqlbackend.CreateMonitorArgs) (*sqlf.Query, error) {
	var userID int32
	var orgID int32
	err := graphqlbackend.UnmarshalNamespaceID(args.Namespace, &userID, &orgID)
	if err != nil {
		return nil, err
	}
	now := s.Now()
	a := actor.FromContext(ctx)
	return sqlf.Sprintf(
		insertCodeMonitorFmtStr,
		now,
		a.UID,
		now,
		a.UID,
		args.Description,
		args.Enabled,
		nilOrInt32(userID),
		nilOrInt32(orgID),
		sqlf.Join(monitorColumns, ", "),
	), nil
}

const updateCodeMonitorFmtStr = `
UPDATE cm_monitors
SET description = %s,
	enabled	= %s,
	namespace_user_id = %s,
	namespace_org_id = %s,
	changed_by = %s,
	changed_at = %s
WHERE id = %s
RETURNING %s;
`

func (s *Store) updateCodeMonitorQuery(ctx context.Context, args *graphqlbackend.UpdateCodeMonitorArgs) (*sqlf.Query, error) {
	var userID int32
	var orgID int32
	err := graphqlbackend.UnmarshalNamespaceID(args.Monitor.Update.Namespace, &userID, &orgID)
	if err != nil {
		return nil, err
	}
	now := s.Now()
	a := actor.FromContext(ctx)
	var monitorID int64
	err = relay.UnmarshalSpec(args.Monitor.Id, &monitorID)
	if err != nil {
		return nil, err
	}
	return sqlf.Sprintf(
		updateCodeMonitorFmtStr,
		args.Monitor.Update.Description,
		args.Monitor.Update.Enabled,
		nilOrInt32(userID),
		nilOrInt32(orgID),
		a.UID,
		now,
		monitorID,
		sqlf.Join(monitorColumns, ", "),
	), nil
}

const deleteMonitorFmtStr = `DELETE FROM cm_monitors WHERE id = %s`

func (s *Store) deleteMonitorQuery(ctx context.Context, args *graphqlbackend.DeleteCodeMonitorArgs) (*sqlf.Query, error) {
	var monitorID int64
	err := relay.UnmarshalSpec(args.Id, &monitorID)
	if err != nil {
		return nil, err
	}
	query := sqlf.Sprintf(
		deleteMonitorFmtStr,
		monitorID,
	)
	return query, nil
}

func scanMonitors(rows *sql.Rows) ([]*Monitor, error) {
	var ms []*Monitor
	for rows.Next() {
		m := &Monitor{}
		if err := rows.Scan(
			&m.ID,
			&m.CreatedBy,
			&m.CreatedAt,
			&m.ChangedBy,
			&m.ChangedAt,
			&m.Description,
			&m.Enabled,
			&m.NamespaceUserID,
			&m.NamespaceOrgID,
		); err != nil {
			return nil, err
		}
		ms = append(ms, m)
	}
	err := rows.Close()
	if err != nil {
		return nil, err
	}
	// Rows.Err will report the last error encountered by Rows.Scan.
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ms, nil
}

func (s *Store) runMonitorQuery(ctx context.Context, q *sqlf.Query) (*Monitor, error) {
	rows, err := s.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ms, err := scanMonitors(rows)
	if err != nil {
		return nil, err
	}
	if len(ms) == 0 {
		return nil, errors.Errorf("operation failed. Query should have returned 1 row")
	}
	return ms[0], nil
}
