package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"

	"github.com/cockroachdb/errors"
	"github.com/gorilla/mux"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/globals"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/gitserver/protocol"
	"github.com/sourcegraph/sourcegraph/internal/jsonc"
	searchbackend "github.com/sourcegraph/sourcegraph/internal/search/backend"
	"github.com/sourcegraph/sourcegraph/internal/txemail"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
	"github.com/sourcegraph/sourcegraph/schema"
)

func serveReposGetByName(db dbutil.DB) func(w http.ResponseWriter, r *http.Request) error {
	return func(w http.ResponseWriter, r *http.Request) error {
		repoName := api.RepoName(mux.Vars(r)["RepoName"])
		repo, err := backend.NewRepos(db).GetByName(r.Context(), repoName)
		if err != nil {
			return err
		}
		data, err := json.Marshal(repo)
		if err != nil {
			return err
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return nil
	}
}

func servePhabricatorRepoCreate(db dbutil.DB) func(w http.ResponseWriter, r *http.Request) error {
	return func(w http.ResponseWriter, r *http.Request) error {
		var repo api.PhabricatorRepoCreateRequest
		err := json.NewDecoder(r.Body).Decode(&repo)
		if err != nil {
			return err
		}
		phabRepo, err := database.Phabricator(db).CreateOrUpdate(r.Context(), repo.Callsign, repo.RepoName, repo.URL)
		if err != nil {
			return err
		}
		data, err := json.Marshal(phabRepo)
		if err != nil {
			return err
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return nil
	}
}

// serveExternalServiceConfigs serves a JSON response that is an array of all
// external service configs that match the requested kind.
func serveExternalServiceConfigs(w http.ResponseWriter, r *http.Request) error {
	var req api.ExternalServiceConfigsRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return err
	}

	options := database.ExternalServicesListOptions{
		Kinds:   []string{req.Kind},
		AfterID: int64(req.AfterID),
	}
	if req.Limit > 0 {
		options.LimitOffset = &database.LimitOffset{
			Limit: req.Limit,
		}
	}

	services, err := database.GlobalExternalServices.List(r.Context(), options)
	if err != nil {
		return err
	}

	// Instead of returning an intermediate response type, we directly return
	// the array of configs (which are themselves JSON objects).
	// This makes it possible for the caller to directly unmarshal the response into
	// a slice of connection configurations for this external service kind.
	configs := make([]map[string]interface{}, 0, len(services))
	for _, service := range services {
		var config map[string]interface{}
		// Raw configs may have comments in them so we have to use a json parser
		// that supports comments in json.
		if err := jsonc.Unmarshal(service.Config, &config); err != nil {
			log15.Error(
				"ignoring external service config that has invalid json",
				"id", service.ID,
				"displayName", service.DisplayName,
				"config", service.Config,
				"err", err,
			)
			continue
		}
		configs = append(configs, config)
	}
	return json.NewEncoder(w).Encode(configs)
}

// serveExternalServicesList serves a JSON response that is an array of all external services
// of the given kind
func serveExternalServicesList(w http.ResponseWriter, r *http.Request) error {
	var req api.ExternalServicesListRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return err
	}

	if len(req.Kinds) == 0 {
		req.Kinds = append(req.Kinds, req.Kind)
	}

	options := database.ExternalServicesListOptions{
		Kinds:   []string{req.Kind},
		AfterID: int64(req.AfterID),
	}
	if req.Limit > 0 {
		options.LimitOffset = &database.LimitOffset{
			Limit: req.Limit,
		}
	}

	services, err := database.GlobalExternalServices.List(r.Context(), options)
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(services)
}

func serveConfiguration(w http.ResponseWriter, r *http.Request) error {
	raw, err := globals.ConfigurationServerFrontendOnly.Source.Read(r.Context())
	if err != nil {
		return err
	}
	err = json.NewEncoder(w).Encode(raw)
	if err != nil {
		return errors.Wrap(err, "Encode")
	}
	return nil
}

