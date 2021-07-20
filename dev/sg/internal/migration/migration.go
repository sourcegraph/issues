package migration

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/httpfs"
	"github.com/jackc/pgx/v4/stdlib"

	"github.com/sourcegraph/sourcegraph/dev/sg/internal/command"
	"github.com/sourcegraph/sourcegraph/dev/sg/internal/db"
	"github.com/sourcegraph/sourcegraph/dev/sg/internal/stdout"
	"github.com/sourcegraph/sourcegraph/dev/sg/root"
	"github.com/sourcegraph/sourcegraph/lib/output"
)

var once sync.Once
var out *output.Output = stdout.Out

// RunUp will migrate up the given number of steps.
// If n is nil then all migrations are ran.
func RunUp(database db.Database, n *int) error {
	return doRunMigrations(database, n, "up", func(m *migrate.Migrate) error { return m.Up() })
}

// RunDown will migrate down the given number of steps.
func RunDown(database db.Database, n *int) error {
	// Negative ints mean that we're going down.
	if n != nil {
		*n = -1 * *n
	}

	return doRunMigrations(database, n, "down", func(m *migrate.Migrate) error { return m.Down() })
}

// RunFixup will run the fixup command.
// The run parameter controls whether changes are actually executed, or just calculated.
// When run is false, no changes are made.
func RunFixup(database db.Database, main string, run bool) error {
	out.Write("")

	mainFiles, err := getMigrationFilesFromGit(database, main)
	if err != nil {
		return err
	}

	mainMigrations, err := getMigrationsForFiles(mainFiles, map[int]migration{})
	if err != nil {
		return err
	}

	localFiles, err := getMigrationFilesFromDisk(database)
	if err != nil {
		return err
	}

	localMigrations, err := getMigrationsForFiles(localFiles, mainMigrations)
	if err != nil {
		return err
	}

	block := out.Block(output.Linef("", output.StyleItalic, "Checking for conflicting migrations in '%s'...", database.Name))
	defer block.Close()

	_, missing, err := findConflictingMigrations(mainMigrations, localMigrations)
	if err != nil {
		return err
	}

	for _, missed := range missing {
		block.WriteLine(output.Linef(
			output.EmojiWarning,
			output.StyleReset,
			"Missing migration '%d_%s' from %s branch. Consider rebasing.", missed.ID, missed.Name, main,
		))
	}

	// TODO: It would be cool to prompt and ask the user if they want to continue if they have missing migrations.
	//       A lot of the time you probably want to stop here...

	mainMaxID := getMaxMigrationID(mainMigrations)
	block.Writef("Latest migration in the main branch  : %d (%s)", mainMaxID, mainMigrations[mainMaxID].Name)

	localMaxID := getMaxMigrationID(localMigrations)
	block.Writef("Latest migration in your local branch: %d (%s)", localMaxID, localMigrations[localMaxID].Name)

	version, err := getDatabaseMigrationVersion(database)
	if err != nil {
		return err
	}
	block.Writef("Current migration version in database: %d", version)

	operations, err := getMigrationOperations(mainMigrations, localMigrations, version)
	if err != nil {
		return err
	}

	if len(operations) > 0 {
		block.Write("")
		block.Write("Pending Operations:")
		for _, op := range operations {
			block.Writef("  %s", showPendingOperation(op))
		}

		if !run {
			block.Write("")
			block.WriteLine(output.Linef(output.EmojiInfo, output.StyleBold, "Not running current operations. Set 'run=true' to execute"))
			return nil
		}

		block.Write("")
		block.Write("Executing Operations:")

		options := OperationOptions{
			Database:       database,
			MaxMigrationID: mainMaxID,
			Run:            run,
			Block:          block,
		}

		var errIdx int
		for currentIdx, op := range operations {
			if err := op.Execute(&options); err != nil {
				errIdx = currentIdx
				break
			}
		}

		// Walk back the operations if we had an error.
		//    Each operation should implement a `Reset` command that undoes their actions.
		if err != nil {
			if true {
				// TODO: https://github.com/sourcegraph/sourcegraph/issues/22775
				panic("Unimplemented! https://github.com/sourcegraph/sourcegraph/issues/22775")
			}

			block.WriteLine(output.Linef(output.EmojiFailure, output.StyleBold, "An operation has failed. Reversing operations now"))
			for index := errIdx; errIdx >= 0; index-- {
				op := operations[index]
				if err := op.Reset(&options); err != nil {
					panic("Oh no no no, everything has gone wrong")
				}
			}

			return err
		}
	}

	block.Write("")
	block.WriteLine(output.Linef(output.EmojiSuccess, output.StyleReset, "Migrations and database are synced."))

	return nil
}

