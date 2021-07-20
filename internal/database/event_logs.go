package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/keegancsmith/sqlf"
	"github.com/lib/pq"

	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/featureflag"
	"github.com/sourcegraph/sourcegraph/internal/timeutil"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/internal/version"
)

const (
	integrationSource = "CODEHOSTINTEGRATION"
)

type EventLogStore struct {
	*basestore.Store
}

// EventLogs instantiates and returns a new EventLogStore with prepared statements.
func EventLogs(db dbutil.DB) *EventLogStore {
	return &EventLogStore{Store: basestore.NewWithDB(db, sql.TxOptions{})}
}

// NewEventLogStoreWithDB instantiates and returns a new EventLogStore using the other store handle.
func EventLogsWith(other basestore.ShareableStore) *EventLogStore {
	return &EventLogStore{Store: basestore.NewWithHandle(other.Handle())}
}

func (l *EventLogStore) With(other basestore.ShareableStore) *EventLogStore {
	return &EventLogStore{Store: l.Store.With(other)}
}

func (l *EventLogStore) Transact(ctx context.Context) (*EventLogStore, error) {
	txBase, err := l.Store.Transact(ctx)
	return &EventLogStore{Store: txBase}, err
}

// Event contains information needed for logging an event.
type Event struct {
	Name            string
	URL             string
	UserID          uint32
	AnonymousUserID string
	Argument        json.RawMessage
	Source          string
	Timestamp       time.Time
	FeatureFlags    featureflag.FlagSet
	CohortID        *string // date in YYYY-MM-DD format
}

func (l *EventLogStore) Insert(ctx context.Context, e *Event) error {
	argument := e.Argument
	if argument == nil {
		argument = json.RawMessage([]byte(`{}`))
	}

	featureFlags, err := json.Marshal(e.FeatureFlags)
	if err != nil {
		return err
	}

	_, err = l.Handle().DB().ExecContext(
		ctx,
		"INSERT INTO event_logs(name, url, user_id, anonymous_user_id, source, argument, version, timestamp, feature_flags, cohort_id) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)",
		e.Name,
		e.URL,
		e.UserID,
		e.AnonymousUserID,
		e.Source,
		argument,
		version.Version(),
		e.Timestamp.UTC(),
		featureFlags,
		e.CohortID,
	)
	if err != nil {
		return errors.Wrap(err, "INSERT")
	}
	return nil
}

