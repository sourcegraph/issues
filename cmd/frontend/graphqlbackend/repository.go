package graphqlbackend

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/externallink"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/phabricator"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/gitserver/protocol"
	"github.com/sourcegraph/sourcegraph/internal/search/result"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/internal/vcs"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
	"github.com/sourcegraph/sourcegraph/schema"
)

type RepositoryResolver struct {
	hydration sync.Once
	err       error

	// Invariant: Name and ID of RepoMatch are always set and safe to use. They are
	// used to hydrate the inner repo, and should always be the same as the name and
	// id of the inner repo, but referring to the inner repo directly is unsafe
	// because it may cause a race during hydration.
	result.RepoMatch

	db dbutil.DB

	// innerRepo may only contain ID and Name information.
	// To access any other repo information, use repo() instead.
	innerRepo *types.Repo

	defaultBranchOnce sync.Once
	defaultBranch     *GitRefResolver
	defaultBranchErr  error
}

func NewRepositoryResolver(db dbutil.DB, repo *types.Repo) *RepositoryResolver {
	// Protect against a nil repo
	var name api.RepoName
	var id api.RepoID
	if repo != nil {
		name = repo.Name
		id = repo.ID
	}

	return &RepositoryResolver{
		db:        db,
		innerRepo: repo,
		RepoMatch: result.RepoMatch{
			Name: name,
			ID:   id,
		},
	}
}

func (r *RepositoryResolver) ID() graphql.ID {
	return MarshalRepositoryID(r.IDInt32())
}

func (r *RepositoryResolver) IDInt32() api.RepoID {
	return r.RepoMatch.ID
}

func MarshalRepositoryID(repo api.RepoID) graphql.ID { return relay.MarshalID("Repository", repo) }

func UnmarshalRepositoryID(id graphql.ID) (repo api.RepoID, err error) {
	err = relay.UnmarshalSpec(id, &repo)
	return
}

// repo makes sure the repo is hydrated before returning it.
func (r *RepositoryResolver) repo(ctx context.Context) (*types.Repo, error) {
	err := r.hydrate(ctx)
	return r.innerRepo, err
}

func (r *RepositoryResolver) RepoName() api.RepoName {
	return r.RepoMatch.Name
}

func (r *RepositoryResolver) Name() string {
	return string(r.RepoMatch.Name)
}

func (r *RepositoryResolver) ExternalRepo(ctx context.Context) (*api.ExternalRepoSpec, error) {
	repo, err := r.repo(ctx)
	return &repo.ExternalRepo, err
}

func (r *RepositoryResolver) IsFork(ctx context.Context) (bool, error) {
	repo, err := r.repo(ctx)
	return repo.Fork, err
}

func (r *RepositoryResolver) IsArchived(ctx context.Context) (bool, error) {
	repo, err := r.repo(ctx)
	return repo.Archived, err
}

func (r *RepositoryResolver) IsPrivate(ctx context.Context) (bool, error) {
	repo, err := r.repo(ctx)
	return repo.Private, err
}

func (r *RepositoryResolver) URI(ctx context.Context) (string, error) {
	repo, err := r.repo(ctx)
	return repo.URI, err
}

func (r *RepositoryResolver) Description(ctx context.Context) (string, error) {
	repo, err := r.repo(ctx)
	return repo.Description, err
}

func (r *RepositoryResolver) ViewerCanAdminister(ctx context.Context) (bool, error) {
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx, r.db); err != nil {
		if err == backend.ErrMustBeSiteAdmin || err == backend.ErrNotAuthenticated {
			return false, nil // not an error
		}
		return false, err
	}
	return true, nil
}

func (r *RepositoryResolver) CloneInProgress(ctx context.Context) (bool, error) {
	return r.MirrorInfo().CloneInProgress(ctx)
}

type RepositoryCommitArgs struct {
	Rev          string
	InputRevspec *string
}