func repoRankFromConfig(siteConfig schema.SiteConfiguration, repoName string) float64 {
	val := 0.0
	if siteConfig.ExperimentalFeatures == nil || siteConfig.ExperimentalFeatures.Ranking == nil {
		return val
	}
	scores := siteConfig.ExperimentalFeatures.Ranking.RepoScores
	if len(scores) == 0 {
		return val
	}
	// try every "directory" in the repo name to assign it a value, so a repoName like
	// "github.com/sourcegraph/zoekt" will have "github.com", "github.com/sourcegraph",
	// and "github.com/sourcegraph/zoekt" tested.
	for i := 0; i < len(repoName); i++ {
		if repoName[i] == '/' {
			val += scores[repoName[:i]]
		}
	}
	val += scores[repoName]
	return val
}

// serveSearchConfiguration is _only_ used by the zoekt index server. Zoekt does
// not depend on frontend and therefore does not have access to `conf.Watch`.
// Additionally, it only cares about certain search specific settings so this
// search specific endpoint is used rather than serving the entire site settings
// from /.internal/configuration.
//
// This endpoint also supports batch requests to avoid managing concurrency in
// zoekt. On vertically scaled instances we have observed zoekt requesting
// this endpoint concurrently leading to socket starvation.
func serveSearchConfiguration(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	siteConfig := conf.Get().SiteConfiguration
	getRepoIndexOptions := func(repoName string) (*searchbackend.RepoIndexOptions, error) {
		repo, err := database.GlobalRepos.GetByName(ctx, api.RepoName(repoName))
		if err != nil {
			return nil, err
		}

		getVersion := func(branch string) (string, error) {
			// Do not to trigger a repo-updater lookup since this is a batch job.
			commitID, err := git.ResolveRevision(ctx, repo.Name, branch, git.ResolveRevisionOptions{})
			if err != nil && errcode.HTTP(err) == http.StatusNotFound {
				// GetIndexOptions wants an empty rev for a missing rev or empty
				// repo.
				return "", nil
			}
			return string(commitID), err
		}

		priority := float64(repo.Stars) + repoRankFromConfig(siteConfig, repoName)

		return &searchbackend.RepoIndexOptions{
			RepoID:     int32(repo.ID),
			Public:     !repo.Private,
			Priority:   priority,
			GetVersion: getVersion,
		}, nil
	}

	sc := database.SearchContexts(dbconn.Global)
	getSearchContextRevisions := func(repoID int32) ([]string, error) {
		return sc.GetAllRevisionsForRepo(ctx, repoID)
	}

	if err := r.ParseForm(); err != nil {
		return err
	}

	b := searchbackend.GetIndexOptions(&siteConfig, getRepoIndexOptions, getSearchContextRevisions, r.Form["repo"]...)
	_, _ = w.Write(b)
	return nil
}

type reposListServer struct {
	// SourcegraphDotComMode is true if this instance of Sourcegraph is http://sourcegraph.com
	SourcegraphDotComMode bool

	// Repos is the subset of backend.Repos methods we use. Declared as an
	// interface for testing.
	Repos interface {
		// ListIndexable returns the repositories to index on Sourcegraph.com
		ListIndexable(context.Context) ([]types.RepoName, error)
		// List returns a list of repositories
		List(context.Context, database.ReposListOptions) ([]*types.Repo, error)
	}

	// Indexers is the subset of searchbackend.Indexers methods we
	// use. reposListServer is used by indexed-search to get the list of
	// repositories to index. These methods are used to return the correct
	// subset for horizontal indexed search. Declared as an interface for
	// testing.
	Indexers interface {
		// ReposSubset returns the subset of repoNames that hostname should
		// index.
		ReposSubset(ctx context.Context, hostname string, indexed map[string]struct{}, repoNames []string) ([]string, error)
		// Enabled is true if horizontal indexed search is enabled.
		Enabled() bool
	}
}

