package janitor

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testRoot(t *testing.T) string {
	bundleDir, err := ioutil.TempDir("", "precise-code-intel-bundle-manager-")
	if err != nil {
		t.Fatalf("unexpected error creating test directory: %s", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(bundleDir)
	})

	for _, dir := range []string{"", "uploads", "dbs", "upload-parts", "db-parts"} {
		path := filepath.Join(bundleDir, dir)
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			t.Fatalf("unexpected error creating test directory: %s", err)
		}
	}

	return bundleDir
}

func makeFile(path string, mtimes time.Time) error {
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	file.Close()

	return os.Chtimes(path, mtimes, mtimes)
}

func getFilenames(root string) ([]string, error) {
	var paths []string
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			paths = append(paths, strings.TrimPrefix(strings.TrimPrefix(path, root), string(os.PathSeparator)))
		}

		return err
	}); err != nil {
		return nil, err
	}

	return paths, nil
}