func (l *EventLogStore) getBySQL(ctx context.Context, querySuffix *sqlf.Query) ([]*types.Event, error) {
	q := sqlf.Sprintf("SELECT id, name, url, user_id, anonymous_user_id, source, argument, version, timestamp FROM event_logs %s", querySuffix)
	rows, err := l.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []*types.Event{}
	for rows.Next() {
		r := types.Event{}
		err := rows.Scan(&r.ID, &r.Name, &r.URL, &r.UserID, &r.AnonymousUserID, &r.Source, &r.Argument, &r.Version, &r.Timestamp)
		if err != nil {
			return nil, err
		}
		events = append(events, &r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

// EventLogsListOptions specifies the options for listing event logs.
type EventLogsListOptions struct {
	// UserID specifies the user whose events should be included.
	UserID int32

	*LimitOffset

	EventName *string
}

// ListAll gets all event logs in descending order of timestamp.
func (l *EventLogStore) ListAll(ctx context.Context, opt EventLogsListOptions) ([]*types.Event, error) {
	conds := []*sqlf.Query{sqlf.Sprintf("TRUE")}
	if opt.UserID != 0 {
		conds = append(conds, sqlf.Sprintf("user_id = %d", opt.UserID))
	}
	if opt.EventName != nil {
		conds = append(conds, sqlf.Sprintf("name = %s", opt.EventName))
	}
	return l.getBySQL(ctx, sqlf.Sprintf("WHERE %s ORDER BY timestamp DESC %s", sqlf.Join(conds, "AND"), opt.LimitOffset.SQL()))
}

// LatestPing returns the most recently recorded ping event.
func (l *EventLogStore) LatestPing(ctx context.Context) (*types.Event, error) {
	if Mocks.EventLogs.LatestPing != nil {
		return Mocks.EventLogs.LatestPing(ctx)
	}

	rows, err := l.getBySQL(ctx, sqlf.Sprintf(`WHERE name='ping' ORDER BY id DESC LIMIT 1`))
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, sql.ErrNoRows
	}
	return rows[0], err
}

// CountByUserID gets a count of events logged by a given user.
func (l *EventLogStore) CountByUserID(ctx context.Context, userID int32) (int, error) {
	return l.countBySQL(ctx, sqlf.Sprintf("WHERE user_id = %d", userID))
}

// CountByUserIDAndEventName gets a count of events logged by a given user and with a given event name.
func (l *EventLogStore) CountByUserIDAndEventName(ctx context.Context, userID int32, name string) (int, error) {
	return l.countBySQL(ctx, sqlf.Sprintf("WHERE user_id = %d AND name = %s", userID, name))
}

// CountByUserIDAndEventNamePrefix gets a count of events logged by a given user and with a given event name prefix.
func (l *EventLogStore) CountByUserIDAndEventNamePrefix(ctx context.Context, userID int32, namePrefix string) (int, error) {
	return l.countBySQL(ctx, sqlf.Sprintf("WHERE user_id = %d AND name LIKE %s", userID, namePrefix+"%"))
}

// CountByUserIDAndEventNames gets a count of events logged by a given user that match a list of given event names.
func (l *EventLogStore) CountByUserIDAndEventNames(ctx context.Context, userID int32, names []string) (int, error) {
	items := []*sqlf.Query{}
	for _, v := range names {
		items = append(items, sqlf.Sprintf("%s", v))
	}
	return l.countBySQL(ctx, sqlf.Sprintf("WHERE user_id = %d AND name IN (%s)", userID, sqlf.Join(items, ",")))
}

// countBySQL gets a count of event logs.
func (l *EventLogStore) countBySQL(ctx context.Context, querySuffix *sqlf.Query) (int, error) {
	q := sqlf.Sprintf("SELECT COUNT(*) FROM event_logs %s", querySuffix)
	r := l.QueryRow(ctx, q)
	var count int
	err := r.Scan(&count)
	return count, err
}

// MaxTimestampByUserID gets the max timestamp among event logs for a given user.
func (l *EventLogStore) MaxTimestampByUserID(ctx context.Context, userID int32) (*time.Time, error) {
	return l.maxTimestampBySQL(ctx, sqlf.Sprintf("WHERE user_id = %d", userID))
}

// MaxTimestampByUserIDAndSource gets the max timestamp among event logs for a given user and event source.
func (l *EventLogStore) MaxTimestampByUserIDAndSource(ctx context.Context, userID int32, source string) (*time.Time, error) {
	return l.maxTimestampBySQL(ctx, sqlf.Sprintf("WHERE user_id = %d AND source = %s", userID, source))
}

// maxTimestampBySQL gets the max timestamp among event logs.
func (l *EventLogStore) maxTimestampBySQL(ctx context.Context, querySuffix *sqlf.Query) (*time.Time, error) {
	q := sqlf.Sprintf("SELECT MAX(timestamp) FROM event_logs %s", querySuffix)
	r := l.QueryRow(ctx, q)

	var t time.Time
	err := r.Scan(&dbutil.NullTime{Time: &t})
	if t.IsZero() {
		return nil, err
	}
	return &t, err
}

// UsageValue is a single count of usage for a time period starting on a given date.
type UsageValue struct {
	Start time.Time
	Count int
}

// PeriodType is the type of period in which to count events and unique users.
type PeriodType string

const (
	// Daily is used to get a count of events or unique users within a day.
	Daily PeriodType = "daily"
	// Weekly is used to get a count of events or unique users within a week.
	Weekly PeriodType = "weekly"
	// Monthly is used to get a count of events or unique users within a month.
	Monthly PeriodType = "monthly"
)

// intervalByPeriodType is a map of generate_series step values by period type.
var intervalByPeriodType = map[PeriodType]*sqlf.Query{
	Daily:   sqlf.Sprintf("'1 day'"),
	Weekly:  sqlf.Sprintf("'1 week'"),
	Monthly: sqlf.Sprintf("'1 month'"),
}

// periodByPeriodType is a map of SQL fragments that produce a timestamp bucket by period
// type. This assumes the existence of a  field named `timestamp` in the enclosing query.
var periodByPeriodType = map[PeriodType]*sqlf.Query{
	Daily:   sqlf.Sprintf(makeDateTruncExpression("day", "timestamp")),
	Weekly:  sqlf.Sprintf(makeDateTruncExpression("week", "timestamp")),
	Monthly: sqlf.Sprintf(makeDateTruncExpression("month", "timestamp")),
}

// calcStartDate calculates the the starting date of a number of periods given the period type.
// from the current time supplied as `now`. Returns a second false value if the period type is
// illegal.
func calcStartDate(now time.Time, periodType PeriodType, periods int) (time.Time, bool) {
	periodsAgo := periods - 1

	switch periodType {
	case Daily:
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -periodsAgo), true
	case Weekly:
		return timeutil.StartOfWeek(now, periodsAgo), true
	case Monthly:
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -periodsAgo, 0), true
	}
	return time.Time{}, false
}

// calcEndDate calculates the the ending date of a number of periods given the period type.
// Returns a second false value if the period type is illegal.
func calcEndDate(startDate time.Time, periodType PeriodType, periods int) (time.Time, bool) {
	periodsAgo := periods - 1

	switch periodType {
	case Daily:
		return startDate.AddDate(0, 0, periodsAgo), true
	case Weekly:
		return startDate.AddDate(0, 0, 7*periodsAgo), true
	case Monthly:
		return startDate.AddDate(0, periodsAgo, 0), true
	}

	return time.Time{}, false
}

// CountUniqueUsersOptions provides options for counting unique users.
type CountUniqueUsersOptions struct {
	// If true, only include registered users. Otherwise, include all users.
	RegisteredOnly bool
	// If true, only include code host integration users. Otherwise, include all users.
	IntegrationOnly bool
	// If set, adds additional restrictions on the event types.
	EventFilters *EventFilterOptions
}

// EventFilterOptions provides options for filtering events.
type EventFilterOptions struct {
	// If set, only include events with a given prefix.
	ByEventNamePrefix string
	// If set, only include events with the given name.
	ByEventName string
	// If not empty, only include events that matche a list of given event names
	ByEventNames []string
	// Must be used with ByEventName
	//
	// If set, only include events that match a specified condition.
	ByEventNameWithCondition *sqlf.Query
}

// EventArgumentMatch provides the options for matching an event with
// a specific JSON value passed as an argument.
type EventArgumentMatch struct {
	// The name of the JSON key to match against.
	ArgumentName string
	// The actual value passed to the JSON key to match.
	ArgumentValue string
}

// PercentileValue is a slice of Nth percentile values calculated from a field of events
// in a time period starting on a given date.
type PercentileValue struct {
	Start  time.Time
	Values []float64
}

// CountUniqueUsersPerPeriod provides a count of unique active users in a given time span, broken up into periods of
// a given type. The value of `now` should be the current time in UTC. Returns an array of length `periods`, with one
// entry for each period in the time span.
func (l *EventLogStore) CountUniqueUsersPerPeriod(ctx context.Context, periodType PeriodType, now time.Time, periods int, opt *CountUniqueUsersOptions) ([]UsageValue, error) {
	startDate, ok := calcStartDate(now, periodType, periods)
	if !ok {
		return nil, errors.Errorf("periodType must be \"daily\", \"weekly\", or \"monthly\". Got %s", periodType)
	}

	endDate, ok := calcEndDate(startDate, periodType, periods)
	if !ok {
		return nil, errors.Errorf("periodType must be \"daily\", \"weekly\", or \"monthly\". Got %s", periodType)
	}

	conds := []*sqlf.Query{sqlf.Sprintf("TRUE")}
	if opt != nil {
		if opt.RegisteredOnly {
			conds = append(conds, sqlf.Sprintf("user_id > 0"))
		}
		if opt.IntegrationOnly {
			conds = append(conds, sqlf.Sprintf("source = %s", integrationSource))
		}
		if opt.EventFilters != nil {
			if opt.EventFilters.ByEventNamePrefix != "" {
				conds = append(conds, sqlf.Sprintf("name LIKE %s", opt.EventFilters.ByEventNamePrefix+"%"))
			}
			if opt.EventFilters.ByEventName != "" {
				conds = append(conds, sqlf.Sprintf("name = %s", opt.EventFilters.ByEventName))
			}
			if opt.EventFilters.ByEventNameWithCondition != nil {
				conds = append(conds, opt.EventFilters.ByEventNameWithCondition)
			}
			if len(opt.EventFilters.ByEventNames) > 0 {
				items := []*sqlf.Query{}
				for _, v := range opt.EventFilters.ByEventNames {
					items = append(items, sqlf.Sprintf("%s", v))
				}
				conds = append(conds, sqlf.Sprintf("name IN (%s)", sqlf.Join(items, ",")))
			}
		}
	}

	return l.countUniqueUsersPerPeriodBySQL(ctx, intervalByPeriodType[periodType], periodByPeriodType[periodType], startDate, endDate, conds)
}

func (l *EventLogStore) countUniqueUsersPerPeriodBySQL(ctx context.Context, interval, period *sqlf.Query, startDate, endDate time.Time, conds []*sqlf.Query) ([]UsageValue, error) {
	return l.countPerPeriodBySQL(ctx, sqlf.Sprintf("DISTINCT "+userIDQueryFragment), interval, period, startDate, endDate, conds)
}

