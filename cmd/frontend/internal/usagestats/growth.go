// Package usagestats provides an interface to update and access information about
// individual and aggregate Sourcegraph users' activity levels.
package usagestats

import (
	"context"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
)

// source: cmd/frontend/internal/usagestats/growth.go:GetGrowthStatistics
func GetGrowthStatistics(ctx context.Context) (*types.GrowthStatistics, error) {
	const q = `
	WITH
  all_usage_by_user_and_month AS (
  SELECT
    user_id,
    DATE_TRUNC('month', timestamp) AS month_active
  FROM
    event_logs
  GROUP BY
    user_id,
    month_active ),
  recent_usage_by_user AS (
  SELECT
    users.id,
    BOOL_OR(CASE
      WHEN DATE_TRUNC('month', month_active) = DATE_TRUNC('month', now()) THEN TRUE
    ELSE
    FALSE
  END
    ) AS current_month,
    BOOL_OR(CASE
      WHEN DATE_TRUNC('month', month_active) = DATE_TRUNC('month', now()) - INTERVAL '1 month' THEN TRUE
    ELSE
    FALSE
  END
    ) AS previous_month,
    DATE_TRUNC('month', DATE(users.created_at)) AS created_month,
    DATE_TRUNC('month', DATE(users.deleted_at)) AS deleted_month
  FROM
    users
  LEFT JOIN
    all_usage_by_user_and_month
  ON
    all_usage_by_user_and_month.user_id = users.id
  GROUP BY
    id,
    created_month,
    deleted_month )
SELECT
  COUNT(*) FILTER (
  WHERE
    recent_usage_by_user.created_month = DATE_TRUNC('month', now())) AS created_users,
  COUNT(*) FILTER (
  WHERE
    recent_usage_by_user.deleted_month = DATE_TRUNC('month', now())) AS deleted_users,
  COUNT(*) FILTER (
  WHERE
    current_month = TRUE
    AND previous_month = FALSE
    AND created_month < DATE_TRUNC('month', now())
    AND (deleted_month < DATE_TRUNC('month', now())
      OR deleted_month IS NULL)) AS resurrected_users,
  COUNT(*) FILTER (
  WHERE
    current_month = FALSE
    AND previous_month = TRUE
    AND created_month < DATE_TRUNC('month', now())
    AND (deleted_month < DATE_TRUNC('month', now())
      OR deleted_month IS NULL)) AS churned_users,
  COUNT(*) FILTER (
  WHERE
    current_month = TRUE
    AND previous_month = TRUE
    AND created_month < DATE_TRUNC('month', now())
    AND (deleted_month < DATE_TRUNC('month', now())
      OR deleted_month IS NULL)) AS retained_users
FROM
  recent_usage_by_user
	`
	var (
		createdUsers     int
		deletedUsers     int
		resurrectedUsers int
		churnedUsers     int
		retainedUsers    int
	)
	if err := dbconn.Global.QueryRowContext(ctx, q).Scan(
		&createdUsers,
		&deletedUsers,
		&resurrectedUsers,
		&churnedUsers,
		&retainedUsers,
	); err != nil {
		return nil, err
	}

	return &types.GrowthStatistics{
		DeletedUsers:     int32(deletedUsers),
		CreatedUsers:     int32(createdUsers),
		ResurrectedUsers: int32(resurrectedUsers),
		ChurnedUsers:     int32(churnedUsers),
		RetainedUsers:    int32(retainedUsers),
	}, nil
}