// RunAdd creates a new up/down migration file pair for the given database and
// returns the names of the new files. If there was an error, the filesystem should remain
// unmodified.
func RunAdd(database db.Database, migrationName string) (up, down string, _ error) {
	baseDir, err := MigrationDirectoryForDatabase(database)
	if err != nil {
		return "", "", err
	}

	// TODO: We can probably convert to migrations and use getMaxMigrationID
	names, err := ReadFilenamesNamesInDirectory(baseDir)
	if err != nil {
		return "", "", err
	}

	lastMigrationIndex, ok := ParseLastMigrationIndex(names)
	if !ok {
		return "", "", errors.New("no previous migrations exist")
	}

	upPath, downPath, err := MakeMigrationFilenames(database, lastMigrationIndex+1, migrationName)
	if err != nil {
		return "", "", err
	}

	if err := writeMigrationFiles(upPath, downPath); err != nil {
		return "", "", err
	}

	return upPath, downPath, nil
}

// MigrationDirectoryForDatabase returns the directory where migration files are stored for the
// given database.
func MigrationDirectoryForDatabase(database db.Database) (string, error) {
	repoRoot, err := root.RepositoryRoot()
	if err != nil {
		return "", err
	}

	return filepath.Join(repoRoot, "migrations", database.Name), nil
}

// MakeMigrationFilenames makes a pair of (absolute) paths to migration files with the
// given migration index and name.
func MakeMigrationFilenames(database db.Database, migrationIndex int, migrationName string) (up string, down string, _ error) {
	baseDir, err := MigrationDirectoryForDatabase(database)
	if err != nil {
		return "", "", err
	}

	upPath := filepath.Join(baseDir, fmt.Sprintf("%d_%s.up.sql", migrationIndex, migrationName))
	downPath := filepath.Join(baseDir, fmt.Sprintf("%d_%s.down.sql", migrationIndex, migrationName))
	return upPath, downPath, nil
}

// ParseMigrationIndex parse a filename and returns the migration index if the filename
// looks like a migration. Each migration filename has the form {unique_id}_{name}.{dir}.sql.
// This function returns a false-valued flag on failure. Leading directories are stripped
// from the input, so a basename or a full path can be supplied.
func ParseMigrationIndex(name string) (int, bool) {
	index, err := strconv.Atoi(strings.Split(filepath.Base(name), "_")[0])
	if err != nil {
		return 0, false
	}

	return index, true
}

// ParseLastMigrationIndex parses a list of filenames and returns the highest migration
// index available.
func ParseLastMigrationIndex(names []string) (int, bool) {
	indices := make([]int, 0, len(names))
	for _, name := range names {
		if index, ok := ParseMigrationIndex(name); ok {
			indices = append(indices, index)
		}
	}
	sort.Ints(indices)

	if len(indices) == 0 {
		return 0, false
	}

	return indices[len(indices)-1], true
}

// Returns the migration name from a filepath (e.g., 12341234_hello.up.sql -> hello).
func ParseMigrationName(name string) (string, bool) {
	names := strings.SplitN(filepath.Base(name), "_", 2)
	if len(names) < 2 {
		return "", false
	}

	baseName := names[1]
	migrationName := strings.ReplaceAll(strings.ReplaceAll(baseName, ".down.sql", ""), ".up.sql", "")
	return migrationName, true
}

const migrationFileTemplate = `
BEGIN;

-- Insert migration here. See README.md. Highlights:
--  * Always use IF EXISTS. eg: DROP TABLE IF EXISTS global_dep_private;
--  * All migrations must be backward-compatible. Old versions of Sourcegraph
--    need to be able to read/write post migration.
--  * Historically we advised against transactions since we thought the
--    migrate library handled it. However, it does not! /facepalm

COMMIT;
`

// writeMigrationFiles writes the contents of migrationFileTemplate to the given filepaths.
func writeMigrationFiles(paths ...string) (err error) {
	defer func() {
		if err != nil {
			for _, path := range paths {
				// undo any changes to the fs on error
				_ = os.Remove(path)
			}
		}
	}()

	for _, path := range paths {
		if err := os.WriteFile(path, []byte(migrationFileTemplate), os.ModePerm); err != nil {
			return err
		}
	}

	return nil
}

