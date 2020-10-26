package graphqlbackend

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/externallink"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/graphqlutil"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
)

func gitCommitByID(ctx context.Context, id graphql.ID) (*GitCommitResolver, error) {
	repoID, commitID, err := unmarshalGitCommitID(id)
	if err != nil {
		return nil, err
	}
	repo, err := repositoryByID(ctx, repoID)
	if err != nil {
		return nil, err
	}
	return repo.Commit(ctx, &RepositoryCommitArgs{Rev: string(commitID)})
}

type GitCommitResolver struct {
	repoResolver *RepositoryResolver

	// inputRev is the Git revspec that the user originally requested that resolved to this Git commit. It is used
	// to avoid redirecting a user browsing a revision "mybranch" to the absolute commit ID as they follow links in the UI.
	inputRev *string

	// fetch + serve sourcegraph stored user information
	includeUserInfo bool

	// oid MUST be specified and a 40-character Git SHA.
	oid GitObjectID

	author    signatureResolver
	committer *signatureResolver
	message   string
	parents   []api.CommitID

	// once ensures that fetching git commit information occurs once
	once sync.Once
	err  error
}

func toGitCommitResolver(repo *RepositoryResolver, commit *git.Commit) *GitCommitResolver {
	res := &GitCommitResolver{
		repoResolver:    repo,
		includeUserInfo: true,
		oid:             GitObjectID(commit.ID),
	}
	res.once.Do(func() {
		res.consumeCommit(commit)
	})
	return res
}

func (r *GitCommitResolver) resolveCommit(ctx context.Context) {
	if r.err != nil {
		return
	}

	r.once.Do(func() {
		var cachedRepo *gitserver.Repo
		cachedRepo, r.err = backend.CachedGitRepo(ctx, r.repoResolver.repo)
		if r.err != nil {
			return
		}

		var commit *git.Commit
		commit, r.err = git.GetCommit(ctx, *cachedRepo, nil, api.CommitID(r.oid), git.ResolveRevisionOptions{})
		if r.err != nil {
			return
		}

		r.consumeCommit(commit)
	})
}

func (r *GitCommitResolver) consumeCommit(commit *git.Commit) {
	r.author = *toSignatureResolver(&commit.Author, r.includeUserInfo)
	r.committer = toSignatureResolver(commit.Committer, r.includeUserInfo)
	r.message = commit.Message
	r.parents = commit.Parents
}

// gitCommitGQLID is a type used for marshaling and unmarshaling a Git commit's
// GraphQL ID.
type gitCommitGQLID struct {
	Repository graphql.ID  `json:"r"`
	CommitID   GitObjectID `json:"c"`
}

func marshalGitCommitID(repo graphql.ID, commitID GitObjectID) graphql.ID {
	return relay.MarshalID("GitCommit", gitCommitGQLID{Repository: repo, CommitID: commitID})
}

func unmarshalGitCommitID(id graphql.ID) (repoID graphql.ID, commitID GitObjectID, err error) {
	var spec gitCommitGQLID
	err = relay.UnmarshalSpec(id, &spec)
	return spec.Repository, spec.CommitID, err
}

func (r *GitCommitResolver) ID() graphql.ID {
	return marshalGitCommitID(r.repoResolver.ID(), r.oid)
}

func (r *GitCommitResolver) Repository() *RepositoryResolver { return r.repoResolver }

func (r *GitCommitResolver) OID() GitObjectID { return r.oid }

func (r *GitCommitResolver) AbbreviatedOID() string {
	return string(r.oid)[:7]
}
func (r *GitCommitResolver) Author(ctx context.Context) (*signatureResolver, error) {
	r.resolveCommit(ctx)
	if r.err != nil {
		return nil, r.err
	}
	return &r.author, nil
}
func (r *GitCommitResolver) Committer(ctx context.Context) (*signatureResolver, error) {
	r.resolveCommit(ctx)
	return r.committer, r.err
}
func (r *GitCommitResolver) Message(ctx context.Context) (string, error) {
	r.resolveCommit(ctx)
	return r.message, r.err
}
func (r *GitCommitResolver) Subject(ctx context.Context) (string, error) {
	r.resolveCommit(ctx)
	return GitCommitSubject(r.message), r.err
}
func (r *GitCommitResolver) Body(ctx context.Context) (*string, error) {
	r.resolveCommit(ctx)
	if r.err != nil {
		return nil, r.err
	}

	body := GitCommitBody(r.message)
	if body == "" {
		return nil, nil
	}
	return &body, nil
}

func (r *GitCommitResolver) Parents(ctx context.Context) ([]*GitCommitResolver, error) {
	r.resolveCommit(ctx)
	if r.err != nil {
		return nil, r.err
	}

	resolvers := make([]*GitCommitResolver, len(r.parents))
	for i, parent := range r.parents {
		var err error
		resolvers[i], err = r.repoResolver.Commit(ctx, &RepositoryCommitArgs{Rev: string(parent)})
		if err != nil {
			return nil, err
		}
	}
	return resolvers, nil
}

func (r *GitCommitResolver) URL() (string, error) {
	return r.repoResolver.URL() + "/-/commit/" + string(r.inputRevOrImmutableRev()), nil
}

func (r *GitCommitResolver) CanonicalURL() (string, error) {
	return r.repoResolver.URL() + "/-/commit/" + string(r.oid), nil
}

func (r *GitCommitResolver) ExternalURLs(ctx context.Context) ([]*externallink.Resolver, error) {
	return externallink.Commit(ctx, r.repoResolver.repo, api.CommitID(r.oid))
}

