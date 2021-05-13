// Package repoupdater implements the repo-updater service HTTP handler.
package repoupdater

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/inconshreveable/log15"
	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/awscodecommit"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/bitbucketserver"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/github"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitlab"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
	"github.com/sourcegraph/sourcegraph/internal/repos"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater/protocol"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

// Server is a repoupdater server.
type Server struct {
	*repos.Store
	*repos.Syncer
	SourcegraphDotComMode bool
	GithubDotComSource    interface {
		GetRepo(ctx context.Context, nameWithOwner string) (*types.Repo, error)
	}
	GitLabDotComSource interface {
		GetRepo(ctx context.Context, projectWithNamespace string) (*types.Repo, error)
	}
	MavenSource interface {
		GetRepo(ctx context.Context, artifactName string) (*types.Repo, error)
	}
	Scheduler interface {
		UpdateOnce(id api.RepoID, name api.RepoName)
		ScheduleInfo(id api.RepoID) *protocol.RepoUpdateSchedulerInfoResult
	}
	GitserverClient interface {
		ListCloned(context.Context) ([]string, error)
	}
	ChangesetSyncRegistry interface {
		// EnqueueChangesetSyncs will queue the supplied changesets to sync ASAP.
		EnqueueChangesetSyncs(ctx context.Context, ids []int64) error
		// HandleExternalServiceSync should be called when an external service changes so that
		// the registry can start or stop the syncer associated with the service
		HandleExternalServiceSync(es api.ExternalService)
	}
	RateLimitSyncer interface {
		// SyncRateLimiters should be called when an external service changes so that
		// our internal rate limiters are kept in sync
		SyncRateLimiters(ctx context.Context) error
	}
	PermsSyncer interface {
		// ScheduleUsers schedules new permissions syncing requests for given users.
		ScheduleUsers(ctx context.Context, userIDs ...int32)
		// ScheduleRepos schedules new permissions syncing requests for given repositories.
		ScheduleRepos(ctx context.Context, repoIDs ...api.RepoID)
	}
}

// Handler returns the http.Handler that should be used to serve requests.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/repo-update-scheduler-info", s.handleRepoUpdateSchedulerInfo)
	mux.HandleFunc("/repo-lookup", s.handleRepoLookup)
	mux.HandleFunc("/enqueue-repo-update", s.handleEnqueueRepoUpdate)
	mux.HandleFunc("/sync-external-service", s.handleExternalServiceSync)
	mux.HandleFunc("/enqueue-changeset-sync", s.handleEnqueueChangesetSync)
	mux.HandleFunc("/schedule-perms-sync", s.handleSchedulePermsSync)
	return mux
}

// TODO(tsenart): Reuse this function in all handlers.
func respond(w http.ResponseWriter, code int, v interface{}) {
	switch val := v.(type) {
	case error:
		if val != nil {
			log15.Error(val.Error())
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(code)
			fmt.Fprintf(w, "%v", val)
		}
	default:
		w.Header().Set("Content-Type", "application/json")
		bs, err := json.Marshal(v)
		if err != nil {
			respond(w, http.StatusInternalServerError, err)
			return
		}

		w.WriteHeader(code)
		if _, err = w.Write(bs); err != nil {
			log15.Error("failed to write response", "error", err)
		}
	}
}