func doRunMigrations(database db.Database, n *int, name string, f func(*migrate.Migrate) error) error {
	var line output.FancyLine
	if n == nil {
		line = output.Linef("", output.StyleBold, "Running all %s migrations", name)
	} else {
		line = output.Linef("", output.StyleBold, "Running %s migrations: %d", name, *n)
	}

	block := out.Block(line)
	defer block.Close()

	logger := mLogger{block: block, prefix: "  applying: "}
	m, err := getMigrate(database, logger)
	if err != nil {
		block.WriteLine(output.Linef(output.EmojiFailure, output.StyleBold, "Could not get database"))
		return err
	}

	if n == nil {
		migrateErr := f(m)
		if migrateErr == migrate.ErrNoChange {
			block.WriteLine(output.Line(output.EmojiSuccess, output.StyleSuccess, "No Changes"))
			return nil
		} else if migrateErr != nil {
			block.WriteLine(output.Line(output.EmojiFailure, output.StyleSuccess, "ERR"))
			return migrateErr
		}
	} else {
		stepsErr := m.Steps(*n)
		if stepsErr != nil {
			block.WriteLine(output.Linef(output.EmojiFailure, output.StyleWarning, "Failed to apply migration steps: %s", stepsErr))
			return stepsErr
		}
	}

	block.WriteLine(output.Line(output.EmojiSuccess, output.StyleSuccess, "Succesfully ran migrations"))
	return nil
}

// makePostgresDSN returns a PostresDSN for the given database. For all databases except
// the default, the PG* environment variables are prefixed with the database name. The
// resulting address depends on the environment.
func makePostgresDSN(database db.Database) string {
	var prefix string
	if database.Name != db.DefaultDatabase.Name {
		prefix = strings.ToUpper(database.Name) + "_"
	}

	var port string
	if value := os.Getenv(fmt.Sprintf("%sPGPORT", prefix)); value != "" {
		port = ":" + value
	}

	return fmt.Sprintf(
		"postgres://%s%s/%s",
		os.Getenv(fmt.Sprintf("%sPGHOST", prefix)),
		port,
		os.Getenv(fmt.Sprintf("%sPGDATABASE", prefix)),
	)
}

// ReadFilenamesNamesInDirectory returns a list of names in the given directory.
func ReadFilenamesNamesInDirectory(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}

	return names, nil
}

type migration struct {
	ID   int
	Name string

	UpName   string
	DownName string
}

type migrationConflict struct {
	ID    int
	Main  migration
	Local migration
}

func getMigrationFilesFromGit(database db.Database, revision string) ([]string, error) {
	baseDir, err := MigrationDirectoryForDatabase(database)
	if err != nil {
		return nil, err
	}

	output, err := command.RunGit("ls-tree", "--name-only", "-r", revision, baseDir)
	if err != nil {
		return nil, err
	}
	files := strings.Split(output, "\n")

	return files, nil
}

func getMigrationFilesFromDisk(database db.Database) ([]string, error) {
	baseDir, err := MigrationDirectoryForDatabase(database)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0, len(entries))
	for _, fileEntry := range entries {
		files = append(files, fileEntry.Name())
	}

	return files, nil
}

func findConflictingMigrations(mainMigrations, localMigrations map[int]migration) ([]migrationConflict, []migration, error) {
	repoRoot, err := root.RepositoryRoot()
	if err != nil {
		return nil, nil, err
	}

	conflicts := []migrationConflict{}
	missing := []migration{}

	for migrationID, mainMigration := range mainMigrations {
		localMigration, ok := localMigrations[migrationID]
		if !ok {
			missing = append(missing, mainMigration)
			continue
		}

		if !fileExists(filepath.Join(repoRoot, mainMigration.UpName)) {
			missing = append(missing, mainMigration)
			continue
		}

		if mainMigration.Name != localMigration.Name {
			conflicts = append(conflicts, migrationConflict{
				ID:    migrationID,
				Main:  mainMigration,
				Local: localMigration,
			})
		}
	}

	return conflicts, missing, nil
}