func (r *GitCommitResolver) Tree(ctx context.Context, args *struct {
	Path      string
	Recursive bool
}) (*GitTreeEntryResolver, error) {
	cachedRepo, err := backend.CachedGitRepo(ctx, r.repoResolver.repo)
	if err != nil {
		return nil, err
	}
	stat, err := git.Stat(ctx, *cachedRepo, api.CommitID(r.oid), args.Path)
	if err != nil {
		return nil, err
	}
	if !stat.Mode().IsDir() {
		return nil, fmt.Errorf("not a directory: %q", args.Path)
	}
	return &GitTreeEntryResolver{
		commit:      r,
		stat:        stat,
		isRecursive: args.Recursive,
	}, nil
}

func (r *GitCommitResolver) Blob(ctx context.Context, args *struct {
	Path string
}) (*GitTreeEntryResolver, error) {
	cachedRepo, err := backend.CachedGitRepo(ctx, r.repoResolver.repo)
	if err != nil {
		return nil, err
	}
	stat, err := git.Stat(ctx, *cachedRepo, api.CommitID(r.oid), args.Path)
	if err != nil {
		return nil, err
	}
	if !stat.Mode().IsRegular() {
		return nil, fmt.Errorf("not a blob: %q", args.Path)
	}
	return &GitTreeEntryResolver{
		commit: r,
		stat:   stat,
	}, nil
}

func (r *GitCommitResolver) File(ctx context.Context, args *struct {
	Path string
}) (*GitTreeEntryResolver, error) {
	return r.Blob(ctx, args)
}

func (r *GitCommitResolver) Languages(ctx context.Context) ([]string, error) {
	inventory, err := backend.Repos.GetInventory(ctx, r.repoResolver.repo, api.CommitID(r.oid), false)
	if err != nil {
		return nil, err
	}

	names := make([]string, len(inventory.Languages))
	for i, l := range inventory.Languages {
		names[i] = l.Name
	}
	return names, nil
}

func (r *GitCommitResolver) LanguageStatistics(ctx context.Context) ([]*languageStatisticsResolver, error) {
	inventory, err := backend.Repos.GetInventory(ctx, r.repoResolver.repo, api.CommitID(r.oid), false)
	if err != nil {
		return nil, err
	}
	stats := make([]*languageStatisticsResolver, 0, len(inventory.Languages))
	for _, lang := range inventory.Languages {
		stats = append(stats, &languageStatisticsResolver{
			l: lang,
		})
	}
	return stats, nil
}

func (r *GitCommitResolver) Ancestors(ctx context.Context, args *struct {
	graphqlutil.ConnectionArgs
	Query *string
	Path  *string
	After *string
}) (*gitCommitConnectionResolver, error) {
	return &gitCommitConnectionResolver{
		revisionRange: string(r.oid),
		first:         args.ConnectionArgs.First,
		query:         args.Query,
		path:          args.Path,
		after:         args.After,
		repo:          r.repoResolver,
	}, nil
}

func (r *GitCommitResolver) BehindAhead(ctx context.Context, args *struct {
	Revspec string
}) (*behindAheadCountsResolver, error) {
	cachedRepo, err := backend.CachedGitRepo(ctx, r.repoResolver.repo)
	if err != nil {
		return nil, err
	}
	counts, err := git.GetBehindAhead(ctx, *cachedRepo, args.Revspec, string(r.oid))
	if err != nil {
		return nil, err
	}
	return &behindAheadCountsResolver{
		behind: int32(counts.Behind),
		ahead:  int32(counts.Ahead),
	}, nil
}

type behindAheadCountsResolver struct{ behind, ahead int32 }

func (r *behindAheadCountsResolver) Behind() int32 { return r.behind }
func (r *behindAheadCountsResolver) Ahead() int32  { return r.ahead }

// inputRevOrImmutableRev returns the input revspec, if it is provided and nonempty. Otherwise it returns the
// canonical OID for the revision.
func (r *GitCommitResolver) inputRevOrImmutableRev() string {
	if r.inputRev != nil && *r.inputRev != "" {
		return escapeRevspecForURL(*r.inputRev)
	}
	return string(r.oid)
}

// repoRevURL returns the URL path prefix to use when constructing URLs to resources at this
// revision. Unlike inputRevOrImmutableRev, it does NOT use the OID if no input revspec is
// given. This is because the convention in the frontend is for repo-rev URLs to omit the "@rev"
// portion (unlike for commit page URLs, which must include some revspec in
// "/REPO/-/commit/REVSPEC").
func (r *GitCommitResolver) repoRevURL() (string, error) {
	url := r.repoResolver.URL()
	var rev string
	if r.inputRev != nil {
		rev = *r.inputRev // use the original input rev from the user
	} else {
		rev = string(r.oid)
	}
	if rev != "" {
		return url + "@" + escapeRevspecForURL(rev), nil
	}
	return url, nil
}

func (r *GitCommitResolver) canonicalRepoRevURL() (string, error) {
	return r.repoResolver.URL() + "@" + string(r.oid), nil
}

// GitCommitSubject returns the first line of the Git commit message.
func GitCommitSubject(message string) string {
	i := strings.Index(message, "\n")
	if i == -1 {
		return message
	}
	return message[:i]
}

// GitCommitBody returns the contents of the Git commit message after the subject.
func GitCommitBody(message string) string {
	i := strings.Index(message, "\n")
	if i == -1 {
		return ""
	}
	return strings.TrimSpace(message[i:])
}
