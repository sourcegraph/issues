// Package repoupdater implements the repo-updater service HTTP handler.
package repoupdater

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/inconshreveable/log15"
	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/awscodecommit"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/bitbucketserver"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/github"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitlab"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater/protocol"
	"github.com/sourcegraph/sourcegraph/internal/trace"
)

// Server is a repoupdater server.
type Server struct {
	repos.Store
	*repos.Syncer
	SourcegraphDotComMode bool
	GithubDotComSource    interface {
		GetRepo(ctx context.Context, nameWithOwner string) (*repos.Repo, error)
	}
	GitLabDotComSource interface {
		GetRepo(ctx context.Context, projectWithNamespace string) (*repos.Repo, error)
	}
	Scheduler interface {
		UpdateOnce(id api.RepoID, name api.RepoName, url string)
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
	mux.HandleFunc("/repo-external-services", s.handleRepoExternalServices)
	mux.HandleFunc("/enqueue-repo-update", s.handleEnqueueRepoUpdate)
	mux.HandleFunc("/exclude-repo", s.handleExcludeRepo)
	mux.HandleFunc("/sync-external-service", s.handleExternalServiceSync)
	mux.HandleFunc("/status-messages", s.handleStatusMessages)
	mux.HandleFunc("/enqueue-changeset-sync", s.handleEnqueueChangesetSync)
	mux.HandleFunc("/schedule-perms-sync", s.handleSchedulePermsSync)
	return mux
}

func (s *Server) handleRepoExternalServices(w http.ResponseWriter, r *http.Request) {
	var req protocol.RepoExternalServicesRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusInternalServerError, err)
		return
	}

	rs, err := s.Store.ListRepos(r.Context(), repos.StoreListReposArgs{
		IDs: []api.RepoID{req.ID},
	})
	if err != nil {
		respond(w, http.StatusInternalServerError, err)
		return
	}

	if len(rs) == 0 {
		respond(w, http.StatusNotFound, errors.Errorf("repository with ID %v does not exist", req.ID))
		return
	}

	var resp protocol.RepoExternalServicesResponse

	svcIDs := rs[0].ExternalServiceIDs()
	if len(svcIDs) == 0 {
		respond(w, http.StatusOK, resp)
		return
	}

	args := repos.StoreListExternalServicesArgs{
		IDs: svcIDs,
	}

	es, err := s.Store.ListExternalServices(r.Context(), args)
	if err != nil {
		respond(w, http.StatusInternalServerError, err)
		return
	}

	resp.ExternalServices = newExternalServices(es...)

	respond(w, http.StatusOK, resp)
}

func (s *Server) handleExcludeRepo(w http.ResponseWriter, r *http.Request) {
	var req protocol.ExcludeRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusInternalServerError, err)
		return
	}

	rs, err := s.Store.ListRepos(r.Context(), repos.StoreListReposArgs{
		IDs: []api.RepoID{req.ID},
	})
	if err != nil {
		respond(w, http.StatusInternalServerError, err)
		return
	}

	var resp protocol.ExcludeRepoResponse
	if len(rs) == 0 {
		log15.Warn("exclude-repo: repo not found. skipping", "repo.id", req.ID)
		respond(w, http.StatusOK, resp)
		return
	}

	args := repos.StoreListExternalServicesArgs{
		Kinds: repos.Repos(rs).Kinds(),
	}

	es, err := s.Store.ListExternalServices(r.Context(), args)
	if err != nil {
		respond(w, http.StatusInternalServerError, err)
		return
	}

	for _, e := range es {
		if err := e.Exclude(rs...); err != nil {
			respond(w, http.StatusInternalServerError, err)
			return
		}
	}

	err = s.Store.UpsertExternalServices(r.Context(), es...)
	if err != nil {
		respond(w, http.StatusInternalServerError, err)
		return
	}

	resp.ExternalServices = newExternalServices(es...)

	respond(w, http.StatusOK, resp)
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

