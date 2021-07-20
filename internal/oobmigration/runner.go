package oobmigration

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/derision-test/glock"
	"github.com/inconshreveable/log15"
	"github.com/opentracing/opentracing-go/log"

	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/internal/observation"
)

// Runner correlates out-of-band migration records in the database with a migrator instance,
// and will run each migration that has no yet completed: either reached 100% in the forward
// direction or 0% in the reverse direction.
type Runner struct {
	store         storeIface
	refreshTicker glock.Ticker
	operations    *operations
	migrators     map[int]migratorAndOption
	ctx           context.Context    // root context passed to the handler
	cancel        context.CancelFunc // cancels the root context
	finished      chan struct{}      // signals that Start has finished
}

type migratorAndOption struct {
	Migrator
	migratorOptions
}

var _ goroutine.BackgroundRoutine = &Runner{}

func NewRunnerWithDB(db dbutil.DB, refreshInterval time.Duration, observationContext *observation.Context) *Runner {
	return newRunner(NewStoreWithDB(db), glock.NewRealTicker(refreshInterval), observationContext)
}

func newRunner(store storeIface, refreshTicker glock.Ticker, observationContext *observation.Context) *Runner {
	ctx, cancel := context.WithCancel(context.Background())

	return &Runner{
		store:         store,
		refreshTicker: refreshTicker,
		operations:    newOperations(observationContext),
		migrators:     map[int]migratorAndOption{},
		ctx:           ctx,
		cancel:        cancel,
		finished:      make(chan struct{}),
	}
}

// ErrMigratorConflict occurs when multiple migrator instances are registered to the same
// out-of-band migration identifier.
var ErrMigratorConflict = errors.New("migrator already registered")

// MigratorOptions configures the behavior of a registered migrator.
type MigratorOptions struct {
	// Interval specifies the time between invocations of an active migration.
	Interval time.Duration

	// ticker mocks periodic behavior for tests.
	ticker glock.Ticker
}

// Register correlates the given migrator with the given migration identifier. An error is
// returned if a migrator is already associated with this migration.
func (r *Runner) Register(id int, migrator Migrator, options MigratorOptions) error {
	if _, ok := r.migrators[id]; ok {
		return ErrMigratorConflict
	}

	if options.Interval == 0 {
		options.Interval = time.Second
	}
	if options.ticker == nil {
		options.ticker = glock.NewRealTicker(options.Interval)
	}

	r.migrators[id] = migratorAndOption{migrator, migratorOptions{
		ticker: options.ticker,
	}}
	return nil
}

type migrationStatusError struct {
	id               int
	expectedProgress float64
	actualProgress   float64
}

func newMigrationStatusError(id int, expectedProgress, actualProgress float64) error {
	return migrationStatusError{
		id:               id,
		expectedProgress: expectedProgress,
		actualProgress:   actualProgress,
	}
}

func (e migrationStatusError) Error() string {
	return fmt.Sprintf("migration %d expected to be at %.2f%% (at %.2f%%)", e.id, e.expectedProgress*100, e.actualProgress*100)
}

// Validate checks the migration records present in the database (including their progress) and returns
// an error if there are unfinished migrations relative to the given version. Specifically, it is illegal
// for a Sourcegraph instance to start up with a migration that has one of the following properties:
//
// - A migration with progress != 0 is introduced _after_ the given version
// - A migration with progress != 1 is deprecated _on or before_ the given version
//
// This error is used to block startup of the application with an informative message indicating that
// the site admin must either (1) run the previous version of Sourcegraph longer to allow the unfinished
// migrations to complete in the case of a premature upgrade, or (2) run a standalone migration utility
// to rewind changes on an unmoving database in the case of a premature downgrade.
func (r *Runner) Validate(ctx context.Context, currentVersion, firstVersion Version) error {
	migrations, err := r.store.List(ctx)
	if err != nil {
		return err
	}

	errs := make([]error, 0, len(migrations))
	for _, migration := range migrations {
		currentVersionCmpIntroduced, err := compareVersions(currentVersion, migration.Introduced)
		if err != nil {
			return err
		}
		if currentVersionCmpIntroduced == VersionOrderBefore && migration.Progress != 0 {
			// Unfinished rollback: currentVersion before introduced version and progress > 0
			errs = append(errs, newMigrationStatusError(migration.ID, 0, migration.Progress))
		}

		if migration.Deprecated == nil {
			continue
		}

		firstVersionCmpDeprecated, err := compareVersions(firstVersion, *migration.Deprecated)
		if err != nil {
			return err
		}
		if firstVersionCmpDeprecated != VersionOrderBefore {
			// Edge case: sourcegraph instance booted on or after deprecation version
			continue
		}

		currentVersionCmpDeprecated, err := compareVersions(currentVersion, *migration.Deprecated)
		if err != nil {
			return err
		}
		if currentVersionCmpDeprecated != VersionOrderBefore && migration.Progress != 1 {
			// Unfinished migration: currentVersion on or after deprecated version, progress < 1
			errs = append(errs, newMigrationStatusError(migration.ID, 1, migration.Progress))
		}
	}

	return wrapMigrationErrors(errs...)
}

