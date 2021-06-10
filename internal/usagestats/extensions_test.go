package usagestats

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

func TestExtensionsUsageStatistics(t *testing.T) {
	ctx := context.Background()

	defer func() {
		timeNow = time.Now
	}()

	weekStart := time.Date(2021, 1, 25, 0, 0, 0, 0, time.UTC)
	now := time.Date(2021, 1, 28, 0, 0, 0, 0, time.UTC)
	mockTimeNow(now)

	db := dbtest.NewDB(t, "")

	_, err := db.Exec(`
		INSERT INTO event_logs
			(id, name, argument, url, user_id, anonymous_user_id, source, version, timestamp)
		VALUES
			(1, 'ExtensionActivation', '{"extension_id": "sourcegraph/codecov"}', 'https://sourcegraph.test:3443/search', 1, '420657f0-d443-4d16-ac7d-003d8cdc91ef', 'WEB', '3.23.0', $1::timestamp - interval '1 day'),
			(2, 'ExtensionActivation', '{"extension_id": "sourcegraph/link-preview-expander"}', 'https://sourcegraph.test:3443/search', 1, '420657f0-d443-4d16-ac7d-003d8cdc91ef', 'WEB', '3.23.0', $1::timestamp - interval '1 day'),
			(3, 'ExtensionActivation', '{"extension_id": "sourcegraph/link-preview-expander"}', 'https://sourcegraph.test:3443/search', 2, '420657f0-d443-4d16-ac7d-003d8cdc19ac', 'WEB', '3.23.0', $1::timestamp - interval '1 day'),
			(4, 'ExtensionActivation', '{"extension_id": "sourcegraph/link-preview-expander"}', 'https://sourcegraph.test:3443/search', 2, '420657f0-d443-4d16-ac7d-003d8cdc19ac', 'WEB', '3.23.0', $1::timestamp - interval '1 day'),
			(5, 'ExtensionActivation', '{"extension_id": "sourcegraph/link-preview-expander"}', 'https://sourcegraph.test:3443/search', 2, '420657f0-d443-4d16-ac7d-003d8cdc19ac', 'WEB', '3.23.0', $1::timestamp - interval '8 days')
	`, now)
	if err != nil {
		t.Fatal(err)
	}

	have, err := GetExtensionsUsageStatistics(ctx, db)
	if err != nil {
		t.Fatal(err)
	}

	oneFloat := float64(1)
	oneAndAHalfFloat := float64(1.5)
	oneInt := int32(1)
	twoInt := int32(2)

	codecovID := "sourcegraph/codecov"
	lpeID := "sourcegraph/link-preview-expander"

	usageStatisticsByExtension := []*types.ExtensionUsageStatistics{
		{
			UserCount:          &oneInt,
			AverageActivations: &oneFloat,
			ExtensionID:        &codecovID,
		},
		{
			UserCount:          &twoInt,
			AverageActivations: &oneAndAHalfFloat,
			ExtensionID:        &lpeID,
		},
	}

	want := &types.ExtensionsUsageStatistics{
		WeekStart:                   weekStart,
		UsageStatisticsByExtension:  usageStatisticsByExtension,
		AverageNonDefaultExtensions: &oneAndAHalfFloat,
		NonDefaultExtensionUsers:    &twoInt,
	}
	if diff := cmp.Diff(want, have); diff != "" {
		t.Fatal(diff)
	}
}
