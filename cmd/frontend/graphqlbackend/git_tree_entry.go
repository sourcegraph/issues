package graphqlbackend

import (
	"context"
	"fmt"
	neturl "net/url"
	"os"
	"path"
	"sync"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/globals"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/externallink"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf/reposource"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/highlight"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
	"github.com/sourcegraph/sourcegraph/schema"
)

var metricLabels = []string{"origin"}
var codeIntelRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "src_lsif_requests",
	Help: "Counts LSIF requests.",
}, metricLabels)

func init() {
	prometheus.MustRegister(codeIntelRequests)
}

// GitTreeEntryResolver resolves an entry in a Git tree in a repository. The entry can be any Git
// object type that is valid in a tree.
type GitTreeEntryResolver struct {
	commit *GitCommitResolver

	contentOnce sync.Once
	content     []byte
	contentErr  error

	// stat is this tree entry's file info. Its Name method must return the full path relative to
	// the root, not the basename.
	stat os.FileInfo

	isRecursive   bool  // whether entries is populated recursively (otherwise just current level of hierarchy)
	isSingleChild *bool // whether this is the single entry in its parent. Only set by the (&GitTreeEntryResolver) entries.
}

func NewGitTreeEntryResolver(commit *GitCommitResolver, stat os.FileInfo) *GitTreeEntryResolver {
	return &GitTreeEntryResolver{commit: commit, stat: stat}
}

func (r *GitTreeEntryResolver) Path() string { return r.stat.Name() }
func (r *GitTreeEntryResolver) Name() string { return path.Base(r.stat.Name()) }

func (r *GitTreeEntryResolver) ToGitTree() (*GitTreeEntryResolver, bool) { return r, true }
func (r *GitTreeEntryResolver) ToGitBlob() (*GitTreeEntryResolver, bool) { return r, true }

func (r *GitTreeEntryResolver) ToVirtualFile() (*virtualFileResolver, bool) { return nil, false }

func (r *GitTreeEntryResolver) ByteSize(ctx context.Context) (int32, error) {
	content, err := r.Content(ctx)
	if err != nil {
		return 0, err
	}
	return int32(len([]byte(content))), nil
}

func (r *GitTreeEntryResolver) Content(ctx context.Context) (string, error) {
	r.contentOnce.Do(func() {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		cachedRepo, err := backend.CachedGitRepo(ctx, r.commit.repoResolver.repo)
		if err != nil {
			r.contentErr = err
		}

		r.content, r.contentErr = git.ReadFile(ctx, *cachedRepo, api.CommitID(r.commit.OID()), r.Path(), 0)
	})

	return string(r.content), r.contentErr
}

func (r *GitTreeEntryResolver) RichHTML(ctx context.Context) (string, error) {
	content, err := r.Content(ctx)
	if err != nil {
		return "", err
	}
	return richHTML(content, path.Ext(r.Path()))
}

func (r *GitTreeEntryResolver) Binary(ctx context.Context) (bool, error) {
	content, err := r.Content(ctx)
	if err != nil {
		return false, err
	}
	return highlight.IsBinary([]byte(content)), nil
}

func (r *GitTreeEntryResolver) Highlight(ctx context.Context, args *HighlightArgs) (*highlightedFileResolver, error) {
	content, err := r.Content(ctx)
	if err != nil {
		return nil, err
	}
	return highlightContent(ctx, args, content, r.Path(), highlight.Metadata{
		RepoName: string(r.commit.repoResolver.repo.Name),
		Revision: string(r.commit.oid),
	})
}

func (r *GitTreeEntryResolver) Commit() *GitCommitResolver { return r.commit }

func (r *GitTreeEntryResolver) Repository() *RepositoryResolver { return r.commit.repoResolver }

func (r *GitTreeEntryResolver) IsRecursive() bool { return r.isRecursive }

func (r *GitTreeEntryResolver) URL(ctx context.Context) (string, error) {
	if submodule := r.Submodule(); submodule != nil {
		repoName, err := cloneURLToRepoName(ctx, submodule.URL())
		if err != nil {
			log15.Error("Failed to resolve submodule repository name from clone URL", "cloneURL", submodule.URL(), "err", err)
			return "", nil
		}
		return "/" + repoName + "@" + submodule.Commit(), nil
	}
	url, err := r.commit.repoRevURL()
	if err != nil {
		return "", err
	}
	return r.urlPath(url)
}

func (r *GitTreeEntryResolver) CanonicalURL() (string, error) {
	url, err := r.commit.canonicalRepoRevURL()
	if err != nil {
		return "", err
	}
	return r.urlPath(url)
}

func (r *GitTreeEntryResolver) urlPath(prefix string) (string, error) {
	if r.IsRoot() {
		return prefix, nil
	}

	u, err := neturl.Parse(prefix)
	if err != nil {
		return "", err
	}

	typ := "blob"
	if r.IsDirectory() {
		typ = "tree"
	}

	u.Path = path.Join(u.Path, "-", typ, r.Path())
	return u.String(), nil
}

func (r *GitTreeEntryResolver) IsDirectory() bool { return r.stat.Mode().IsDir() }

func (r *GitTreeEntryResolver) ExternalURLs(ctx context.Context) ([]*externallink.Resolver, error) {
	return externallink.FileOrDir(ctx, r.commit.repoResolver.repo, r.commit.inputRevOrImmutableRev(), r.Path(), r.stat.Mode().IsDir())
}

