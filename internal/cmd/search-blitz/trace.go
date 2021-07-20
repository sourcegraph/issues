package main

import (
	"compress/gzip"
	"context"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/inconshreveable/log15"
)

// traceStore fetches jaeger traces and stores them gzipped locally for future
// consumption. Additionally it will delete old traces to prevent filling up
// the disk.
type traceStore struct {
	// Dir is the directory to store traces.
	Dir string

	// Token is the Sourcegraph Access token.
	Token string

	// MaxTotalTraceBytes is the maximum number of bytes on disk all traces
	// should take. CleanupLoop will remove old traces to keep this invariant.
	MaxTotalTraceBytes int64

	// MaxFetchAttempts is the maximum number of attempts to try fetching a trace
	// before failing. Defaults to 1 when zero valued.
	MaxFetchAttempts int

	// JaegerServerURL if non-empty will be used as the non-path of the URL
	// for queries. If set, Token will not be used. In production we can
	// internally access jaeger-query instead of needing an admin access token
	// + the jaeger proxy. Environment variable we use is JAEGER_SERVER_URL.
	JaegerServerURL string
}

// Fetch and store the trace.
func (t *traceStore) Fetch(ctx context.Context, traceURL string) (err error) {
	attempts := t.MaxFetchAttempts
	if attempts == 0 {
		attempts = 1
	}

	began := time.Now()
	attempt := 0

	defer func() {
		fetchDurationSeconds.WithLabelValues(
			strconv.FormatBool(err != nil),
			strconv.FormatInt(int64(attempt), 10),
		).Observe(time.Since(began).Seconds())
	}()

	for ; attempt < attempts; attempt++ {
		if err = t.fetch(ctx, traceURL); err != nil {
			time.Sleep(time.Second)
			continue
		}
		return nil
	}

	return errors.Errorf("failed to fetch trace %q after %d attempts: %v", traceURL, t.MaxFetchAttempts, err)
}

func (t *traceStore) fetch(ctx context.Context, traceURL string) (err error) {
	// prevent jaeger misbehaving stopping the next run of the query.
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	u, err := url.Parse(traceURL)
	if err != nil {
		return err
	}

	// translate trace url to json endpoint. Example:
	// before: https://sourcegraph.com/-/debug/jaeger/trace/5fd3f3b7e7206687
	// after:  https://sourcegraph.com/-/debug/jaeger/api/traces/5fd3f3b7e7206687
	traceID := path.Base(u.Path)
	u.Path = path.Dir(path.Dir(u.Path)) + "/api/traces/" + traceID

	// use internal URL vs public url. Example:
	// before: https://sourcegraph.com/-/debug/jaeger/api/traces/5fd3f3b7e7206687
	// after:  http://jaeger-query.prod:16686/-/debug/jaeger/api/traces/5fd3f3b7e7206687
	if t.JaegerServerURL != "" {
		p := u.Path
		u, err = url.Parse(t.JaegerServerURL)
		if err != nil {
			return err
		}
		u.Path = p
	}

	traceURL = u.String()

	req, err := http.NewRequest("GET", traceURL, nil)
	if err != nil {
		return err
	}

	req = req.WithContext(ctx)

	if t.JaegerServerURL == "" && t.Token != "" {
		req.Header.Set("Authorization", "token "+t.Token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return errors.Errorf("unexpected status code %d: %s", resp.StatusCode, b)
	}

	dst := filepath.Join(t.Dir, traceID+".json.gz")
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil && !os.IsExist(err) {
		return err
	}

	f, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	gz := gzip.NewWriter(f)

	_, err = io.Copy(gz, resp.Body)
	if err == nil {
		err = gz.Close()
	}

	if err != nil {
		_ = f.Close()
		_ = os.Remove(dst)
		return err
	}

	return f.Close()
}

// CleanupLoop periodically will remove old traces from disk such that we are
// using less than MaxTotalTraceBytes.
func (t *traceStore) CleanupLoop(ctx context.Context) {
	for {
		err := t.doCleanup()
		if err != nil {
			log15.Error("trace store cleanup failed", "error", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Minute):
		}
	}
}

func (t *traceStore) doCleanup() error {
	names, err := filepath.Glob(t.Dir + "/*.json.gz")
	if err != nil {
		return err
	}

	traces := make([]fs.FileInfo, 0, len(names))
	for _, name := range names {
		info, err := os.Lstat(name)
		if err != nil {
			return err
		}
		traces = append(traces, info)
	}

	var size int64
	for _, info := range traces {
		size += info.Size()
	}

	if size < t.MaxTotalTraceBytes {
		return nil
	}

	// sort by age so we can remove oldest
	sort.Slice(traces, func(i, j int) bool {
		return traces[i].ModTime().Before(traces[j].ModTime())
	})

	var (
		target       = int64(float64(t.MaxTotalTraceBytes) * 0.9)
		removed      int
		removedBytes int64
	)
	for _, info := range traces {
		if err := os.Remove(filepath.Join(t.Dir, info.Name())); err != nil {
			return err
		}

		removed++
		removedBytes += info.Size()
		size -= info.Size()
		if size <= target {
			break
		}
	}

	log15.Info("removed old traces", "removed", removed, "removedBytes", removedBytes, "aliveBytes", size)

	return nil
}
