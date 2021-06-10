package cli

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/Masterminds/semver"
	"github.com/inconshreveable/log15"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/oobmigration"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/internal/version"
)

// newOutOfBandMigrationRunner creates and validates an out of band migrator instance.
// This method may issue a `log.Fatal` when there are migrations left in an unexpected
// state for the current application version.
func newOutOfBandMigrationRunner(ctx context.Context, db *sql.DB) *oobmigration.Runner {
	outOfBandMigrationRunner := oobmigration.NewRunnerWithDB(db, time.Second*30, &observation.Context{
		Logger:     log15.Root(),
		Tracer:     &trace.Tracer{Tracer: opentracing.GlobalTracer()},
		Registerer: prometheus.DefaultRegisterer,
	})

	validateOutOfBandMigrationRunner(ctx, db, outOfBandMigrationRunner)
	return outOfBandMigrationRunner
}

func validateOutOfBandMigrationRunner(ctx context.Context, db dbutil.DB, outOfBandMigrationRunner *oobmigration.Runner) {
	if os.Getenv("SRC_DISABLE_OOBMIGRATION_VALIDATION") != "" {
		log15.Warn("Skipping out-of-band migrations check")
		return
	}

	if version.IsDev(version.Version()) {
		log15.Warn("Skipping out-of-band migrations check (dev mode)", "version", version.Version())
		return
	}
	currentSemver, err := semver.NewVersion(version.Version())
	if err != nil {
		log15.Warn("Skipping out-of-band migrations check", "version", version.Version(), "error", err)
		return
	}

	firstVersion, err := backend.GetFirstServiceVersion(ctx, db, "frontend")
	if err != nil {
		log.Fatalf("Failed to retrieve first instance version: %v", err)
	}
	firstVersionSemver, err := semver.NewVersion(firstVersion)
	if err != nil {
		log15.Warn("Skipping out-of-band migrations check", "version", version.Version(), "error", err)
		return
	}

	// Ensure that there are no unfinished migrations that would cause inconsistent
	// results. If there are unfinished migrations, the site-admin needs to run the
	// previous version of Sourcegraph for longer while the migrations finish.
	//
	// This condition should only be hit when the site-admin prematurely updates to
	// a version that requires the migration process to be already finished. There
	// are warnings on the site-admin migration page indicating this danger.
	if err := outOfBandMigrationRunner.Validate(
		ctx,
		oobmigration.NewVersion(int(currentSemver.Major()), int(currentSemver.Minor())),
		oobmigration.NewVersion(int(firstVersionSemver.Major()), int(firstVersionSemver.Minor())),
	); err != nil {
		log.Fatalf("ERROR: %v", err)
	}
}
