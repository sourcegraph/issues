package usagestatsdeprecated

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
)

func init() {
	// Prevent background GC from running
	gcOnce.Do(func() {})
}

func TestUserUsageStatistics_None(t *testing.T) {
	setupForTest(t)

	want := &types.UserUsageStatistics{
		UserID: 42,
	}
	got, err := GetByUserID(42)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("got %+v != %+v", got, want)
	}
}

func TestUserUsageStatistics_LogPageView(t *testing.T) {
	setupForTest(t)

	user := types.User{
		ID: 1,
	}
	err := LogActivity(true, user.ID, "test-cookie-id", "PAGEVIEW")
	if err != nil {
		t.Fatal(err)
	}

	a, err := GetByUserID(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if wantViews := int32(1); a.PageViews != wantViews {
		t.Errorf("got %d, want %d", a.PageViews, wantViews)
	}
	diff := (*a.LastActiveTime).Unix() - time.Now().Unix()
	if wantMaxDiff := 10; diff > int64(wantMaxDiff) || diff < -int64(wantMaxDiff) {
		t.Errorf("got %d seconds apart, wanted less than %d seconds apart", diff, wantMaxDiff)
	}
}

func TestUserUsageStatistics_LogSearchQuery(t *testing.T) {
	setupForTest(t)

	user := types.User{
		ID: 1,
	}
	err := LogActivity(true, user.ID, "test-cookie-id", "SEARCHQUERY")
	if err != nil {
		t.Fatal(err)
	}

	a, err := GetByUserID(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if want := int32(1); a.SearchQueries != want {
		t.Errorf("got %d, want %d", a.SearchQueries, want)
	}
}

func TestUserUsageStatistics_LogCodeIntelAction(t *testing.T) {
	setupForTest(t)

	user := types.User{
		ID: 1,
	}
	err := LogActivity(true, user.ID, "test-cookie-id", "CODEINTEL")
	if err != nil {
		t.Fatal(err)
	}

	a, err := GetByUserID(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if want := int32(1); a.CodeIntelligenceActions != want {
		t.Errorf("got %d, want %d", a.CodeIntelligenceActions, want)
	}
}

func TestUserUsageStatistics_LogCodeHostIntegrationUsage(t *testing.T) {
	setupForTest(t)

	user := types.User{
		ID: 1,
	}
	err := LogActivity(true, user.ID, "test-cookie-id", "CODEINTELINTEGRATION")
	if err != nil {
		t.Fatal(err)
	}

	a, err := GetByUserID(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	diff := (*a.LastCodeHostIntegrationTime).Unix() - time.Now().Unix()
	if wantMaxDiff := 10; diff > int64(wantMaxDiff) || diff < -int64(wantMaxDiff) {
		t.Errorf("got %d seconds apart, wanted less than %d seconds apart", diff, wantMaxDiff)
	}
}

func TestUserUsageStatistics_getUsersActiveToday(t *testing.T) {
	setupForTest(t)

	user1 := types.User{
		ID: 1,
	}
	user2 := types.User{
		ID: 2,
	}

	// Test single user
	err := LogActivity(true, user1.ID, "test-cookie-id-1", "PAGEVIEW")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	n, err := GetUsersActiveTodayCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if want := 1; n != want {
		t.Errorf("got %d, want %d", n, want)
	}

	// Test multiple users, with repeats
	err = LogActivity(true, user2.ID, "test-cookie-id-2", "PAGEVIEW")
	if err != nil {
		t.Fatal(err)
	}
	err = LogActivity(true, user1.ID, "test-cookie-id-1", "PAGEVIEW")
	if err != nil {
		t.Fatal(err)
	}
	err = LogActivity(false, 0, "test-cookie-id-3", "PAGEVIEW")
	if err != nil {
		t.Fatal(err)
	}
	err = LogActivity(true, user2.ID, "test-cookie-id-2", "PAGEVIEW")
	if err != nil {
		t.Fatal(err)
	}

	n, err = GetUsersActiveTodayCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if want := 3; n != want {
		t.Errorf("got %d, want %d", n, want)
	}
}

func TestUserUsageStatistics_DAUs_WAUs_MAUs(t *testing.T) {
	defer func() {
		timeNow = time.Now
	}()

	setupForTest(t)

	user1 := types.User{
		ID: 1,
	}
	user2 := types.User{
		ID: 2,
	}

	// hardcode "now" as 2018/03/31
	now := time.Date(2018, 3, 31, 12, 0, 0, 0, time.UTC)
	oneMonthFourDaysAgo := now.AddDate(0, -1, -4)
	oneMonthThreeDaysAgo := now.AddDate(0, -1, -3)
	twoWeeksTwoDaysAgo := now.AddDate(0, 0, -2*7-2)
	twoWeeksAgo := now.AddDate(0, 0, -2*7)
	fiveDaysAgo := now.AddDate(0, 0, -5)
	threeDaysAgo := now.AddDate(0, 0, -3)

	// 2018/02/27 (2 users, 1 registered)
	mockTimeNow(oneMonthFourDaysAgo)
	err := LogActivity(true, user1.ID, "test-1", "PAGEVIEW")
	if err != nil {
		t.Fatal(err)
	}
	err = LogActivity(false, 0, "068ccbfa-8529-4fa7-859e-2c3514af2434", "PAGEVIEW")
	if err != nil {
		t.Fatal(err)
	}
	// This should not be visible, as code host integration usage is ONLY recorded for registered users.
	err = LogActivity(false, 0, "068ccbfa-8529-4fa7-859e-2c3514af2434", "CODEINTELINTEGRATION")
	if err != nil {
		t.Fatal(err)
	}

	// 2018/02/28 (2 users, 1 registered)
	mockTimeNow(oneMonthThreeDaysAgo)
	err = LogActivity(true, user1.ID, "test-1", "PAGEVIEW")
	if err != nil {
		t.Fatal(err)
	}
	err = LogActivity(false, 0, "30dd2661-2e73-4774-bc2b-7a126f360734", "PAGEVIEW")
	if err != nil {
		t.Fatal(err)
	}

	// 2018/03/15 (2 users, 1 registered)
	mockTimeNow(twoWeeksTwoDaysAgo)
	err = LogActivity(true, user2.ID, "test-2", "PAGEVIEW")
	if err != nil {
		t.Fatal(err)
	}
	err = LogActivity(false, 0, "068ccbfa-8529-4fa7-859e-2c3514af2434", "PAGEVIEW")
	if err != nil {
		t.Fatal(err)
	}

	// 2018/03/17 (2 users, 1 registered)
	mockTimeNow(twoWeeksAgo)
	err = LogActivity(true, user2.ID, "test-2", "PAGEVIEW")
	if err != nil {
		t.Fatal(err)
	}
	err = LogActivity(false, 0, "b309dad0-b6f9-440d-bf0a-4cf38030ca70", "PAGEVIEW")
	if err != nil {
		t.Fatal(err)
	}
	err = LogActivity(true, user2.ID, "test-2", "CODEINTELINTEGRATION")
	if err != nil {
		t.Fatal(err)
	}

	// 2018/03/26 (1 user, 1 registered)
	mockTimeNow(fiveDaysAgo)
	err = LogActivity(true, user1.ID, "test-1", "PAGEVIEW")
	if err != nil {
		t.Fatal(err)
	}

	// 2018/03/28 (2 users, 2 registered)
	mockTimeNow(threeDaysAgo)
	err = LogActivity(true, user1.ID, "test-1", "PAGEVIEW")
	if err != nil {
		t.Fatal(err)
	}
	err = LogActivity(true, user2.ID, "test-2", "PAGEVIEW")
	if err != nil {
		t.Fatal(err)
	}
	err = LogActivity(true, user1.ID, "test-1", "CODEINTELINTEGRATION")
	if err != nil {
		t.Fatal(err)
	}

	wantMAUs := []*types.SiteActivityPeriod{
		{
			StartTime:            time.Date(2018, 3, 1, 0, 0, 0, 0, time.UTC),
			UserCount:            4,
			RegisteredUserCount:  2,
			AnonymousUserCount:   2,
			IntegrationUserCount: 2,
		},
		{
			StartTime:            time.Date(2018, 2, 1, 0, 0, 0, 0, time.UTC),
			UserCount:            3,
			RegisteredUserCount:  1,
			AnonymousUserCount:   2,
			IntegrationUserCount: 0,
		},
		{
			StartTime: time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	wantWAUs := []*types.SiteActivityPeriod{
		{
			StartTime:            time.Date(2018, 3, 25, 0, 0, 0, 0, time.UTC),
			UserCount:            2,
			RegisteredUserCount:  2,
			AnonymousUserCount:   0,
			IntegrationUserCount: 1,
		},
		{
			StartTime: time.Date(2018, 3, 18, 0, 0, 0, 0, time.UTC),
		},
		{
			StartTime:            time.Date(2018, 3, 11, 0, 0, 0, 0, time.UTC),
			UserCount:            3,
			RegisteredUserCount:  1,
			AnonymousUserCount:   2,
			IntegrationUserCount: 1,
		},
		{
			StartTime: time.Date(2018, 3, 04, 0, 0, 0, 0, time.UTC),
		},
	}

	wantDAUs := []*types.SiteActivityPeriod{
		{
			StartTime: time.Date(2018, 3, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			StartTime: time.Date(2018, 3, 30, 0, 0, 0, 0, time.UTC),
		},
		{
			StartTime: time.Date(2018, 3, 29, 0, 0, 0, 0, time.UTC),
		},
		{
			StartTime:            time.Date(2018, 3, 28, 0, 0, 0, 0, time.UTC),
			UserCount:            2,
			RegisteredUserCount:  2,
			AnonymousUserCount:   0,
			IntegrationUserCount: 1,
		},
		{
			StartTime: time.Date(2018, 3, 27, 0, 0, 0, 0, time.UTC),
		},
		{
			StartTime:            time.Date(2018, 3, 26, 0, 0, 0, 0, time.UTC),
			UserCount:            1,
			RegisteredUserCount:  1,
			AnonymousUserCount:   0,
			IntegrationUserCount: 0,
		},
		{
			StartTime: time.Date(2018, 3, 25, 0, 0, 0, 0, time.UTC),
		},
	}

	want := &types.SiteUsageStatistics{
		DAUs: wantDAUs,
		WAUs: wantWAUs,
		MAUs: wantMAUs,
	}

	mockTimeNow(now)
	days, weeks, months := 7, 4, 3
	siteActivity, err := GetSiteUsageStatistics(&SiteUsageStatisticsOptions{
		DayPeriods:   &days,
		WeekPeriods:  &weeks,
		MonthPeriods: &months,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = siteActivityCompare(siteActivity, want)
	if err != nil {
		t.Error(err)
	}
}

func setupForTest(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	keyPrefix = "__test__" + t.Name() + ":"
	pool = &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", "localhost:6379")
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
	c := pool.Get()
	defer c.Close()
	_, err := c.Do("EVAL", `local keys = redis.call('keys', ARGV[1])
if #keys > 0 then
	return redis.call('del', unpack(keys))
else
	return ''
end`, 0, keyPrefix+"*")
	if err != nil {
		t.Log("Could not clear test prefix:", err)
	}
}

func mockTimeNow(t time.Time) {
	timeNow = func() time.Time {
		return t
	}
}

func siteActivityCompare(a, b *types.SiteUsageStatistics) error {
	if a == nil || b == nil {
		return errors.New("site activities can not be nil")
	}
	if a == b {
		return nil
	}
	if len(a.DAUs) != len(b.DAUs) || len(a.WAUs) != len(b.WAUs) || len(a.MAUs) != len(b.MAUs) {
		return fmt.Errorf("site activities must be same length, got %d want %d (DAUs), got %d want %d (WAUs), got %d want %d (MAUs)", len(a.DAUs), len(b.DAUs), len(a.WAUs), len(b.WAUs), len(a.MAUs), len(b.MAUs))
	}
	if err := siteActivityPeriodSliceCompare("DAUs", a.DAUs, b.DAUs); err != nil {
		return err
	}
	if err := siteActivityPeriodSliceCompare("WAUs", a.WAUs, b.WAUs); err != nil {
		return err
	}
	if err := siteActivityPeriodSliceCompare("MAUs", a.MAUs, b.MAUs); err != nil {
		return err
	}
	return nil
}

func siteActivityPeriodSliceCompare(label string, a, b []*types.SiteActivityPeriod) error {
	if a == nil || b == nil {
		return fmt.Errorf("%v slices can not be nil", label)
	}
	for i, v := range a {
		if err := siteActivityPeriodCompare(label, v, b[i]); err != nil {
			return err
		}
	}
	return nil
}

func siteActivityPeriodCompare(label string, a, b *types.SiteActivityPeriod) error {
	if a == nil || b == nil {
		return errors.New("site activity periods can not be nil")
	}
	if a == b {
		return nil
	}
	if a.StartTime != b.StartTime || a.UserCount != b.UserCount || a.RegisteredUserCount != b.RegisteredUserCount || a.AnonymousUserCount != b.AnonymousUserCount || a.IntegrationUserCount != b.IntegrationUserCount {
		return fmt.Errorf("[%v] got %+v want %+v", label, a, b)
	}
	return nil
}
