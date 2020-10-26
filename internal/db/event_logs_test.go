package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/version"
)

func TestEventLogs_ValidInfo(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	var testCases = []struct {
		name  string
		event *Event
		err   string // Stringified error
	}{
		{
			name:  "EmptyName",
			event: &Event{UserID: 1, URL: "http://sourcegraph.com", Source: "WEB"},
			err:   `INSERT: pq: new row for relation "event_logs" violates check constraint "event_logs_check_name_not_empty"`,
		},
		{
			name:  "InvalidUser",
			event: &Event{Name: "test_event", URL: "http://sourcegraph.com", Source: "WEB"},
			err:   `INSERT: pq: new row for relation "event_logs" violates check constraint "event_logs_check_has_user"`,
		},
		{
			name:  "EmptySource",
			event: &Event{Name: "test_event", URL: "http://sourcegraph.com", UserID: 1},
			err:   `INSERT: pq: new row for relation "event_logs" violates check constraint "event_logs_check_source_not_empty"`,
		},

		{
			name:  "ValidInsert",
			event: &Event{Name: "test_event", UserID: 1, URL: "http://sourcegraph.com", Source: "WEB"},
			err:   "<nil>",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := EventLogs.Insert(ctx, tc.event)

			if have, want := fmt.Sprint(err), tc.err; have != want {
				t.Errorf("have %+v, want %+v", have, want)
			}
		})
	}
}

func TestEventLogs_CountUniqueUsersPerPeriod(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	now := time.Now()
	startDate, _ := calcStartDate(now, Daily, 3)
	secondDay := startDate.Add(time.Hour * 24)
	thirdDay := startDate.Add(time.Hour * 24 * 2)

	events := []*Event{
		makeTestEvent(&Event{UserID: 1, Timestamp: startDate}),
		makeTestEvent(&Event{UserID: 1, Timestamp: startDate}),
		makeTestEvent(&Event{UserID: 2, Timestamp: startDate}),
		makeTestEvent(&Event{UserID: 2, Timestamp: startDate}),

		makeTestEvent(&Event{UserID: 1, Timestamp: secondDay}),
		makeTestEvent(&Event{UserID: 2, Timestamp: secondDay}),
		makeTestEvent(&Event{UserID: 3, Timestamp: secondDay}),
		makeTestEvent(&Event{UserID: 1, Timestamp: secondDay}),

		makeTestEvent(&Event{UserID: 5, Timestamp: thirdDay}),
		makeTestEvent(&Event{UserID: 6, Timestamp: thirdDay}),
		makeTestEvent(&Event{UserID: 7, Timestamp: thirdDay}),
		makeTestEvent(&Event{UserID: 8, Timestamp: thirdDay}),
	}

	for _, e := range events {
		if err := EventLogs.Insert(ctx, e); err != nil {
			t.Fatal(err)
		}
	}

	values, err := EventLogs.CountUniqueUsersPerPeriod(ctx, Daily, now, 3, nil)
	if err != nil {
		t.Fatal(err)
	}

	assertUsageValue(t, values[0], startDate.Add(time.Hour*24*2), 4)
	assertUsageValue(t, values[1], startDate.Add(time.Hour*24), 3)
	assertUsageValue(t, values[2], startDate, 2)
}

func TestEventLogs_UsersUsageCounts(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	now := time.Now()

	startDate, _ := calcStartDate(now, Daily, 3)
	secondDay := startDate.Add(time.Hour * 24)
	thirdDay := startDate.Add(time.Hour * 24 * 2)

	days := []time.Time{startDate, secondDay, thirdDay}
	names := []string{"SearchResultsQueried", "codeintel"}
	users := []uint32{1, 2}

	for _, day := range days {
		for _, user := range users {
			for _, name := range names {
				for i := 0; i < 25; i++ {
					e := &Event{
						UserID:    user,
						Name:      name,
						URL:       "test",
						Source:    "test",
						Timestamp: day.Add(time.Minute * time.Duration(rand.Intn(60*12))),
					}

					if err := EventLogs.Insert(ctx, e); err != nil {
						t.Fatal(err)
					}
				}
			}
		}
	}

	have, err := EventLogs.UsersUsageCounts(ctx)
	if err != nil {
		t.Fatal(err)
	}

	want := []types.UserUsageCounts{
		{Date: days[2], UserID: users[0], SearchCount: 25, CodeIntelCount: 25},
		{Date: days[2], UserID: users[1], SearchCount: 25, CodeIntelCount: 25},
		{Date: days[1], UserID: users[0], SearchCount: 25, CodeIntelCount: 25},
		{Date: days[1], UserID: users[1], SearchCount: 25, CodeIntelCount: 25},
		{Date: days[0], UserID: users[0], SearchCount: 25, CodeIntelCount: 25},
		{Date: days[0], UserID: users[1], SearchCount: 25, CodeIntelCount: 25},
	}

	if diff := cmp.Diff(want, have); diff != "" {
		t.Error(diff)
	}
}

