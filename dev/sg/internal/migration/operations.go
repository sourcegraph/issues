package migration

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/golang-migrate/migrate/v4"
	"github.com/sourcegraph/sourcegraph/dev/sg/internal/db"
	"github.com/sourcegraph/sourcegraph/lib/output"
)

func getMaxMigrationID(migrations map[int]migration) int {
	maxID := 0
	for migrationID := range migrations {
		maxID = int(math.Max(float64(maxID), float64(migrationID)))
	}

	return maxID
}

func getMigrationOperations(mainMigrations, localMigrations map[int]migration, currentVersion int) ([]MigrationOperation, error) {
	conflicts, missing, _ := findConflictingMigrations(mainMigrations, localMigrations)

	sort.Slice(conflicts, func(i, j int) bool {
		return conflicts[i].ID < conflicts[j].ID
	})

	if len(missing) > 0 {
		return nil, errors.New(
			"You have missing migrations. Rebase first before trying fixup.\n" +
				"You may use 'sg migration up' or 'sg migration down' to manage migrations in your current branch.",
		)
	}

	operations := []MigrationOperation{}

	// If there are no conflicts and we're at exactly the right migration, nothing else to add.
	if len(conflicts) == 0 && currentVersion == getMaxMigrationID(localMigrations) {
		return operations, nil
	}

	// branchingVersion is the migration ID in which main & local have their first conflict.
	var branchingVersion = getMaxMigrationID(mainMigrations)
	for _, conflict := range conflicts {
		branchingVersion = min(branchingVersion, conflict.ID)
	}

	shouldChangeDB := false
	for _, conflict := range conflicts {
		if conflict.ID <= currentVersion {
			shouldChangeDB = true
			break
		}
	}

	if shouldChangeDB {
		// Must move conflicting migrations from main out of the directory
		//
		// This is because golang-migrate will fail to do anything if we have two migration files
		// with the same ID.
		//
		// So we will just hide them for a bit (they are reversed with `OpReverseTempfile`)
		for _, conflict := range conflicts {
			operations = append(operations, OpTempfile{
				Migration: conflict.Main,
			})
		}

		// You have to apply the down migrations in reverse.
		maxLocalID := getMaxMigrationID(localMigrations)
		for i := maxLocalID; i >= branchingVersion; i-- {
			migration, ok := localMigrations[i]
			if !ok {
				continue
			}

			if migration.ID <= currentVersion {
				operations = append(operations, OpDatabaseDown{
					Migration: migration,
				})
			}
		}

		// Now we can reverse the tempfiles.
		//    We don't need the golang-migrate to work during this stage,
		//    because we're going to immediately fixup the files after this.
		//
		//    Then once we're done, we'll be able to sync the database.
		for _, conflict := range conflicts {
			operations = append(operations, OpReverseTempfile{
				Migration: conflict.Main,
			})
		}
	}

	// resolve them conflicts
	minFixup := getMaxMigrationID(mainMigrations)
	for _, conflict := range conflicts {
		operations = append(operations, OpFileFixup{
			Conflict: conflict,
		})
	}

	// Will have to shift migrations accordingly
	if len(conflicts) > 0 {
		shiftExtra := len(conflicts)
		for i := minFixup; i < getMaxMigrationID(localMigrations); i++ {
			operations = append(operations, OpFileShift{
				Migration: localMigrations[i+1],
				Magnitude: shiftExtra,
			})
		}
	}

	// And then make the database be synced w/ current migrations
	operations = append(operations, OpDatabaseSync{})

	return operations, nil
}

type OperationOptions struct {
	// The database we're acting upon
	Database db.Database

	// The current maximum migration ID. Can be changed throughout one run.
	MaxMigrationID int

	// Whether or not to actually touch the database
	Run bool

	// Where to print things.
	Block *output.Block
}

type MigrationOperation interface {
	Execute(options *OperationOptions) error
	Reset(options *OperationOptions) error
	show() string
}

