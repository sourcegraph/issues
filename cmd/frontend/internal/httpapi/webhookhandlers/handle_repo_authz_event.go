package webhookhandlers

import (
	"context"
	"fmt"

	gh "github.com/google/go-github/v28/github"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/globals"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater/protocol"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

// handleGithubRepoAuthzEvent handles any github event containing a repository field, and enqueues the contained
// repo for permissions synchronisation.
func handleGitHubRepoAuthzEvent(db dbutil.DB) func(ctx context.Context, extSvc *types.ExternalService, payload interface{}) error {
	return func(ctx context.Context, extSvc *types.ExternalService, payload interface{}) error {
		if !conf.ExperimentalFeatures().EnablePermissionsWebhooks {
			return nil
		}
		if globals.PermissionsUserMapping().Enabled {
			return nil
		}

		log15.Debug("handleGitHubRepoAuthzEvent: Got github event", "type", fmt.Sprintf("%T", payload))

		e, ok := payload.(repoGetter)
		if !ok {
			return fmt.Errorf("incorrect event type sent to github event handler: %T", payload)
		}
		return scheduleRepoUpdate(ctx, db, e.GetRepo())
	}
}

type repoGetter interface {
	GetRepo() *gh.Repository
}

// scheduleRepoUpdate finds an internal repo from a github repo, and posts it to repo-updater to
// schedule a permissions update
// 🚨 SECURITY: we want to be able to find any private repo here, so the DB call uses internal actor
func scheduleRepoUpdate(ctx context.Context, db dbutil.DB, repo *gh.Repository) error {
	if repo == nil {
		return nil
	}

	// 🚨 SECURITY: we want to be able to find any private repo here, so set internal actor
	ctx = actor.WithInternalActor(ctx)
	r, err := database.Repos(db).GetByName(ctx, api.RepoName("github.com/"+repo.GetFullName()))
	if err != nil {
		return err
	}

	log15.Debug("scheduleRepoUpdate: Dispatching permissions update", "repos", repo.GetFullName())

	c := repoupdater.DefaultClient
	return c.SchedulePermsSync(ctx, protocol.PermsSyncRequest{
		RepoIDs: []api.RepoID{r.ID},
	})
}