func (l *EventLogStore) countPerPeriodBySQL(ctx context.Context, countExpr, interval, period *sqlf.Query, startDate, endDate time.Time, conds []*sqlf.Query) ([]UsageValue, error) {
	allPeriods := sqlf.Sprintf("SELECT generate_series((%s)::timestamp, (%s)::timestamp, (%s)::interval) AS period", startDate, endDate, interval)
	countByPeriod := sqlf.Sprintf(`SELECT (%s) AS period, COUNT(%s) AS count
		FROM event_logs
		WHERE (%s)
		GROUP BY period`, period, countExpr, sqlf.Join(conds, ") AND ("))
	q := sqlf.Sprintf(`WITH all_periods AS (%s), count_by_period AS (%s)
		SELECT all_periods.period, COALESCE(count, 0)
		FROM all_periods
		LEFT OUTER JOIN count_by_period ON all_periods.period = (count_by_period.period)::timestamp
		ORDER BY period DESC`, allPeriods, countByPeriod)
	rows, err := l.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := []UsageValue{}
	for rows.Next() {
		var v UsageValue
		err := rows.Scan(&v.Start, &v.Count)
		if err != nil {
			return nil, err
		}
		v.Start = v.Start.UTC()
		counts = append(counts, v)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return counts, nil
}

// CountUniqueUsersAll provides a count of unique active users in a given time span.
func (l *EventLogStore) CountUniqueUsersAll(ctx context.Context, startDate, endDate time.Time) (int, error) {
	return l.countUniqueUsersBySQL(ctx, startDate, endDate, nil)
}

// CountUniqueUsersByEventNamePrefix provides a count of unique active users in a given time span that logged an event with a given prefix.
func (l *EventLogStore) CountUniqueUsersByEventNamePrefix(ctx context.Context, startDate, endDate time.Time, namePrefix string) (int, error) {
	return l.countUniqueUsersBySQL(ctx, startDate, endDate, sqlf.Sprintf("AND name LIKE %s ", namePrefix+"%"))
}

// CountUniqueUsersByEventName provides a count of unique active users in a given time span that logged a given event.
func (l *EventLogStore) CountUniqueUsersByEventName(ctx context.Context, startDate, endDate time.Time, name string) (int, error) {
	return l.countUniqueUsersBySQL(ctx, startDate, endDate, sqlf.Sprintf("AND name = %s", name))
}

// CountUniqueUsersByEventNames provides a count of unique active users in a given time span that logged any event that matches a list of given event names
func (l *EventLogStore) CountUniqueUsersByEventNames(ctx context.Context, startDate, endDate time.Time, names []string) (int, error) {
	items := []*sqlf.Query{}
	for _, v := range names {
		items = append(items, sqlf.Sprintf("%s", v))
	}
	return l.countUniqueUsersBySQL(ctx, startDate, endDate, sqlf.Sprintf("AND name IN (%s)", sqlf.Join(items, ",")))
}

func (l *EventLogStore) countUniqueUsersBySQL(ctx context.Context, startDate, endDate time.Time, querySuffix *sqlf.Query) (int, error) {
	if querySuffix == nil {
		querySuffix = sqlf.Sprintf("")
	}
	q := sqlf.Sprintf(`SELECT COUNT(DISTINCT `+userIDQueryFragment+`)
		FROM event_logs
		WHERE (DATE(TIMEZONE('UTC'::text, timestamp)) >= %s) AND (DATE(TIMEZONE('UTC'::text, timestamp)) <= %s) %s`, startDate, endDate, querySuffix)
	r := l.QueryRow(ctx, q)
	var count int
	err := r.Scan(&count)
	return count, err
}

func (l *EventLogStore) ListUniqueUsersAll(ctx context.Context, startDate, endDate time.Time) ([]int32, error) {
	rows, err := l.Handle().DB().QueryContext(ctx, `SELECT user_id
		FROM event_logs
		WHERE user_id > 0 AND DATE(TIMEZONE('UTC'::text, timestamp)) >= $1 AND DATE(TIMEZONE('UTC'::text, timestamp)) <= $2
		GROUP BY user_id`, startDate, endDate)
	if err != nil {
		return nil, err
	}
	var users []int32
	defer rows.Close()
	for rows.Next() {
		var userID int32
		err := rows.Scan(&userID)
		if err != nil {
			return nil, err
		}
		users = append(users, userID)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

// UsersUsageCounts returns a list of UserUsageCounts for all active users that produced 'SearchResultsQueried' and any
// '%codeintel%' events in the event_logs table.
func (l *EventLogStore) UsersUsageCounts(ctx context.Context) (counts []types.UserUsageCounts, err error) {
	rows, err := l.Handle().DB().QueryContext(ctx, usersUsageCountsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var c types.UserUsageCounts

		err := rows.Scan(
			&c.Date,
			&c.UserID,
			&dbutil.NullInt32{N: &c.SearchCount},
			&dbutil.NullInt32{N: &c.CodeIntelCount},
		)

		if err != nil {
			return nil, err
		}

		counts = append(counts, c)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return counts, nil
}

const usersUsageCountsQuery = `
-- source: internal/database/event_logs.go:UsersUsageCounts
SELECT
  DATE(timestamp),
  user_id,
  COUNT(*) FILTER (WHERE event_logs.name ='SearchResultsQueried') as search_count,
  COUNT(*) FILTER (WHERE event_logs.name LIKE '%codeintel%') as codeintel_count
FROM event_logs
GROUP BY 1, 2
ORDER BY 1 DESC, 2 ASC;
`

func (l *EventLogStore) SiteUsage(ctx context.Context) (types.SiteUsageSummary, error) {
	return l.siteUsage(ctx, time.Now().UTC())
}

func (l *EventLogStore) siteUsage(ctx context.Context, now time.Time) (summary types.SiteUsageSummary, err error) {
	query := sqlf.Sprintf(siteUsageQuery, now, now, now, now)

	err = l.QueryRow(ctx, query).Scan(
		&summary.Month,
		&summary.Week,
		&summary.Day,
		&summary.UniquesMonth,
		&summary.UniquesWeek,
		&summary.UniquesDay,
		&summary.RegisteredUniquesMonth,
		&summary.RegisteredUniquesWeek,
		&summary.RegisteredUniquesDay,
		&summary.IntegrationUniquesMonth,
		&summary.IntegrationUniquesWeek,
		&summary.IntegrationUniquesDay,
		&summary.ManageUniquesMonth,
		&summary.CodeUniquesMonth,
		&summary.VerifyUniquesMonth,
		&summary.MonitorUniquesMonth,
		&summary.ManageUniquesWeek,
		&summary.CodeUniquesWeek,
		&summary.VerifyUniquesWeek,
		&summary.MonitorUniquesWeek,
	)

	return summary, err
}

var siteUsageQuery = `
SELECT
  current_month,
  current_week,
  current_day,

  COUNT(DISTINCT user_id) FILTER (WHERE month = current_month) AS uniques_month,
  COUNT(DISTINCT user_id) FILTER (WHERE week = current_week) AS uniques_week,
  COUNT(DISTINCT user_id) FILTER (WHERE day = current_day) AS uniques_day,
  COUNT(DISTINCT user_id) FILTER (WHERE month = current_month AND registered) AS registered_uniques_month,
  COUNT(DISTINCT user_id) FILTER (WHERE week = current_week AND registered) AS registered_uniques_week,
  COUNT(DISTINCT user_id) FILTER (WHERE day = current_day AND registered) AS registered_uniques_day,
  COUNT(DISTINCT user_id) FILTER (WHERE month = current_month AND source = 'CODEHOSTINTEGRATION')
  	AS integration_uniques_month,
  COUNT(DISTINCT user_id) FILTER (WHERE week = current_week AND source = 'CODEHOSTINTEGRATION')
  	AS integration_uniques_week,
  COUNT(DISTINCT user_id) FILTER (WHERE day = current_day AND source = 'CODEHOSTINTEGRATION')
  	AS integration_uniques_day,

  COUNT(DISTINCT user_id) FILTER (
    WHERE month = current_month AND name LIKE 'ViewSiteAdmin%%%%'
  ) AS manage_uniques_month,

  COUNT(DISTINCT user_id) FILTER (
    WHERE month = current_month AND name IN (
      'ViewRepository',
      'ViewBlob',
      'ViewTree',
      'SearchResultsQueried'
    )
  ) AS code_uniques_month,

  COUNT(DISTINCT user_id) FILTER (
    WHERE month = current_month AND name IN (
      'SavedSearchEmailClicked',
      'SavedSearchSlackClicked',
      'SavedSearchEmailNotificationSent'
    )
  ) AS verify_uniques_month,

  COUNT(DISTINCT user_id) FILTER (
    WHERE month = current_month AND name IN (
      'DiffSearchResultsQueried'
    )
  ) AS monitor_uniques_month,

  COUNT(DISTINCT user_id) FILTER (
    WHERE week = current_week AND name LIKE 'ViewSiteAdmin%%%%'
  ) AS manage_uniques_week,

  COUNT(DISTINCT user_id) FILTER (
    WHERE week = current_week AND name IN (
      'ViewRepository',
      'ViewBlob',
      'ViewTree',
      'SearchResultsQueried'
    )
  ) AS code_uniques_week,

  COUNT(DISTINCT user_id) FILTER (
    WHERE week = current_week AND name IN (
      'SavedSearchEmailClicked',
      'SavedSearchSlackClicked',
      'SavedSearchEmailNotificationSent'
    )
  ) AS verify_uniques_week,

  COUNT(DISTINCT user_id) FILTER (
    WHERE week = current_week AND name IN (
      'DiffSearchResultsQueried'
    )
  ) AS monitor_uniques_week

FROM (
  -- This sub-query is here to avoid re-doing this work above on each aggregation.
  SELECT
    name,
    user_id != 0 as registered,
    ` + aggregatedUserIDQueryFragment + ` AS user_id,
    source,
    ` + makeDateTruncExpression("month", "timestamp") + ` as month,
    ` + makeDateTruncExpression("week", "timestamp") + ` as week,
    ` + makeDateTruncExpression("day", "timestamp") + ` as day,
    ` + makeDateTruncExpression("month", "%s::timestamp") + ` as current_month,
    ` + makeDateTruncExpression("week", "%s::timestamp") + ` as current_week,
    ` + makeDateTruncExpression("day", "%s::timestamp") + ` as current_day
  FROM event_logs
  WHERE timestamp >= ` + makeDateTruncExpression("month", "%s::timestamp") + `
) events

GROUP BY current_month, current_week, current_day
`

// CodeIntelligencePreciseWAUs returns the WAU (current week) with precise-based code intelligence events.
func (l *EventLogStore) CodeIntelligencePreciseWAUs(ctx context.Context) (int, error) {
	eventNames := []string{
		"codeintel.lsifHover",
		"codeintel.lsifDefinitions",
		"codeintel.lsifReferences",
	}

	return l.codeIntelligenceWeeklyUsersCount(ctx, eventNames, time.Now().UTC())
}

// CodeIntelligenceSearchBasedWAUs returns the WAU (current week) with searched-base code intelligence events.
func (l *EventLogStore) CodeIntelligenceSearchBasedWAUs(ctx context.Context) (int, error) {
	eventNames := []string{
		"codeintel.searchHover",
		"codeintel.searchDefinitions",
		"codeintel.searchReferences",
	}

	return l.codeIntelligenceWeeklyUsersCount(ctx, eventNames, time.Now().UTC())
}

// CodeIntelligenceWAUs returns the WAU (current week) with any (precise or search-based) code intelligence event.
func (l *EventLogStore) CodeIntelligenceWAUs(ctx context.Context) (int, error) {
	eventNames := []string{
		"codeintel.lsifHover",
		"codeintel.lsifDefinitions",
		"codeintel.lsifReferences",
		"codeintel.searchHover",
		"codeintel.searchDefinitions",
		"codeintel.searchReferences",
	}

	return l.codeIntelligenceWeeklyUsersCount(ctx, eventNames, time.Now().UTC())
}

// CodeIntelligenceCrossRepositoryWAUs returns the WAU (current week) with any (precise or search-based) cross-repository code intelligence event.
func (l *EventLogStore) CodeIntelligenceCrossRepositoryWAUs(ctx context.Context) (int, error) {
	eventNames := []string{
		"codeintel.lsifDefinitions.xrepo",
		"codeintel.lsifReferences.xrepo",
		"codeintel.searchDefinitions.xrepo",
		"codeintel.searchReferences.xrepo",
	}

	return l.codeIntelligenceWeeklyUsersCount(ctx, eventNames, time.Now().UTC())
}

// CodeIntelligencePreciseCrossRepositoryWAUs returns the WAU (current week) with precise-based cross-repository code intelligence events.
func (l *EventLogStore) CodeIntelligencePreciseCrossRepositoryWAUs(ctx context.Context) (int, error) {
	eventNames := []string{
		"codeintel.lsifDefinitions.xrepo",
		"codeintel.lsifReferences.xrepo",
	}

	return l.codeIntelligenceWeeklyUsersCount(ctx, eventNames, time.Now().UTC())
}

// CodeIntelligenceSearchBasedCrossRepositoryWAUs returns the WAU (current week) with searched-base cross-repository code intelligence events.
func (l *EventLogStore) CodeIntelligenceSearchBasedCrossRepositoryWAUs(ctx context.Context) (int, error) {
	eventNames := []string{
		"codeintel.searchDefinitions.xrepo",
		"codeintel.searchReferences.xrepo",
	}

	return l.codeIntelligenceWeeklyUsersCount(ctx, eventNames, time.Now().UTC())
}

func (l *EventLogStore) codeIntelligenceWeeklyUsersCount(ctx context.Context, eventNames []string, now time.Time) (wau int, _ error) {
	var names []*sqlf.Query
	for _, name := range eventNames {
		names = append(names, sqlf.Sprintf("%s", name))
	}

	if err := l.QueryRow(ctx, sqlf.Sprintf(codeIntelWeeklyUsersQuery, now, sqlf.Join(names, ", "))).Scan(&wau); err != nil {
		return 0, err
	}

	return wau, nil
}

var codeIntelWeeklyUsersQuery = `
-- source: internal/database/event_logs.go:codeIntelligenceWeeklyUsersCount
SELECT COUNT(DISTINCT ` + userIDQueryFragment + `)
FROM event_logs
WHERE
  timestamp >= ` + makeDateTruncExpression("week", "%s::timestamp") + `
  AND name IN (%s);
`

// CodeIntelligenceRepositoryCounts returns the number of repositories with and without an associated
// and up-to-date code intelligence upload.
func (l *EventLogStore) CodeIntelligenceRepositoryCounts(ctx context.Context) (withUploads int, withoutUploads int, err error) {
	var totalRepositories int

	rows, err := l.Query(ctx, sqlf.Sprintf(codeIntelligenceRepositoryCountsQuery))
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&totalRepositories, &withUploads); err != nil {
			return 0, 0, err
		}
	}

	if err := rows.Err(); err != nil {
		return 0, 0, err
	}

	return withUploads, totalRepositories - withUploads, nil
}

var codeIntelligenceRepositoryCountsQuery = `
-- source: internal/database/event_logs.go:CodeIntelligenceRepositoryCounts
WITH active_repositories AS (
	SELECT count(*) AS count FROM repo WHERE deleted_at IS NULL
),
active_repositories_with_upload AS (
	SELECT count(DISTINCT repository_id) AS count
	FROM lsif_dumps u
	JOIN repo r ON r.id = u.repository_id
	WHERE r.deleted_at IS NULL
)
SELECT
	(SELECT count FROM active_repositories) AS total_repositories,
	(SELECT count FROM active_repositories_with_upload) AS repositories_with_uploads
`

// AggregatedCodeIntelEvents calculates CodeIntelAggregatedEvent for each every unique event type related to code intel.
func (l *EventLogStore) AggregatedCodeIntelEvents(ctx context.Context) ([]types.CodeIntelAggregatedEvent, error) {
	return l.aggregatedCodeIntelEvents(ctx, time.Now().UTC())
}

func (l *EventLogStore) aggregatedCodeIntelEvents(ctx context.Context, now time.Time) (events []types.CodeIntelAggregatedEvent, err error) {
	var eventNames = []string{
		"codeintel.lsifHover",
		"codeintel.lsifDefinitions",
		"codeintel.lsifDefinitions.xrepo",
		"codeintel.lsifReferences",
		"codeintel.lsifReferences.xrepo",
		"codeintel.searchHover",
		"codeintel.searchDefinitions",
		"codeintel.searchDefinitions.xrepo",
		"codeintel.searchReferences",
		"codeintel.searchReferences.xrepo",
	}

	var eventNameQueries []*sqlf.Query
	for _, name := range eventNames {
		eventNameQueries = append(eventNameQueries, sqlf.Sprintf("%s", name))
	}

	query := sqlf.Sprintf(aggregatedCodeIntelEventsQuery, now, now, sqlf.Join(eventNameQueries, ", "))

	rows, err := l.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var event types.CodeIntelAggregatedEvent
		err := rows.Scan(
			&event.Name,
			&event.LanguageID,
			&event.Week,
			&event.TotalWeek,
			&event.UniquesWeek,
		)
		if err != nil {
			return nil, err
		}

		events = append(events, event)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

var aggregatedCodeIntelEventsQuery = `
-- source: internal/database/event_logs.go:aggregatedCodeIntelEvents
WITH events AS (
  SELECT
    name,
    (argument->>'languageId')::text as language_id,
    ` + aggregatedUserIDQueryFragment + ` AS user_id,
    ` + makeDateTruncExpression("week", "%s::timestamp") + ` as current_week
  FROM event_logs
  WHERE
    timestamp >= ` + makeDateTruncExpression("week", "%s::timestamp") + `
    AND name IN (%s)
)
SELECT
  name,
  language_id,
  current_week,
  COUNT(*) AS total_week,
  COUNT(DISTINCT user_id) AS uniques_week
FROM events GROUP BY name, current_week, language_id;
`

// AggregatedSearchEvents calculates SearchAggregatedEvent for each every unique event type related to search.
func (l *EventLogStore) AggregatedSearchEvents(ctx context.Context, now time.Time) ([]types.SearchAggregatedEvent, error) {
	latencyEvents, err := l.aggregatedSearchEvents(ctx, aggregatedSearchLatencyEventsQuery, now)
	if err != nil {
		return nil, err
	}

	usageEvents, err := l.aggregatedSearchEvents(ctx, aggregatedSearchUsageEventsQuery, now)
	if err != nil {
		return nil, err
	}
	return append(latencyEvents, usageEvents...), nil
}

func (l *EventLogStore) aggregatedSearchEvents(ctx context.Context, queryString string, now time.Time) (events []types.SearchAggregatedEvent, err error) {
	query := sqlf.Sprintf(queryString, now, now, now, now)

	rows, err := l.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var event types.SearchAggregatedEvent
		err := rows.Scan(
			&event.Name,
			&event.Month,
			&event.Week,
			&event.Day,
			&event.TotalMonth,
			&event.TotalWeek,
			&event.TotalDay,
			&event.UniquesMonth,
			&event.UniquesWeek,
			&event.UniquesDay,
			pq.Array(&event.LatenciesMonth),
			pq.Array(&event.LatenciesWeek),
			pq.Array(&event.LatenciesDay),
		)
		if err != nil {
			return nil, err
		}

		events = append(events, event)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

var searchLatencyEventNames = []string{
	"'search.latencies.literal'",
	"'search.latencies.regexp'",
	"'search.latencies.structural'",
	"'search.latencies.file'",
	"'search.latencies.repo'",
	"'search.latencies.diff'",
	"'search.latencies.commit'",
	"'search.latencies.symbol'",
}

var aggregatedSearchLatencyEventsQuery = `
-- source: internal/database/event_logs.go:aggregatedSearchLatencyEvents
WITH events AS (
  SELECT
    name,
    -- Postgres 9.6 needs to go from text to integer (i.e. can't go directly to integer)
    (argument->'durationMs')::text::integer as latency,
    ` + aggregatedUserIDQueryFragment + ` AS user_id,
    ` + makeDateTruncExpression("month", "timestamp") + ` as month,
    ` + makeDateTruncExpression("week", "timestamp") + ` as week,
    ` + makeDateTruncExpression("day", "timestamp") + ` as day,
    ` + makeDateTruncExpression("month", "%s::timestamp") + ` as current_month,
    ` + makeDateTruncExpression("week", "%s::timestamp") + ` as current_week,
    ` + makeDateTruncExpression("day", "%s::timestamp") + ` as current_day
  FROM event_logs
  WHERE
    timestamp >= ` + makeDateTruncExpression("month", "%s::timestamp") + `
    AND name IN (` + strings.Join(searchLatencyEventNames, ", ") + `)
)
SELECT
  name,
  current_month,
  current_week,
  current_day,
  COUNT(*) FILTER (WHERE month = current_month) AS total_month,
  COUNT(*) FILTER (WHERE week = current_week) AS total_week,
  COUNT(*) FILTER (WHERE day = current_day) AS total_day,
  COUNT(DISTINCT user_id) FILTER (WHERE month = current_month) AS uniques_month,
  COUNT(DISTINCT user_id) FILTER (WHERE week = current_week) AS uniques_week,
  COUNT(DISTINCT user_id) FILTER (WHERE day = current_day) AS uniques_day,
  PERCENTILE_CONT(ARRAY[0.50, 0.90, 0.99]) WITHIN GROUP (ORDER BY latency) FILTER (WHERE month = current_month) AS latencies_month,
  PERCENTILE_CONT(ARRAY[0.50, 0.90, 0.99]) WITHIN GROUP (ORDER BY latency) FILTER (WHERE week = current_week) AS latencies_week,
  PERCENTILE_CONT(ARRAY[0.50, 0.90, 0.99]) WITHIN GROUP (ORDER BY latency) FILTER (WHERE day = current_day) AS latencies_day
FROM events GROUP BY name, current_month, current_week, current_day
`

var aggregatedSearchUsageEventsQuery = `
-- source: internal/database/event_logs.go:aggregatedSearchUsageEvents
WITH events AS (
  SELECT
    json.key::text,
    json.value::text,
    ` + aggregatedUserIDQueryFragment + ` AS user_id,
    ` + makeDateTruncExpression("month", "timestamp") + ` as month,
    ` + makeDateTruncExpression("week", "timestamp") + ` as week,
    ` + makeDateTruncExpression("day", "timestamp") + ` as day,
    ` + makeDateTruncExpression("month", "%s::timestamp") + ` as current_month,
    ` + makeDateTruncExpression("week", "%s::timestamp") + ` as current_week,
    ` + makeDateTruncExpression("day", "%s::timestamp") + ` as current_day
  FROM event_logs
  CROSS JOIN LATERAL jsonb_each(argument->'code_search'->'query_data'->'query') json
  WHERE
    timestamp >= ` + makeDateTruncExpression("month", "%s::timestamp") + `
    AND name = 'SearchResultsQueried'
)
SELECT
  key,
  current_month,
  current_week,
  current_day,
  SUM(case when month = current_month then value::int else 0 end) AS total_month,
  SUM(case when week = current_week then value::int else 0 end) AS total_week,
  SUM(case when day = current_day then value::int else 0 end) AS total_day,
  COUNT(DISTINCT user_id) FILTER (WHERE month = current_month) AS uniques_month,
  COUNT(DISTINCT user_id) FILTER (WHERE week = current_week) AS uniques_week,
  COUNT(DISTINCT user_id) FILTER (WHERE day = current_day) AS uniques_day,
  NULL,
  NULL,
  NULL
FROM events
WHERE key IN
  (
	'count_or',
	'count_and',
	'count_not',
	'count_select_repo',
	'count_select_file',
	'count_select_content',
	'count_select_symbol',
	'count_select_commit_diff_added',
	'count_select_commit_diff_removed',
	'count_repo_contains',
	'count_repo_contains_file',
	'count_repo_contains_content',
	'count_repo_contains_commit_after',
	'count_count_all',
	'count_non_global_context',
	'count_only_patterns',
	'count_only_patterns_three_or_more'
  )
GROUP BY key, current_month, current_week, current_day
`

// userIDQueryFragment is a query fragment that can be used to return the anonymous user ID
// when the user ID is not set (i.e. 0).
const userIDQueryFragment = `
CASE WHEN user_id = 0
  THEN anonymous_user_id
  ELSE CAST(user_id AS TEXT)
END
`

// aggregatedUserIDQueryFragment is a query fragment that can be used to canonicalize the
// values of the user_id and anonymous_user_id fields (assumed in scope) int a unified value.
const aggregatedUserIDQueryFragment = `
CASE WHEN user_id = 0
  -- It's faster to group by an int rather than text, so we convert
  -- the anonymous_user_id to an int, rather than the user_id to text.
  THEN ('x' || substr(md5(anonymous_user_id), 1, 8))::bit(32)::int
  ELSE user_id
END
`

// makeDateTruncExpression returns an expresson that converts the given
// SQL expression into the start of the containing date container specified
// by the unit parameter (e.g. day, week, month, or).
func makeDateTruncExpression(unit, expr string) string {
	if unit == "week" {
		return fmt.Sprintf(`DATE_TRUNC('%s', TIMEZONE('UTC', %s) + '1 day'::interval) - '1 day'::interval`, unit, expr)
	}

	return fmt.Sprintf(`DATE_TRUNC('%s', TIMEZONE('UTC', %s))`, unit, expr)
}
