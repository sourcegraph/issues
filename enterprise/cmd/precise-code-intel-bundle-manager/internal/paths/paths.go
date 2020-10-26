package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

const uploadDir = "uploads"
const uploadPartsDir = "upload-parts"
const dbsDir = "dbs"
const dbPartsDir = "db-parts"
const dbBackupsDir = "db-backups"
const migrationMarkersDir = "migration-markers"

// PrepDirectories creates the root directories within the given bundle dir.
func PrepDirectories(bundleDir string) error {
	rootDirs := []string{
		uploadDir,
		uploadPartsDir,
		dbsDir,
		dbPartsDir,
		dbBackupsDir,
		migrationMarkersDir,
	}

	for _, dir := range rootDirs {
		if err := os.MkdirAll(filepath.Join(bundleDir, dir), os.ModePerm); err != nil {
			return err
		}
	}

	return nil
}

// UploadsDir returns the path of the directory containing upload files.
func UploadsDir(bundleDir string) string {
	return filepath.Join(bundleDir, uploadDir)
}

// UploadFilename returns the path of the upload with the given identifier.
func UploadFilename(bundleDir string, id int64) string {
	return filepath.Join(bundleDir, uploadDir, fmt.Sprintf("%d.gz", id))
}

// UploadPartsDir returns the path of the directory containing upload part files.
func UploadPartsDir(bundleDir string) string {
	return filepath.Join(bundleDir, uploadPartsDir)
}

// UploadPartFilename returns the path of the upload with the given identifier and part index.
func UploadPartFilename(bundleDir string, id, index int64) string {
	return filepath.Join(bundleDir, uploadPartsDir, fmt.Sprintf("%d.%d.gz", id, index))
}

// DBsDir returns the path of the directory containing db file trees.
func DBsDir(bundleDir string) string {
	return filepath.Join(bundleDir, dbsDir)
}

// DBDir returns the path of the directory containing files for a given bundle identifier.
func DBDir(bundleDir string, id int64) string {
	return filepath.Join(bundleDir, dbsDir, strconv.FormatInt(id, 10))
}

// SQLiteDBFilename returns the path of the SQLite db for the given bundle identifier.
func SQLiteDBFilename(bundleDir string, id int64) string {
	return filepath.Join(bundleDir, dbsDir, strconv.FormatInt(id, 10), "sqlite.db")
}

// DBPartsDir returns the path of the directory containing db part files.
func DBPartsDir(bundleDir string) string {
	return filepath.Join(bundleDir, dbPartsDir)
}

// DBPartFilename returns the path of the db with the given identifier and part index.
func DBPartFilename(bundleDir string, id, index int64) string {
	return filepath.Join(bundleDir, dbPartsDir, fmt.Sprintf("%d.%d.gz", id, index))
}

// DBBackupsDir returns the path of the directory containing db backup files.
func DBBackupsDir(bundleDir string) string {
	return filepath.Join(bundleDir, dbBackupsDir)
}

// DBBackupFilename returns the path of the backup SQLite db for the given bundle identifier.
func DBBackupFilename(bundleDir string, id int64) string {
	return filepath.Join(bundleDir, dbBackupsDir, strconv.FormatInt(id, 10)+".db")
}

// MigrationMarkerFilename returns the path to the file that marks a migration has been performed.
func MigrationMarkerFilename(bundleDir string, version int) string {
	return filepath.Join(bundleDir, migrationMarkersDir, fmt.Sprintf("v%d", version))
}