func TestEventLogs_SiteUsage(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	// This unix timestamp is equivalent to `Friday, May 15, 2020 10:30:00 PM GMT` and is set to
	// be a consistent value so that the tests don't fail when someone runs it at some particular
	// time that falls too near the edge of a week.
	now := time.Unix(1589581800, 0).UTC()

	days := map[time.Time]struct {
		users   []uint32
		names   []string
		sources []string
	}{
		// Today
		now: {
			[]uint32{1, 2, 3, 4, 5},
			[]string{"ViewSiteAdminX"},
			[]string{"test", "CODEHOSTINTEGRATION"},
		},
		// This week
		now.Add(-time.Hour * 24 * 3): {
			[]uint32{0, 2, 3, 5},
			[]string{"ViewRepository", "ViewTree"},
			[]string{"test", "CODEHOSTINTEGRATION"},
		},
		// This week
		now.Add(-time.Hour * 24 * 4): {
			[]uint32{1, 3, 5, 7},
			[]string{"ViewSiteAdminX", "SavedSearchSlackClicked"},
			[]string{"test", "CODEHOSTINTEGRATION"},
		},
		// This month
		now.Add(-time.Hour * 24 * 6): {
			[]uint32{0, 1, 8, 9},
			[]string{"ViewSiteAdminX"},
			[]string{"test", "CODEHOSTINTEGRATION"},
		},
		// This month
		now.Add(-time.Hour * 24 * 12): {
			[]uint32{1, 2, 3, 4, 5, 6, 11},
			[]string{"ViewTree", "SavedSearchSlackClicked"},
			[]string{"test", "CODEHOSTINTEGRATION"},
		},
		// Previous month
		now.Add(-time.Hour * 24 * 40): {
			[]uint32{0, 1, 5, 6, 13},
			[]string{"SearchResultsQueried", "DiffSearchResultsQueried"},
			[]string{"test", "CODEHOSTINTEGRATION"},
		},
	}

	for day, data := range days {
		for _, user := range data.users {
			for _, name := range data.names {
				for _, source := range data.sources {
					for i := 0; i < 5; i++ {
						e := &Event{
							UserID: user,
							Name:   name,
							URL:    "test",
							Source: source,
							// Jitter current time +/- 30 minutes
							Timestamp: day.Add(time.Minute * time.Duration(rand.Intn(60)-30)),
						}

						if user == 0 {
							e.AnonymousUserID = "deadbeef"
						}

						if err := EventLogs.Insert(ctx, e); err != nil {
							t.Fatal(err)
						}
					}
				}
			}
		}
	}

	summary, err := EventLogs.siteUsage(ctx, now)
	if err != nil {
		t.Fatal(err)
	}

	expectedSummary := types.SiteUsageSummary{
		Month:                   time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC),
		Week:                    now.Truncate(time.Hour * 24).Add(-time.Hour * 24 * 5), // the previous Sunday
		Day:                     now.Truncate(time.Hour * 24),
		UniquesMonth:            11,
		UniquesWeek:             7,
		UniquesDay:              5,
		RegisteredUniquesMonth:  10,
		RegisteredUniquesWeek:   6,
		RegisteredUniquesDay:    5,
		IntegrationUniquesMonth: 11,
		IntegrationUniquesWeek:  7,
		IntegrationUniquesDay:   5,
		ManageUniquesMonth:      9,
		CodeUniquesMonth:        8,
		VerifyUniquesMonth:      8,
		MonitorUniquesMonth:     0,
		ManageUniquesWeek:       6,
		CodeUniquesWeek:         4,
		VerifyUniquesWeek:       4,
		MonitorUniquesWeek:      0,
	}
	if diff := cmp.Diff(expectedSummary, summary); diff != "" {
		t.Fatal(diff)
	}
}