func showPendingOperation(op MigrationOperation) string {
	return fmt.Sprintf("%-18s: %s", strings.Replace(fmt.Sprintf("%T", op), "migration.", "", 1), op.show())
}

func makeTempPath(path string) string {
	dir, file := filepath.Split(path)
	return filepath.Join(dir, fmt.Sprintf("_%s", file))
}

func makeTempMigration(database db.Database, migration migration, block *output.Block) error {
	upPath, downPath, err := MakeMigrationFilenames(database, migration.ID, migration.Name)
	if err != nil {
		return err
	}

	upTempPath := makeTempPath(upPath)
	downTempPath := makeTempPath(downPath)

	if err := os.Rename(upPath, upTempPath); err != nil {
		return err
	}
	if err := os.Rename(downPath, downTempPath); err != nil {
		return err
	}

	block.Writef("  OpTempfile %d (%s)", migration.ID, migration.Name)

	return nil
}

func reverseTempMigration(database db.Database, migration migration, block *output.Block) error {
	upPath, downPath, err := MakeMigrationFilenames(database, migration.ID, migration.Name)
	if err != nil {
		return err
	}

	upTempPath := makeTempPath(upPath)
	downTempPath := makeTempPath(downPath)

	if err := os.Rename(upTempPath, upPath); err != nil {
		return err
	}
	if err := os.Rename(downTempPath, downPath); err != nil {
		return err
	}

	block.Writef("  OpReverseTempfile %d (%s)", migration.ID, migration.Name)

	return nil
}

// Reverse temp file moves a "<migration>.sql" file -> "_<migration>.sql"
// This makes it so that golang-migrate ignores the sql file, which is good for us.
type OpTempfile struct {
	Migration migration
}

func (op OpTempfile) Execute(options *OperationOptions) error {
	return makeTempMigration(options.Database, op.Migration, options.Block)
}
func (op OpTempfile) Reset(options *OperationOptions) error {
	return reverseTempMigration(options.Database, op.Migration, options.Block)
}
func (op OpTempfile) show() string {
	return fmt.Sprintf("%d %s", op.Migration.ID, op.Migration.Name)
}

// Reverse temp file moves a "_<migration>.sql" -> "<migration>.sql" file
type OpReverseTempfile struct {
	Migration migration
}

func (op OpReverseTempfile) Execute(options *OperationOptions) error {
	return reverseTempMigration(options.Database, op.Migration, options.Block)
}
func (op OpReverseTempfile) Reset(options *OperationOptions) error {
	return makeTempMigration(options.Database, op.Migration, options.Block)
}
func (op OpReverseTempfile) show() string {
	return fmt.Sprintf("%d %s", op.Migration.ID, op.Migration.Name)
}

func resolveOneConflict(database db.Database, conflict migrationConflict, maxID int, block *output.Block) error {
	newUp, newDown, err := MakeMigrationFilenames(database, maxID, conflict.Local.Name)
	if err != nil {
		return err
	}

	oldUp, oldDown, err := MakeMigrationFilenames(database, conflict.ID, conflict.Local.Name)
	if err != nil {
		return err
	}

	block.Writef("  Resolving migration conflict: %d %s", conflict.ID, conflict.Main.Name)

	// Check to make sure both up and down exist for this migration
	if isFileInRepo(oldUp, block) != nil || isFileInRepo(oldDown, block) != nil {
		// This should not be possible :) We should have died earlier but it's good to confirm
		return errors.Newf(
			"could not find both migration files for migration (%d): %s || %s",
			conflict.ID,
			oldUp,
			oldDown,
		)
	}

	if err := os.Rename(oldUp, newUp); err != nil {
		return err
	}
	if err := os.Rename(oldDown, newDown); err != nil {
		return err
	}
	block.Writef("  Moved migration %d -> %d (%s)", conflict.ID, maxID, conflict.Local.Name)

	return nil
}

