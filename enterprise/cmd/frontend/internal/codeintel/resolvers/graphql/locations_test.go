package graphql

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	gql "github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/codeintel/resolvers"
	store "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/lsifstore"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

const numRoutines = 5
const numRepositories = 10
const numCommits = 10 // per repo
const numPaths = 10   // per commit

func TestCachedLocationResolver(t *testing.T) {
	db := new(dbtesting.MockDB)

	t.Cleanup(func() {
		database.Mocks.Repos.Get = nil
		git.Mocks.ResolveRevision = nil
		backend.Mocks.Repos.GetCommit = nil
	})

	var repoCalls uint32
	database.Mocks.Repos.Get = func(v0 context.Context, id api.RepoID) (*types.Repo, error) {
		atomic.AddUint32(&repoCalls, 1)
		return &types.Repo{ID: id, CreatedAt: time.Now()}, nil
	}

	git.Mocks.ResolveRevision = func(spec string, opt git.ResolveRevisionOptions) (api.CommitID, error) {
		return api.CommitID(spec), nil
	}

	var commitCalls uint32
	git.Mocks.GetCommit = func(commitID api.CommitID) (*git.Commit, error) {
		atomic.AddUint32(&commitCalls, 1)
		return &git.Commit{ID: commitID}, nil
	}

	cachedResolver := NewCachedLocationResolver(db)

	var repositoryIDs []api.RepoID
	for i := 1; i <= numRepositories; i++ {
		repositoryIDs = append(repositoryIDs, api.RepoID(i))
	}

	var commits []string
	for i := 1; i <= numCommits; i++ {
		commits = append(commits, fmt.Sprintf("%040d", i))
	}

	var paths []string
	for i := 1; i <= numPaths; i++ {
		paths = append(paths, fmt.Sprintf("/foo/%d/bar/baz.go", i))
	}

	type resolverPair struct {
		key      string
		resolver *gql.GitTreeEntryResolver
	}
	resolvers := make(chan resolverPair, numRoutines*len(repositoryIDs)*len(commits)*len(paths))

	var wg sync.WaitGroup
	errs := make(chan error, numRoutines)
	for i := 0; i < numRoutines; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for _, repositoryID := range repositoryIDs {
				repositoryResolver, err := cachedResolver.Repository(context.Background(), repositoryID)
				if err != nil {
					errs <- err
					return
				}
				repoID, err := gql.UnmarshalRepositoryID(repositoryResolver.ID())
				if err != nil {
					errs <- err
					return
				}
				if repoID != repositoryID {
					errs <- errors.Errorf("unexpected repository id. want=%d have=%d", repositoryID, repoID)
					return
				}
			}

			for _, repositoryID := range repositoryIDs {
				for _, commit := range commits {
					commitResolver, err := cachedResolver.Commit(context.Background(), repositoryID, commit)
					if err != nil {
						errs <- err
						return
					}
					if commitResolver.OID() != graphqlbackend.GitObjectID(commit) {
						errs <- errors.Errorf("unexpected commit. want=%s have=%s", commit, commitResolver.OID())
						return
					}
				}
			}

			for _, repositoryID := range repositoryIDs {
				for _, commit := range commits {
					for _, path := range paths {
						treeResolver, err := cachedResolver.Path(context.Background(), repositoryID, commit, path)
						if err != nil {
							errs <- err
							return
						}
						if treeResolver.Path() != path {
							errs <- errors.Errorf("unexpected path. want=%s have=%s", path, treeResolver.Path())
							return
						}

						resolvers <- resolverPair{key: fmt.Sprintf("%d:%s:%s", repositoryID, commit, path), resolver: treeResolver}
					}
				}
			}
		}()
	}
	wg.Wait()

	close(errs)
	for err := range errs {
		t.Error(err)
	}

	if val := atomic.LoadUint32(&repoCalls); val != uint32(len(repositoryIDs)) {
		t.Errorf("unexpected number of repo calls. want=%d have=%d", len(repositoryIDs), val)
	}

	// We don't need to load commits from git-server unless we ask for fields like author or committer.
	// Since we already know this commit exists, and we only need it's already known commit ID, we assert
	// that zero calls to git.GetCommit where done. Check the gitCommitResolver lazy loading logic.
	if val := atomic.LoadUint32(&commitCalls); val != 0 {
		t.Errorf("unexpected number of commit calls. want=%d have=%d", 0, val)
	}

	close(resolvers)
	resolversByKey := map[string][]*gql.GitTreeEntryResolver{}
	for pair := range resolvers {
		resolversByKey[pair.key] = append(resolversByKey[pair.key], pair.resolver)
	}

	for _, vs := range resolversByKey {
		for _, v := range vs {
			if v != vs[0] {
				t.Errorf("resolvers for same key unexpectedly have differing addresses: %p and %p", v, vs[0])
			}
		}
	}
}

