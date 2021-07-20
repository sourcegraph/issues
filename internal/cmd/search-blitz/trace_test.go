package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/cockroachdb/errors"
)

func TestTraceStore(t *testing.T) {
	traceID := "5fd3f3b7e7206687"
	tracePath := "/-/debug/jaeger/trace/" + traceID
	wantPath := "/-/debug/jaeger/api/traces/" + traceID
	wantAuth := "token s3cr3t"
	payload := []byte(`{"hello": "world"}`)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != wantPath {
			http.Error(w, "bad path: "+r.URL.Path, 404)
			return
		}
		if got := r.Header.Get("Authorization"); got != wantAuth {
			http.Error(w, "bad auth: "+got, 401)
			return
		}
		_, _ = w.Write(payload)
	}))
	t.Cleanup(ts.Close)

	store := &traceStore{
		Dir:   t.TempDir(),
		Token: "s3cr3t",
	}

	err := store.Fetch(context.Background(), ts.URL+tracePath)
	if err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(store.Dir, traceID+".json.gz")
	checkStoredPayload := func() error {
		got, err := readFileGZ(dst)
		if err != nil {
			return err
		}
		if !bytes.Equal(payload, got) {
			return errors.Errorf("unexpected payload on disk:\nwant: %s\ngot:  %s", payload, got)
		}
		return nil
	}

	if err := checkStoredPayload(); err != nil {
		t.Fatal(err)
	}

	// Now test the JaegerServerURL feature which doesn't use auth
	if err := os.Remove(dst); err != nil {
		t.Fatal(err)
	}
	store.JaegerServerURL = ts.URL
	wantAuth = ""
	err = store.Fetch(context.Background(), "https://sourcegraph.com"+tracePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := checkStoredPayload(); err != nil {
		t.Fatal(err)
	}

	// Test we don't cleanup
	store.MaxTotalTraceBytes = 10000
	if err := store.doCleanup(); err != nil {
		t.Fatal(err)
	}
	if err := checkStoredPayload(); err != nil {
		t.Fatal(err)
	}

	// Now make low enough to cleanup
	store.MaxTotalTraceBytes = 1
	if err := store.doCleanup(); err != nil {
		t.Fatal(err)
	}
	if err := checkStoredPayload(); !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

func readFileGZ(p string) ([]byte, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}

	return io.ReadAll(gz)
}