// serveIndex is used by zoekt to get the list of repositories for it to
// index.
func (h *reposListServer) serveIndex(w http.ResponseWriter, r *http.Request) error {
	var opt struct {
		// Hostname is used to determine the subset of repos to return
		Hostname string
		// Indexed is the repository names of indexed repos by Hostname.
		Indexed []string
	}
	if err := json.NewDecoder(r.Body).Decode(&opt); err != nil {
		return err
	}

	var names []string
	if h.SourcegraphDotComMode {
		res, err := h.Repos.ListIndexable(r.Context())
		if err != nil {
			return errors.Wrap(err, "listing repos")
		}
		names = make([]string, len(res))
		for i, r := range res {
			names[i] = string(r.Name)
		}
	} else {
		trueP := true
		res, err := h.Repos.List(r.Context(), database.ReposListOptions{Index: &trueP})
		if err != nil {
			return errors.Wrap(err, "listing repos")
		}
		names = make([]string, len(res))
		for i, r := range res {
			names[i] = string(r.Name)
		}
	}

	if h.Indexers.Enabled() {
		indexed := make(map[string]struct{}, len(opt.Indexed))
		for _, name := range opt.Indexed {
			indexed[name] = struct{}{}
		}

		var err error
		names, err = h.Indexers.ReposSubset(r.Context(), opt.Hostname, indexed, names)
		if err != nil {
			return err
		}
	}

	data := struct {
		RepoNames []string
	}{
		RepoNames: names,
	}

	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(&data)
}

func serveReposListEnabled(w http.ResponseWriter, r *http.Request) error {
	names, err := database.GlobalRepos.ListEnabledNames(r.Context())
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(names)
}

func serveSavedQueriesListAll(db dbutil.DB) func(w http.ResponseWriter, r *http.Request) error {
	return func(w http.ResponseWriter, r *http.Request) error {
		// List settings for all users, orgs, etc.
		settings, err := database.SavedSearches(db).ListAll(r.Context())
		if err != nil {
			return errors.Wrap(err, "database.SavedSearches.ListAll")
		}

		queries := make([]api.SavedQuerySpecAndConfig, 0, len(settings))
		for _, s := range settings {
			var spec api.SavedQueryIDSpec
			if s.Config.UserID != nil {
				spec = api.SavedQueryIDSpec{Subject: api.SettingsSubject{User: s.Config.UserID}, Key: s.Config.Key}
			} else if s.Config.OrgID != nil {
				spec = api.SavedQueryIDSpec{Subject: api.SettingsSubject{Org: s.Config.OrgID}, Key: s.Config.Key}
			}

			queries = append(queries, api.SavedQuerySpecAndConfig{
				Spec:   spec,
				Config: s.Config,
			})
		}

		if err := json.NewEncoder(w).Encode(queries); err != nil {
			return errors.Wrap(err, "Encode")
		}

		return nil
	}
}

func serveSavedQueriesGetInfo(db dbutil.DB) func(w http.ResponseWriter, r *http.Request) error {
	return func(w http.ResponseWriter, r *http.Request) error {
		var query string
		err := json.NewDecoder(r.Body).Decode(&query)
		if err != nil {
			return errors.Wrap(err, "Decode")
		}
		info, err := database.QueryRunnerState(db).Get(r.Context(), query)
		if err != nil {
			return errors.Wrap(err, "SavedQueries.Get")
		}
		if err := json.NewEncoder(w).Encode(info); err != nil {
			return errors.Wrap(err, "Encode")
		}
		return nil
	}
}

