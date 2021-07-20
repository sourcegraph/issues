package background

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/sourcegraph/sourcegraph/internal/insights"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/insights/compression"

	"golang.org/x/time/rate"

	"github.com/hashicorp/go-multierror"
	"github.com/inconshreveable/log15"
	"github.com/xhit/go-str2duration/v2"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/insights/background/queryrunner"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/insights/discovery"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/insights/store"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbcache"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/internal/metrics"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/internal/vcs"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

// The historical enqueuer takes regular search insights like a search for `errorf` and runs them
// in the past to find how many results for that query occurred in the past. It does this using
// live/unindexed searches slowly in the background, by finding an old Git commit closest to the
// time we're interested in. See the docstring on the historicalEnqueuer struct for an explanation
// of how that works.
//
// There are some major pros/cons of the implementation as it stands today. Pros:
//
// 1. It works and is reliable.
// 2. It is pretty extensively covered by tests.
// 3. It will not harm the rest of Sourcegraph (e.g. by enqueueing too much work, running too many search queries, etc.)
//
// The cons are:
//
// 1. It's a huge glorified series of nested for loops, which makes it complex and hard to read and
//    understand. I spent two full weeks refactoring various parts of it to make it nicer, but it's
//    really challenging to structure this code in a nice way because the problem and solution we're
//    fundamentally representing here is complex. i.e., the code is complex because the problem is.
// 2. The tests are a bit complex/difficult to read. This is a symptom of the algorithmic complexity
//    at play here. I considered testing individual units of the code more aggressively, but the
//    reality is that the individual units (e.g. each for loop) is not complex - it is the aggregate
//    of them that is. If you can find a more clear way to represent this, you are smarter than me.
//
// If you're reading this and frustrated or confused, message @slimsag and I'll help you out.

// newInsightHistoricalEnqueuer returns a background goroutine which will periodically find all of the search
// insights across all user settings, and determine for which dates they do not have data and attempt
// to backfill them by enqueueing work for executing searches with `before:` and `after:` filter
// ranges.
func newInsightHistoricalEnqueuer(ctx context.Context, workerBaseStore *basestore.Store, settingStore discovery.SettingStore, insightsStore *store.Store, observationContext *observation.Context) goroutine.BackgroundRoutine {
	metrics := metrics.NewOperationMetrics(
		observationContext.Registerer,
		"insights_historical_enqueuer",
		metrics.WithCountHelp("Total number of insights historical enqueuer executions"),
	)
	operation := observationContext.Operation(observation.Op{
		Name:    "HistoricalEnqueuer.Run",
		Metrics: metrics,
	})

	defaultRateLimit := rate.Limit(10.0)
	getRateLimit := getRateLimit(defaultRateLimit)

	limiter := rate.NewLimiter(getRateLimit(), 1)

	go conf.Watch(func() {
		val := getRateLimit()
		log15.Info(fmt.Sprintf("Updating insights/historical-worker rate limit value=%v", val))
		limiter.SetLimit(val)
	})

	repoStore := database.Repos(workerBaseStore.Handle().DB())

	framesToBackfill := func() int {
		if frames := conf.Get().InsightsHistoricalFrames; frames != 0 {
			return frames
		}
		return 6 // 6 one-month frames.
	}

	frameLength := func() time.Duration {
		defaultLen := 30 * 24 * time.Hour
		if s := conf.Get().InsightsHistoricalFrameLength; s != "" {
			parsed, err := str2duration.ParseDuration(s)
			if err != nil {
				log15.Error("insights: failed to parse site config insights.historical.frameLength", "error", err)
				return defaultLen
			}
			return parsed
		}
		return defaultLen
	}

	maxTime := time.Now().Add(-time.Duration(framesToBackfill()) * frameLength())

	historicalEnqueuer := &historicalEnqueuer{
		now:           time.Now,
		settingStore:  settingStore,
		insightsStore: insightsStore,
		loader:        insights.NewLoader(repoStore.Handle().DB()),
		repoStore:     database.Repos(workerBaseStore.Handle().DB()),
		limiter:       limiter,
		enqueueQueryRunnerJob: func(ctx context.Context, job *queryrunner.Job) error {
			_, err := queryrunner.EnqueueJob(ctx, workerBaseStore, job)
			return err
		},
		gitFirstEverCommit:   (&cachedGitFirstEverCommit{impl: git.FirstEverCommit}).gitFirstEverCommit,
		gitFindNearestCommit: git.FindNearestCommit,

		// Fill e.g. the last 52 weeks of data, recording 1 point per week.
		framesToBackfill: framesToBackfill,
		frameLength:      frameLength,

		frameFilter: compression.NewHistoricalFilter(true, maxTime, insightsStore.Handle().DB()),

		allReposIterator: (&discovery.AllReposIterator{
			IndexableReposLister:  dbcache.NewIndexableReposLister(repoStore),
			RepoStore:             repoStore,
			Clock:                 time.Now,
			SourcegraphDotComMode: envvar.SourcegraphDotComMode(),

			// If a new repository is added to Sourcegraph, it can take 0-15m for it to be picked
			// up for backfilling.
			RepositoryListCacheTime: 15 * time.Minute,
		}).ForEach,
	}

	// We use a periodic goroutine here just for metrics tracking. We specify 5s here so it runs as
	// fast as possible without wasting CPU cycles, but in reality the handler itself can take
	// minutes to hours to complete as it intentionally enqueues work slowly to avoid putting
	// pressure on the system.
	return goroutine.NewPeriodicGoroutineWithMetrics(ctx, 15*time.Minute, goroutine.NewHandlerWithErrorMessage(
		"insights_historical_enqueuer",
		historicalEnqueuer.Handler,
	), operation)
}