func wrapMigrationErrors(errs ...error) error {
	if len(errs) == 0 {
		return nil
	}

	descriptions := make([]string, 0, len(errs))
	for _, err := range errs {
		descriptions = append(descriptions, fmt.Sprintf("  - %s\n", err))
	}
	sort.Strings(descriptions)

	return errors.Errorf(
		"Unfinished migrations. Please revert Sourcegraph to the previous version and wait for the following migrations to complete.\n\n%s\n",
		strings.Join(descriptions, "\n"),
	)
}

// Start runs registered migrators on a loop until they complete. This method will periodically
// re-read from the database in order to refresh its current view of the migrations.
func (r *Runner) Start() {
	defer close(r.finished)

	ctx := r.ctx
	var wg sync.WaitGroup
	migrationProcesses := map[int]chan Migration{}

	// Periodically read the complete set of out-of-band migrations from the database
	for migrations := range r.listMigrations(ctx) {
		for i := range migrations {
			id := migrations[i].ID
			migrator, ok := r.migrators[id]
			if !ok {
				continue
			}

			// Ensure we have a migration routine running for this migration
			r.ensureProcessorIsRunning(&wg, migrationProcesses, id, func(ch <-chan Migration) {
				runMigrator(ctx, r.store, migrator.Migrator, ch, migrator.migratorOptions, r.operations)
			})

			// Send the new migration to the processor routine. This loop guarantees
			// that either (1) the routine can immediately write the new value into the
			// free buffer slot, in which case we immediately break; (2) the routine
			// cannot immediately write because the buffer slot is full with a migration
			// value that is comparatively out of date.
			//
			// In this second case we'll read from the channel to free the buffer slot
			// of the old value, then write our new value there.
			//
			// Note: This loop breaks after two iterations (at most).
		loop:
			for {
				select {
				case migrationProcesses[id] <- migrations[i]:
					break loop
				case <-migrationProcesses[id]:
				}
			}
		}
	}

	// Unblock all processor routines
	for _, ch := range migrationProcesses {
		close(ch)
	}

	// Wait for processor routines to finish
	wg.Wait()
}