func serveSavedQueriesSetInfo(db dbutil.DB) func(w http.ResponseWriter, r *http.Request) error {
	return func(w http.ResponseWriter, r *http.Request) error {
		var info *api.SavedQueryInfo
		err := json.NewDecoder(r.Body).Decode(&info)
		if err != nil {
			return errors.Wrap(err, "Decode")
		}
		err = database.QueryRunnerState(db).Set(r.Context(), &database.SavedQueryInfo{
			Query:        info.Query,
			LastExecuted: info.LastExecuted,
			LatestResult: info.LatestResult,
			ExecDuration: info.ExecDuration,
		})
		if err != nil {
			return errors.Wrap(err, "SavedQueries.Set")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
		return nil
	}
}

func serveSavedQueriesDeleteInfo(db dbutil.DB) func(w http.ResponseWriter, r *http.Request) error {
	return func(w http.ResponseWriter, r *http.Request) error {
		var query string
		err := json.NewDecoder(r.Body).Decode(&query)
		if err != nil {
			return errors.Wrap(err, "Decode")
		}
		err = database.QueryRunnerState(db).Delete(r.Context(), query)
		if err != nil {
			return errors.Wrap(err, "SavedQueries.Delete")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
		return nil
	}
}

func serveSettingsGetForSubject(db dbutil.DB) func(w http.ResponseWriter, r *http.Request) error {
	return func(w http.ResponseWriter, r *http.Request) error {
		var subject api.SettingsSubject
		if err := json.NewDecoder(r.Body).Decode(&subject); err != nil {
			return errors.Wrap(err, "Decode")
		}
		settings, err := database.Settings(db).GetLatest(r.Context(), subject)
		if err != nil {
			return errors.Wrap(err, "Settings.GetLatest")
		}
		if err := json.NewEncoder(w).Encode(settings); err != nil {
			return errors.Wrap(err, "Encode")
		}
		return nil
	}
}

func serveOrgsListUsers(db dbutil.DB) func(w http.ResponseWriter, r *http.Request) error {
	return func(w http.ResponseWriter, r *http.Request) error {
		var orgID int32
		err := json.NewDecoder(r.Body).Decode(&orgID)
		if err != nil {
			return errors.Wrap(err, "Decode")
		}
		orgMembers, err := database.OrgMembers(db).GetByOrgID(r.Context(), orgID)
		if err != nil {
			return errors.Wrap(err, "OrgMembers.GetByOrgID")
		}
		users := make([]int32, 0, len(orgMembers))
		for _, member := range orgMembers {
			users = append(users, member.UserID)
		}
		if err := json.NewEncoder(w).Encode(users); err != nil {
			return errors.Wrap(err, "Encode")
		}
		return nil
	}
}

func serveOrgsGetByName(db dbutil.DB) func(w http.ResponseWriter, r *http.Request) error {
	return func(w http.ResponseWriter, r *http.Request) error {
		var orgName string
		err := json.NewDecoder(r.Body).Decode(&orgName)
		if err != nil {
			return errors.Wrap(err, "Decode")
		}
		org, err := database.Orgs(db).GetByName(r.Context(), orgName)
		if err != nil {
			return errors.Wrap(err, "Orgs.GetByName")
		}
		if err := json.NewEncoder(w).Encode(org.ID); err != nil {
			return errors.Wrap(err, "Encode")
		}
		return nil
	}
}

func serveUsersGetByUsername(w http.ResponseWriter, r *http.Request) error {
	var username string
	err := json.NewDecoder(r.Body).Decode(&username)
	if err != nil {
		return errors.Wrap(err, "Decode")
	}
	user, err := database.GlobalUsers.GetByUsername(r.Context(), username)
	if err != nil {
		return errors.Wrap(err, "Users.GetByUsername")
	}
	if err := json.NewEncoder(w).Encode(user.ID); err != nil {
		return errors.Wrap(err, "Encode")
	}
	return nil
}

func serveUserEmailsGetEmail(w http.ResponseWriter, r *http.Request) error {
	var userID int32
	err := json.NewDecoder(r.Body).Decode(&userID)
	if err != nil {
		return errors.Wrap(err, "Decode")
	}
	email, _, err := database.GlobalUserEmails.GetPrimaryEmail(r.Context(), userID)
	if err != nil {
		return errors.Wrap(err, "UserEmails.GetEmail")
	}
	if err := json.NewEncoder(w).Encode(email); err != nil {
		return errors.Wrap(err, "Encode")
	}
	return nil
}

func serveExternalURL(w http.ResponseWriter, r *http.Request) error {
	if err := json.NewEncoder(w).Encode(globals.ExternalURL().String()); err != nil {
		return errors.Wrap(err, "Encode")
	}
	return nil
}

func serveCanSendEmail(w http.ResponseWriter, r *http.Request) error {
	if err := json.NewEncoder(w).Encode(conf.CanSendEmail()); err != nil {
		return errors.Wrap(err, "Encode")
	}
	return nil
}

func serveSendEmail(w http.ResponseWriter, r *http.Request) error {
	var msg txemail.Message
	err := json.NewDecoder(r.Body).Decode(&msg)
	if err != nil {
		return err
	}
	return txemail.Send(r.Context(), msg)
}

func serveGitResolveRevision(w http.ResponseWriter, r *http.Request) error {
	// used by zoekt-sourcegraph-mirror
	vars := mux.Vars(r)
	name := api.RepoName(vars["RepoName"])
	spec := vars["Spec"]

	// Do not to trigger a repo-updater lookup since this is a batch job.
	commitID, err := git.ResolveRevision(r.Context(), name, spec, git.ResolveRevisionOptions{})
	if err != nil {
		return err
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(commitID))
	return nil
}

func serveGitTar(w http.ResponseWriter, r *http.Request) error {
	// used by zoekt-sourcegraph-mirror
	vars := mux.Vars(r)
	name := vars["RepoName"]
	spec := vars["Commit"]

	// Ensure commit exists. Do not want to trigger a repo-updater lookup since this is a batch job.
	repo := api.RepoName(name)
	commit, err := git.ResolveRevision(r.Context(), repo, spec, git.ResolveRevisionOptions{})
	if err != nil {
		return err
	}

	opts := gitserver.ArchiveOptions{
		Treeish: string(commit),
		Format:  "tar",
	}

	location := gitserver.DefaultClient.ArchiveURL(repo, opts)

	w.Header().Set("Location", location.String())
	w.WriteHeader(http.StatusFound)

	return nil
}

func serveGitExec(w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()
	req := protocol.ExecRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.Wrap(err, "Decode")
	}

	vars := mux.Vars(r)
	repoID, err := strconv.ParseInt(vars["RepoID"], 10, 64)
	if err != nil {
		http.Error(w, "illegal repository id: "+err.Error(), http.StatusBadRequest)
		return nil
	}

	repo, err := database.GlobalRepos.Get(r.Context(), api.RepoID(repoID))
	if err != nil {
		return err
	}

	// Set repo name in gitserver request payload
	req.Repo = repo.Name

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		return errors.Wrap(err, "Encode")
	}

	// Find the correct shard to query
	addr := gitserver.DefaultClient.AddrForRepo(repo.Name)

	director := func(req *http.Request) {
		req.URL.Scheme = "http"
		req.URL.Host = addr
		req.URL.Path = "/exec"
		req.Body = io.NopCloser(bytes.NewReader(buf.Bytes()))
		req.ContentLength = int64(buf.Len())
	}

	gitserver.DefaultReverseProxy.ServeHTTP(repo.Name, "POST", "exec", director, w, r)
	return nil
}

