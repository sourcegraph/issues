package db

import (
	"context"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/authz"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/db/query"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/github"
)

/*
 * Helpers
 */

func sortedRepoNames(repos []*types.Repo) []api.RepoName {
	names := repoNames(repos)
	sort.Slice(names, func(i, j int) bool { return names[i] < names[j] })
	return names
}

func repoNames(repos []*types.Repo) []api.RepoName {
	var names []api.RepoName
	for _, repo := range repos {
		names = append(names, repo.Name)
	}
	return names
}

func createRepo(ctx context.Context, t *testing.T, repo *types.Repo) {
	t.Helper()

	op := InsertRepoOp{Name: repo.Name}

	if repo.RepoFields != nil {
		op.Description = repo.Description
		op.Fork = repo.Fork
		op.Cloned = repo.Cloned
		op.Archived = repo.Archived
	}

	if err := Repos.Upsert(ctx, op); err != nil {
		t.Fatal(err)
	}
}

func mustCreate(ctx context.Context, t *testing.T, repos ...*types.Repo) []*types.Repo {
	var createdRepos []*types.Repo
	for _, repo := range repos {
		createRepo(ctx, t, repo)
		repo, err := Repos.GetByName(ctx, repo.Name)
		if err != nil {
			t.Fatal(err)
		}
		createdRepos = append(createdRepos, repo)
	}
	return createdRepos
}

// InsertRepoOp represents an operation to insert a repository.
type InsertRepoOp struct {
	Name         api.RepoName
	Description  string
	Fork         bool
	Archived     bool
	Cloned       bool
	ExternalRepo api.ExternalRepoSpec
}

const upsertSQL = `
WITH upsert AS (
  UPDATE repo
  SET
    name                  = $1,
    description           = $2,
    fork                  = $3,
    external_id           = NULLIF(BTRIM($4), ''),
    external_service_type = NULLIF(BTRIM($5), ''),
    external_service_id   = NULLIF(BTRIM($6), ''),
    archived              = $7,
    cloned                = $8
  WHERE name = $1 OR (
    external_id IS NOT NULL
    AND external_service_type IS NOT NULL
    AND external_service_id IS NOT NULL
    AND NULLIF(BTRIM($4), '') IS NOT NULL
    AND NULLIF(BTRIM($5), '') IS NOT NULL
    AND NULLIF(BTRIM($6), '') IS NOT NULL
    AND external_id = NULLIF(BTRIM($4), '')
    AND external_service_type = NULLIF(BTRIM($5), '')
    AND external_service_id = NULLIF(BTRIM($6), '')
  )
  RETURNING repo.name
)

INSERT INTO repo (
  name,
  description,
  fork,
  external_id,
  external_service_type,
  external_service_id,
  archived,
  cloned
) (
  SELECT
    $1 AS name,
    $2 AS description,
    $3 AS fork,
    NULLIF(BTRIM($4), '') AS external_id,
    NULLIF(BTRIM($5), '') AS external_service_type,
    NULLIF(BTRIM($6), '') AS external_service_id,
    $7 AS archived,
    $8 AS cloned
  WHERE NOT EXISTS (SELECT 1 FROM upsert)
)`

// Upsert updates the repository if it already exists (keyed on name) and
// inserts it if it does not.
//
// Upsert exists for testing purposes only. Repository mutations are managed
// by repo-updater.
func (s *repos) Upsert(ctx context.Context, op InsertRepoOp) error {
	insert := false

	// We optimistically assume the repo is already in the table, so first
	// check if it is. We then fallback to the upsert functionality. The
	// upsert is logged as a modification to the DB, even if it is a no-op. So
	// we do this check to avoid log spam if postgres is configured with
	// log_statement='mod'.
	r, err := s.GetByName(ctx, op.Name)
	if err != nil {
		if _, ok := err.(*RepoNotFoundErr); !ok {
			return err
		}
		insert = true // missing
	} else {
		insert = (op.Description != r.Description) ||
			(op.Fork != r.Fork) ||
			(!op.ExternalRepo.Equal(&r.ExternalRepo))
	}

	if !insert {
		return nil
	}

	_, err = dbconn.Global.ExecContext(
		ctx,
		upsertSQL,
		op.Name,
		op.Description,
		op.Fork,
		op.ExternalRepo.ID,
		op.ExternalRepo.ServiceType,
		op.ExternalRepo.ServiceID,
		op.Archived,
		op.Cloned,
	)

	return err
}

