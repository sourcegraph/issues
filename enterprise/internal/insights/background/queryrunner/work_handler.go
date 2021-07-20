package queryrunner

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/time/rate"

	"github.com/cockroachdb/errors"
	"github.com/graph-gophers/graphql-go"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/insights/store"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
)

var _ workerutil.Handler = &workHandler{}

// workHandler implements the dbworker.Handler interface by executing search queries and
// inserting insights about them to the insights Timescale database.
type workHandler struct {
	workerBaseStore *basestore.Store
	insightsStore   *store.Store
	limiter         *rate.Limiter
}

func (r *workHandler) Handle(ctx context.Context, record workerutil.Record) (err error) {
	defer func() {
		if err != nil {
			log15.Error("insights.queryrunner.workHandler", "error", err)
		}
	}()

	// Dequeue the job to get information about it, like what search query to perform.
	job, err := dequeueJob(ctx, r.workerBaseStore, record.RecordID())
	if err != nil {
		return err
	}

	err = r.limiter.Wait(ctx)
	if err != nil {
		return err
	}
	// Actually perform the search query.
	//
	// 🚨 SECURITY: The request is performed without authentication, we get back results from every
	// repository on Sourcegraph - so we must be careful to only record insightful information that
	// is OK to expose to every user on Sourcegraph (e.g. total result counts are fine, exposing
	// that a repository exists may or may not be fine, exposing individual results is definitely
	// not, etc.)
	var results *gqlSearchResponse
	results, err = search(ctx, job.SearchQuery)
	if err != nil {
		return err
	}

	// TODO(slimsag): future: Logs are not a good way to surface these errors to users.
	if len(results.Errors) > 0 {
		return errors.Errorf("GraphQL errors: %v", results.Errors)
	}
	if alert := results.Data.Search.Results.Alert; alert != nil {
		if alert.Title == "No repositories satisfied your repo: filter" {
			// We got zero results and no repositories matched. This could be for a few reasons:
			//
			// 1. The repo hasn't been cloned by Sourcegraph yet.
			// 2. The repo has been cloned by Sourcegraph, but the user hasn't actually pushed it
			//    to the code host yet so it's empty.
			// 3. This is a search query for backfilling data, and the repository is a fork/archive
			//    which are excluded from search results by default (and the user didn't put `fork:yes`
			//    etc. in their search query.)
			//
			// In any case, this is not a problem - we want to record that we got zero results in
			// general.
		} else {
			// Maybe the user's search query is actually wrong.
			return errors.Errorf("insights query issue: alert: %v query=%q", alert, job.SearchQuery)
		}
	}
	if results.Data.Search.Results.LimitHit {
		log15.Error("insights query issue", "problem", "limit hit", "query", job.SearchQuery)
	}
	if cloning := len(results.Data.Search.Results.Cloning); cloning > 0 {
		log15.Error("insights query issue", "cloning_repos", cloning, "query", job.SearchQuery)
	}
	if missing := len(results.Data.Search.Results.Missing); missing > 0 {
		log15.Error("insights query issue", "missing_repos", missing, "query", job.SearchQuery)
	}
	if timedout := len(results.Data.Search.Results.Timedout); timedout > 0 {
		log15.Error("insights query issue", "timedout_repos", timedout, "query", job.SearchQuery)
	}

	// 🚨 SECURITY: The request is performed without authentication, we get back results from every
	// repository on Sourcegraph - so we must be careful to only record insightful information that
	// is OK to expose to every user on Sourcegraph (e.g. total result counts are fine, exposing
	// that a repository exists may just barely be fine, exposing individual results is definitely
	// not, etc.) OR record only data that we later restrict to only users who have access to those
	// repositories.
	recordTime := time.Now()
	if job.RecordTime != nil {
		recordTime = *job.RecordTime
	}

	// Figure out how many matches we got for every unique repository returned in the search
	// results.
	matchesPerRepo := make(map[string]int, len(results.Data.Search.Results.Results)*4)
	for _, result := range results.Data.Search.Results.Results {
		decoded, err := decodeResult(result)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf(`for query "%s"`, job.SearchQuery))
		}
		matchesPerRepo[decoded.repoID()] = matchesPerRepo[decoded.repoID()] + decoded.matchCount()
	}

	// Record the number of results we got, one data point per-repository.
	repoStore := database.Repos(r.workerBaseStore.Handle().DB())
	for graphQLRepoID, matchCount := range matchesPerRepo {
		dbRepoID, err := graphqlbackend.UnmarshalRepositoryID(graphql.ID(graphQLRepoID))
		if err != nil {
			return errors.Wrap(err, "UnmarshalRepositoryID")
		}
		repo, err := repoStore.Get(ctx, dbRepoID)
		if err != nil {
			return errors.Wrap(err, "RepoStore.GetByID")
		}

		repoName := string(repo.Name)
		err = r.insightsStore.RecordSeriesPoint(ctx, store.RecordSeriesPointArgs{
			SeriesID: job.SeriesID,
			Point: store.SeriesPoint{
				Time:  recordTime,
				Value: float64(matchCount),
			},
			RepoName: &repoName,
			RepoID:   &repo.ID,
		})
		if err != nil {
			return errors.Wrap(err, "RecordSeriesPoint")
		}
	}
	return nil
}
