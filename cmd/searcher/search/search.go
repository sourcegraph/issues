// Package search is a service which exposes an API to text search a repo at
// a specific commit.
//
// Architecture Notes:
// * Archive is fetched from gitserver
// * Simple HTTP API exposed
// * Currently no concept of authorization
// * On disk cache of fetched archives to reduce load on gitserver
// * Run search on archive. Rely on OS file buffers
// * Simple to scale up since stateless
// * Use ingress with affinity to increase local cache hit ratio
package search

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	nettrace "golang.org/x/net/trace"

	"github.com/cockroachdb/errors"
	"github.com/gorilla/schema"
	"github.com/inconshreveable/log15"
	"github.com/opentracing/opentracing-go/ext"
	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/sourcegraph/sourcegraph/cmd/searcher/protocol"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/store"
	"github.com/sourcegraph/sourcegraph/internal/trace/ot"
)

const (
	// maxLimit is a hard-coded maximum for total number of matches we return.
	// This may be increased in the future pending stability evaluation, or may
	// be removed entirely once streaming is in place and we don't need to buffer
	// the whole result set in memory.
	maxLimit = 100_000

	// numWorkers is how many concurrent readerGreps run in the case of
	// regexSearch, and the number of parallel workers in the case of
	// structuralSearch.
	numWorkers = 8
)

// Service is the search service. It is an http.Handler.
type Service struct {
	Store *store.Store
	Log   log15.Logger
}

var decoder = schema.NewDecoder()

func init() {
	decoder.IgnoreUnknownKeys(true)
}