func newExternalServices(es ...*repos.ExternalService) []api.ExternalService {
	svcs := make([]api.ExternalService, 0, len(es))

	for _, e := range es {
		svc := api.ExternalService{
			ID:          e.ID,
			Kind:        e.Kind,
			DisplayName: e.DisplayName,
			Config:      e.Config,
			CreatedAt:   e.CreatedAt,
			UpdatedAt:   e.UpdatedAt,
		}

		if e.IsDeleted() {
			svc.DeletedAt = &e.DeletedAt
		}

		svcs = append(svcs, svc)
	}

	return svcs
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

	rs, err := s.Store.ListRepos(ctx, repos.StoreListReposArgs{Names: []string{string(req.Repo)}})
	if err != nil {
		return nil, http.StatusInternalServerError, errors.Wrap(err, "store.list-repos")
	}

	if len(rs) != 1 {
		return nil, http.StatusNotFound, errors.Errorf("repo %q not found in store", req.Repo)
	}

	repo := rs[0]
	if req.URL == "" {
		if urls := repo.CloneURLs(); len(urls) > 0 {
			req.URL = urls[0]
		}
	}
	s.Scheduler.UpdateOnce(repo.ID, req.Repo, req.URL)

	return &protocol.RepoUpdateResponse{
		ID:   repo.ID,
		Name: repo.Name,
		URL:  req.URL,
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

	s.Syncer.TriggerEnqueueSyncJobs()

	err := externalServiceValidate(ctx, &req)
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

func externalServiceValidate(ctx context.Context, req *protocol.ExternalServiceSyncRequest) error {
	if req.ExternalService.DeletedAt != nil {
		// We don't need to check deleted services.
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	src, err := repos.NewSource(&repos.ExternalService{
		ID:          req.ExternalService.ID,
		Kind:        req.ExternalService.Kind,
		DisplayName: req.ExternalService.DisplayName,
		Config:      req.ExternalService.Config,
	}, httpcli.NewExternalHTTPClientFactory())
	if err != nil {
		return err
	}

	results := make(chan repos.SourceResult)

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

	list, err := s.Store.ListRepos(ctx, repos.StoreListReposArgs{
		Names: []string{string(args.Repo)},
	})
	if err != nil {
		return nil, err
	}
	var repo *repos.Repo
	if len(list) > 0 {
		repo = list[0]
	}

	// If we are sourcegraph.com we don't run a global Sync since there are
	// too many repos. Instead we use an incremental approach where we check
	// for changes everytime a user browses a repo. RepoLookup is the signal
	// we rely on to check metadata.
	codehost := extsvc.CodeHostOf(args.Repo, extsvc.PublicCodeHosts...)
	if s.SourcegraphDotComMode && codehost != nil {
		// TODO a queue with single flighting to speak to remote for args.Repo?
		if repo != nil {
			// We have (potentially stale) data we can return to the user right
			// now. Do that rather than blocking.
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
				defer cancel()
				repoResult, err := s.remoteRepoSync(ctx, codehost, string(args.Repo))
				if err != nil {
					log15.Error("async remoteRepoSync failed", "repo", args.Repo, "error", err)
				}

				// Since we don't support private repositories on Cloud,
				// we can safely assume that when a repository stored in the database is not accessible anymore,
				// no other external service should have access to it, we can then remove it.
				if repoResult.ErrorNotFound || repoResult.ErrorUnauthorized {
					err = s.Store.UpsertRepos(ctx, repo.With(func(r *repos.Repo) {
						r.DeletedAt = s.Now()
					}))
					if err != nil {
						log15.Error("failed to delete inaccessible repo", "repo", args.Repo, "error", err)
					}
				}
			}()
		} else {
			// Try and find this repo on the remote host. Block on the remote
			// request.
			return s.remoteRepoSync(ctx, codehost, string(args.Repo))
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
	var repo *repos.Repo
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
			if isUnauthorized(err) {
				return &protocol.RepoLookupResult{
					ErrorUnauthorized: true,
				}, nil
			}
			if isTemporarilyUnavailable(err) {
				return &protocol.RepoLookupResult{
					ErrorTemporarilyUnavailable: true,
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

func (s *Server) handleStatusMessages(w http.ResponseWriter, r *http.Request) {
	resp := protocol.StatusMessagesResponse{
		Messages: []protocol.StatusMessage{},
	}

	notCloned, err := s.Store.CountNotClonedRepos(r.Context())
	if err != nil {
		respond(w, http.StatusInternalServerError, err)
		return
	}

	if notCloned != 0 {
		resp.Messages = append(resp.Messages, protocol.StatusMessage{
			Cloning: &protocol.CloningProgress{
				Message: fmt.Sprintf("%d repositories enqueued for cloning...", notCloned),
			},
		})
	}

	for _, e := range s.Syncer.SyncErrors() {
		if multiErr, ok := errors.Cause(e).(*multierror.Error); ok {
			for _, e := range multiErr.Errors {
				if sourceErr, ok := e.(*repos.SourceError); ok {
					resp.Messages = append(resp.Messages, protocol.StatusMessage{
						ExternalServiceSyncError: &protocol.ExternalServiceSyncError{
							Message:           sourceErr.Err.Error(),
							ExternalServiceId: sourceErr.ExtSvc.ID,
						},
					})
				} else {
					resp.Messages = append(resp.Messages, protocol.StatusMessage{
						SyncError: &protocol.SyncError{Message: e.Error()},
					})
				}
			}
		} else {
			resp.Messages = append(resp.Messages, protocol.StatusMessage{
				SyncError: &protocol.SyncError{Message: e.Error()},
			})
		}
	}

	messagesSummary := func() string {
		cappedMsg := resp.Messages
		if len(cappedMsg) > 10 {
			cappedMsg = cappedMsg[:10]
		}

		jMsg, err := json.MarshalIndent(cappedMsg, "", " ")
		if err != nil {
			return "error summarizing messages for logging"
		}
		return string(jMsg)
	}

	log15.Debug("TRACE handleStatusMessages", "messages", log15.Lazy{Fn: messagesSummary})

	respond(w, http.StatusOK, resp)
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

func newRepoInfo(r *repos.Repo) (*protocol.RepoInfo, error) {
	urls := r.CloneURLs()
	if len(urls) == 0 {
		return nil, fmt.Errorf("no clone urls for repo id=%q name=%q", r.ID, r.Name)
	}

	info := protocol.RepoInfo{
		Name:         api.RepoName(r.Name),
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