func getRateLimit(defaultValue rate.Limit) func() rate.Limit {
	return func() rate.Limit {
		val := conf.Get().InsightsHistoricalWorkerRateLimit

		var result rate.Limit
		if val == nil {
			result = defaultValue
		} else {
			result = rate.Limit(*val)
		}

		return result
	}
}

// RepoStore is a subset of the API exposed by the database.Repos() store (only the subset used by
// historicalEnqueuer.)
type RepoStore interface {
	GetByName(ctx context.Context, name api.RepoName) (*types.Repo, error)
}

// historicalEnqueuer effectively enqueues jobs that generate historical data for insights. Right
// now, it only supports search insights. It does this by adjusting the user's search query to be
// for a specific repo and commit like `repo:<repo>@<commit>`, where `<repo>` is every repository
// on Sourcegraph (one search per) and `<commit>` is a Git commit closest in time to the historical
// point in time we're trying to generate data for. A lot of effort is placed into doing the work
// slowly, linearly, and consistently over time without harming any other part of Sourcegraph
// (including the search API, by performing searches slowly and on single repositories at a time
// only.)
//
// It works roughly like this:
//
//   * For every repository on Sourcegraph (a subset on Sourcegraph.com):
//     * Build a list of time frames that we should consider
//	   * Check the commit index to see if any timeframes can be discarded (if they didn't change)
//     * For each frame:
//       * Find the oldest commit in the repository.
//         * For every unique search insight series (i.e. search query):
//           * Consider yielding/sleeping.
//           * If the series has data for this timeframe+repo already, nothing to do.
//           * If the timeframe we're generating data for is before the oldest commit in the repo, record a zero value.
//           * Else, locate the commit nearest to the point in time we're trying to get data for and
//             enqueue a queryrunner job to search that repository commit - recording historical data
//            for it.
//
// As you can no doubt see, there is much complexity and potential room for duplicative API calls
// here (e.g. "for every timeframe we list every repository"). For this exact reason, we do two
// things:
//
// 1. Cache duplicative calls to prevent performing heavy operations multiple times.
// 2. Lift heavy operations to the layer/loop one level higher, when it is sane to do so.
// 3. Ensure we perform work slowly, linearly, and with yielding/sleeping between any substantial
//    work being performed.
//
type historicalEnqueuer struct {
	// Required fields used for mocking in tests.
	now                   func() time.Time
	settingStore          discovery.SettingStore
	insightsStore         store.Interface
	loader                insights.Loader
	repoStore             RepoStore
	enqueueQueryRunnerJob func(ctx context.Context, job *queryrunner.Job) error
	gitFirstEverCommit    func(ctx context.Context, repoName api.RepoName) (*git.Commit, error)
	gitFindNearestCommit  func(ctx context.Context, repoName api.RepoName, revSpec string, target time.Time) (*git.Commit, error)
	frameFilter           compression.DataFrameFilter

	// framesToBackfill describes the number of historical timeframes to backfill data for.
	framesToBackfill func() int

	// frameLength describes the length of each timeframe to backfill data for.
	frameLength func() time.Duration

	// The iterator to use for walking over all repositories on Sourcegraph.
	allReposIterator func(ctx context.Context, each func(repoName string) error) error
	limiter          *rate.Limiter
}