// listMigrations returns a channel that will asynchronously receive the full list of out-of-band
// migrations that exist in the database. This channel will receive a value periodically as long
// as the given context is active.
func (r *Runner) listMigrations(ctx context.Context) <-chan []Migration {
	ch := make(chan []Migration)

	go func() {
		defer close(ch)

		for {
			migrations, err := r.store.List(ctx)
			if err != nil {
				log15.Error("Failed to list out-of-band migrations", "error", err)
			}

			select {
			case ch <- migrations:
			case <-ctx.Done():
				return
			}

			select {
			case <-r.refreshTicker.Chan():
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch
}

// ensureProcessorIsRunning ensures that there is a non-nil channel at m[id]. If this key
// is not set, a new channel is created and stored in this key. The channel is then passed
// to runMigrator in a goroutine.
//
// This method logs the execution of the migration processor in the given wait group.
func (r *Runner) ensureProcessorIsRunning(wg *sync.WaitGroup, m map[int]chan Migration, id int, runMigrator func(<-chan Migration)) {
	if _, ok := m[id]; ok {
		return
	}

	wg.Add(1)
	ch := make(chan Migration, 1)
	m[id] = ch

	go func() {
		runMigrator(ch)
		wg.Done()
	}()
}

// Stop will cancel the context used in Start, then blocks until Start has returned.
func (r *Runner) Stop() {
	r.cancel()
	<-r.finished
}

type migratorOptions struct {
	ticker glock.Ticker
}

// runMigrator runs the given migrator function periodically (on each read from ticker)
// while the migration is not complete. We will periodically (on each read from migrations)
// update our current view of the migration progress and (more importantly) its direction.
func runMigrator(ctx context.Context, store storeIface, migrator Migrator, migrations <-chan Migration, options migratorOptions, operations *operations) {
	// Get initial migration. This channel will close when the context
	// is canceled, so we don't need to do any more complex select here.
	migration, ok := <-migrations
	if !ok {
		return
	}

	// We're just starting up - refresh our progress before migrating
	if err := updateProgress(ctx, store, &migration, migrator); err != nil {
		log15.Error("Failed to determine migration progress", "migrationID", migration.ID, "error", err)
	}

	for {
		select {
		case migration = <-migrations:
			// We just got a new version of the migration from the database. We need to check
			// the actual progress based on the migrator in case the progress as stored in the
			// migrations table has been de-synchronized from the actual progress.
			if err := updateProgress(ctx, store, &migration, migrator); err != nil {
				log15.Error("Failed to determine migration progress", "migrationID", migration.ID, "error", err)
			}

		case <-options.ticker.Chan():
			if !migration.Complete() {
				// Run the migration only if there's something left to do
				if err := runMigrationFunction(ctx, store, &migration, migrator, operations); err != nil {
					log15.Error("Failed migration action", "migrationID", migration.ID, "error", err)
				}
			}

		case <-ctx.Done():
			return
		}
	}
}

// runMigrationFunction invokes the Up or Down method on the given migrator depending on the migration
// direction. If an error occurs, it will be associated in the database with the migration record.
// Regardless of the success of the migration function, the progress function on the migrator will be
// invoked and the progress written to the database.
func runMigrationFunction(ctx context.Context, store storeIface, migration *Migration, migrator Migrator, operations *operations) error {
	migrationFunc := runMigrationUp
	if migration.ApplyReverse {
		migrationFunc = runMigrationDown
	}

	if migrationErr := migrationFunc(ctx, migration, migrator, operations); migrationErr != nil {
		log15.Error("Failed to perform migration", "migrationID", migration.ID, "error", migrationErr)

		// Migration resulted in an error. All we'll do here is add this error to the migration's error
		// message list. Unless _that_ write to the database fails, we'll continue along the happy path
		// in order to update the migration, which could have made additional progress before failing.

		if err := store.AddError(ctx, migration.ID, migrationErr.Error()); err != nil {
			return err
		}
	}

	return updateProgress(ctx, store, migration, migrator)
}

// updateProgress invokes the Progress method on the given migrator, updates the Progress field of the
// given migration record, and updates the record in the database.
func updateProgress(ctx context.Context, store storeIface, migration *Migration, migrator Migrator) error {
	progress, err := migrator.Progress(ctx)
	if err != nil {
		return err
	}

	if err := store.UpdateProgress(ctx, migration.ID, progress); err != nil {
		return err
	}

	migration.Progress = progress
	return nil
}

func runMigrationUp(ctx context.Context, migration *Migration, migrator Migrator, operations *operations) (err error) {
	ctx, endObservation := operations.upForMigration(migration.ID).With(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("migrationID", migration.ID),
	}})
	defer endObservation(1, observation.Args{})

	log15.Debug("Running up migration", "migrationID", migration.ID)
	return migrator.Up(ctx)
}

func runMigrationDown(ctx context.Context, migration *Migration, migrator Migrator, operations *operations) (err error) {
	ctx, endObservation := operations.downForMigration(migration.ID).With(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("migrationID", migration.ID),
	}})
	defer endObservation(1, observation.Args{})

	log15.Debug("Running down migration", "migrationID", migration.ID)
	return migrator.Down(ctx)
}