// gitServiceHandler are handlers which redirect git clone requests to the
// gitserver for the repo.
type gitServiceHandler struct {
	Gitserver interface {
		AddrForRepo(api.RepoName) string
	}
}

func (s *gitServiceHandler) serveInfoRefs(w http.ResponseWriter, r *http.Request) {
	s.redirectToGitServer(w, r, "/info/refs")
}

func (s *gitServiceHandler) serveGitUploadPack(w http.ResponseWriter, r *http.Request) {
	s.redirectToGitServer(w, r, "/git-upload-pack")
}

func (s *gitServiceHandler) redirectToGitServer(w http.ResponseWriter, r *http.Request, gitPath string) {
	repo := mux.Vars(r)["RepoName"]

	u := &url.URL{
		Scheme:   "http",
		Host:     s.Gitserver.AddrForRepo(api.RepoName(repo)),
		Path:     path.Join("/git", repo, gitPath),
		RawQuery: r.URL.RawQuery,
	}

	http.Redirect(w, r, u.String(), http.StatusTemporaryRedirect)
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "could not parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	for _, service := range r.Form["service"] {
		switch service {
		case "gitserver":
			if err := gitserver.DefaultClient.WaitForGitServers(r.Context()); err != nil {
				http.Error(w, "wait for gitservers failed: "+err.Error(), http.StatusBadGateway)
				return
			}

		default:
			http.Error(w, "unknown service: "+service, http.StatusBadRequest)
			return
		}
	}

	_, _ = w.Write([]byte("pong"))
}
