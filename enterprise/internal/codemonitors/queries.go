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

type MonitorQuery struct {
	Id           int64
	Monitor      int64
	QueryString  string
	NextRun      time.Time
	LatestResult *time.Time
	CreatedBy    int32
	CreatedAt    time.Time
	ChangedBy    int32
	ChangedAt    time.Time
}

var queryColumns = []*sqlf.Query{
	sqlf.Sprintf("cm_queries.id"),
	sqlf.Sprintf("cm_queries.monitor"),
	sqlf.Sprintf("cm_queries.query"),
	sqlf.Sprintf("cm_queries.next_run"),
	sqlf.Sprintf("cm_queries.created_by"),
	sqlf.Sprintf("cm_queries.created_at"),
	sqlf.Sprintf("cm_queries.changed_by"),
	sqlf.Sprintf("cm_queries.changed_at"),
}

func (s *Store) CreateTriggerQuery(ctx context.Context, monitorID int64, args *graphqlbackend.CreateTriggerArgs) (err error) {
	var q *sqlf.Query
	q, err = s.createTriggerQueryQuery(ctx, monitorID, args)
	if err != nil {
		return err
	}
	return s.Exec(ctx, q)
}

func (s *Store) UpdateTriggerQuery(ctx context.Context, args *graphqlbackend.UpdateCodeMonitorArgs) (err error) {
	var q *sqlf.Query
	q, err = s.updateTriggerQueryQuery(ctx, args)
	if err != nil {
		return err
	}
	return s.Exec(ctx, q)
}

const triggerQueryByMonitorFmtStr = `
SELECT id, monitor, query, next_run, latest_result, created_by, created_at, changed_by, changed_at
FROM cm_queries
WHERE monitor = %s;
`

func (s *Store) TriggerQueryByMonitorIDInt64(ctx context.Context, monitorID int64) (*MonitorQuery, error) {
	return s.runTriggerQuery(ctx, sqlf.Sprintf(triggerQueryByMonitorFmtStr, monitorID))
}

const triggerQueryByIDFmtStr = `
SELECT id, monitor, query, next_run, latest_result, created_by, created_at, changed_by, changed_at
FROM cm_queries
WHERE id = %s;
`

func (s *Store) triggerQueryByIDInt64(ctx context.Context, queryID int64) (*MonitorQuery, error) {
	return s.runTriggerQuery(ctx, sqlf.Sprintf(triggerQueryByIDFmtStr, queryID))
}

const resetTriggerQueryTimestamps = `
UPDATE cm_queries
SET latest_result = null,
    next_run = %s
WHERE id = %s;
`

func (s *Store) ResetTriggerQueryTimestamps(ctx context.Context, queryID int64) error {
	return s.Exec(ctx, sqlf.Sprintf(resetTriggerQueryTimestamps, s.Now(), queryID))
}

const createTriggerQueryFmtStr = `
INSERT INTO cm_queries
(monitor, query, created_by, created_at, changed_by, changed_at, next_run, latest_result)
VALUES (%s,%s,%s,%s,%s,%s,%s,%s)
RETURNING %s;
`

func (s *Store) createTriggerQueryQuery(ctx context.Context, monitorID int64, args *graphqlbackend.CreateTriggerArgs) (*sqlf.Query, error) {
	now := s.Now()
	a := actor.FromContext(ctx)
	return sqlf.Sprintf(
		createTriggerQueryFmtStr,
		monitorID,
		args.Query,
		a.UID,
		now,
		a.UID,
		now,
		now,
		now,
		sqlf.Join(queryColumns, ", "),
	), nil
}

const updateTriggerQueryFmtStr = `
UPDATE cm_queries
SET query = %s,
	changed_by = %s,
	changed_at = %s,
	latest_result = %s
WHERE id = %s
AND monitor = %s
RETURNING %s;
`

func (s *Store) updateTriggerQueryQuery(ctx context.Context, args *graphqlbackend.UpdateCodeMonitorArgs) (q *sqlf.Query, err error) {
	now := s.Now()
	a := actor.FromContext(ctx)

	var triggerID int64
	err = relay.UnmarshalSpec(args.Trigger.Id, &triggerID)
	if err != nil {
		return nil, err
	}

	var monitorID int64
	err = relay.UnmarshalSpec(args.Monitor.Id, &monitorID)
	if err != nil {
		return nil, err
	}

	return sqlf.Sprintf(
		updateTriggerQueryFmtStr,
		args.Trigger.Update.Query,
		a.UID,
		now,
		now,
		triggerID,
		monitorID,
		sqlf.Join(queryColumns, ", "),
	), nil
}

const getQueryByRecordIDFmtStr = `
SELECT q.id, q.monitor, q.query, q.next_run, q.latest_result, q.created_by, q.created_at, q.changed_by, q.changed_at
FROM cm_queries q INNER JOIN cm_trigger_jobs j ON q.id = j.query
WHERE j.id = %s
`

func (s *Store) GetQueryByRecordID(ctx context.Context, recordID int) (query *MonitorQuery, err error) {
	q := sqlf.Sprintf(
		getQueryByRecordIDFmtStr,
		recordID,
	)
	var ms []*MonitorQuery
	var rows *sql.Rows
	rows, err = s.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	ms, err = scanTriggerQueries(rows)
	if err != nil {
		return nil, err
	}
	if len(ms) != 1 {
		return nil, errors.Errorf("query should have returned 1 row")
	}
	return ms[0], nil
}

const setTriggerQueryNextRunFmtStr = `
UPDATE cm_queries
SET next_run = %s,
latest_result = %s
WHERE id = %s
`

func (s *Store) SetTriggerQueryNextRun(ctx context.Context, triggerQueryID int64, next time.Time, latestResults time.Time) error {
	q := sqlf.Sprintf(
		setTriggerQueryNextRunFmtStr,
		next,
		latestResults,
		triggerQueryID,
	)
	return s.Exec(ctx, q)
}

func scanTriggerQueries(rows *sql.Rows) (ms []*MonitorQuery, err error) {
	for rows.Next() {
		m := &MonitorQuery{}
		if err := rows.Scan(
			&m.Id,
			&m.Monitor,
			&m.QueryString,
			&m.NextRun,
			&m.LatestResult,
			&m.CreatedBy,
			&m.CreatedAt,
			&m.ChangedBy,
			&m.ChangedAt,
		); err != nil {
			return nil, err
		}
		ms = append(ms, m)
	}
	err = rows.Close()
	if err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ms, nil
}

func (s *Store) runTriggerQuery(ctx context.Context, q *sqlf.Query) (*MonitorQuery, error) {
	rows, err := s.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ms, err := scanTriggerQueries(rows)
	if err != nil {
		return nil, err
	}
	if len(ms) == 0 {
		return nil, errors.Errorf("operation failed. Query should have returned 1 row")
	}
	return ms[0], nil
}
