package usagestats

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

func TestRetentionUsageStatistics(t *testing.T) {
	ctx := context.Background()

	defer func() {
		timeNow = time.Now
	}()

	eventDate := time.Date(2020, 11, 3, 0, 0, 0, 0, time.UTC)
	userCreationDate := time.Date(2020, 10, 26, 0, 0, 0, 0, time.UTC)

	mockTimeNow(eventDate)
	db := dbtest.NewDB(t, "")

	events := []database.Event{{
		Name:      "ViewHome",
		URL:       "https://sourcegraph.test:3443/search",
		UserID:    1,
		Source:    "WEB",
		Timestamp: userCreationDate,
	}, {
		Name:      "ViewHome",
		URL:       "https://sourcegraph.test:3443/search",
		UserID:    1,
		Source:    "WEB",
		Timestamp: eventDate,
	}}

	for _, event := range events {
		err := database.EventLogs(db).Insert(ctx, &event)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Insert user
	_, err := db.Exec(
		`INSERT INTO users(username, display_name, avatar_url, created_at, updated_at, passwd, invalidated_sessions_at, site_admin)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8)`,
		"test", "test", nil, userCreationDate, userCreationDate, "foobar", userCreationDate, true)
	if err != nil {
		t.Fatal(err)
	}

	have, err := GetRetentionStatistics(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	one := int32(1)
	oneFloat := float64(1)
	zeroFloat := float64(0)
	weekly := []*types.WeeklyRetentionStats{
		{
			WeekStart:  userCreationDate.UTC(),
			CohortSize: &one,
			Week0:      &oneFloat,
			Week1:      &oneFloat,
			Week2:      &zeroFloat,
			Week3:      &zeroFloat,
			Week4:      &zeroFloat,
			Week5:      &zeroFloat,
			Week6:      &zeroFloat,
			Week7:      &zeroFloat,
			Week8:      &zeroFloat,
			Week9:      &zeroFloat,
			Week10:     &zeroFloat,
			Week11:     &zeroFloat,
		},
	}

	for i := 1; i <= 11; i++ {
		weekly = append(weekly, &types.WeeklyRetentionStats{
			WeekStart:  userCreationDate.Add(time.Hour * time.Duration(168*i) * -1).UTC(),
			CohortSize: nil,
			Week0:      nil,
			Week1:      nil,
			Week2:      nil,
			Week3:      nil,
			Week4:      nil,
			Week5:      nil,
			Week6:      nil,
			Week7:      nil,
			Week8:      nil,
			Week9:      nil,
			Week10:     nil,
			Week11:     nil,
		})
	}

	want := &types.RetentionStats{
		Weekly: weekly,
	}

	if diff := cmp.Diff(want, have); diff != "" {
		t.Fatal(diff)
	}

}
