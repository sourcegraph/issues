package usagestats

import (
	"context"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
)

func GetSavedSearches(ctx context.Context) (*types.SavedSearches, error) {
	const q = `
	SELECT
	(SELECT COUNT(*) FROM saved_searches) AS totalSavedSearches,
	(SELECT COUNT(DISTINCT user_id) FROM saved_searches) AS uniqueUsers,
	(SELECT COUNT(*) FROM event_logs WHERE event_logs.name = 'SavedSearchEmailNotificationSent') AS notificationsSent,
	(SELECT COUNT(*) FROM event_logs WHERE event_logs.name = 'SavedSearchEmailClicked') AS notificationsClicked,
	(SELECT COUNT(DISTINCT user_id) FROM event_logs WHERE event_logs.name = 'ViewSavedSearchListPage') AS uniqueUserPageViews,
	(SELECT COUNT(*) FROM saved_searches WHERE org_id IS NOT NULL) AS orgSavedSearches	
	`
	var (
		totalSavedSearches   int
		uniqueUsers          int
		notificationsSent    int
		notificationsClicked int
		uniqueUserPageViews  int
		orgSavedSearches     int
	)
	if err := dbconn.Global.QueryRowContext(ctx, q).Scan(
		&totalSavedSearches,
		&uniqueUsers,
		&notificationsSent,
		&notificationsClicked,
		&uniqueUserPageViews,
		&orgSavedSearches,
	); err != nil {
		return nil, err
	}

	return &types.SavedSearches{
		TotalSavedSearches:   int32(totalSavedSearches),
		UniqueUsers:          int32(uniqueUsers),
		NotificationsSent:    int32(notificationsSent),
		NotificationsClicked: int32(notificationsClicked),
		UniqueUserPageViews:  int32(uniqueUserPageViews),
		OrgSavedSearches:     int32(orgSavedSearches),
	}, nil
}
