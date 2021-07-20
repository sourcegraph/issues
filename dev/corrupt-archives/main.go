package main

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/cockroachdb/errors"
)

func corruptArchives(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	files := make([]fs.FileInfo, len(entries))
	for i := range entries {
		files[i], err = entries[i].Info()
		if err != nil {
			return err
		}
	}

	archives := []fs.FileInfo{}
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".zip") {
			archives = append(archives, f)
		}
	}

	for _, f := range archives {
		if err := corruptArchive(filepath.Join(dir, f.Name()), f.Size()); err != nil {
			return err
		}
	}

	return nil
}

func corruptArchive(path string, size int64) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return errors.Errorf("open err: %v", err)
	}
	defer file.Close()

	err = file.Truncate(size / 2)
	if err != nil {
		return err
	}
	_, err = file.Write([]byte(strings.Repeat("corrupt", 100)))

	return err
}

func main() {
	if err := corruptArchives(os.Args[len(os.Args)-1]); err != nil {
		log.Fatal(err)
	}
}