func (h *historicalEnqueuer) Handler(ctx context.Context) error {
	// Discover all insights on the instance.
	foundInsights, err := discovery.Discover(ctx, h.settingStore, h.loader, discovery.InsightFilterArgs{})
	if err != nil {
		return errors.Wrap(err, "Discover")
	}

	// Deduplicate series that may be unique (e.g. different name/description) but do not have
	// unique data (i.e. use the same exact search query or webhook URL.)
	var (
		uniqueSeries    = map[string]insights.TimeSeries{}
		sortedSeriesIDs []string
		multi           error
	)
	for _, insight := range foundInsights {
		for _, series := range insight.Series {
			seriesID := discovery.Encode(series)
			if err != nil {
				multi = multierror.Append(multi, err)
				continue
			}
			_, exists := uniqueSeries[seriesID]
			if exists {
				continue
			}
			uniqueSeries[seriesID] = series
			sortedSeriesIDs = append(sortedSeriesIDs, seriesID)
		}
	}
	if err := h.buildFrames(ctx, uniqueSeries, sortedSeriesIDs); err != nil {
		return multierror.Append(multi, err)
	}
	return nil
}

// buildFrames is invoked to build historical data for all past timeframes that we care about
// backfilling data for. This is done in small chunks, specifically so that we perform work incrementally.
//
// It is only called if there is at least one insights series defined.
//
// It will return instantly if there are no unique series.
func (h *historicalEnqueuer) buildFrames(ctx context.Context, uniqueSeries map[string]insights.TimeSeries, sortedSeriesIDs []string) error {
	if len(uniqueSeries) == 0 {
		return nil // nothing to do.
	}
	var multi error

	frames := Frames(h.framesToBackfill(), h.frameLength(), h.now())

	hardErr := h.allReposIterator(ctx, h.buildForRepo(ctx, uniqueSeries, sortedSeriesIDs, frames, multi))
	return hardErr
}

