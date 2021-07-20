package store

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
)

func TestPrepareZip(t *testing.T) {
	s, cleanup := tmpStore(t)
	defer cleanup()

	wantRepo := api.RepoName("foo")
	wantCommit := api.CommitID("deadbeefdeadbeefdeadbeefdeadbeefdeadbeef")

	returnFetch := make(chan struct{})
	var gotRepo api.RepoName
	var gotCommit api.CommitID
	var fetchZipCalled int64
	s.FetchTar = func(ctx context.Context, repo api.RepoName, commit api.CommitID) (io.ReadCloser, error) {
		<-returnFetch
		atomic.AddInt64(&fetchZipCalled, 1)
		gotRepo = repo
		gotCommit = commit
		return emptyTar(t), nil
	}

	// Fetch same commit in parallel to ensure single-flighting works
	startPrepareZip := make(chan struct{})
	prepareZipErr := make(chan error)
	for i := 0; i < 10; i++ {
		go func() {
			<-startPrepareZip
			_, err := s.PrepareZip(context.Background(), wantRepo, wantCommit)
			prepareZipErr <- err
		}()
	}
	close(startPrepareZip)
	close(returnFetch)
	for i := 0; i < 10; i++ {
		err := <-prepareZipErr
		if err != nil {
			t.Fatal("expected PrepareZip to succeed:", err)
		}
	}

	if gotCommit != wantCommit {
		t.Errorf("fetched wrong commit. got=%v want=%v", gotCommit, wantCommit)
	}
	if gotRepo != wantRepo {
		t.Errorf("fetched wrong repo. got=%v want=%v", gotRepo, wantRepo)
	}

	// Wait for item to appear on disk cache, then test again to ensure we
	// use the disk cache.
	onDisk := false
	for i := 0; i < 500; i++ {
		files, _ := os.ReadDir(s.Path)
		if len(files) != 0 {
			onDisk = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !onDisk {
		t.Fatal("timed out waiting for items to appear in cache at", s.Path)
	}
	_, err := s.PrepareZip(context.Background(), wantRepo, wantCommit)
	if err != nil {
		t.Fatal("expected PrepareZip to succeed:", err)
	}
}

func TestPrepareZip_fetchTarFail(t *testing.T) {
	fetchErr := errors.New("test")
	s, cleanup := tmpStore(t)
	defer cleanup()
	s.FetchTar = func(ctx context.Context, repo api.RepoName, commit api.CommitID) (io.ReadCloser, error) {
		return nil, fetchErr
	}
	_, err := s.PrepareZip(context.Background(), "foo", "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	if !errors.Is(err, fetchErr) {
		t.Fatalf("expected PrepareZip to fail with %v, failed with %v", fetchErr, err)
	}
}

func TestPrepareZip_errHeader(t *testing.T) {
	s, cleanup := tmpStore(t)
	defer cleanup()
	s.FetchTar = func(ctx context.Context, repo api.RepoName, commit api.CommitID) (io.ReadCloser, error) {
		buf := new(bytes.Buffer)
		w := tar.NewWriter(buf)
		w.Flush()
		buf.WriteString("oh yeah")
		err := w.Close()
		if err != nil {
			t.Fatal(err)
		}
		return io.NopCloser(bytes.NewReader(buf.Bytes())), nil
	}
	_, err := s.PrepareZip(context.Background(), "foo", "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	if have, want := errors.Cause(err).Error(), tar.ErrHeader.Error(); have != want {
		t.Fatalf("expected PrepareZip to fail with tar.ErrHeader, failed with %v", err)
	}
	if !errcode.IsTemporary(err) {
		t.Fatalf("expected PrepareZip to fail with a temporary error, failed with %v", err)
	}
}

func TestIngoreSizeMax(t *testing.T) {
	patterns := []string{
		"foo",
		"foo.*",
		"foo_*",
		"*.foo",
		"bar.baz",
	}
	tests := []struct {
		name    string
		ignored bool
	}{
		// Pass
		{"foo", true},
		{"foo.bar", true},
		{"foo_bar", true},
		{"bar.baz", true},
		{"bar.foo", true},
		// Fail
		{"baz.foo.bar", false},
		{"bar_baz", false},
		{"baz.baz", false},
	}

	for _, test := range tests {
		if got, want := ignoreSizeMax(test.name, patterns), test.ignored; got != want {
			t.Errorf("case %s got %v want %v", test.name, got, want)
		}
	}
}

func tmpStore(t *testing.T) (*Store, func()) {
	d, err := os.MkdirTemp("", "store_test")
	if err != nil {
		t.Fatal(err)
	}
	return &Store{
		Path: d,
	}, func() { os.RemoveAll(d) }
}

func emptyTar(t *testing.T) io.ReadCloser {
	buf := new(bytes.Buffer)
	w := tar.NewWriter(buf)
	err := w.Close()
	if err != nil {
		t.Fatal(err)
	}
	return io.NopCloser(bytes.NewReader(buf.Bytes()))
}
