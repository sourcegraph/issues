package batches

import (
	"context"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/enterprise"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/batches/resolvers"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/background"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/store"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types/scheduler/window"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/webhooks"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/encryption/keyring"
	"github.com/sourcegraph/sourcegraph/internal/oobmigration"
)

// Init initializes the given enterpriseServices to include the required
// resolvers for Batch Changes and sets up webhook handlers for changeset
// events.
func Init(ctx context.Context, db dbutil.DB, outOfBandMigrationRunner *oobmigration.Runner, enterpriseServices *enterprise.Services) error {
	// Validate site configuration.
	conf.ContributeValidator(func(c conf.Unified) (problems conf.Problems) {
		if _, err := window.NewConfiguration(c.BatchChangesRolloutWindows); err != nil {
			problems = append(problems, conf.NewSiteProblem(err.Error()))
		}

		return
	})

	cstore := store.New(db, keyring.Default().BatchChangesCredentialKey)

	enterpriseServices.BatchChangesResolver = resolvers.New(cstore)
	enterpriseServices.GitHubWebhook = webhooks.NewGitHubWebhook(cstore)
	enterpriseServices.BitbucketServerWebhook = webhooks.NewBitbucketServerWebhook(cstore)
	enterpriseServices.GitLabWebhook = webhooks.NewGitLabWebhook(cstore)

	return background.RegisterMigrations(cstore, outOfBandMigrationRunner)
}
