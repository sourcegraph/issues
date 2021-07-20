package testutil

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/net/context/ctxhttp"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/internal/api"
)

func FetchTarFromGithub(ctx context.Context, repo api.RepoName, commit api.CommitID) (io.ReadCloser, error) {
	// key is a sha256 hash since we want to use it for the disk name
	h := sha256.Sum256([]byte(string(repo) + " " + string(commit)))
	key := hex.EncodeToString(h[:])
	path := filepath.Join("/tmp/search_test/codeload/", key+".tar.gz")

	// Check codeload cache first
	r, err := openGzipReader(path)
	if err == nil {
		return r, nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}

	// Fetch archive to a temporary path
	tmpPath := path + ".part"
	url := fmt.Sprintf("https://codeload.%s/tar.gz/%s", string(repo), string(commit))
	fmt.Println("fetching", url)
	resp, err := ctxhttp.Get(ctx, nil, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("github repo archive: URL %s returned HTTP %d", url, resp.StatusCode)
	}
	f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil, err
	}
	defer func() { os.Remove(tmpPath) }()
	_, err = io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		return nil, err
	}

	// Ensure contents are written to disk
	if err := fsync(tmpPath); err != nil {
		return nil, err
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return nil, err
	}

	// Ensure rename is written to disk
	if err := fsync(filepath.Dir(path)); err != nil {
		return nil, err
	}

	return openGzipReader(path)
}

func openGzipReader(name string) (io.ReadCloser, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	r, err := gzip.NewReader(f)
	if err != nil {
		f.Close()
		return nil, err
	}
	return &gzipReadCloser{f: f, r: r}, nil
}

func fsync(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	err = f.Sync()
	if err1 := f.Close(); err == nil {
		err = err1
	}
	return err
}

type gzipReadCloser struct {
	f *os.File
	r *gzip.Reader
}

func (z *gzipReadCloser) Read(p []byte) (int, error) {
	return z.r.Read(p)
}

func (z *gzipReadCloser) Close() error {
	err := z.r.Close()
	if err1 := z.f.Close(); err == nil {
		err = err1
	}
	return err
}