func TestEventLogs_AggregatedEvents(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	names := []string{"codeintel.searchHover", "search.latencies.literal", "unknown event"}
	users := []uint32{1, 2}
	durations := []int{40, 65, 72}

	// This unix timestamp is equivalent to `Friday, May 15, 2020 10:30:00 PM GMT` and is set to
	// be a consistent value so that the tests don't fail when someone runs it at some particular
	// time that falls too near the edge of a week.
	now := time.Unix(1589581800, 0).UTC()

	days := []time.Time{
		now,                           // Today
		now.Add(-time.Hour * 24 * 3),  // This week
		now.Add(-time.Hour * 24 * 4),  // This week
		now.Add(-time.Hour * 24 * 6),  // This month
		now.Add(-time.Hour * 24 * 12), // This month
		now.Add(-time.Hour * 24 * 40), // Previous month
	}

	durationOffset := 0
	for _, user := range users {
		for _, name := range names {
			for _, duration := range durations {
				for _, day := range days {
					for i := 0; i < 25; i++ {
						durationOffset++

						e := &Event{
							UserID: user,
							Name:   name,
							URL:    "test",
							Source: "test",
							// Make durations non-uniform to test percent_cont. The values
							// in this test were hand-checked before being added to the assertion.
							// Adding additional events or changing parameters will require these
							// values to be checked again.
							Argument: json.RawMessage(fmt.Sprintf(`{"durationMs": %d}`, duration+durationOffset)),
							// Jitter current time +/- 30 minutes
							Timestamp: day.Add(time.Minute * time.Duration(rand.Intn(60)-30)),
						}

						if err := EventLogs.Insert(ctx, e); err != nil {
							t.Fatal(err)
						}
					}
				}
			}
		}
	}

	events, err := EventLogs.aggregatedEvents(ctx, now)
	if err != nil {
		t.Fatal(err)
	}

	expectedEvents := []types.AggregatedEvent{
		{
			Name:           "codeintel.searchHover",
			Month:          time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC),
			Week:           now.Truncate(time.Hour * 24).Add(-time.Hour * 24 * 5), // the previous Sunday
			Day:            now.Truncate(time.Hour * 24),
			TotalMonth:     int32(len(users) * len(durations) * 25 * 5), // 5 days in month
			TotalWeek:      int32(len(users) * len(durations) * 25 * 3), // 3 days in week
			TotalDay:       int32(len(users) * len(durations) * 25),
			UniquesMonth:   2,
			UniquesWeek:    2,
			UniquesDay:     2,
			LatenciesMonth: []float64{944, 1772.1, 1839.51},
			LatenciesWeek:  []float64{919, 1752.1, 1792.51},
			LatenciesDay:   []float64{894, 1732.1, 1745.51},
		},
		{
			Name:           "search.latencies.literal",
			Month:          time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC),
			Week:           now.Truncate(time.Hour * 24).Add(-time.Hour * 24 * 5), // the previous Sunday
			Day:            now.Truncate(time.Hour * 24),
			TotalMonth:     int32(len(users) * len(durations) * 25 * 5), // 5 days in month
			TotalWeek:      int32(len(users) * len(durations) * 25 * 3), // 3 days in week
			TotalDay:       int32(len(users) * len(durations) * 25),
			UniquesMonth:   2,
			UniquesWeek:    2,
			UniquesDay:     2,
			LatenciesMonth: []float64{1394, 2222.1, 2289.51},
			LatenciesWeek:  []float64{1369, 2202.1, 2242.51},
			LatenciesDay:   []float64{1344, 2182.1, 2195.51},
		},
	}
	if diff := cmp.Diff(expectedEvents, events); diff != "" {
		t.Fatal(diff)
	}
}

func TestEventLogs_AggregatedEventsSparseEvents(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	// This unix timestamp is equivalent to `Friday, May 15, 2020 10:30:00 PM GMT` and is set to
	// be a consistent value so that the tests don't fail when someone runs it at some particular
	// time that falls too near the edge of a week.
	now := time.Unix(1589581800, 0).UTC()

	for i := 0; i < 5; i++ {
		e := &Event{
			UserID: 1,
			Name:   "codeintel.searchHover",
			URL:    "test",
			Source: "test",
			// Make durations non-uniform to test percent_cont. The values
			// in this test were hand-checked before being added to the assertion.
			// Adding additional events or changing parameters will require these
			// values to be checked again.
			Argument:  json.RawMessage(fmt.Sprintf(`{"durationMs": %d}`, 50)),
			Timestamp: now.Add(-time.Hour * 24 * 6), // This month
		}

		if err := EventLogs.Insert(ctx, e); err != nil {
			t.Fatal(err)
		}
	}

	events, err := EventLogs.aggregatedEvents(ctx, now)
	if err != nil {
		t.Fatal(err)
	}

	expectedEvents := []types.AggregatedEvent{
		{
			Name:           "codeintel.searchHover",
			Month:          time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC),
			Week:           now.Truncate(time.Hour * 24).Add(-time.Hour * 24 * 5), // the previous Sunday
			Day:            now.Truncate(time.Hour * 24),
			TotalMonth:     5,
			TotalWeek:      0,
			TotalDay:       0,
			UniquesMonth:   1,
			UniquesWeek:    0,
			UniquesDay:     0,
			LatenciesMonth: []float64{50, 50, 50},
			LatenciesWeek:  nil,
			LatenciesDay:   nil,
		},
	}
	if diff := cmp.Diff(expectedEvents, events); diff != "" {
		t.Fatal(diff)
	}
}