func (h *historicalEnqueuer) buildForRepo(ctx context.Context, uniqueSeries map[string]insights.TimeSeries, sortedSeriesIDs []string, frames []compression.Frame, softErr error) func(repoName string) error {
	return func(repoName string) error {
		// Lookup the repository (we need its database ID)
		repo, err := h.repoStore.GetByName(ctx, api.RepoName(repoName))
		if err != nil {
			// Ignore RepoNotFoundErr because it could just be that the repository was actually
			// deleted and allReposIterator had it cached.
			if errors.HasType(err, &database.RepoNotFoundErr{}) {
				return err // hard DB error
			} else {
				return nil
			}
		}

		log15.Info("insights: starting frames", "starting_frames", frames)

		filtered := h.frameFilter.FilterFrames(ctx, frames, repo.ID)
		log15.Info("insights: sampling historical data frames", "repo_id", repo.ID, "frames", frames)

		// Find the first commit made to the repository on the default branch.
		firstHEADCommit, err := h.gitFirstEverCommit(ctx, api.RepoName(repoName))
		if err != nil {
			if errors.HasType(err, &gitserver.RevisionNotFoundError{}) || vcs.IsRepoNotExist(err) {
				return nil // no error - repo may not be cloned yet (or not even pushed to code host yet)
			}
			if strings.Contains(err.Error(), `failed (output: "usage: git rev-list [OPTION] <commit-id>...`) {
				return nil // repository is empty
			}
			// soft error, repo may be in a bad state but others might be OK.
			softErr = multierror.Append(softErr, errors.Wrap(err, "FirstEverCommit "+repoName))
			return nil
		}

		// For every series that we want to potentially gather historical data for, try.
		for _, seriesID := range sortedSeriesIDs {
			series := uniqueSeries[seriesID]

			for i := len(filtered) - 1; i >= 0; i-- {
				currentFrame := filtered[i]

				err := h.limiter.Wait(ctx)
				if err != nil {
					return err
				}

				// If we already have data for this frame+repo+series, then there's nothing to do.
				var numDataPoints int
				numDataPoints, err = h.insightsStore.CountData(ctx, store.CountDataOpts{
					From:     &currentFrame.From,
					To:       &currentFrame.To,
					SeriesID: &seriesID,
					RepoID:   &repo.ID,
				})
				if err != nil {
					softErr = multierror.Append(softErr, err)
					// In this case we will assume the point does not exist and query for it anyway.
				} else if numDataPoints > 0 {
					continue
				}

				// Build historical data for this unique timeframe+repo+series.
				hardErr, err := h.buildSeries(ctx, &buildSeriesContext{
					from:            currentFrame.From,
					to:              currentFrame.To,
					repo:            repo,
					firstHEADCommit: firstHEADCommit,
					seriesID:        seriesID,
					series:          series,
				})
				if err != nil {
					softErr = multierror.Append(softErr, err)
					continue
				}
				if hardErr != nil {
					return multierror.Append(softErr, hardErr)
				}
			}

		}
		return nil
	}
}

// buildSeriesContext describes context/parameters for a call to buildSeries()
type buildSeriesContext struct {
	// The timeframe we're building historical data for.
	from, to time.Time

	// The repository we're building historical data for.
	repo *types.Repo

	// The first commit made in the repository on the default branch.
	firstHEADCommit *git.Commit

	// The series we're building historical data for.
	seriesID string
	series   insights.TimeSeries
}

func Frames(numFrames int, frameLength time.Duration, current time.Time) []compression.Frame {
	frames := make([]compression.Frame, 0)

	for frame := 0; frame < numFrames; frame++ {
		// Determine the exact start and end time of this timeframe.
		from := current.Add(-time.Duration(frame+1) * frameLength)
		to := current.Add(-time.Duration(frame) * frameLength)

		frames = append([]compression.Frame{{
			From: from,
			To:   to,
		}}, frames...)
	}
	return frames
}