func getMigrationsForFiles(files []string, existing map[int]migration) (map[int]migration, error) {
	shouldSkip := func(migrations map[int]string, migrationID int, migrationName string) bool {
		_, ok := migrations[migrationID]
		if !ok {
			return false
		}

		exist, ok := existing[migrationID]
		if !ok {
			return false
		}

		return exist.Name == migrationName
	}

	upMigrations := make(map[int]string)
	downMigrations := make(map[int]string)

	for _, file := range files {
		if file == "" {
			continue
		}

		if strings.HasPrefix(file, "_") {
			continue
		}

		migrationID, ok := ParseMigrationIndex(file)
		if !ok {
			return nil, errors.Newf("bad migration file format: %s", file)
		}

		migrationName, ok := ParseMigrationName(file)
		if !ok {
			return nil, errors.Newf("bad migration file name: %s", file)
		}

		if strings.HasSuffix(file, ".down.sql") {
			if shouldSkip(downMigrations, migrationID, migrationName) {
				continue
			}

			downMigrations[migrationID] = file
		} else if strings.HasSuffix(file, ".up.sql") {
			if shouldSkip(upMigrations, migrationID, migrationName) {
				continue
			}

			upMigrations[migrationID] = file
		} else if strings.HasSuffix(file, ".sql") {
			return nil, errors.Newf("sql file that doesn't fit migration file format: %s", file)
		}
	}

	if len(upMigrations) != len(downMigrations) {
		return nil, errors.Newf(
			"not the same number of up (%d) and down (%d) migrations. Check for corresponding up & down scripts for each migration.",
			len(upMigrations),
			len(downMigrations),
		)
	}

	migrations := make(map[int]migration)
	for migrationID := range upMigrations {
		upMigration, ok := upMigrations[migrationID]
		if !ok {
			return nil, errors.Newf("missing up migration for ID: %d", migrationID)
		}

		downMigration, ok := downMigrations[migrationID]
		if !ok {
			return nil, errors.Newf("missing down migration for ID: %d", migrationID)
		}

		migrationName, ok := ParseMigrationName(upMigration)
		if !ok {
			return nil, errors.Newf("bad migration file name: %s", upMigration)
		}

		migrations[migrationID] = migration{
			ID:       migrationID,
			Name:     migrationName,
			UpName:   upMigration,
			DownName: downMigration,
		}

	}

	return migrations, nil
}

func isFileInRepo(path string, block *output.Block) error {
	repoRoot, err := root.RepositoryRoot()
	if err != nil {
		return err
	}

	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		relativePath, _ := filepath.Rel(repoRoot, path)
		block.WriteLine(output.Linef(output.EmojiFailure, output.StyleItalic, "File does not exist: %s", relativePath))
	}

	return err
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist)
}

func getPostgresDB(database db.Database) (*sql.DB, error) {
	once.Do(func() {
		sql.Register("postgres-proxy", stdlib.GetDefaultDriver())
	})

	db, err := sql.Open("postgres-proxy", makePostgresDSN(database))
	if err != nil {
		return nil, errors.Wrap(err, "sql.Open")
	}

	return db, nil
}

func getDatabaseMigrationVersion(database db.Database) (int, error) {
	sqlDB, err := getPostgresDB(database)
	if err != nil {
		return 0, err
	}

	var version int
	row := sqlDB.QueryRow(fmt.Sprintf("SELECT version FROM %s", database.MigrationsTable))
	if err := row.Scan(&version); err != nil {
		return 0, err
	}

	return version, nil
}

func getMigrate(database db.Database, logger mLogger) (*migrate.Migrate, error) {
	db, err := getPostgresDB(database)
	if err != nil {
		return nil, err
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{
		MigrationsTable: database.MigrationsTable,
	})
	if err != nil {
		return nil, errors.Wrap(err, "postgres.WithInstance")
	}

	d, err := httpfs.New(http.FS(database.FS), ".")
	if err != nil {
		return nil, errors.Wrap(err, "httpfs.New")
	}

	m, err := migrate.NewWithInstance("httpfs", d, "postgres", driver)
	m.Log = logger

	return m, err
}

// mLogger implements the logger struct for migrate.Migrate
type mLogger struct {
	block  *output.Block
	prefix string
}

func (m mLogger) Printf(f string, i ...interface{}) {
	m.block.Writef(m.prefix+strings.TrimSpace(f), i...)
}
func (mLogger) Verbose() bool {
	return false
}