func TestCachedLocationResolverUnknownRepository(t *testing.T) {
	db := new(dbtesting.MockDB)

	t.Cleanup(func() {
		database.Mocks.Repos.Get = nil
		git.Mocks.ResolveRevision = nil
	})

	database.Mocks.Repos.Get = func(v0 context.Context, id api.RepoID) (*types.Repo, error) {
		return nil, &database.RepoNotFoundErr{ID: id}
	}

	repositoryResolver, err := NewCachedLocationResolver(db).Repository(context.Background(), 50)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if repositoryResolver != nil {
		t.Errorf("unexpected non-nil resolver")
	}

	// Ensure no dereference in child resolvers either
	pathResolver, err := NewCachedLocationResolver(db).Path(context.Background(), 50, "deadbeef", "main.go")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if pathResolver != nil {
		t.Errorf("unexpected non-nil resolver")
	}
}

func TestCachedLocationResolverUnknownCommit(t *testing.T) {
	db := new(dbtesting.MockDB)

	t.Cleanup(func() {
		database.Mocks.Repos.Get = nil
		git.Mocks.ResolveRevision = nil
	})

	database.Mocks.Repos.Get = func(v0 context.Context, id api.RepoID) (*types.Repo, error) {
		return &types.Repo{ID: id}, nil
	}

	git.Mocks.ResolveRevision = func(spec string, opt git.ResolveRevisionOptions) (api.CommitID, error) {
		return "", &gitserver.RevisionNotFoundError{}
	}

	commitResolver, err := NewCachedLocationResolver(db).Commit(context.Background(), 50, "deadbeef")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if commitResolver != nil {
		t.Errorf("unexpected non-nil resolver")
	}

	// Ensure no dereference in child resolvers either
	pathResolver, err := NewCachedLocationResolver(db).Path(context.Background(), 50, "deadbeef", "main.go")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if pathResolver != nil {
		t.Errorf("unexpected non-nil resolver")
	}
}

func TestResolveLocations(t *testing.T) {
	db := new(dbtesting.MockDB)

	t.Cleanup(func() {
		database.Mocks.Repos.Get = nil
		git.Mocks.ResolveRevision = nil
		backend.Mocks.Repos.GetCommit = nil
	})

	database.Mocks.Repos.Get = func(v0 context.Context, id api.RepoID) (*types.Repo, error) {
		return &types.Repo{ID: id, Name: api.RepoName(fmt.Sprintf("repo%d", id))}, nil
	}

	git.Mocks.ResolveRevision = func(spec string, opt git.ResolveRevisionOptions) (api.CommitID, error) {
		if spec == "deadbeef3" {
			return "", &gitserver.RevisionNotFoundError{}
		}
		return api.CommitID(spec), nil
	}

	backend.Mocks.Repos.GetCommit = func(v0 context.Context, repo *types.Repo, commitID api.CommitID) (*git.Commit, error) {
		return &git.Commit{ID: commitID}, nil
	}

	r1 := lsifstore.Range{Start: lsifstore.Position{Line: 11, Character: 12}, End: lsifstore.Position{Line: 13, Character: 14}}
	r2 := lsifstore.Range{Start: lsifstore.Position{Line: 21, Character: 22}, End: lsifstore.Position{Line: 23, Character: 24}}
	r3 := lsifstore.Range{Start: lsifstore.Position{Line: 31, Character: 32}, End: lsifstore.Position{Line: 33, Character: 34}}
	r4 := lsifstore.Range{Start: lsifstore.Position{Line: 41, Character: 42}, End: lsifstore.Position{Line: 43, Character: 44}}

	locations, err := resolveLocations(context.Background(), NewCachedLocationResolver(db), []resolvers.AdjustedLocation{
		{Dump: store.Dump{RepositoryID: 50}, AdjustedCommit: "deadbeef1", AdjustedRange: r1, Path: "p1"},
		{Dump: store.Dump{RepositoryID: 51}, AdjustedCommit: "deadbeef2", AdjustedRange: r2, Path: "p2"},
		{Dump: store.Dump{RepositoryID: 52}, AdjustedCommit: "deadbeef3", AdjustedRange: r3, Path: "p3"},
		{Dump: store.Dump{RepositoryID: 53}, AdjustedCommit: "deadbeef4", AdjustedRange: r4, Path: "p4"},
	})
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	if len(locations) != 3 {
		t.Fatalf("unexpected length. want=%d have=%d", 3, len(locations))
	}
	if url := locations[0].CanonicalURL(); url != "/repo50@deadbeef1/-/tree/p1?L12:13-14:15" {
		t.Errorf("unexpected canonical url. want=%s have=%s", "/repo50@deadbeef1/-/tree/p1?L12:13-14:15", url)
	}
	if url := locations[1].CanonicalURL(); url != "/repo51@deadbeef2/-/tree/p2?L22:23-24:25" {
		t.Errorf("unexpected canonical url. want=%s have=%s", "/repo51@deadbeef2/-/tree/p2?L22:23-24:25", url)
	}
	if url := locations[2].CanonicalURL(); url != "/repo53@deadbeef4/-/tree/p4?L42:43-44:45" {
		t.Errorf("unexpected canonical url. want=%s have=%s", "/repo53@deadbeef4/-/tree/p4?L42:43-44:45", url)
	}
}
