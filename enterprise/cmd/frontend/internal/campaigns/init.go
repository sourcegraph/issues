package campaigns

import (
	"context"
	"database/sql"
	"time"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/enterprise"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns/resolvers"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/globalstatedb"
)

func Init(ctx context.Context, enterpriseServices *enterprise.Services) error {
	globalState, err := globalstatedb.Get(ctx)
	if err != nil {
		return err
	}

	campaignsStore := campaigns.NewStoreWithClock(dbconn.Global, msResolutionClock)
	repositories := repos.NewDBStore(dbconn.Global, sql.TxOptions{})

	enterpriseServices.CampaignsResolver = resolvers.NewResolver(dbconn.Global)
	enterpriseServices.GitHubWebhook = campaigns.NewGitHubWebhook(campaignsStore, repositories, msResolutionClock)
	enterpriseServices.BitbucketServerWebhook = campaigns.NewBitbucketServerWebhook(
		campaignsStore,
		repositories,
		msResolutionClock,
		"sourcegraph-"+globalState.SiteID,
	)
	enterpriseServices.GitLabWebhook = campaigns.NewGitLabWebhook(campaignsStore, repositories, msResolutionClock)

	return nil
}

var msResolutionClock = func() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }
