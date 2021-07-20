package insights

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strconv"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/enterprise"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/insights/resolvers"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/oobmigration"
)

// IsEnabled tells if code insights are enabled or not.
func IsEnabled() bool {
	if v, _ := strconv.ParseBool(os.Getenv("DISABLE_CODE_INSIGHTS")); v {
		// Code insights can always be disabled. This can be a helpful escape hatch if e.g. there
		// are issues with (or connecting to) the codeinsights-db deployment and it is preventing
		// the Sourcegraph frontend or repo-updater from starting.
		//
		// It is also useful in dev environments if you do not wish to spend resources running Code
		// Insights.
		return false
	}
	if conf.IsDeployTypeSingleDockerContainer(conf.DeployType()) {
		// Code insights is not supported in single-container Docker demo deployments.
		return false
	}
	return true
}

// Init initializes the given enterpriseServices to include the required resolvers for insights.
func Init(ctx context.Context, postgres dbutil.DB, outOfBandMigrationRunner *oobmigration.Runner, enterpriseServices *enterprise.Services) error {
	if !IsEnabled() {
		if conf.IsDeployTypeSingleDockerContainer(conf.DeployType()) {
			enterpriseServices.InsightsResolver = resolvers.NewDisabledResolver("backend-run code insights are not available on single-container deployments")
		} else {
			enterpriseServices.InsightsResolver = resolvers.NewDisabledResolver("code insights has been disabled")
		}
		return nil
	}
	timescale, err := InitializeCodeInsightsDB("frontend")
	if err != nil {
		return err
	}
	enterpriseServices.InsightsResolver = resolvers.New(timescale, postgres)
	return nil
}

// InitializeCodeInsightsDB connects to and initializes the Code Insights Timescale DB, running
// database migrations before returning. It is safe to call from multiple services/containers (in
// which case, one's migration will win and the other caller will receive an error and should exit
// and restart until the other finishes.)
func InitializeCodeInsightsDB(app string) (*sql.DB, error) {
	timescaleDSN := conf.Get().ServiceConnections.CodeInsightsTimescaleDSN
	conf.Watch(func() {
		if newDSN := conf.Get().ServiceConnections.CodeInsightsTimescaleDSN; timescaleDSN != newDSN {
			log.Fatalf("Detected codeinsights database DSN change, restarting to take effect: %s", newDSN)
		}
	})

	db, err := dbconn.New(dbconn.Opts{DSN: timescaleDSN, DBName: "codeinsights", AppName: app})
	if err != nil {
		return nil, errors.Errorf("Failed to connect to codeinsights database: %s", err)
	}

	if err := dbconn.MigrateDB(db, dbconn.CodeInsights); err != nil {
		return nil, errors.Errorf("Failed to perform codeinsights database migration: %s", err)
	}
	return db, nil
}