/*
 * Tests
 */

func TestRepos_Get(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	now := time.Now()

	service := types.ExternalService{
		Kind:        extsvc.KindGitHub,
		DisplayName: "Github - Test",
		Config:      `{"url": "https://github.com", "repositoryQuery": ["none"], "token": "abc"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Create a new external service
	confGet := func() *conf.Unified {
		return &conf.Unified{}
	}

	err := ExternalServices.Create(ctx, confGet, &service)
	if err != nil {
		t.Fatal(err)
	}

	want := mustCreate(ctx, t, &types.Repo{
		ExternalRepo: api.ExternalRepoSpec{
			ID:          "r",
			ServiceType: extsvc.TypeGitHub,
			ServiceID:   "https://github.com",
		},
		Name:    "name",
		Private: true,
		RepoFields: &types.RepoFields{
			URI:         "uri",
			Description: "description",
			Fork:        true,
			Archived:    true,
			Cloned:      true,
			CreatedAt:   now,
			UpdatedAt:   now,
			Metadata:    new(github.Repository),
			Sources: map[string]*types.SourceInfo{
				service.URN(): {
					ID:       service.URN(),
					CloneURL: "git@github.com:foo/bar.git",
				},
			},
		},
	})

	repo, err := Repos.Get(ctx, want[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if !jsonEqual(t, repo, want[0]) {
		t.Errorf("got %v, want %v", repo, want[0])
	}
}

func TestRepos_GetByIDs(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	want := mustCreate(ctx, t, &types.Repo{
		Name: "r",
		ExternalRepo: api.ExternalRepoSpec{
			ID:          "a",
			ServiceType: "b",
			ServiceID:   "c",
		},
	})

	repos, err := Repos.GetByIDs(ctx, want[0].ID, 404)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 {
		t.Fatalf("got %d repos, but want 1", len(repos))
	}

	// We don't need the RepoFields to identify a repository.
	want[0].RepoFields = nil
	if !jsonEqual(t, repos[0], want[0]) {
		t.Errorf("got %v, want %v", repos[0], want[0])
	}
}

func TestRepos_List(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	MockAuthzFilter = func(ctx context.Context, repos []*types.Repo, p authz.Perms) ([]*types.Repo, error) {
		return repos, nil
	}
	defer func() { MockAuthzFilter = nil }()

	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()
	ctx = actor.WithActor(ctx, &actor.Actor{})

	now := time.Now()

	service := types.ExternalService{
		Kind:        extsvc.KindGitHub,
		DisplayName: "Github - Test",
		Config:      `{"url": "https://github.com", "repositoryQuery": ["none"], "token": "abc"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Create a new external service
	confGet := func() *conf.Unified {
		return &conf.Unified{}
	}

	err := ExternalServices.Create(ctx, confGet, &service)
	if err != nil {
		t.Fatal(err)
	}

	want := mustCreate(ctx, t, &types.Repo{
		ExternalRepo: api.ExternalRepoSpec{
			ID:          "r",
			ServiceType: extsvc.TypeGitHub,
			ServiceID:   "https://github.com",
		},
		Name:    "name",
		Private: true,
		RepoFields: &types.RepoFields{
			URI:         "uri",
			Description: "description",
			Fork:        true,
			Archived:    true,
			Cloned:      true,
			CreatedAt:   now,
			UpdatedAt:   now,
			Metadata:    new(github.Repository),
			Sources: map[string]*types.SourceInfo{
				service.URN(): {
					ID:       service.URN(),
					CloneURL: "git@github.com:foo/bar.git",
				},
			},
		},
	})

	repos, err := Repos.List(ctx, ReposListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !jsonEqual(t, repos, want) {
		t.Errorf("got %v, want %v", repos, want)
	}
}

func TestRepos_List_fork(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	MockAuthzFilter = func(ctx context.Context, repos []*types.Repo, p authz.Perms) ([]*types.Repo, error) {
		return repos, nil
	}
	defer func() { MockAuthzFilter = nil }()
	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()
	ctx = actor.WithActor(ctx, &actor.Actor{})

	mine := mustCreate(ctx, t, &types.Repo{Name: "a/r", RepoFields: &types.RepoFields{Fork: false}})
	yours := mustCreate(ctx, t, &types.Repo{Name: "b/r", RepoFields: &types.RepoFields{Fork: true}})

	{
		repos, err := Repos.List(ctx, ReposListOptions{OnlyForks: true})
		if err != nil {
			t.Fatal(err)
		}
		assertJSONEqual(t, yours, repos)
	}
	{
		repos, err := Repos.List(ctx, ReposListOptions{NoForks: true})
		if err != nil {
			t.Fatal(err)
		}
		assertJSONEqual(t, mine, repos)
	}
	{
		repos, err := Repos.List(ctx, ReposListOptions{NoForks: true, OnlyForks: true})
		if err != nil {
			t.Fatal(err)
		}
		assertJSONEqual(t, nil, repos)
	}
	{
		repos, err := Repos.List(ctx, ReposListOptions{})
		if err != nil {
			t.Fatal(err)
		}
		assertJSONEqual(t, append(append([]*types.Repo(nil), mine...), yours...), repos)
	}
}

func TestRepos_List_cloned(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	MockAuthzFilter = func(ctx context.Context, repos []*types.Repo, p authz.Perms) ([]*types.Repo, error) {
		return repos, nil
	}
	defer func() { MockAuthzFilter = nil }()
	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()
	ctx = actor.WithActor(ctx, &actor.Actor{})

	mine := mustCreate(ctx, t, &types.Repo{Name: "a/r", RepoFields: &types.RepoFields{Cloned: false}})
	yours := mustCreate(ctx, t, &types.Repo{Name: "b/r", RepoFields: &types.RepoFields{Cloned: true}})

	tests := []struct {
		name string
		opt  ReposListOptions
		want []*types.Repo
	}{
		{"OnlyCloned", ReposListOptions{OnlyCloned: true}, yours},
		{"NoCloned", ReposListOptions{NoCloned: true}, mine},
		{"NoCloned && OnlyCloned", ReposListOptions{NoCloned: true, OnlyCloned: true}, nil},
		{"Default", ReposListOptions{}, append(append([]*types.Repo(nil), mine...), yours...)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repos, err := Repos.List(ctx, test.opt)
			if err != nil {
				t.Fatal(err)
			}
			assertJSONEqual(t, test.want, repos)
		})
	}
}

func TestRepos_List_pagination(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	MockAuthzFilter = func(ctx context.Context, repos []*types.Repo, p authz.Perms) ([]*types.Repo, error) {
		return repos, nil
	}
	defer func() { MockAuthzFilter = nil }()
	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()
	ctx = actor.WithActor(ctx, &actor.Actor{})

	createdRepos := []*types.Repo{
		{Name: "r1"},
		{Name: "r2"},
		{Name: "r3"},
	}
	for _, repo := range createdRepos {
		mustCreate(ctx, t, repo)
	}

	type testcase struct {
		limit  int
		offset int
		exp    []api.RepoName
	}
	tests := []testcase{
		{limit: 1, offset: 0, exp: []api.RepoName{"r1"}},
		{limit: 1, offset: 1, exp: []api.RepoName{"r2"}},
		{limit: 1, offset: 2, exp: []api.RepoName{"r3"}},
		{limit: 2, offset: 0, exp: []api.RepoName{"r1", "r2"}},
		{limit: 2, offset: 2, exp: []api.RepoName{"r3"}},
		{limit: 3, offset: 0, exp: []api.RepoName{"r1", "r2", "r3"}},
		{limit: 3, offset: 3, exp: nil},
		{limit: 4, offset: 0, exp: []api.RepoName{"r1", "r2", "r3"}},
		{limit: 4, offset: 4, exp: nil},
	}
	for _, test := range tests {
		repos, err := Repos.List(ctx, ReposListOptions{LimitOffset: &LimitOffset{Limit: test.limit, Offset: test.offset}})
		if err != nil {
			t.Fatal(err)
		}
		if got := sortedRepoNames(repos); !reflect.DeepEqual(got, test.exp) {
			t.Errorf("for test case %v, got %v (want %v)", test, got, test.exp)
		}
	}
}

// TestRepos_List_query tests the behavior of Repos.List when called with
// a query.
// Test batch 1 (correct filtering)
func TestRepos_List_query1(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	MockAuthzFilter = func(ctx context.Context, repos []*types.Repo, p authz.Perms) ([]*types.Repo, error) {
		return repos, nil
	}
	defer func() { MockAuthzFilter = nil }()
	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()
	ctx = actor.WithActor(ctx, &actor.Actor{})

	createdRepos := []*types.Repo{
		{Name: "abc/def"},
		{Name: "def/ghi"},
		{Name: "jkl/mno/pqr"},
		{Name: "github.com/abc/xyz"},
	}
	for _, repo := range createdRepos {
		createRepo(ctx, t, repo)
	}
	tests := []struct {
		query string
		want  []api.RepoName
	}{
		{"def", []api.RepoName{"abc/def", "def/ghi"}},
		{"ABC/DEF", []api.RepoName{"abc/def"}},
		{"xyz", []api.RepoName{"github.com/abc/xyz"}},
		{"mno/p", []api.RepoName{"jkl/mno/pqr"}},
	}
	for _, test := range tests {
		repos, err := Repos.List(ctx, ReposListOptions{Query: test.query})
		if err != nil {
			t.Fatal(err)
		}
		if got := repoNames(repos); !reflect.DeepEqual(got, test.want) {
			t.Errorf("%q: got repos %q, want %q", test.query, got, test.want)
		}
	}
}

// Test batch 2 (correct ranking)
func TestRepos_List_query2(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	MockAuthzFilter = func(ctx context.Context, repos []*types.Repo, p authz.Perms) ([]*types.Repo, error) {
		return repos, nil
	}
	defer func() { MockAuthzFilter = nil }()
	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()
	ctx = actor.WithActor(ctx, &actor.Actor{})

	createdRepos := []*types.Repo{
		{Name: "a/def"},
		{Name: "b/def"},
		{Name: "c/def"},
		{Name: "def/ghi"},
		{Name: "def/jkl"},
		{Name: "def/mno"},
		{Name: "abc/m"},
	}
	for _, repo := range createdRepos {
		createRepo(ctx, t, repo)
	}
	tests := []struct {
		query string
		want  []api.RepoName
	}{
		{"def", []api.RepoName{"a/def", "b/def", "c/def", "def/ghi", "def/jkl", "def/mno"}},
		{"b/def", []api.RepoName{"b/def"}},
		{"def/", []api.RepoName{"def/ghi", "def/jkl", "def/mno"}},
		{"def/m", []api.RepoName{"def/mno"}},
	}
	for _, test := range tests {
		repos, err := Repos.List(ctx, ReposListOptions{Query: test.query})
		if err != nil {
			t.Fatal(err)
		}
		if got := repoNames(repos); !reflect.DeepEqual(got, test.want) {
			t.Errorf("Unexpected repo result for query %q:\ngot:  %q\nwant: %q", test.query, got, test.want)
		}
	}
}

// Test sort
func TestRepos_List_sort(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	MockAuthzFilter = func(ctx context.Context, repos []*types.Repo, p authz.Perms) ([]*types.Repo, error) {
		return repos, nil
	}
	defer func() { MockAuthzFilter = nil }()
	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()
	ctx = actor.WithActor(ctx, &actor.Actor{})

	createdRepos := []*types.Repo{
		{Name: "c/def"},
		{Name: "def/mno"},
		{Name: "b/def"},
		{Name: "abc/m"},
		{Name: "abc/def"},
		{Name: "def/jkl"},
		{Name: "def/ghi"},
	}
	for _, repo := range createdRepos {
		createRepo(ctx, t, repo)
	}
	tests := []struct {
		query   string
		orderBy RepoListOrderBy
		want    []api.RepoName
	}{
		{
			query: "",
			orderBy: RepoListOrderBy{{
				Field: RepoListName,
			}},
			want: []api.RepoName{"abc/def", "abc/m", "b/def", "c/def", "def/ghi", "def/jkl", "def/mno"},
		},
		{
			query: "",
			orderBy: RepoListOrderBy{{
				Field: RepoListCreatedAt,
			}},
			want: []api.RepoName{"c/def", "def/mno", "b/def", "abc/m", "abc/def", "def/jkl", "def/ghi"},
		},
		{
			query: "",
			orderBy: RepoListOrderBy{{
				Field:      RepoListCreatedAt,
				Descending: true,
			}},
			want: []api.RepoName{"def/ghi", "def/jkl", "abc/def", "abc/m", "b/def", "def/mno", "c/def"},
		},
		{
			query: "def",
			orderBy: RepoListOrderBy{{
				Field:      RepoListCreatedAt,
				Descending: true,
			}},
			want: []api.RepoName{"def/ghi", "def/jkl", "abc/def", "b/def", "def/mno", "c/def"},
		},
	}
	for _, test := range tests {
		repos, err := Repos.List(ctx, ReposListOptions{Query: test.query, OrderBy: test.orderBy})
		if err != nil {
			t.Fatal(err)
		}
		if got := repoNames(repos); !reflect.DeepEqual(got, test.want) {
			t.Errorf("Unexpected repo result for query %q, orderBy %v:\ngot:  %q\nwant: %q", test.query, test.orderBy, got, test.want)
		}
	}
}

// TestRepos_List_patterns tests the behavior of Repos.List when called with
// IncludePatterns and ExcludePattern.
func TestRepos_List_patterns(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	MockAuthzFilter = func(ctx context.Context, repos []*types.Repo, p authz.Perms) ([]*types.Repo, error) {
		return repos, nil
	}
	defer func() { MockAuthzFilter = nil }()
	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()
	ctx = actor.WithActor(ctx, &actor.Actor{})

	createdRepos := []*types.Repo{
		{Name: "a/b"},
		{Name: "c/d"},
		{Name: "e/f"},
		{Name: "g/h"},
	}
	for _, repo := range createdRepos {
		createRepo(ctx, t, repo)
	}
	tests := []struct {
		includePatterns []string
		excludePattern  string
		want            []api.RepoName
	}{
		{
			includePatterns: []string{"(a|c)"},
			want:            []api.RepoName{"a/b", "c/d"},
		},
		{
			includePatterns: []string{"(a|c)", "b"},
			want:            []api.RepoName{"a/b"},
		},
		{
			includePatterns: []string{"(a|c)"},
			excludePattern:  "d",
			want:            []api.RepoName{"a/b"},
		},
		{
			excludePattern: "(d|e)",
			want:           []api.RepoName{"a/b", "g/h"},
		},
	}
	for _, test := range tests {
		repos, err := Repos.List(ctx, ReposListOptions{
			IncludePatterns: test.includePatterns,
			ExcludePattern:  test.excludePattern,
		})
		if err != nil {
			t.Fatal(err)
		}
		if got := repoNames(repos); !reflect.DeepEqual(got, test.want) {
			t.Errorf("include %q exclude %q: got repos %q, want %q", test.includePatterns, test.excludePattern, got, test.want)
		}
	}
}

// TestRepos_List_patterns tests the behavior of Repos.List when called with
// a QueryPattern.
func TestRepos_List_queryPattern(t *testing.T) {
	MockAuthzFilter = func(ctx context.Context, repos []*types.Repo, p authz.Perms) ([]*types.Repo, error) {
		return repos, nil
	}
	defer func() { MockAuthzFilter = nil }()
	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()
	ctx = actor.WithActor(ctx, &actor.Actor{})

	createdRepos := []*types.Repo{
		{Name: "a/b"},
		{Name: "c/d"},
		{Name: "e/f"},
		{Name: "g/h"},
	}
	for _, repo := range createdRepos {
		createRepo(ctx, t, repo)
	}
	tests := []struct {
		q    query.Q
		want []api.RepoName
		err  string
	}{
		// These are the same tests as TestRepos_List_patterns, but in an
		// expression form.
		{
			q:    "(a|c)",
			want: []api.RepoName{"a/b", "c/d"},
		},
		{
			q:    query.And("(a|c)", "b"),
			want: []api.RepoName{"a/b"},
		},
		{
			q:    query.And("(a|c)", query.Not("d")),
			want: []api.RepoName{"a/b"},
		},
		{
			q:    query.Not("(d|e)"),
			want: []api.RepoName{"a/b", "g/h"},
		},

		// Some extra tests which test the pattern compiler
		{
			q:    "",
			want: []api.RepoName{"a/b", "c/d", "e/f", "g/h"},
		},
		{
			q:    "^a/b$",
			want: []api.RepoName{"a/b"},
		},
		{
			// Should match only e/f, but pattern compiler doesn't handle this
			// so matches nothing.
			q:    "[a-zA-Z]/e",
			want: nil,
		},

		// Test OR support
		{
			q:    query.Or(query.Not("(d|e)"), "d"),
			want: []api.RepoName{"a/b", "c/d", "g/h"},
		},

		// Test deeply nested
		{
			q: query.Or(
				query.And(
					true,
					query.Not(query.Or("a", "c"))),
				query.And(query.Not("e"), query.Not("a"))),
			want: []api.RepoName{"c/d", "e/f", "g/h"},
		},

		// Corner cases for Or
		{
			q:    query.Or(), // empty Or is false
			want: nil,
		},
		{
			q:    query.Or("a"),
			want: []api.RepoName{"a/b"},
		},

		// Corner cases for And
		{
			q:    query.And(), // empty And is true
			want: []api.RepoName{"a/b", "c/d", "e/f", "g/h"},
		},
		{
			q:    query.And("a"),
			want: []api.RepoName{"a/b"},
		},
		{
			q:    query.And("a", "d"),
			want: nil,
		},

		// Bad pattern
		{
			q:   query.And("a/b", ")*"),
			err: "error parsing regexp",
		},
		// Only want strings
		{
			q:   query.And("a/b", 1),
			err: "unexpected token",
		},
	}
	for _, test := range tests {
		repos, err := Repos.List(ctx, ReposListOptions{
			PatternQuery: test.q,
		})
		if err != nil {
			if test.err == "" {
				t.Fatal(err)
			}
			if !strings.Contains(err.Error(), test.err) {
				t.Errorf("expected error to contain %q, got: %v", test.err, err)
			}
			continue
		}
		if test.err != "" {
			t.Errorf("%s: expected error", query.Print(test.q))
			continue
		}
		if got := repoNames(repos); !reflect.DeepEqual(got, test.want) {
			t.Errorf("%s: got repos %q, want %q", query.Print(test.q), got, test.want)
		}
	}
}

func TestRepos_List_queryAndPatternsMutuallyExclusive(t *testing.T) {
	ctx := context.Background()
	wantErr := "Query and IncludePatterns/ExcludePattern options are mutually exclusive"

	t.Run("Query and IncludePatterns", func(t *testing.T) {
		_, err := Repos.List(ctx, ReposListOptions{Query: "x", IncludePatterns: []string{"y"}})
		if err == nil || !strings.Contains(err.Error(), wantErr) {
			t.Fatalf("got error %v, want it to contain %q", err, wantErr)
		}
	})

	t.Run("Query and ExcludePattern", func(t *testing.T) {
		_, err := Repos.List(ctx, ReposListOptions{Query: "x", ExcludePattern: "y"})
		if err == nil || !strings.Contains(err.Error(), wantErr) {
			t.Fatalf("got error %v, want it to contain %q", err, wantErr)
		}
	})
}

func TestRepos_Create(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	// Add a repo.
	createRepo(ctx, t, &types.Repo{
		Name:       "a/b",
		RepoFields: &types.RepoFields{Description: "test"}})

	repo, err := Repos.GetByName(ctx, "a/b")
	if err != nil {
		t.Fatal(err)
	}

	if got, want := repo.Name, api.RepoName("a/b"); got != want {
		t.Fatalf("got Name %q, want %q", got, want)
	}
	if got, want := repo.Description, "test"; got != want {
		t.Fatalf("got Description %q, want %q", got, want)
	}
}

func TestRepos_Create_dupe(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	// Add a repo.
	createRepo(ctx, t, &types.Repo{Name: "a/b"})

	// Add another repo with the same name.
	createRepo(ctx, t, &types.Repo{Name: "a/b"})
}
