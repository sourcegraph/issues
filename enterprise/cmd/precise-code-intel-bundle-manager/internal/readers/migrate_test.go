package readers

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSQLitePaths(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("unexpected error creating temp dir: %s", err)
	}

	for i := 0; i < 20; i++ {
		filename := filepath.Join(tempDir, "dbs", strconv.Itoa(i), "sqlite.db")

		if err := os.MkdirAll(filepath.Dir(filename), os.ModePerm); err != nil {
			t.Fatalf("unexpected error creating dir: %s", err)
		}

		if err := ioutil.WriteFile(filename, make([]byte, (i%5)*100+(i/5)), os.ModePerm); err != nil {
			t.Fatalf("unexpected error writing file: %s", err)
		}
	}

	paths, err := sqlitePaths(tempDir)
	if err != nil {
		t.Fatalf("unexpected error getting sqlite paths: %s", err)
	}

	expectedPaths := []string{
		filepath.Join(tempDir, "dbs", "19", "sqlite.db"),
		filepath.Join(tempDir, "dbs", "14", "sqlite.db"),
		filepath.Join(tempDir, "dbs", "9", "sqlite.db"),
		filepath.Join(tempDir, "dbs", "4", "sqlite.db"),
		filepath.Join(tempDir, "dbs", "18", "sqlite.db"),
		filepath.Join(tempDir, "dbs", "13", "sqlite.db"),
		filepath.Join(tempDir, "dbs", "8", "sqlite.db"),
		filepath.Join(tempDir, "dbs", "3", "sqlite.db"),
		filepath.Join(tempDir, "dbs", "17", "sqlite.db"),
		filepath.Join(tempDir, "dbs", "12", "sqlite.db"),
		filepath.Join(tempDir, "dbs", "7", "sqlite.db"),
		filepath.Join(tempDir, "dbs", "2", "sqlite.db"),
		filepath.Join(tempDir, "dbs", "16", "sqlite.db"),
		filepath.Join(tempDir, "dbs", "11", "sqlite.db"),
		filepath.Join(tempDir, "dbs", "6", "sqlite.db"),
		filepath.Join(tempDir, "dbs", "1", "sqlite.db"),
		filepath.Join(tempDir, "dbs", "15", "sqlite.db"),
		filepath.Join(tempDir, "dbs", "10", "sqlite.db"),
		filepath.Join(tempDir, "dbs", "5", "sqlite.db"),
		filepath.Join(tempDir, "dbs", "0", "sqlite.db"),
	}
	if diff := cmp.Diff(expectedPaths, paths); diff != "" {
		t.Errorf("unexpected paths (-want +got):\n%s", diff)
	}
}