func (r *GitTreeEntryResolver) RawZipArchiveURL() string {
	return globals.ExternalURL().ResolveReference(&neturl.URL{
		Path:     path.Join(r.Repository().URL(), "-/raw/", r.Path()),
		RawQuery: "format=zip",
	}).String()
}

func (r *GitTreeEntryResolver) Submodule() *gitSubmoduleResolver {
	if submoduleInfo, ok := r.stat.Sys().(git.Submodule); ok {
		return &gitSubmoduleResolver{submodule: submoduleInfo}
	}
	return nil
}

func cloneURLToRepoName(ctx context.Context, cloneURL string) (string, error) {
	repoName, err := reposourceCloneURLToRepoName(ctx, cloneURL)
	if err != nil {
		return "", err
	}
	if repoName == "" {
		return "", fmt.Errorf("No matching code host found for %s", cloneURL)
	}
	return string(repoName), nil
}

// reposourceCloneURLToRepoName maps a Git clone URL (format documented here:
// https://git-scm.com/docs/git-clone#_git_urls_a_id_urls_a) to the corresponding repo name if there
// exists a code host configuration that matches the clone URL. Implicitly, it includes a code host
// configuration for github.com, even if one is not explicitly specified. Returns the empty string and nil
// error if a matching code host could not be found. This function does not actually check the code
// host to see if the repository actually exists.
func reposourceCloneURLToRepoName(ctx context.Context, cloneURL string) (repoName api.RepoName, err error) {
	if repoName := reposource.CustomCloneURLToRepoName(cloneURL); repoName != "" {
		return repoName, nil
	}

	opt := db.ExternalServicesListOptions{
		Kinds: []string{
			extsvc.KindGitHub,
			extsvc.KindGitLab,
			extsvc.KindBitbucketServer,
			extsvc.KindAWSCodeCommit,
			extsvc.KindGitolite,
		},
		LimitOffset: &db.LimitOffset{
			Limit: 500, // The number is randomly chosen
		},
	}
	for {
		svcs, err := db.ExternalServices.List(ctx, opt)
		if err != nil {
			return "", errors.Wrap(err, "list")
		}
		if len(svcs) == 0 {
			break // No more results, exiting
		}
		opt.AfterID = svcs[len(svcs)-1].ID // Advance the cursor

		for _, svc := range svcs {
			cfg, err := extsvc.ParseConfig(svc.Kind, svc.Config)
			if err != nil {
				return "", errors.Wrap(err, "parse config")
			}

			var rs reposource.RepoSource
			switch c := cfg.(type) {
			case *schema.GitHubConnection:
				rs = reposource.GitHub{GitHubConnection: c}
			case *schema.GitLabConnection:
				rs = reposource.GitLab{GitLabConnection: c}
			case *schema.BitbucketServerConnection:
				rs = reposource.BitbucketServer{BitbucketServerConnection: c}
			case *schema.AWSCodeCommitConnection:
				rs = reposource.AWS{AWSCodeCommitConnection: c}
			case *schema.GitoliteConnection:
				rs = reposource.Gitolite{GitoliteConnection: c}
			default:
				return "", errors.Errorf("unexpected connection type: %T", cfg)
			}

			repoName, err := rs.CloneURLToRepoName(cloneURL)
			if err != nil {
				return "", err
			}
			if repoName != "" {
				return repoName, nil
			}
		}

		if len(svcs) < opt.Limit {
			break // Less results than limit means we've reached end
		}
	}

	// Fallback for github.com
	rs := reposource.GitHub{
		GitHubConnection: &schema.GitHubConnection{
			Url: "https://github.com",
		},
	}
	return rs.CloneURLToRepoName(cloneURL)
}

func CreateFileInfo(path string, isDir bool) os.FileInfo {
	return fileInfo{path: path, isDir: isDir}
}

func (r *GitTreeEntryResolver) IsSingleChild(ctx context.Context, args *gitTreeEntryConnectionArgs) (bool, error) {
	if !r.IsDirectory() {
		return false, nil
	}
	if r.isSingleChild != nil {
		return *r.isSingleChild, nil
	}
	cachedRepo, err := backend.CachedGitRepo(ctx, r.commit.repoResolver.repo)
	if err != nil {
		return false, err
	}
	entries, err := git.ReadDir(ctx, *cachedRepo, api.CommitID(r.commit.OID()), path.Dir(r.Path()), false)
	if err != nil {
		return false, err
	}
	return len(entries) == 1, nil
}

func (r *GitTreeEntryResolver) LSIF(ctx context.Context, args *struct{ ToolName *string }) (GitBlobLSIFDataResolver, error) {
	codeIntelRequests.WithLabelValues(trace.RequestOrigin(ctx)).Inc()

	var toolName string
	if args.ToolName != nil {
		toolName = *args.ToolName
	}

	return EnterpriseResolvers.codeIntelResolver.GitBlobLSIFData(ctx, &GitBlobLSIFDataArgs{
		Repo:      r.Repository().Type(),
		Commit:    api.CommitID(r.Commit().OID()),
		Path:      r.Path(),
		ExactPath: !r.stat.IsDir(),
		ToolName:  toolName,
	})
}

type fileInfo struct {
	path  string
	size  int64
	isDir bool
}

func (f fileInfo) Name() string { return f.path }
func (f fileInfo) Size() int64  { return f.size }
func (f fileInfo) IsDir() bool  { return f.isDir }
func (f fileInfo) Mode() os.FileMode {
	if f.IsDir() {
		return os.ModeDir
	}
	return 0
}
func (f fileInfo) ModTime() time.Time { return time.Now() }
func (f fileInfo) Sys() interface{}   { return interface{}(nil) }