func (s *Server) handleRepoUpdateSchedulerInfo(w http.ResponseWriter, r *http.Request) {
	var args protocol.RepoUpdateSchedulerInfoArgs
	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result := s.Scheduler.ScheduleInfo(args.ID)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleRepoLookup(w http.ResponseWriter, r *http.Request) {
	var args protocol.RepoLookupArgs
	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := s.repoLookup(r.Context(), args)
	if err != nil {
		if r.Context().Err() != nil {
			http.Error(w, "request canceled", http.StatusGatewayTimeout)
			return
		}
		log15.Error("repoLookup failed", "args", &args, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleEnqueueRepoUpdate(w http.ResponseWriter, r *http.Request) {
	var req protocol.RepoUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, err)
		return
	}
	result, status, err := s.enqueueRepoUpdate(r.Context(), &req)
	if err != nil {
		log15.Error("enqueueRepoUpdate failed", "req", req, "error", err)
		respond(w, status, err)
		return
	}
	respond(w, status, result)
}

func (s *Server) enqueueRepoUpdate(ctx context.Context, req *protocol.RepoUpdateRequest) (resp *protocol.RepoUpdateResponse, httpStatus int, err error) {
	tr, ctx := trace.New(ctx, "enqueueRepoUpdate", req.String())
	defer func() {
		log15.Debug("enqueueRepoUpdate", "httpStatus", httpStatus, "resp", resp, "error", err)
		if resp != nil {
			tr.LogFields(
				otlog.Int32("resp.id", int32(resp.ID)),
				otlog.String("resp.name", resp.Name),
				otlog.String("resp.url", resp.URL),
			)
		}
		tr.SetError(err)
		tr.Finish()
	}()

	rs, err := s.Store.RepoStore.List(ctx, database.ReposListOptions{Names: []string{string(req.Repo)}})
	if err != nil {
		return nil, http.StatusInternalServerError, errors.Wrap(err, "store.list-repos")
	}

	if len(rs) != 1 {
		return nil, http.StatusNotFound, errors.Errorf("repo %q not found in store", req.Repo)
	}

	repo := rs[0]

	s.Scheduler.UpdateOnce(repo.ID, repo.Name)

	return &protocol.RepoUpdateResponse{
		ID:   repo.ID,
		Name: string(repo.Name),
	}, http.StatusOK, nil
}

func (s *Server) handleExternalServiceSync(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	var req protocol.ExternalServiceSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	src, err := repos.NewSource(&types.ExternalService{
		ID:          req.ExternalService.ID,
		Kind:        req.ExternalService.Kind,
		DisplayName: req.ExternalService.DisplayName,
		Config:      req.ExternalService.Config,
	}, httpcli.NewExternalHTTPClientFactory())
	if err != nil {
		log15.Error("server.external-service-sync", "kind", req.ExternalService.Kind, "error", err)
		return
	}

	err = externalServiceValidate(ctx, req, src)
	if err == github.ErrIncompleteResults {
		log15.Info("server.external-service-sync", "kind", req.ExternalService.Kind, "error", err)
		syncResult := &protocol.ExternalServiceSyncResult{
			ExternalService: req.ExternalService,
			Error:           err.Error(),
		}
		respond(w, http.StatusOK, syncResult)
		return
	} else if ctx.Err() != nil {
		// client is gone
		return
	} else if err != nil {
		log15.Error("server.external-service-sync", "kind", req.ExternalService.Kind, "error", err)
		respond(w, http.StatusInternalServerError, err)
		return
	}

	if err := s.Syncer.TriggerExternalServiceSync(ctx, req.ExternalService.ID); err != nil {
		log15.Warn("Enqueueing external service sync job", "error", err, "id", req.ExternalService.ID)
	}

	if s.RateLimitSyncer != nil {
		err = s.RateLimitSyncer.SyncRateLimiters(ctx)
		if err != nil {
			log15.Warn("Handling rate limiter sync", "err", err)
		}
	}
	if s.ChangesetSyncRegistry != nil {
		s.ChangesetSyncRegistry.HandleExternalServiceSync(req.ExternalService)
	}

	log15.Info("server.external-service-sync", "synced", req.ExternalService.Kind)
	respond(w, http.StatusOK, &protocol.ExternalServiceSyncResult{
		ExternalService: req.ExternalService,
	})
}

func externalServiceValidate(ctx context.Context, req protocol.ExternalServiceSyncRequest, src repos.Source) error {
	if !req.ExternalService.DeletedAt.IsZero() {
		// We don't need to check deleted services.
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make(chan repos.SourceResult)

	if v, ok := src.(repos.UserSource); ok {
		if err := v.ValidateAuthenticator(ctx); err != nil {
			return err
		}
	} else {
		go func() {
			src.ListRepos(ctx, results)
			close(results)
		}()

		for res := range results {
			if res.Err != nil {
				// Send error to user before waiting for all results, but drain
				// the rest of the results to not leak a blocked goroutine
				go func() {
					for range results {
					}
				}()
				return res.Err
			}
		}
	}

	return nil
}

var mockRepoLookup func(protocol.RepoLookupArgs) (*protocol.RepoLookupResult, error)

func (s *Server) repoLookup(ctx context.Context, args protocol.RepoLookupArgs) (result *protocol.RepoLookupResult, err error) {
	// Sourcegraph.com: this is on the user path, do not block for ever if codehost is being
	// bad. Ideally block before cloudflare 504s the request (1min).
	// Other: we only speak to our database, so response should be in a few ms.
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	tr, ctx := trace.New(ctx, "repoLookup", args.String())
	defer func() {
		log15.Debug("repoLookup", "result", result, "error", err)
		if result != nil {
			tr.LazyPrintf("result: %s", result)
		}
		tr.SetError(err)
		tr.Finish()
	}()

	if args.Repo == "" {
		return nil, errors.New("Repo must be set (is blank)")
	}

	if mockRepoLookup != nil {
		return mockRepoLookup(args)
	}

	list, err := s.Store.RepoStore.List(ctx, database.ReposListOptions{
		Names: []string{string(args.Repo)},
	})
	if err != nil {
		return nil, err
	}
	var repo *types.Repo
	if len(list) > 0 {
		repo = list[0]
	}

	// If we are sourcegraph.com we don't sync our site level code hosts in the background
	// since there are too many repos. Instead we use an incremental approach where we check
	// for changes everytime a user browses a repo. RepoLookup is the signal
	// we rely on to check metadata.

	codehost := extsvc.CodeHostOf(args.Repo, extsvc.PublicCodeHosts...)
	if s.SourcegraphDotComMode && codehost != nil {
		if repo == nil {
			// Try and find this repo on the remote host. Block on the remote
			// request.
			return s.remoteRepoSync(ctx, codehost, string(args.Repo))
		}

		// TODO a queue with single flighting to speak to remote for args.Repo?

		// We have (potentially stale) data we can return to the user right now. Do that
		// rather than blocking. This should only happen for public repos, private repos
		// are ignored since if they do exist in our DB they would have been added by a
		// user owned code host connection in which case they'll be kept up to date by
		// our background syncer.
		if !repo.Private {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
				defer cancel()
				repoResult, err := s.remoteRepoSync(ctx, codehost, string(args.Repo))
				if err != nil {
					log15.Error("async remoteRepoSync failed", "repo", args.Repo, "error", err)
					return
				}

				// Since we are only dealing with public repos here we can safely assume that
				// when a repository stored in the database is not accessible anymore, no other
				// external service should have access to it, we can then remove it.
				if repoResult.ErrorNotFound || repoResult.ErrorUnauthorized {
					err = s.Store.UpsertRepos(ctx, repo.With(func(r *types.Repo) {
						r.DeletedAt = s.Now()
					}))
					if err != nil {
						log15.Error("failed to delete inaccessible repo", "repo", args.Repo, "error", err)
					}
				}
			}()
		}
	}

	if repo == nil {
		return &protocol.RepoLookupResult{
			ErrorNotFound: true,
		}, nil
	}

	repoInfo, err := newRepoInfo(repo)
	if err != nil {
		return nil, err
	}

	return &protocol.RepoLookupResult{
		Repo: repoInfo,
	}, nil
}

// remoteRepoSync is used by Sourcegraph.com to incrementally sync metadata
// for remoteName on codehost.
func (s *Server) remoteRepoSync(ctx context.Context, codehost *extsvc.CodeHost, remoteName string) (*protocol.RepoLookupResult, error) {
	var repo *types.Repo
	var err error
	switch codehost {
	case extsvc.GitHubDotCom:
		nameWithOwner := strings.TrimPrefix(remoteName, "github.com/")
		repo, err = s.GithubDotComSource.GetRepo(ctx, nameWithOwner)
		if err != nil {
			if github.IsNotFound(err) {
				return &protocol.RepoLookupResult{
					ErrorNotFound: true,
				}, nil
			}
			// This check needs to come before isUnauthorized since GitHub returns 403 when
			// rate limit has been exceeded.
			if isTemporarilyUnavailable(err) {
				return &protocol.RepoLookupResult{
					ErrorTemporarilyUnavailable: true,
				}, nil
			}
			if isUnauthorized(err) {
				return &protocol.RepoLookupResult{
					ErrorUnauthorized: true,
				}, nil
			}
			return nil, err
		}

	case extsvc.GitLabDotCom:
		projectWithNamespace := strings.TrimPrefix(remoteName, "gitlab.com/")
		repo, err = s.GitLabDotComSource.GetRepo(ctx, projectWithNamespace)
		if err != nil {
			if gitlab.IsNotFound(err) {
				return &protocol.RepoLookupResult{
					ErrorNotFound: true,
				}, nil
			}
			if isUnauthorized(err) {
				return &protocol.RepoLookupResult{
					ErrorUnauthorized: true,
				}, nil
			}
			return nil, err
		}
	case extsvc.Maven:
		artifactPath := strings.TrimPrefix(remoteName, "maven/")
		repo, err = s.MavenSource.GetRepo(ctx, artifactPath)
		if err != nil {
			if errcode.IsNotFound(err) {
				return &protocol.RepoLookupResult{
					ErrorNotFound: true,
				}, nil
			}
			return nil, err
		}
	}

	if repo.Private {
		return &protocol.RepoLookupResult{
			ErrorNotFound: true,
		}, nil
	}

	err = s.Syncer.SyncRepo(ctx, s.Store, repo)
	if err != nil {
		return nil, err
	}

	repoInfo, err := newRepoInfo(repo)
	if err != nil {
		return nil, err
	}

	return &protocol.RepoLookupResult{
		Repo: repoInfo,
	}, nil
}

func (s *Server) handleEnqueueChangesetSync(w http.ResponseWriter, r *http.Request) {
	if s.ChangesetSyncRegistry == nil {
		log15.Warn("ChangesetSyncer is nil")
		respond(w, http.StatusForbidden, nil)
		return
	}

	var req protocol.ChangesetSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, err)
		return
	}
	if len(req.IDs) == 0 {
		respond(w, http.StatusBadRequest, errors.New("no ids provided"))
		return
	}
	err := s.ChangesetSyncRegistry.EnqueueChangesetSyncs(r.Context(), req.IDs)
	if err != nil {
		resp := protocol.ChangesetSyncResponse{Error: err.Error()}
		respond(w, http.StatusInternalServerError, resp)
		return
	}
	respond(w, http.StatusOK, nil)
}

