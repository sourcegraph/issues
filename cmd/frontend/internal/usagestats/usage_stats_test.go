package usagestats

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"
	"time"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
)

func TestGetArchive(t *testing.T) {
	setupForTest(t)

	now := time.Now().UTC()
	ctx := context.Background()

	user, err := db.Users.Create(ctx, db.NewUser{
		Email:           "foo@bar.com",
		Username:        "admin",
		EmailIsVerified: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	event := &db.Event{
		Name:      "SearchResultsQueried",
		URL:       "test",
		UserID:    uint32(user.ID),
		Source:    "test",
		Timestamp: now,
	}

	err = db.EventLogs.Insert(ctx, event)
	if err != nil {
		t.Fatal(err)
	}

	dates, err := db.Users.ListDates(ctx)
	if err != nil {
		t.Fatal(err)
	}

	archive, err := GetArchive(ctx)
	if err != nil {
		t.Fatal(err)
	}

	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]string{
		"UsersUsageCounts.csv": fmt.Sprintf("date,user_id,search_count,code_intel_count\n%s,%d,%d,%d\n",
			time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
			event.UserID,
			1,
			0,
		),
		"UsersDates.csv": fmt.Sprintf("user_id,created_at,deleted_at\n%d,%s,%s\n",
			dates[0].UserID,
			dates[0].CreatedAt.Format(time.RFC3339),
			"NULL",
		),
	}

	for _, f := range zr.File {
		content, ok := want[f.Name]
		if !ok {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}

		have, err := ioutil.ReadAll(rc)
		if err != nil {
			t.Fatal(err)
		}

		delete(want, f.Name)

		if content != string(have) {
			t.Errorf("%q has wrong content:\nwant: %s\nhave: %s", f.Name, content, string(have))
		}
	}

	for file := range want {
		t.Errorf("Missing file from ZIP archive %q", file)
	}
}

func TestUserUsageStatistics_None(t *testing.T) {
	setupForTest(t)

	want := &types.UserUsageStatistics{
		UserID: 42,
	}
	got, err := GetByUserID(context.Background(), 42)
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
	err := logLocalEvent(context.Background(), "ViewRepo", "https://sourcegraph.example.com/", user.ID, "test-cookie-id", "WEB", nil)
	if err != nil {
		t.Fatal(err)
	}

	a, err := GetByUserID(context.Background(), user.ID)
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

	// Set searchOccurred to true to prevent using redis to log all-time stats during tests.
	searchOccurred = 1
	defer func() {
		searchOccurred = 0
	}()

	user := types.User{
		ID: 1,
	}
	err := logLocalEvent(context.Background(), "SearchResultsQueried", "https://sourcegraph.example.com/", user.ID, "test-cookie-id", "WEB", nil)
	if err != nil {
		t.Fatal(err)
	}

	a, err := GetByUserID(context.Background(), user.ID)
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
	err := logLocalEvent(context.Background(), "hover", "https://sourcegraph.example.com/", user.ID, "test-cookie-id", "WEB", nil)
	if err != nil {
		t.Fatal(err)
	}

	a, err := GetByUserID(context.Background(), user.ID)
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
	err := logLocalEvent(context.Background(), "hover", "https://sourcegraph.example.com/", user.ID, "test-cookie-id", "CODEHOSTINTEGRATION", nil)
	if err != nil {
		t.Fatal(err)
	}

	a, err := GetByUserID(context.Background(), user.ID)
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

	ctx := context.Background()

	user1 := types.User{
		ID: 1,
	}
	user2 := types.User{
		ID: 2,
	}

	// Test single user
	err := logLocalEvent(ctx, "ViewBlob", "https://sourcegraph.example.com/", user1.ID, "test-cookie-id-1", "WEB", nil)
	if err != nil {
		t.Fatal(err)
	}

	n, err := GetUsersActiveTodayCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if want := 1; n != want {
		t.Errorf("got %d, want %d", n, want)
	}

	// Test multiple users, with repeats
	err = logLocalEvent(ctx, "ViewBlob", "https://sourcegraph.example.com/", user2.ID, "test-cookie-id-2", "WEB", nil)
	if err != nil {
		t.Fatal(err)
	}
	err = logLocalEvent(ctx, "ViewBlob", "https://sourcegraph.example.com/", user1.ID, "test-cookie-id-1", "WEB", nil)
	if err != nil {
		t.Fatal(err)
	}
	err = logLocalEvent(ctx, "ViewBlob", "https://sourcegraph.example.com/", 0, "test-cookie-id-3", "WEB", nil)
	if err != nil {
		t.Fatal(err)
	}
	err = logLocalEvent(ctx, "ViewBlob", "https://sourcegraph.example.com/", user2.ID, "test-cookie-id-2", "WEB", nil)

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
	ctx := context.Background()

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
	err := logLocalEvent(ctx, "ViewBlob", "https://sourcegraph.example.com/", user1.ID, "test-cookie-id-1", "WEB", nil)
	if err != nil {
		t.Fatal(err)
	}
	err = logLocalEvent(ctx, "ViewBlob", "https://sourcegraph.example.com/", 0, "068ccbfa-8529-4fa7-859e-2c3514af2434", "WEB", nil)
	if err != nil {
		t.Fatal(err)
	}
	err = logLocalEvent(ctx, "hover", "https://sourcegraph.example.com/", 0, "068ccbfa-8529-4fa7-859e-2c3514af2434", "CODEHOSTINTEGRATION", nil)
	if err != nil {
		t.Fatal(err)
	}

	// 2018/02/28 (2 users, 1 registered)
	mockTimeNow(oneMonthThreeDaysAgo)
	err = logLocalEvent(ctx, "ViewBlob", "https://sourcegraph.example.com/", user1.ID, "test-cookie-id-1", "WEB", nil)
	if err != nil {
		t.Fatal(err)
	}
	err = logLocalEvent(ctx, "ViewBlob", "https://sourcegraph.example.com/", 0, "30dd2661-2e73-4774-bc2b-7a126f360734", "WEB", nil)
	if err != nil {
		t.Fatal(err)
	}

	// 2018/03/15 (2 users, 1 registered)
	mockTimeNow(twoWeeksTwoDaysAgo)
	err = logLocalEvent(ctx, "ViewBlob", "https://sourcegraph.example.com/", user2.ID, "test-cookie-id-2", "WEB", nil)
	if err != nil {
		t.Fatal(err)
	}
	err = logLocalEvent(ctx, "ViewBlob", "https://sourcegraph.example.com/", 0, "068ccbfa-8529-4fa7-859e-2c3514af2434", "WEB", nil)
	if err != nil {
		t.Fatal(err)
	}

	// 2018/03/17 (2 users, 1 registered)
	mockTimeNow(twoWeeksAgo)
	err = logLocalEvent(ctx, "ViewBlob", "https://sourcegraph.example.com/", user2.ID, "test-cookie-id-2", "WEB", nil)
	if err != nil {
		t.Fatal(err)
	}
	err = logLocalEvent(ctx, "ViewBlob", "https://sourcegraph.example.com/", 0, "b309dad0-b6f9-440d-bf0a-4cf38030ca70", "WEB", nil)
	if err != nil {
		t.Fatal(err)
	}
	err = logLocalEvent(ctx, "hover", "https://sourcegraph.example.com/", user2.ID, "test-cookie-id-2", "CODEHOSTINTEGRATION", nil)
	if err != nil {
		t.Fatal(err)
	}

	// 2018/03/26 (1 user, 1 registered)
	mockTimeNow(fiveDaysAgo)
	err = logLocalEvent(ctx, "ViewBlob", "https://sourcegraph.example.com/", user1.ID, "test-cookie-id-1", "WEB", nil)
	if err != nil {
		t.Fatal(err)
	}

	// 2018/03/28 (2 users, 2 registered)
	mockTimeNow(threeDaysAgo)
	err = logLocalEvent(ctx, "ViewBlob", "https://sourcegraph.example.com/", user1.ID, "test-cookie-id-1", "WEB", nil)
	if err != nil {
		t.Fatal(err)
	}
	err = logLocalEvent(ctx, "ViewBlob", "https://sourcegraph.example.com/", user2.ID, "test-cookie-id-2", "WEB", nil)
	if err != nil {
		t.Fatal(err)
	}
	err = logLocalEvent(ctx, "hover", "https://sourcegraph.example.com/", user1.ID, "test-cookie-id-1", "CODEHOSTINTEGRATION", nil)
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
			IntegrationUserCount: 1,
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
	siteActivity, err := GetSiteUsageStatistics(context.Background(), &SiteUsageStatisticsOptions{
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

	dbtesting.SetupGlobalTestDB(t)
}

func mockTimeNow(t time.Time) {
	timeNow = func() time.Time {
		return t
	}
}

func siteActivityCompare(got, want *types.SiteUsageStatistics) error {
	if got == nil || want == nil {
		return errors.New("site activities can not be nil")
	}
	if got == want {
		return nil
	}
	if len(got.DAUs) != len(want.DAUs) || len(got.WAUs) != len(want.WAUs) || len(got.MAUs) != len(want.MAUs) {
		return fmt.Errorf("site activities must be same length, got %d want %d (DAUs), got %d want %d (WAUs), got %d want %d (MAUs)", len(got.DAUs), len(want.DAUs), len(got.WAUs), len(want.WAUs), len(got.MAUs), len(want.MAUs))
	}
	if err := siteActivityPeriodSliceCompare("DAUs", got.DAUs, want.DAUs); err != nil {
		return err
	}
	if err := siteActivityPeriodSliceCompare("WAUs", got.WAUs, want.WAUs); err != nil {
		return err
	}
	if err := siteActivityPeriodSliceCompare("MAUs", got.MAUs, want.MAUs); err != nil {
		return err
	}
	return nil
}

func siteActivityPeriodSliceCompare(label string, got, want []*types.SiteActivityPeriod) error {
	if got == nil || want == nil {
		return fmt.Errorf("%v slices can not be nil", label)
	}
	for i, v := range got {
		if err := siteActivityPeriodCompare(label, v, want[i]); err != nil {
			return err
		}
	}
	return nil
}

func siteActivityPeriodCompare(label string, got, want *types.SiteActivityPeriod) error {
	if got == nil || want == nil {
		return errors.New("site activity periods can not be nil")
	}
	if got == want {
		return nil
	}
	if got.StartTime != want.StartTime || got.UserCount != want.UserCount || got.RegisteredUserCount != want.RegisteredUserCount || got.AnonymousUserCount != want.AnonymousUserCount || got.IntegrationUserCount != want.IntegrationUserCount {
		return fmt.Errorf("[%v] got %+v want %+v", label, got, want)
	}
	return nil
}
