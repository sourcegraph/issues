package backend

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater/protocol"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

func TestRepos_ResolveRev_noRevSpecified_getsDefaultBranch(t *testing.T) {
	ctx := testContext()

	const wantRepo = "a"
	want := strings.Repeat("a", 40)

	calledRepoLookup := false
	repoupdater.MockRepoLookup = func(args protocol.RepoLookupArgs) (*protocol.RepoLookupResult, error) {
		calledRepoLookup = true
		if args.Repo != wantRepo {
			t.Errorf("got %q, want %q", args.Repo, wantRepo)
		}
		return &protocol.RepoLookupResult{
			Repo: &protocol.RepoInfo{Name: wantRepo},
		}, nil
	}
	defer func() { repoupdater.MockRepoLookup = nil }()
	var calledVCSRepoResolveRevision bool
	git.Mocks.ResolveRevision = func(rev string, opt git.ResolveRevisionOptions) (api.CommitID, error) {
		calledVCSRepoResolveRevision = true
		return api.CommitID(want), nil
	}
	defer git.ResetMocks()

	// (no rev/branch specified)
	commitID, err := NewRepos(&dbtesting.MockDB{}).ResolveRev(ctx, &types.Repo{Name: "a"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if calledRepoLookup {
		t.Error("calledRepoLookup")
	}
	if !calledVCSRepoResolveRevision {
		t.Error("!calledVCSRepoResolveRevision")
	}
	if string(commitID) != want {
		t.Errorf("got resolved commit %q, want %q", commitID, want)
	}
}

func TestRepos_ResolveRev_noCommitIDSpecified_resolvesRev(t *testing.T) {
	ctx := testContext()

	const wantRepo = "a"
	want := strings.Repeat("a", 40)

	calledRepoLookup := false
	repoupdater.MockRepoLookup = func(args protocol.RepoLookupArgs) (*protocol.RepoLookupResult, error) {
		calledRepoLookup = true
		if args.Repo != wantRepo {
			t.Errorf("got %q, want %q", args.Repo, wantRepo)
		}
		return &protocol.RepoLookupResult{
			Repo: &protocol.RepoInfo{Name: wantRepo},
		}, nil
	}
	defer func() { repoupdater.MockRepoLookup = nil }()
	var calledVCSRepoResolveRevision bool
	git.Mocks.ResolveRevision = func(rev string, opt git.ResolveRevisionOptions) (api.CommitID, error) {
		calledVCSRepoResolveRevision = true
		return api.CommitID(want), nil
	}
	defer git.ResetMocks()

	commitID, err := NewRepos(&dbtesting.MockDB{}).ResolveRev(ctx, &types.Repo{Name: "a"}, "b")
	if err != nil {
		t.Fatal(err)
	}
	if calledRepoLookup {
		t.Error("calledRepoLookup")
	}
	if !calledVCSRepoResolveRevision {
		t.Error("!calledVCSRepoResolveRevision")
	}
	if string(commitID) != want {
		t.Errorf("got resolved commit %q, want %q", commitID, want)
	}
}

func TestRepos_ResolveRev_commitIDSpecified_resolvesCommitID(t *testing.T) {
	ctx := testContext()

	const wantRepo = "a"
	want := strings.Repeat("a", 40)

	calledRepoLookup := false
	repoupdater.MockRepoLookup = func(args protocol.RepoLookupArgs) (*protocol.RepoLookupResult, error) {
		calledRepoLookup = true
		if args.Repo != wantRepo {
			t.Errorf("got %q, want %q", args.Repo, wantRepo)
		}
		return &protocol.RepoLookupResult{
			Repo: &protocol.RepoInfo{Name: wantRepo},
		}, nil
	}
	defer func() { repoupdater.MockRepoLookup = nil }()
	var calledVCSRepoResolveRevision bool
	git.Mocks.ResolveRevision = func(rev string, opt git.ResolveRevisionOptions) (api.CommitID, error) {
		calledVCSRepoResolveRevision = true
		return api.CommitID(want), nil
	}
	defer git.ResetMocks()

	commitID, err := NewRepos(&dbtesting.MockDB{}).ResolveRev(ctx, &types.Repo{Name: "a"}, strings.Repeat("a", 40))
	if err != nil {
		t.Fatal(err)
	}
	if calledRepoLookup {
		t.Error("calledRepoLookup")
	}
	if !calledVCSRepoResolveRevision {
		t.Error("!calledVCSRepoResolveRevision")
	}
	if string(commitID) != want {
		t.Errorf("got resolved commit %q, want %q", commitID, want)
	}
}

func TestRepos_ResolveRev_commitIDSpecified_failsToResolve(t *testing.T) {
	ctx := testContext()

	const wantRepo = "a"
	want := errors.New("x")

	calledRepoLookup := false
	repoupdater.MockRepoLookup = func(args protocol.RepoLookupArgs) (*protocol.RepoLookupResult, error) {
		calledRepoLookup = true
		if args.Repo != wantRepo {
			t.Errorf("got %q, want %q", args.Repo, wantRepo)
		}
		return &protocol.RepoLookupResult{
			Repo: &protocol.RepoInfo{Name: wantRepo},
		}, nil
	}
	defer func() { repoupdater.MockRepoLookup = nil }()
	var calledVCSRepoResolveRevision bool
	git.Mocks.ResolveRevision = func(rev string, opt git.ResolveRevisionOptions) (api.CommitID, error) {
		calledVCSRepoResolveRevision = true
		return "", errors.New("x")
	}
	defer git.ResetMocks()

	_, err := NewRepos(&dbtesting.MockDB{}).ResolveRev(ctx, &types.Repo{Name: "a"}, strings.Repeat("a", 40))
	if !reflect.DeepEqual(err, want) {
		t.Fatalf("got err %v, want %v", err, want)
	}
	if calledRepoLookup {
		t.Error("calledRepoLookup")
	}
	if !calledVCSRepoResolveRevision {
		t.Error("!calledVCSRepoResolveRevision")
	}
}

func TestRepos_GetCommit_repoupdaterError(t *testing.T) {
	ctx := testContext()

	const wantRepo = "a"
	want := api.CommitID(strings.Repeat("a", 40))

	calledRepoLookup := false
	repoupdater.MockRepoLookup = func(args protocol.RepoLookupArgs) (*protocol.RepoLookupResult, error) {
		calledRepoLookup = true
		if args.Repo != wantRepo {
			t.Errorf("got %q, want %q", args.Repo, wantRepo)
		}
		return &protocol.RepoLookupResult{ErrorNotFound: true}, nil
	}
	defer func() { repoupdater.MockRepoLookup = nil }()
	var calledVCSRepoGetCommit bool
	git.Mocks.GetCommit = func(commitID api.CommitID) (*git.Commit, error) {
		calledVCSRepoGetCommit = true
		return &git.Commit{ID: want}, nil
	}
	defer git.ResetMocks()

	commit, err := NewRepos(&dbtesting.MockDB{}).GetCommit(ctx, &types.Repo{Name: "a"}, want)
	if err != nil {
		t.Fatal(err)
	}
	if calledRepoLookup {
		t.Error("calledRepoLookup")
	}
	if !calledVCSRepoGetCommit {
		t.Error("!calledVCSRepoGetCommit")
	}
	if commit.ID != want {
		t.Errorf("got commit %q, want %q", commit.ID, want)
	}
}
