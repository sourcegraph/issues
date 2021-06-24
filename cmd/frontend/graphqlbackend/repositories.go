package graphqlbackend

import (
	"context"
	"sync"
	"time"

	"github.com/google/zoekt"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/graphqlutil"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/search"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

type repositoryArgs struct {
	graphqlutil.ConnectionArgs
	Query       *string
	Names       *[]string
	Cloned      bool
	NotCloned   bool
	Indexed     bool
	NotIndexed  bool
	FailedFetch bool
	OrderBy     string
	Descending  bool
	After       *string
}

func (r *schemaResolver) Repositories(args *repositoryArgs) (*repositoryConnectionResolver, error) {
	opt := database.ReposListOptions{
		OrderBy: database.RepoListOrderBy{{
			Field:      toDBRepoListColumn(args.OrderBy),
			Descending: args.Descending,
		}},
	}
	if args.Names != nil {
		opt.Names = *args.Names
	}
	if args.Query != nil {
		opt.Query = *args.Query
	}
	if args.After != nil {
		cursor, err := unmarshalRepositoryCursor(args.After)
		if err != nil {
			return nil, err
		}
		opt.CursorColumn = cursor.Column
		opt.CursorValue = cursor.Value
		opt.CursorDirection = cursor.Direction
	} else {
		opt.CursorColumn = string(toDBRepoListColumn(args.OrderBy))
		opt.CursorValue = ""
		if args.Descending {
			opt.CursorDirection = "prev"
		} else {
			opt.CursorDirection = "next"
		}
	}

	opt.FailedFetch = args.FailedFetch
	args.ConnectionArgs.Set(&opt.LimitOffset)

	return &repositoryConnectionResolver{
		db:          r.db,
		opt:         opt,
		cloned:      args.Cloned,
		notCloned:   args.NotCloned,
		indexed:     args.Indexed,
		notIndexed:  args.NotIndexed,
		failedFetch: args.FailedFetch,
	}, nil
}

type TotalCountArgs struct {
	Precise bool
}

type RepositoryConnectionResolver interface {
	Nodes(ctx context.Context) ([]*RepositoryResolver, error)
	TotalCount(ctx context.Context, args *TotalCountArgs) (*int32, error)
	PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error)
}

var _ RepositoryConnectionResolver = &repositoryConnectionResolver{}

type repositoryConnectionResolver struct {
	db          dbutil.DB
	opt         database.ReposListOptions
	cloned      bool
	notCloned   bool
	indexed     bool
	notIndexed  bool
	failedFetch bool

	// cache results because they are used by multiple fields
	once  sync.Once
	repos []*types.Repo
	err   error
}

func (r *repositoryConnectionResolver) compute(ctx context.Context) ([]*types.Repo, error) {
	r.once.Do(func() {
		opt2 := r.opt

		if envvar.SourcegraphDotComMode() {
			// 🚨 SECURITY: Don't allow non-admins to perform huge queries on Sourcegraph.com.
			if isSiteAdmin := backend.CheckCurrentUserIsSiteAdmin(ctx, r.db) == nil; !isSiteAdmin {
				if opt2.LimitOffset == nil {
					opt2.LimitOffset = &database.LimitOffset{Limit: 1000}
				}
			}
		}

		var indexed map[string]*zoekt.Repository
		searchIndexEnabled := search.Indexed().Enabled()
		isIndexed := func(repo api.RepoName) bool {
			if !searchIndexEnabled {
				return true // do not need index
			}
			_, ok := indexed[string(repo)]
			return ok
		}
		if searchIndexEnabled && (!r.indexed || !r.notIndexed) {
			listCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
			var err error
			indexed, err = search.Indexed().ListAll(listCtx)
			if err != nil {
				r.err = err
				return
			}
			// ensure we fetch at least as many repos as we have indexed
			if opt2.LimitOffset != nil && opt2.LimitOffset.Limit < len(indexed) {
				opt2.LimitOffset.Limit = len(indexed) * 2
			}
		}

		if !r.cloned {
			opt2.NoCloned = true
		} else if !r.notCloned {
			// notCloned is true by default.
			// this condition is valid only if it has been
			// explicitly set to false by the client.
			opt2.OnlyCloned = true
		}
		opt2.FailedFetch = r.failedFetch

		for {
			// Cursor-based pagination requires that we fetch limit+1 records, so
			// that we know whether or not there's an additional page (or more)
			// beyond the current one. We reset the limit immediately afterward for
			// any subsequent calculations.
			if opt2.LimitOffset != nil {
				opt2.LimitOffset.Limit++
			}
			repos, err := backend.NewRepos(r.db).List(ctx, opt2)
			if err != nil {
				r.err = err
				return
			}
			if opt2.LimitOffset != nil {
				opt2.LimitOffset.Limit--
			}
			reposFromDB := len(repos)

			if !r.indexed || !r.notIndexed {
				keepRepos := repos[:0]
				for _, repo := range repos {
					indexed := isIndexed(repo.Name)
					if (r.indexed && indexed) || (r.notIndexed && !indexed) {
						keepRepos = append(keepRepos, repo)
					}
				}
				repos = keepRepos
			}

			r.repos = append(r.repos, repos...)

			if opt2.LimitOffset == nil {
				break
			} else {
				// check if we filtered some repos and if
				// we need to get more from the DB
				if len(repos) >= r.opt.Limit || reposFromDB < r.opt.Limit {
					break
				}
				opt2.Offset += opt2.Limit
			}
		}
	})

	return r.repos, r.err
}