// ServeHTTP handles HTTP based search requests
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	running.Inc()
	defer running.Dec()

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	var p protocol.Request
	err = decoder.Decode(&p, r.Form)
	if err != nil {
		http.Error(w, "failed to decode form: "+err.Error(), http.StatusBadRequest)
		return
	}
	if p.Deadline != "" {
		var deadline time.Time
		if err := deadline.UnmarshalText([]byte(p.Deadline)); err != nil {
			http.Error(w, "invalid deadline: "+err.Error(), http.StatusBadRequest)
			return
		}
		dctx, cancel := context.WithDeadline(ctx, deadline)
		defer cancel()
		ctx = dctx
	}
	if !p.PatternMatchesContent && !p.PatternMatchesPath {
		// BACKCOMPAT: Old frontends send neither of these fields, but we still want to
		// search file content in that case.
		p.PatternMatchesContent = true
	}
	if err = validateParams(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if p.Limit == 0 || p.Limit > maxLimit {
		p.Limit = maxLimit
	}

	ctx, cancel, stream := newLimitedStreamCollector(ctx, p.Limit)
	defer cancel()

	deadlineHit, err := s.search(ctx, &p, stream)
	if err != nil {
		code := http.StatusInternalServerError
		if errcode.IsBadRequest(err) || errors.Is(ctx.Err(), context.Canceled) {
			code = http.StatusBadRequest
		} else if errcode.IsTemporary(err) {
			code = http.StatusServiceUnavailable
		} else {
			log.Printf("internal error serving %#+v: %s", p, err)
		}
		http.Error(w, err.Error(), code)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	resp := protocol.Response{
		Matches:     stream.Collected(),
		LimitHit:    stream.LimitHit(),
		DeadlineHit: deadlineHit,
	}
	// The only reasonable error is the client going away now since we know we
	// can encode resp. This happens relatively often due to our
	// graphqlbackend regularly cancelling in-flight requests. We can't send
	// an error response, so we just ignore.
	_ = json.NewEncoder(w).Encode(&resp)
}

func (s *Service) search(ctx context.Context, p *protocol.Request, sender *limitedStreamCollector) (deadlineHit bool, err error) {
	tr := nettrace.New("search", fmt.Sprintf("%s@%s", p.Repo, p.Commit))
	tr.LazyPrintf("%s", p.Pattern)

	span, ctx := ot.StartSpanFromContext(ctx, "Search")
	ext.Component.Set(span, "service")
	span.SetTag("repo", p.Repo)
	span.SetTag("url", p.URL)
	span.SetTag("commit", p.Commit)
	span.SetTag("pattern", p.Pattern)
	span.SetTag("isRegExp", strconv.FormatBool(p.IsRegExp))
	span.SetTag("isStructuralPat", strconv.FormatBool(p.IsStructuralPat))
	span.SetTag("languages", p.Languages)
	span.SetTag("isWordMatch", strconv.FormatBool(p.IsWordMatch))
	span.SetTag("isCaseSensitive", strconv.FormatBool(p.IsCaseSensitive))
	span.SetTag("pathPatternsAreRegExps", strconv.FormatBool(p.PathPatternsAreRegExps))
	span.SetTag("pathPatternsAreCaseSensitive", strconv.FormatBool(p.PathPatternsAreCaseSensitive))
	span.SetTag("limit", p.Limit)
	span.SetTag("patternMatchesContent", p.PatternMatchesContent)
	span.SetTag("patternMatchesPath", p.PatternMatchesPath)
	span.SetTag("deadline", p.Deadline)
	span.SetTag("indexerEndpoints", p.IndexerEndpoints)
	span.SetTag("select", p.Select)
	defer func(start time.Time) {
		code := "200"
		// We often have canceled and timed out requests. We do not want to
		// record them as errors to avoid noise
		if ctx.Err() == context.Canceled {
			code = "canceled"
			span.SetTag("err", err)
		} else if ctx.Err() == context.DeadlineExceeded {
			code = "timedout"
			span.SetTag("err", err)
			deadlineHit = true
			err = nil // error is fully described by deadlineHit=true return value
		} else if err != nil {
			tr.LazyPrintf("error: %v", err)
			tr.SetError()
			ext.Error.Set(span, true)
			span.SetTag("err", err.Error())
			if errcode.IsBadRequest(err) {
				code = "400"
			} else if errcode.IsTemporary(err) {
				code = "503"
			} else {
				code = "500"
			}
		}
		tr.LazyPrintf("code=%s matches=%d limitHit=%v deadlineHit=%v", code, sender.SentCount(), sender.LimitHit(), deadlineHit)
		tr.Finish()
		requestTotal.WithLabelValues(code).Inc()
		span.LogFields(otlog.Int("matches.len", sender.SentCount()))
		span.SetTag("limitHit", sender.LimitHit())
		span.SetTag("deadlineHit", deadlineHit)
		span.Finish()
		if s.Log != nil {
			s.Log.Debug("search request", "repo", p.Repo, "commit", p.Commit, "pattern", p.Pattern, "isRegExp", p.IsRegExp, "isStructuralPat", p.IsStructuralPat, "languages", p.Languages, "isWordMatch", p.IsWordMatch, "isCaseSensitive", p.IsCaseSensitive, "patternMatchesContent", p.PatternMatchesContent, "patternMatchesPath", p.PatternMatchesPath, "matches", sender.SentCount(), "code", code, "duration", time.Since(start), "indexerEndpoints", p.IndexerEndpoints, "err", err)
		}
	}(time.Now())

	if p.IsStructuralPat && p.Indexed {
		// Execute the new structural search path that directly calls Zoekt.
		// TODO use limit in indexed structural search
		return structuralSearchWithZoekt(ctx, p, sender)
	}

	// Compile pattern before fetching from store incase it is bad.
	var rg *readerGrep
	if !p.IsStructuralPat {
		rg, err = compile(&p.PatternInfo)
		if err != nil {
			return false, badRequestError{err.Error()}
		}
	}

	if p.FetchTimeout == "" {
		p.FetchTimeout = "500ms"
	}
	fetchTimeout, err := time.ParseDuration(p.FetchTimeout)
	if err != nil {
		return false, err
	}
	prepareCtx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	getZf := func() (string, *store.ZipFile, error) {
		path, err := s.Store.PrepareZip(prepareCtx, p.Repo, p.Commit)
		if err != nil {
			return "", nil, err
		}
		zf, err := s.Store.ZipCache.Get(path)
		return path, zf, err
	}

	zipPath, zf, err := store.GetZipFileWithRetry(getZf)
	if err != nil {
		return false, errors.Wrap(err, "failed to get archive")
	}
	defer zf.Close()

	nFiles := uint64(len(zf.Files))
	bytes := int64(len(zf.Data))
	tr.LazyPrintf("files=%d bytes=%d", nFiles, bytes)
	span.LogFields(
		otlog.Uint64("archive.files", nFiles),
		otlog.Int64("archive.size", bytes))
	archiveFiles.Observe(float64(nFiles))
	archiveSize.Observe(float64(bytes))

	if p.IsStructuralPat {
		return false, filteredStructuralSearch(ctx, zipPath, zf, &p.PatternInfo, p.Repo, sender)
	} else {
		return false, regexSearch(ctx, rg, zf, p.Limit, p.PatternMatchesContent, p.PatternMatchesPath, p.IsNegated, sender)
	}
}

func validateParams(p *protocol.Request) error {
	if p.Repo == "" {
		return errors.New("Repo must be non-empty")
	}
	// Surprisingly this is the same sanity check used in the git source.
	if len(p.Commit) != 40 {
		return errors.Errorf("Commit must be resolved (Commit=%q)", p.Commit)
	}
	if p.Pattern == "" && p.ExcludePattern == "" && len(p.IncludePatterns) == 0 {
		return errors.New("At least one of pattern and include/exclude pattners must be non-empty")
	}
	if p.IsNegated && p.IsStructuralPat {
		return errors.New("Negated patterns are not supported for structural searches")
	}
	return nil
}

const megabyte = float64(1000 * 1000)

var (
	running = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "searcher_service_running",
		Help: "Number of running search requests.",
	})
	archiveSize = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "searcher_service_archive_size_bytes",
		Help:    "Observes the size when an archive is searched.",
		Buckets: []float64{1 * megabyte, 10 * megabyte, 100 * megabyte, 500 * megabyte, 1000 * megabyte, 5000 * megabyte},
	})
	archiveFiles = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "searcher_service_archive_files",
		Help:    "Observes the number of files when an archive is searched.",
		Buckets: []float64{100, 1000, 10000, 50000, 100000},
	})
	requestTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "searcher_service_request_total",
		Help: "Number of returned search requests.",
	}, []string{"code"})
)

type badRequestError struct{ msg string }

func (e badRequestError) Error() string    { return e.msg }
func (e badRequestError) BadRequest() bool { return true }