// buildSeries is invoked to build historical data for every unique timeframe * repo * series that
// could need backfilling. Note that this means that for a single search insight, this means this
// function may be called e.g. (52 timeframes) * (500000 repos) * (1 series) times.
//
// It may return both hard errors (e.g. DB connection failure, future series are unlikely to build)
// and soft errors (e.g. user's search query is invalid, future series are likely to build.)
func (h *historicalEnqueuer) buildSeries(ctx context.Context, bctx *buildSeriesContext) (hardErr, softErr error) {
	query := bctx.series.Query
	// TODO(slimsag): future: use the search query parser here to avoid any false-positives like a
	// search query with `content:"repo:"`.
	if strings.Contains(query, "repo:") {
		// We need to specify the repo: filter ourselves, so rewriting their query which already
		// contains this would be complex (we would need to enumerate all repos their query would
		// have matched the same way the search backend would've). We don't support this today.
		//
		// Another possibility is that they are specifying a non-default branch with the `repo:`
		// filter. We would need to handle this below if so - we don't today.
		return nil, nil
	}

	// We're trying to find the # of search results at the middle of the timeframe, ideally.
	frameDuration := bctx.to.Sub(bctx.from)
	frameMidpoint := bctx.from.Add(frameDuration / 2)

	// Optimization: If the timeframe we're building data for ends before the first commit in the
	// repository, then we know there are no results (the repository didn't have any commits at all
	// at that point in time.)
	repoName := string(bctx.repo.Name)
	if bctx.to.Before(bctx.firstHEADCommit.Author.Date) {
		if err := h.insightsStore.RecordSeriesPoint(ctx, store.RecordSeriesPointArgs{
			SeriesID: bctx.seriesID,
			Point: store.SeriesPoint{
				Time:  frameMidpoint,
				Value: 0, // no matches
			},
			RepoName: &repoName,
			RepoID:   &bctx.repo.ID,
		}); err != nil {
			hardErr = errors.Wrap(err, "RecordSeriesPoint")
			return // DB error
		}
		return // success - nothing else to do
	}

	// At this point, we know:
	//
	// 1. We're building data for the `[from, to]` timeframe.
	// 2. We're building data for the search `query`.
	//
	// We need a way to find out in that historical timeframe what the total # of results was.
	// There are only two ways to do that:
	//
	// 1. Run `type:diff` searches, this would give us matching lines added/removed/changed over
	//    time. To use this, we would need to ensure we *start* looking for historical data at the
	//    very first commit in the repo, and keep a running tally of added/removed/changed lines -
	//    this requires a lot of book-keeping.
	// 2. Choose some commits in the timeframe `[from, to]` (or, if none exist in that timeframe,
	//    whatever commit is closest) and perform a live/unindexed search for that `repo:<repo>@commit`
	//    which will effectively search the repo at that point in time.
	//
	// We do the 2nd, and start by trying to locate the commit nearest to the middle of the
	// timeframe we're trying to fill in historical data for.
	nearestCommit, err := h.gitFindNearestCommit(ctx, bctx.repo.Name, "HEAD", frameMidpoint)
	if err != nil {
		if errors.HasType(err, &gitserver.RevisionNotFoundError{}) || vcs.IsRepoNotExist(err) {
			return // no error - repo may not be cloned yet (or not even pushed to code host yet)
		}
		hardErr = errors.Wrap(err, "FindNearestCommit")
		return
	}
	if nearestCommit == nil {
		return // repository has no commits / is empty. Maybe not yet pushed to code host.
	}

	// Build the search query we will run. The most important part here is
	query = withCountUnlimited(query)
	query = fmt.Sprintf("%s repo:^%s$@%s", query, regexp.QuoteMeta(repoName), string(nearestCommit.ID))

	hardErr = h.enqueueQueryRunnerJob(ctx, &queryrunner.Job{
		SeriesID:    bctx.seriesID,
		SearchQuery: query,
		RecordTime:  &frameMidpoint,
		State:       "queued",
	})
	return
}

// cachedGitFirstEverCommit is a simple in-memory cache for gitFirstEverCommit calls. It does so
// using a map, and entries are never evicted because they are expected to be small and in general
// unchanging.
type cachedGitFirstEverCommit struct {
	impl func(ctx context.Context, repoName api.RepoName) (*git.Commit, error)

	mu    sync.Mutex
	cache map[api.RepoName]*git.Commit
}

func (c *cachedGitFirstEverCommit) gitFirstEverCommit(ctx context.Context, repoName api.RepoName) (*git.Commit, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cache == nil {
		c.cache = map[api.RepoName]*git.Commit{}
	}
	if cached, ok := c.cache[repoName]; ok {
		return cached, nil
	}
	entry, err := c.impl(ctx, repoName)
	if err != nil {
		return nil, err
	}
	c.cache[repoName] = entry
	return entry, nil
}
