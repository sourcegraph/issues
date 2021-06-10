package usagestats

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

func TestCodeMonitoringUsageStatistics(t *testing.T) {
	ctx := context.Background()

	defer func() {
		timeNow = time.Now
	}()

	now := time.Date(2021, 1, 28, 0, 0, 0, 0, time.UTC)
	mockTimeNow(now)

	db := dbtest.NewDB(t, "")

	_, err := db.Exec(`
		INSERT INTO event_logs
			(id, name, argument, url, user_id, anonymous_user_id, source, version, timestamp)
		VALUES
			(1, 'ViewCodeMonitoringPage', '{}', '', 1, '420657f0-d443-4d16-ac7d-003d8cdc91ef', 'WEB', '3.23.0', $1::timestamp - interval '1 day'),
			(2, 'ViewCreateCodeMonitorPage', '{"hasTriggerQuery": false}', '', 1, '420657f0-d443-4d16-ac7d-003d8cdc91ef', 'WEB', '3.23.0', $1::timestamp - interval '1 day'),
			(3, 'ViewCreateCodeMonitorPage', '{"hasTriggerQuery": false}', '', 1, '420657f0-d443-4d16-ac7d-003d8cdc91ef', 'WEB', '3.23.0', $1::timestamp - interval '1 day'),
			(4, 'ViewCreateCodeMonitorPage', '{"hasTriggerQuery": true}', '', 2, '420657f0-d443-4d16-ac7d-003d8cdc19ac', 'WEB', '3.23.0', $1::timestamp - interval '1 day'),
			(5, 'ViewManageCodeMonitorPage', '{}', '', 2, '420657f0-d443-4d16-ac7d-003d8cdc19ac', 'WEB', '3.23.0', $1::timestamp - interval '1 day'),
			(6, 'CodeMonitorEmailLinkClicked', '{}', '', 2, '420657f0-d443-4d16-ac7d-003d8cdc19ac', 'WEB', '3.23.0', $1::timestamp - interval '1 day')
	`, now)
	if err != nil {
		t.Fatal(err)
	}

	have, err := GetCodeMonitoringUsageStatistics(ctx, db)
	if err != nil {
		t.Fatal(err)
	}

	oneInt := int32(1)
	twoInt := int32(2)
	threeInt := int32(3)

	want := &types.CodeMonitoringUsageStatistics{
		CodeMonitoringPageViews:                       &oneInt,
		CreateCodeMonitorPageViews:                    &threeInt,
		CreateCodeMonitorPageViewsWithTriggerQuery:    &oneInt,
		CreateCodeMonitorPageViewsWithoutTriggerQuery: &twoInt,
		ManageCodeMonitorPageViews:                    &oneInt,
		CodeMonitorEmailLinkClicks:                    &oneInt,
	}
	if diff := cmp.Diff(want, have); diff != "" {
		t.Fatal(diff)
	}
}
