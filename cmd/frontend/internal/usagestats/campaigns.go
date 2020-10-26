package usagestats

import (
	"context"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
)

// GetCampaignsUsageStatistics returns the current site's campaigns usage.
func GetCampaignsUsageStatistics(ctx context.Context) (*types.CampaignsUsageStatistics, error) {
	const q = `
SELECT
    (SELECT COUNT(*) FROM campaigns) AS campaigns_count,
    COUNT(*) FILTER (WHERE owned_by_campaign_id IS NOT NULL AND publication_state = 'PUBLISHED') AS action_changesets,
    COUNT(*) FILTER (WHERE owned_by_campaign_id IS NOT NULL AND publication_state = 'PUBLISHED' AND external_state = 'MERGED') AS action_changesets_merged,
    COUNT(*) FILTER (WHERE added_to_campaign) AS manual_changesets,
    COUNT(*) FILTER (WHERE added_to_campaign AND external_state = 'MERGED') AS manual_changesets_merged
FROM changesets;
`
	var (
		campaignsCount         int
		actionChangesets       int
		actionChangesetsMerged int
		manualChangesets       int
		manualChangesetsMerged int
	)

	if err := dbconn.Global.QueryRowContext(ctx, q).Scan(
		&campaignsCount,
		&actionChangesets,
		&actionChangesetsMerged,
		&manualChangesets,
		&manualChangesetsMerged,
	); err != nil {
		return nil, err
	}

	return &types.CampaignsUsageStatistics{
		CampaignsCount:              int32(campaignsCount),
		ActionChangesetsCount:       int32(actionChangesets),
		ActionChangesetsMergedCount: int32(actionChangesetsMerged),
		ManualChangesetsCount:       int32(manualChangesets),
		ManualChangesetsMergedCount: int32(manualChangesetsMerged),
	}, nil
}