func (s *Server) handleSchedulePermsSync(w http.ResponseWriter, r *http.Request) {
	if s.PermsSyncer == nil {
		respond(w, http.StatusForbidden, nil)
		return
	}

	var req protocol.PermsSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, err)
		return
	}
	if len(req.UserIDs) == 0 && len(req.RepoIDs) == 0 {
		respond(w, http.StatusBadRequest, errors.New("neither user and repo ids provided"))
		return
	}

	s.PermsSyncer.ScheduleUsers(r.Context(), req.UserIDs...)
	s.PermsSyncer.ScheduleRepos(r.Context(), req.RepoIDs...)

	respond(w, http.StatusOK, nil)
}

func newRepoInfo(r *types.Repo) (*protocol.RepoInfo, error) {
	urls := r.CloneURLs()
	if len(urls) == 0 {
		return nil, fmt.Errorf("no clone urls for repo id=%q name=%q", r.ID, r.Name)
	}

	info := protocol.RepoInfo{
		Name:         r.Name,
		Description:  r.Description,
		Fork:         r.Fork,
		Archived:     r.Archived,
		Private:      r.Private,
		VCS:          protocol.VCSInfo{URL: urls[0]},
		ExternalRepo: r.ExternalRepo,
	}

	typ, _ := extsvc.ParseServiceType(r.ExternalRepo.ServiceType)
	switch typ {
	case extsvc.TypeGitHub:
		ghrepo := r.Metadata.(*github.Repository)
		info.Links = &protocol.RepoLinks{
			Root:   ghrepo.URL,
			Tree:   pathAppend(ghrepo.URL, "/tree/{rev}/{path}"),
			Blob:   pathAppend(ghrepo.URL, "/blob/{rev}/{path}"),
			Commit: pathAppend(ghrepo.URL, "/commit/{commit}"),
		}
	case extsvc.TypeGitLab:
		proj := r.Metadata.(*gitlab.Project)
		info.Links = &protocol.RepoLinks{
			Root:   proj.WebURL,
			Tree:   pathAppend(proj.WebURL, "/tree/{rev}/{path}"),
			Blob:   pathAppend(proj.WebURL, "/blob/{rev}/{path}"),
			Commit: pathAppend(proj.WebURL, "/commit/{commit}"),
		}
	case extsvc.TypeBitbucketServer:
		repo := r.Metadata.(*bitbucketserver.Repo)
		if len(repo.Links.Self) == 0 {
			break
		}

		href := repo.Links.Self[0].Href
		root := strings.TrimSuffix(href, "/browse")
		info.Links = &protocol.RepoLinks{
			Root:   href,
			Tree:   pathAppend(root, "/browse/{path}?at={rev}"),
			Blob:   pathAppend(root, "/browse/{path}?at={rev}"),
			Commit: pathAppend(root, "/commits/{commit}"),
		}
	case extsvc.TypeAWSCodeCommit:
		repo := r.Metadata.(*awscodecommit.Repository)
		if repo.ARN == "" {
			break
		}

		splittedARN := strings.Split(strings.TrimPrefix(repo.ARN, "arn:aws:codecommit:"), ":")
		if len(splittedARN) == 0 {
			break
		}
		region := splittedARN[0]
		webURL := fmt.Sprintf("https://%s.console.aws.amazon.com/codesuite/codecommit/repositories/%s", region, repo.Name)
		info.Links = &protocol.RepoLinks{
			Root:   webURL + "/browse",
			Tree:   webURL + "/browse/{rev}/--/{path}",
			Blob:   webURL + "/browse/{rev}/--/{path}",
			Commit: webURL + "/commit/{commit}",
		}
	}

	return &info, nil
}

func pathAppend(base, p string) string {
	return strings.TrimRight(base, "/") + p
}

func isUnauthorized(err error) bool {
	code := github.HTTPErrorCode(err)
	if code == 0 {
		code = gitlab.HTTPErrorCode(err)
	}
	return code == http.StatusUnauthorized || code == http.StatusForbidden
}

func isTemporarilyUnavailable(err error) bool {
	return github.IsRateLimitExceeded(err)
}
