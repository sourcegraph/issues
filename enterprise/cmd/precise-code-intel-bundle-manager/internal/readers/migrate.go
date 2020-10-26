package readers

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"

	"github.com/inconshreveable/log15"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/precise-code-intel-bundle-manager/internal/paths"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/cache"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/postgres"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/sqlite/migrate"
	"github.com/sourcegraph/sourcegraph/internal/db/basestore"
)

// NumMigrateRoutines is the number of goroutines launched to migrate bundle files.
var NumMigrateRoutines = 1

func Migrate(bundleDir string, storeCache cache.StoreCache, db *sql.DB) error {
	if err := migrateVersions(bundleDir, storeCache); err != nil {
		return fmt.Errorf("failed to migrate SQLite bundle versions: %s", err.Error())
	}

	if err := migrateToPostgres(bundleDir, storeCache, db); err != nil {
		return fmt.Errorf("failed to migrate SQLite bundles to Postgres: %s", err.Error())
	}

	return nil
}

// migrateVersions runs through each SQLite database on disk and opens a store instance which will perform
// any necessary migrations to transform it to the newest schema. Because this may have a non-negligible
// cost cost some intersection of migrations and database size, we try to pay this cost up-front instead
// of being paid on-demand when the database is opened within the query path. This method does not block
// the startup of the bundle manager as it does not change the correctness of the service.
func migrateVersions(bundleDir string, storeCache cache.StoreCache) error {
	version := migrate.CurrentSchemaVersion
	migrationMarkerFilename := paths.MigrationMarkerFilename(bundleDir, version)

	// If a file exists indicating the current schema version, then we've already run a full background
	// migration and can exit early. If the file doesn't exist, we'll run the migration and then write
	// to this file to indicate that we don't need to perform the migration again again in the future.
	if exists, err := paths.PathExists(migrationMarkerFilename); err != nil || exists {
		return err
	}

	paths, err := sqlitePaths(bundleDir)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return nil
	}

	ch := make(chan string, len(paths))
	defer close(ch)

	log15.Info(
		"Migrating bundles in background",
		"version", version,
		"numBundles", len(paths),
	)

	var wg sync.WaitGroup
	for i := 0; i < NumMigrateRoutines; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for filename := range ch {
				log15.Debug("Migrating bundle", "filename", filename)

				if err := storeCache.WithStore(context.Background(), filename, noopHandler); err != nil {
					log15.Error("Failed to migrate bundle", "err", err, "filename", filename)
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		touchFile(migrationMarkerFilename)
		log15.Info("Finished bundle migration", "version", version)
	}()

	for _, path := range paths {
		ch <- path
	}

	return nil
}

// sqlitePaths returns the paths of all SQLite files currently on disk ordered by file size
// (largest first). We order the files this way as we want to do expensive migrations in the
// background rather than on the query path and larger files take longer.
func sqlitePaths(bundleDir string) ([]string, error) {
	fileInfos, err := ioutil.ReadDir(paths.DBsDir(bundleDir))
	if err != nil {
		return nil, err
	}

	sizeByPath := map[string]int64{}
	for _, fileInfo := range fileInfos {
		id, err := strconv.Atoi(fileInfo.Name())
		if err != nil {
			continue
		}

		filename := paths.SQLiteDBFilename(bundleDir, int64(id))
		stat, err := os.Stat(filename)
		if err != nil {
			continue
		}

		sizeByPath[filename] = stat.Size()
	}

	// Construct a slice of the map keys
	paths := make([]string, 0, len(sizeByPath))
	for path := range sizeByPath {
		paths = append(paths, path)
	}

	// Order by descending size
	sort.Slice(paths, func(i, j int) bool {
		return sizeByPath[paths[j]] < sizeByPath[paths[i]]
	})

	return paths, nil
}

// touchFile ensures an empty file exists at the given path.
func touchFile(filename string) {
	file, err := os.Create(filename)
	if err != nil {
		log15.Error("Failed to create migration marker", "err", err)
		return
	}
	if err := file.Close(); err != nil {
		log15.Error("Failed to create migration marker", "err", err)
		return
	}
}

func noopHandler(store persistence.Store) error {
	return nil
}

func migrateToPostgres(bundleDir string, storeCache cache.StoreCache, db *sql.DB) error {
	var migrationCompleteMessage = fmt.Sprintf(
		"Migration to Postgres has completed. All existing LSIF bundles have moved to the path %s and can be removed from the filesystem to reclaim space.",
		paths.DBBackupsDir(bundleDir),
	)

	bundleFilenames, err := sqlitePaths(bundleDir)
	if err != nil {
		return err
	}
	if len(bundleFilenames) == 0 {
		log15.Info(migrationCompleteMessage)
		return nil
	}

	ctx := context.Background()

	bundleIDs, err := basestore.ScanInts(db.QueryContext(ctx, `SELECT dump_id FROM lsif_data_metadata`))
	if err != nil {
		return err
	}

	bundleIDMap := map[int]struct{}{}
	for _, bundleID := range bundleIDs {
		bundleIDMap[bundleID] = struct{}{}
	}

	moveFileToBackupDirectory := func(bundleID int64) {
		filename := paths.SQLiteDBFilename(bundleDir, bundleID)

		if err := os.Rename(filename, paths.DBBackupFilename(bundleDir, bundleID)); err != nil {
			log15.Error("Failed to move bundle to backups directory", "err", err, "filename", filename)
			return
		}

		if err := os.RemoveAll(filepath.Dir(filename)); err != nil {
			log15.Error("Failed to remove outer directory", "err", err, "directory", filepath.Dir(filename))
		}
	}

	updateIDs := map[string]int{}
	for _, filename := range bundleFilenames {
		bundleID, err := strconv.Atoi(filepath.Base(filepath.Dir(filename)))
		if err != nil {
			log15.Error("Failed to extract bundle id from filename", "err", err, "filename", filename)
			continue
		}

		if _, ok := bundleIDMap[bundleID]; ok {
			moveFileToBackupDirectory(int64(bundleID))
			continue
		}

		updateIDs[filename] = bundleID
	}

	if len(updateIDs) == 0 {
		log15.Info(migrationCompleteMessage)
		return nil
	}

	log15.Info(
		"Migrating bundle data to Postgres in background",
		"numBundles", len(updateIDs),
	)

	for filename, bundleID := range updateIDs {
		if err := postgres.MigrateBundleToPostgres(ctx, bundleID, filename, db); err != nil {
			log15.Error("Failed to migrate bundle", "err", err, "filename", filename)
			continue
		}

		moveFileToBackupDirectory(int64(bundleID))
	}

	log15.Info(migrationCompleteMessage)
	return nil
}