// Fixes a conflict when two migrations share the same ID
type OpFileFixup struct {
	Conflict migrationConflict
}

func (op OpFileFixup) Execute(options *OperationOptions) error {
	options.MaxMigrationID += 1

	err := resolveOneConflict(options.Database, op.Conflict, options.MaxMigrationID, options.Block)
	if err != nil {
		return err
	}

	return err
}
func (op OpFileFixup) Reset(options *OperationOptions) error {
	// TODO: https://github.com/sourcegraph/sourcegraph/issues/22775
	return nil
}
func (op OpFileFixup) show() string {
	return fmt.Sprintf(
		"Conflicting migration %d main:%s local:%s",
		op.Conflict.ID,
		op.Conflict.Main.Name,
		op.Conflict.Local.Name,
	)
}

// Moves a migration when there is no conflict,
// but because of another shift and/or fixup, this migration must move.
//
// For an example, see: TestGetOprerationsWithLopsidedMigrations
type OpFileShift struct {
	Migration migration
	Magnitude int
}

func (op OpFileShift) Execute(options *OperationOptions) error {
	oldUp, oldDown, err := MakeMigrationFilenames(options.Database, op.Migration.ID, op.Migration.Name)
	if err != nil {
		return err
	}

	newUp, newDown, err := MakeMigrationFilenames(options.Database, op.Migration.ID+op.Magnitude, op.Migration.Name)
	if err != nil {
		return err
	}

	block := options.Block

	if err := os.Rename(oldUp, newUp); err != nil {
		return err
	}
	if err := os.Rename(oldDown, newDown); err != nil {
		return err
	}

	block.Writef("  Shifted %d -> %d (%s)", op.Migration.ID, op.Migration.ID+op.Magnitude, op.Migration.Name)

	return nil
}
func (op OpFileShift) Reset(options *OperationOptions) error {
	// TODO: https://github.com/sourcegraph/sourcegraph/issues/22775
	return nil
}
func (op OpFileShift) show() string {
	return fmt.Sprintf(
		"Shift migration '%s' from %d -> %d",
		op.Migration.Name,
		op.Migration.ID,
		op.Migration.ID+op.Magnitude,
	)
}

type OpDatabaseDown struct {
	Migration migration
}

func (op OpDatabaseDown) Execute(options *OperationOptions) error {
	logger := mLogger{block: options.Block, prefix: "  OpDbDown: "}
	m, err := getMigrate(options.Database, logger)
	if err != nil {
		return err
	}

	// migrate to the migration one below the desired migration
	if err := m.Migrate(uint(op.Migration.ID) - 1); err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			return err
		}
	}

	return nil
}
func (op OpDatabaseDown) Reset(options *OperationOptions) error {
	logger := mLogger{block: options.Block, prefix: "  OpDbDown - Reset: "}
	m, err := getMigrate(options.Database, logger)
	if err != nil {
		return err
	}

	if err := m.Migrate(uint(op.Migration.ID)); err != nil {
		return err
	}

	version, _ := getDatabaseMigrationVersion(options.Database)
	options.Block.Writef("==> Version: %d", version)

	return nil
}
func (op OpDatabaseDown) show() string {
	return fmt.Sprintf("Migrate down from %d (%s) -> %d", op.Migration.ID, op.Migration.Name, op.Migration.ID-1)
}

type OpDatabaseSync struct{}

func (op OpDatabaseSync) Execute(options *OperationOptions) error {
	logger := mLogger{block: options.Block, prefix: "  OpDatabaseSync: "}
	m, err := getMigrate(options.Database, logger)
	if err != nil {
		return err
	}

	options.Block.Write("  Applying migrations to local database")
	return m.Up()
}
func (op OpDatabaseSync) Reset(options *OperationOptions) error {
	return nil
}
func (op OpDatabaseSync) show() string {
	return "Sync to latest"
}

func min(x int, y int) int {
	return int(math.Min(float64(x), float64(y)))
}