func (r *RepositoryResolver) Commit(ctx context.Context, args *RepositoryCommitArgs) (*GitCommitResolver, error) {
	repo, err := r.repo(ctx)
	if err != nil {
		return nil, err
	}

	commitID, err := backend.NewRepos(r.db).ResolveRev(ctx, repo, args.Rev)
	if err != nil {
		if gitserver.IsRevisionNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return r.CommitFromID(ctx, args, commitID)
}

func (r *RepositoryResolver) CommitFromID(ctx context.Context, args *RepositoryCommitArgs, commitID api.CommitID) (*GitCommitResolver, error) {
	resolver := toGitCommitResolver(r, r.db, commitID, nil)
	if args.InputRevspec != nil {
		resolver.inputRev = args.InputRevspec
	} else {
		resolver.inputRev = &args.Rev
	}
	return resolver, nil
}

func (r *RepositoryResolver) DefaultBranch(ctx context.Context) (*GitRefResolver, error) {
	do := func() (*GitRefResolver, error) {
		refName, err := getDefaultBranchForRepo(ctx, r.RepoName())
		if err != nil {
			return nil, err
		}
		if refName == "" {
			return nil, nil
		}
		return &GitRefResolver{repo: r, name: refName}, nil
	}
	r.defaultBranchOnce.Do(func() {
		r.defaultBranch, r.defaultBranchErr = do()
	})
	return r.defaultBranch, r.defaultBranchErr
}

func getDefaultBranchForRepo(ctx context.Context, repoName api.RepoName) (string, error) {
	refBytes, _, exitCode, err := git.ExecSafe(ctx, repoName, []string{"symbolic-ref", "HEAD"})
	refName := string(bytes.TrimSpace(refBytes))

	if err == nil && exitCode == 0 {
		// Check that our repo is not empty
		_, err = git.ResolveRevision(ctx, repoName, "HEAD", git.ResolveRevisionOptions{NoEnsureRevision: true})
	}

	// If we fail to get the default branch due to cloning or being empty, we return nothing.
	if err != nil {
		if vcs.IsCloneInProgress(err) || gitserver.IsRevisionNotFound(err) {
			return "", nil
		}
		return "", err
	}

	return refName, nil
}

func (r *RepositoryResolver) Language(ctx context.Context) (string, error) {
	// The repository language is the most common language at the HEAD commit of the repository.
	// Note: the repository database field is no longer updated as of
	// https://github.com/sourcegraph/sourcegraph/issues/2586, so we do not use it anymore and
	// instead compute the language on the fly.
	repo, err := r.repo(ctx)
	if err != nil {
		return "", err
	}

	repos := backend.NewRepos(r.db)

	commitID, err := repos.ResolveRev(ctx, repo, "")
	if err != nil {
		// Comment: Should we return a nil error?
		return "", err
	}

	inventory, err := repos.GetInventory(ctx, repo, commitID, false)
	if err != nil {
		return "", err
	}
	if len(inventory.Languages) == 0 {
		return "", err
	}
	return inventory.Languages[0].Name, nil
}

func (r *RepositoryResolver) Enabled() bool { return true }

// No clients that we know of read this field. Additionally on performance profiles
// the marshalling of timestamps is significant in our postgres client. So we
// deprecate the fields and return fake data for created_at.
// https://github.com/sourcegraph/sourcegraph/pull/4668
func (r *RepositoryResolver) CreatedAt() DateTime {
	return DateTime{Time: time.Now()}
}

func (r *RepositoryResolver) UpdatedAt() *DateTime {
	return nil
}

func (r *RepositoryResolver) URL() string {
	return r.RepoMatch.URL().String()
}

func (r *RepositoryResolver) ExternalURLs(ctx context.Context) ([]*externallink.Resolver, error) {
	repo, err := r.repo(ctx)
	if err != nil {
		return nil, err
	}
	return externallink.Repository(ctx, r.db, repo)
}

func (r *RepositoryResolver) Rev() string {
	return r.RepoMatch.Rev
}

func (r *RepositoryResolver) Label() (Markdown, error) {
	var label string
	if r.Rev() != "" {
		label = r.Name() + "@" + r.Rev()
	} else {
		label = r.Name()
	}
	text := "[" + label + "](" + r.URL() + ")"
	return Markdown(text), nil
}

func (r *RepositoryResolver) Detail() Markdown {
	return Markdown("Repository match")
}

func (r *RepositoryResolver) Matches() []*searchResultMatchResolver {
	return nil
}

func (r *RepositoryResolver) ToRepository() (*RepositoryResolver, bool) { return r, true }
func (r *RepositoryResolver) ToFileMatch() (*FileMatchResolver, bool)   { return nil, false }
func (r *RepositoryResolver) ToCommitSearchResult() (*CommitSearchResultResolver, bool) {
	return nil, false
}

func (r *RepositoryResolver) ResultCount() int32 {
	return 1
}

func (r *RepositoryResolver) Type(ctx context.Context) (*types.Repo, error) {
	return r.repo(ctx)
}

func (r *RepositoryResolver) Stars(ctx context.Context) (int32, error) {
	repo, err := r.repo(ctx)
	if err != nil {
		return 0, err
	}
	return int32(repo.Stars), nil
}

func (r *RepositoryResolver) hydrate(ctx context.Context) error {
	r.hydration.Do(func() {
		// Repositories with an empty creation date were created using RepoName.ToRepo(),
		// they only contain ID and name information.
		if r.innerRepo != nil && !r.innerRepo.CreatedAt.IsZero() {
			return
		}

		log15.Debug("RepositoryResolver.hydrate", "repo.ID", r.IDInt32())

		var repo *types.Repo
		repo, r.err = database.Repos(r.db).Get(ctx, r.IDInt32())
		if r.err == nil {
			r.innerRepo = repo
		}
	})

	return r.err
}

func (r *RepositoryResolver) LSIFUploads(ctx context.Context, args *LSIFUploadsQueryArgs) (LSIFUploadConnectionResolver, error) {
	return EnterpriseResolvers.codeIntelResolver.LSIFUploadsByRepo(ctx, &LSIFRepositoryUploadsQueryArgs{
		LSIFUploadsQueryArgs: args,
		RepositoryID:         r.ID(),
	})
}

func (r *RepositoryResolver) LSIFIndexes(ctx context.Context, args *LSIFIndexesQueryArgs) (LSIFIndexConnectionResolver, error) {
	return EnterpriseResolvers.codeIntelResolver.LSIFIndexesByRepo(ctx, &LSIFRepositoryIndexesQueryArgs{
		LSIFIndexesQueryArgs: args,
		RepositoryID:         r.ID(),
	})
}

func (r *RepositoryResolver) IndexConfiguration(ctx context.Context) (IndexConfigurationResolver, error) {
	return EnterpriseResolvers.codeIntelResolver.IndexConfiguration(ctx, r.ID())
}

func (r *RepositoryResolver) CodeIntelligenceCommitGraph(ctx context.Context) (CodeIntelligenceCommitGraphResolver, error) {
	return EnterpriseResolvers.codeIntelResolver.CommitGraph(ctx, r.ID())
}

type AuthorizedUserArgs struct {
	RepositoryID graphql.ID
	Permission   string
	First        int32
	After        *string
}

type RepoAuthorizedUserArgs struct {
	RepositoryID graphql.ID
	*AuthorizedUserArgs
}

func (r *RepositoryResolver) AuthorizedUsers(ctx context.Context, args *AuthorizedUserArgs) (UserConnectionResolver, error) {
	return EnterpriseResolvers.authzResolver.AuthorizedUsers(ctx, &RepoAuthorizedUserArgs{
		RepositoryID:       r.ID(),
		AuthorizedUserArgs: args,
	})
}

func (r *RepositoryResolver) PermissionsInfo(ctx context.Context) (PermissionsInfoResolver, error) {
	return EnterpriseResolvers.authzResolver.RepositoryPermissionsInfo(ctx, r.ID())
}

func (r *schemaResolver) AddPhabricatorRepo(ctx context.Context, args *struct {
	Callsign string
	Name     *string
	// TODO(chris): Remove URI in favor of Name.
	URI *string
	URL string
}) (*EmptyResponse, error) {
	if args.Name != nil {
		args.URI = args.Name
	}

	_, err := database.Phabricator(r.db).CreateIfNotExists(ctx, args.Callsign, api.RepoName(*args.URI), args.URL)
	if err != nil {
		log15.Error("adding phabricator repo", "callsign", args.Callsign, "name", args.URI, "url", args.URL)
	}
	return nil, err
}

func (r *schemaResolver) ResolvePhabricatorDiff(ctx context.Context, args *struct {
	RepoName    string
	DiffID      int32
	BaseRev     string
	Patch       *string
	AuthorName  *string
	AuthorEmail *string
	Description *string
	Date        *string
}) (*GitCommitResolver, error) {
	repo, err := database.Repos(r.db).GetByName(ctx, api.RepoName(args.RepoName))
	if err != nil {
		return nil, err
	}
	targetRef := fmt.Sprintf("phabricator/diff/%d", args.DiffID)
	getCommit := func() (*GitCommitResolver, error) {
		// We first check via the vcsrepo api so that we can toggle
		// NoEnsureRevision. We do this, otherwise RepositoryResolver.Commit
		// will try and fetch it from the remote host. However, this is not on
		// the remote host since we created it.
		_, err = git.ResolveRevision(ctx, repo.Name, targetRef, git.ResolveRevisionOptions{
			NoEnsureRevision: true,
		})
		if err != nil {
			return nil, err
		}
		r := NewRepositoryResolver(r.db, repo)
		return r.Commit(ctx, &RepositoryCommitArgs{Rev: targetRef})
	}

	// If we already created the commit
	if commit, err := getCommit(); commit != nil || (err != nil && !gitserver.IsRevisionNotFound(err)) {
		return commit, err
	}

	origin := ""
	if phabRepo, err := database.Phabricator(r.db).GetByName(ctx, api.RepoName(args.RepoName)); err == nil {
		origin = phabRepo.URL
	}

	if origin == "" {
		return nil, errors.New("unable to resolve the origin of the phabricator instance")
	}

	client, clientErr := makePhabClientForOrigin(ctx, r.db, origin)

	patch := ""
	if args.Patch != nil {
		patch = *args.Patch
	} else if client == nil {
		return nil, clientErr
	} else {
		diff, err := client.GetRawDiff(ctx, int(args.DiffID))
		// No diff contents were given and we couldn't fetch them
		if err != nil {
			return nil, err
		}

		patch = diff
	}

	var info *phabricator.DiffInfo
	if client != nil && (args.AuthorEmail == nil || args.AuthorName == nil || args.Date == nil) {
		info, err = client.GetDiffInfo(ctx, int(args.DiffID))
		// Not all the information was given and we couldn't fetch it
		if err != nil {
			return nil, err
		}
	} else {
		var description, authorName, authorEmail string
		if args.Description != nil {
			description = *args.Description
		}
		if args.AuthorName != nil {
			authorName = *args.AuthorName
		}
		if args.AuthorEmail != nil {
			authorEmail = *args.AuthorEmail
		}
		date, err := phabricator.ParseDate(*args.Date)
		if err != nil {
			return nil, err
		}

		info = &phabricator.DiffInfo{
			AuthorName:  authorName,
			AuthorEmail: authorEmail,
			Message:     description,
			Date:        *date,
		}
	}

	_, err = gitserver.DefaultClient.CreateCommitFromPatch(ctx, protocol.CreateCommitFromPatchRequest{
		Repo:       api.RepoName(args.RepoName),
		BaseCommit: api.CommitID(args.BaseRev),
		TargetRef:  targetRef,
		Patch:      patch,
		CommitInfo: protocol.PatchCommitInfo{
			AuthorName:  info.AuthorName,
			AuthorEmail: info.AuthorEmail,
			Message:     info.Message,
			Date:        info.Date,
		},
	})
	if err != nil {
		return nil, err
	}

	return getCommit()
}

func makePhabClientForOrigin(ctx context.Context, db dbutil.DB, origin string) (*phabricator.Client, error) {
	opt := database.ExternalServicesListOptions{
		Kinds: []string{extsvc.KindPhabricator},
		LimitOffset: &database.LimitOffset{
			Limit: 500, // The number is randomly chosen
		},
	}
	for {
		svcs, err := database.ExternalServices(db).List(ctx, opt)
		if err != nil {
			return nil, errors.Wrap(err, "list")
		}
		if len(svcs) == 0 {
			break // No more results, exiting
		}
		opt.AfterID = svcs[len(svcs)-1].ID // Advance the cursor

		for _, svc := range svcs {
			cfg, err := extsvc.ParseConfig(svc.Kind, svc.Config)
			if err != nil {
				return nil, errors.Wrap(err, "parse config")
			}

			var conn *schema.PhabricatorConnection
			switch c := cfg.(type) {
			case *schema.PhabricatorConnection:
				conn = c
			default:
				log15.Error("makePhabClientForOrigin", "error", errors.Errorf("want *schema.PhabricatorConnection but got %T", cfg))
				continue
			}

			if conn.Url != origin {
				continue
			}

			if conn.Token == "" {
				return nil, errors.Errorf("no phabricator token was given for: %s", origin)
			}

			return phabricator.NewClient(ctx, conn.Url, conn.Token, nil)
		}

		if len(svcs) < opt.Limit {
			break // Less results than limit means we've reached end
		}
	}

	return nil, errors.Errorf("no phabricator was configured for: %s", origin)
}