func TestEventLogs_ListAll(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	now := time.Now()

	startDate, _ := calcStartDate(now, Daily, 3)

	events := []*Event{
		{
			UserID:    1,
			Name:      "SearchResultsQueried",
			URL:       "test",
			Source:    "test",
			Timestamp: startDate,
		}, {
			UserID:    2,
			Name:      "codeintel",
			URL:       "test",
			Source:    "test",
			Timestamp: startDate,
		},
		{
			UserID:    2,
			Name:      "ViewRepository",
			URL:       "test",
			Source:    "test",
			Timestamp: startDate,
		},
		{
			UserID:    2,
			Name:      "SearchResultsQueried",
			URL:       "test",
			Source:    "test",
			Timestamp: startDate,
		}}

	for _, event := range events {
		if err := EventLogs.Insert(ctx, event); err != nil {
			t.Fatal(err)
		}

	}

	searchResultQueriedEvent := "SearchResultsQueried"
	have, err := EventLogs.ListAll(ctx, EventLogsListOptions{EventName: &searchResultQueriedEvent})
	if err != nil {
		t.Fatal(err)
	}

	want := 2

	if diff := cmp.Diff(want, len(have)); diff != "" {
		t.Error(diff)
	}
}

func TestEventLogs_LatestPing(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)

	t.Run("with no pings in database", func(t *testing.T) {
		ctx := context.Background()
		ping, err := EventLogs.LatestPing(ctx)
		if ping != nil {
			t.Fatalf("have ping %+v, expected nil", ping)
		}
		if err != sql.ErrNoRows {
			t.Fatalf("have err %+v, expected no rows error", err)
		}
	})

	t.Run("with existing pings in database", func(t *testing.T) {
		userID := int32(0)
		timestamp := time.Now().UTC().Truncate(time.Microsecond)

		ctx := context.Background()
		events := []*Event{
			{
				UserID:          0,
				Name:            "ping",
				URL:             "test",
				AnonymousUserID: "test",
				Source:          "test",
				Timestamp:       timestamp,
				Argument:        json.RawMessage(`{"key": "value1"}`),
			}, {
				UserID:          0,
				Name:            "ping",
				URL:             "test",
				AnonymousUserID: "test",
				Source:          "test",
				Timestamp:       timestamp,
				Argument:        json.RawMessage(`{"key": "value2"}`),
			},
		}
		for _, event := range events {
			if err := EventLogs.Insert(ctx, event); err != nil {
				t.Fatal(err)
			}
		}

		gotPing, err := EventLogs.LatestPing(ctx)
		if err != nil || gotPing == nil {
			t.Fatal(err)
		}
		expectedPing := &types.Event{
			ID:              2,
			Name:            events[1].Name,
			URL:             events[1].URL,
			UserID:          &userID,
			AnonymousUserID: events[1].AnonymousUserID,
			Version:         version.Version(),
			Argument:        string(events[1].Argument),
			Source:          events[1].Source,
			Timestamp:       timestamp,
		}
		if diff := cmp.Diff(gotPing, expectedPing); diff != "" {
			t.Fatal(diff)
		}
	})
}

// makeTestEvent sets the required (uninteresting) fields that are required on insertion
// due to db constraints. This method will also add some sub-day jitter to the timestamp.
func makeTestEvent(e *Event) *Event {
	if e.UserID == 0 {
		e.UserID = 1
	}
	e.Name = "foo"
	e.URL = "test"
	e.Source = "WEB"
	e.Timestamp = e.Timestamp.Add(time.Minute * time.Duration(rand.Intn(60*12)))
	return e
}

func assertUsageValue(t *testing.T, v UsageValue, start time.Time, count int) {
	if v.Start != start {
		t.Errorf("got Start %q, want %q", v.Start, start)
	}
	if v.Count != count {
		t.Errorf("got Count %d, want %d", v.Count, count)
	}
}
