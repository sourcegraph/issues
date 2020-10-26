// Package usagestats provides an interface to update and access information about
// individual and aggregate Sourcegraph users' activity levels.
package usagestats

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"strconv"
	"time"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/timeutil"
)

var (
	timeNow = time.Now
)

// GetArchive generates and returns a usage statistics ZIP archive containing the CSV
// files defined in RFC 145, or an error in case of failure.
func GetArchive(ctx context.Context) ([]byte, error) {
	counts, err := db.EventLogs.UsersUsageCounts(ctx)
	if err != nil {
		return nil, err
	}

	dates, err := db.Users.ListDates(ctx)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	countsFile, err := zw.Create("UsersUsageCounts.csv")
	if err != nil {
		return nil, err
	}

	countsWriter := csv.NewWriter(countsFile)

	record := []string{
		"date",
		"user_id",
		"search_count",
		"code_intel_count",
	}

	if err := countsWriter.Write(record); err != nil {
		return nil, err
	}

	for _, c := range counts {
		record[0] = c.Date.UTC().Format(time.RFC3339)
		record[1] = strconv.FormatUint(uint64(c.UserID), 10)
		record[2] = strconv.FormatInt(int64(c.SearchCount), 10)
		record[3] = strconv.FormatInt(int64(c.CodeIntelCount), 10)

		if err := countsWriter.Write(record); err != nil {
			return nil, err
		}
	}

	countsWriter.Flush()

	datesFile, err := zw.Create("UsersDates.csv")
	if err != nil {
		return nil, err
	}

	datesWriter := csv.NewWriter(datesFile)

	record = record[:3]
	record[0] = "user_id"
	record[1] = "created_at"
	record[2] = "deleted_at"

	if err := datesWriter.Write(record); err != nil {
		return nil, err
	}

	for _, d := range dates {
		record[0] = strconv.FormatUint(uint64(d.UserID), 10)
		record[1] = d.CreatedAt.UTC().Format(time.RFC3339)
		if d.DeletedAt.IsZero() {
			record[2] = "NULL"
		} else {
			record[2] = d.DeletedAt.UTC().Format(time.RFC3339)
		}

		if err := datesWriter.Write(record); err != nil {
			return nil, err
		}
	}

	datesWriter.Flush()

	if err := zw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

var MockGetByUserID func(userID int32) (*types.UserUsageStatistics, error)

// GetByUserID returns a single user's UserUsageStatistics.
func GetByUserID(ctx context.Context, userID int32) (*types.UserUsageStatistics, error) {
	if MockGetByUserID != nil {
		return MockGetByUserID(userID)
	}

	pageViews, err := db.EventLogs.CountByUserIDAndEventNamePrefix(ctx, userID, "View")
	if err != nil {
		return nil, err
	}
	searchQueries, err := db.EventLogs.CountByUserIDAndEventName(ctx, userID, "SearchResultsQueried")
	if err != nil {
		return nil, err
	}
	codeIntelligenceActions, err := db.EventLogs.CountByUserIDAndEventNames(ctx, userID, []string{"hover", "findReferences", "goToDefinition.preloaded", "goToDefinition"})
	if err != nil {
		return nil, err
	}
	findReferencesActions, err := db.EventLogs.CountByUserIDAndEventName(ctx, userID, "findReferences")
	if err != nil {
		return nil, err
	}
	lastActiveTime, err := db.EventLogs.MaxTimestampByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	lastCodeHostIntegrationTime, err := db.EventLogs.MaxTimestampByUserIDAndSource(ctx, userID, "CODEHOSTINTEGRATION")
	if err != nil {
		return nil, err
	}
	return &types.UserUsageStatistics{
		UserID:                      userID,
		PageViews:                   int32(pageViews),
		SearchQueries:               int32(searchQueries),
		CodeIntelligenceActions:     int32(codeIntelligenceActions),
		FindReferencesActions:       int32(findReferencesActions),
		LastActiveTime:              lastActiveTime,
		LastCodeHostIntegrationTime: lastCodeHostIntegrationTime,
	}, nil
}

// GetUsersActiveTodayCount returns a count of users that have been active today.
func GetUsersActiveTodayCount(ctx context.Context) (int, error) {
	now := timeNow().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return db.EventLogs.CountUniqueUsersAll(ctx, today, today.AddDate(0, 0, 1))
}

// ListRegisteredUsersToday returns a list of the registered users that were active today.
func ListRegisteredUsersToday(ctx context.Context) ([]int32, error) {
	now := timeNow().UTC()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return db.EventLogs.ListUniqueUsersAll(ctx, start, start.AddDate(0, 0, 1))
}

// ListRegisteredUsersThisWeek returns a list of the registered users that were active this week.
func ListRegisteredUsersThisWeek(ctx context.Context) ([]int32, error) {
	start := timeutil.StartOfWeek(timeNow().UTC(), 0)
	return db.EventLogs.ListUniqueUsersAll(ctx, start, start.AddDate(0, 0, 7))
}

// ListRegisteredUsersThisMonth returns a list of the registered users that were active this month.
func ListRegisteredUsersThisMonth(ctx context.Context) ([]int32, error) {
	now := timeNow().UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	return db.EventLogs.ListUniqueUsersAll(ctx, start, start.AddDate(0, 1, 0))
}

// SiteUsageStatisticsOptions contains options for the number of daily, weekly, and monthly periods in
// which to calculate the number of unique users (i.e., how many days of Daily Active Users, or DAUs,
// how many weeks of Weekly Active Users, or WAUs, and how many months of Monthly Active Users, or MAUs).
type SiteUsageStatisticsOptions struct {
	DayPeriods   *int
	WeekPeriods  *int
	MonthPeriods *int
}

// GetSiteUsageStatistics returns the current site's SiteActivity.
func GetSiteUsageStatistics(ctx context.Context, opt *SiteUsageStatisticsOptions) (*types.SiteUsageStatistics, error) {
	var (
		dayPeriods   = defaultDays
		weekPeriods  = defaultWeeks
		monthPeriods = defaultMonths
	)

	if opt != nil {
		if opt.DayPeriods != nil {
			dayPeriods = minIntOrZero(maxStorageDays, *opt.DayPeriods)
		}
		if opt.WeekPeriods != nil {
			weekPeriods = minIntOrZero(maxStorageDays/7, *opt.WeekPeriods)
		}
		if opt.MonthPeriods != nil {
			monthPeriods = minIntOrZero(maxStorageDays/31, *opt.MonthPeriods)
		}
	}

	daus, err := activeUsers(ctx, db.Daily, dayPeriods)
	if err != nil {
		return nil, err
	}
	waus, err := activeUsers(ctx, db.Weekly, weekPeriods)
	if err != nil {
		return nil, err
	}
	maus, err := activeUsers(ctx, db.Monthly, monthPeriods)
	if err != nil {
		return nil, err
	}
	return &types.SiteUsageStatistics{
		DAUs: daus,
		WAUs: waus,
		MAUs: maus,
	}, nil
}

// activeUsers returns counts of active users in the given number of days, weeks, or months, as selected (including the current, partially completed period).
func activeUsers(ctx context.Context, periodType db.PeriodType, periods int) ([]*types.SiteActivityPeriod, error) {
	if periods == 0 {
		return []*types.SiteActivityPeriod{}, nil
	}

	uniqueUsers, err := db.EventLogs.CountUniqueUsersPerPeriod(ctx, periodType, timeNow().UTC(), periods, nil)
	if err != nil {
		return nil, err
	}
	registeredUniqueUsers, err := db.EventLogs.CountUniqueUsersPerPeriod(ctx, periodType, timeNow().UTC(), periods, &db.CountUniqueUsersOptions{
		RegisteredOnly: true,
	})
	if err != nil {
		return nil, err
	}
	integrationUniqueUsers, err := db.EventLogs.CountUniqueUsersPerPeriod(ctx, periodType, timeNow().UTC(), periods, &db.CountUniqueUsersOptions{
		IntegrationOnly: true,
	})
	if err != nil {
		return nil, err
	}

	var activeUsers []*types.SiteActivityPeriod
	for i, u := range uniqueUsers {
		// Pull out data from each period. Note that CountUniqueUsersPerPeriod will
		// always return a slice of length `periods` due to the generate_series in
		// the base query, so it is safe to read the following indices.
		actPer := &types.SiteActivityPeriod{
			StartTime:            u.Start,
			UserCount:            int32(u.Count),
			RegisteredUserCount:  int32(registeredUniqueUsers[i].Count),
			AnonymousUserCount:   int32(u.Count - registeredUniqueUsers[i].Count),
			IntegrationUserCount: int32(integrationUniqueUsers[i].Count),
		}
		activeUsers = append(activeUsers, actPer)
	}

	return activeUsers, nil
}

func minIntOrZero(a, b int) int {
	min := b
	if a < b {
		min = a
	}
	if min < 0 {
		return 0
	}
	return min
}