func (r *repositoryConnectionResolver) Nodes(ctx context.Context) ([]*RepositoryResolver, error) {
	repos, err := r.compute(ctx)
	if err != nil {
		return nil, err
	}
	resolvers := make([]*RepositoryResolver, 0, len(repos))
	for i, repo := range repos {
		if r.opt.LimitOffset != nil && i == r.opt.Limit {
			break
		}

		resolvers = append(resolvers, NewRepositoryResolver(r.db, repo))
	}
	return resolvers, nil
}

func (r *repositoryConnectionResolver) TotalCount(ctx context.Context, args *TotalCountArgs) (countptr *int32, err error) {
	if r.opt.UserID != 0 {
		// 🚨 SECURITY: If filtering by user, restrict to that user
		if err := backend.CheckSameUser(ctx, r.opt.UserID); err != nil {
			return nil, err
		}
	} else {
		// 🚨 SECURITY: Only site admins can list all repos, because a total repository
		// count does not respect repository permissions.
		if err := backend.CheckCurrentUserIsSiteAdmin(ctx, r.db); err != nil {
			return nil, err
		}
	}

	i32ptr := func(v int32) *int32 {
		return &v
	}

	if !r.cloned || !r.notCloned {
		// Don't support counting if filtering by clone status.
		return nil, nil
	}
	if !r.indexed || !r.notIndexed {
		// Don't support counting if filtering by index status.
		return nil, nil
	}

	// Counting repositories is slow on Sourcegraph.com. Don't wait very long for an exact count.
	if !args.Precise && envvar.SourcegraphDotComMode() {
		if len(r.opt.Query) < 4 {
			return nil, nil
		}

		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, 300*time.Millisecond)
		defer cancel()
		defer func() {
			if ctx.Err() == context.DeadlineExceeded {
				countptr = nil
				err = nil
			}
		}()
	}

	count, err := database.Repos(r.db).Count(ctx, r.opt)
	return i32ptr(int32(count)), err
}

func (r *repositoryConnectionResolver) PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error) {
	repos, err := r.compute(ctx)
	if err != nil {
		return nil, err
	}
	if len(repos) == 0 || r.opt.LimitOffset == nil || len(repos) <= r.opt.Limit {
		return graphqlutil.HasNextPage(false), nil
	}

	var value string
	switch r.opt.CursorColumn {
	case string(database.RepoListName):
		value = string(repos[len(repos)-1].Name)
	case string(database.RepoListCreatedAt):
		value = repos[len(repos)-1].CreatedAt.Format("2006-01-02 15:04:05.999999")
	}
	return graphqlutil.NextPageCursor(marshalRepositoryCursor(
		&repositoryCursor{
			Column:    r.opt.CursorColumn,
			Value:     value,
			Direction: r.opt.CursorDirection,
		},
	)), nil
}

func repoNamesToStrings(repoNames []api.RepoName) []string {
	strings := make([]string, len(repoNames))
	for i, repoName := range repoNames {
		strings[i] = string(repoName)
	}
	return strings
}

func toDBRepoListColumn(ob string) database.RepoListColumn {
	switch ob {
	case "REPO_URI", "REPOSITORY_NAME":
		return database.RepoListName
	case "REPO_CREATED_AT", "REPOSITORY_CREATED_AT":
		return database.RepoListCreatedAt
	default:
		return ""
	}
}
