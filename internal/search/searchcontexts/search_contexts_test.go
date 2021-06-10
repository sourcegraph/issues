package searchcontexts

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/schema"
)

func TestResolvingValidSearchContextSpecs(t *testing.T) {
	tests := []struct {
		name                  string
		searchContextSpec     string
		wantSearchContextName string
	}{
		{name: "resolve user search context", searchContextSpec: "@user", wantSearchContextName: "user"},
		{name: "resolve global search context", searchContextSpec: "global", wantSearchContextName: "global"},
		{name: "resolve empty search context as global", searchContextSpec: "", wantSearchContextName: "global"},
		{name: "resolve namespaced search context", searchContextSpec: "@user/test", wantSearchContextName: "test"},
		{name: "resolve namespaced search context with / in name", searchContextSpec: "@user/test/version", wantSearchContextName: "test/version"},
	}

	db := new(dbtesting.MockDB)
	database.Mocks.Namespaces.GetByName = func(ctx context.Context, name string) (*database.Namespace, error) {
		return &database.Namespace{Name: name, User: 1}, nil
	}
	database.Mocks.SearchContexts.GetSearchContext = func(ctx context.Context, opts database.GetSearchContextOptions) (*types.SearchContext, error) {
		return &types.SearchContext{Name: opts.Name}, nil
	}
	defer func() {
		database.Mocks.Namespaces.GetByName = nil
		database.Mocks.SearchContexts.GetSearchContext = nil
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			searchContext, err := ResolveSearchContextSpec(context.Background(), db, tt.searchContextSpec)
			if err != nil {
				t.Fatal(err)
			}
			if searchContext.Name != tt.wantSearchContextName {
				t.Fatalf("got %q, expected %q", searchContext.Name, tt.wantSearchContextName)
			}
		})
	}
}

func TestResolvingInvalidSearchContextSpecs(t *testing.T) {
	tests := []struct {
		name              string
		searchContextSpec string
		wantErr           string
	}{
		{name: "invalid format", searchContextSpec: "+user", wantErr: "search context not found"},
		{name: "user not found", searchContextSpec: "@user", wantErr: "search context \"@user\" not found"},
		{name: "empty user not found", searchContextSpec: "@", wantErr: "search context not found"},
	}

	db := new(dbtesting.MockDB)
	database.Mocks.Namespaces.GetByName = func(ctx context.Context, name string) (*database.Namespace, error) {
		return &database.Namespace{}, nil
	}
	database.Mocks.SearchContexts.GetSearchContext = func(ctx context.Context, opts database.GetSearchContextOptions) (*types.SearchContext, error) {
		return nil, errors.New("search context not found")
	}
	defer func() {
		database.Mocks.Namespaces.GetByName = nil
		database.Mocks.SearchContexts.GetSearchContext = nil
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveSearchContextSpec(context.Background(), db, tt.searchContextSpec)
			if err == nil {
				t.Fatal("Expected error, but there was none")
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("err: got %q, expected %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestConstructingSearchContextSpecs(t *testing.T) {
	tests := []struct {
		name                  string
		searchContext         *types.SearchContext
		wantSearchContextSpec string
	}{
		{name: "global search context", searchContext: GetGlobalSearchContext(), wantSearchContextSpec: "global"},
		{name: "user auto-defined search context", searchContext: &types.SearchContext{Name: "user", NamespaceUserID: 1}, wantSearchContextSpec: "@user"},
		{name: "org auto-defined search context", searchContext: &types.SearchContext{Name: "org", NamespaceOrgID: 1}, wantSearchContextSpec: "@org"},
		{name: "user namespaced search context", searchContext: &types.SearchContext{ID: 1, Name: "context", NamespaceUserID: 1, NamespaceUserName: "user"}, wantSearchContextSpec: "@user/context"},
		{name: "org namespaced search context", searchContext: &types.SearchContext{ID: 1, Name: "context", NamespaceOrgID: 1, NamespaceOrgName: "org"}, wantSearchContextSpec: "@org/context"},
		{name: "instance-level search context", searchContext: &types.SearchContext{ID: 1, Name: "instance-level-context"}, wantSearchContextSpec: "instance-level-context"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			searchContextSpec := GetSearchContextSpec(tt.searchContext)
			if searchContextSpec != tt.wantSearchContextSpec {
				t.Fatalf("got %q, expected %q", searchContextSpec, tt.wantSearchContextSpec)
			}
		})
	}
}

func TestGettingSearchContextFromVersionContext(t *testing.T) {
	tests := []struct {
		name              string
		versionContext    *schema.VersionContext
		wantSearchContext *types.SearchContext
	}{
		{
			name:              "simple version context",
			versionContext:    &schema.VersionContext{Name: "vc1", Description: "vc1 description"},
			wantSearchContext: &types.SearchContext{Name: "vc1", Description: "vc1 description", Public: true},
		},
		{
			name:              "version context with spaces in name",
			versionContext:    &schema.VersionContext{Name: "Version Context  2", Description: "Version Context 2 description"},
			wantSearchContext: &types.SearchContext{Name: "Version_Context_2", Description: "Version Context 2 description", Public: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSearchContext := getSearchContextFromVersionContext(tt.versionContext)
			if !reflect.DeepEqual(tt.wantSearchContext, gotSearchContext) {
				t.Fatalf("want %+v, got %+v", tt.wantSearchContext, gotSearchContext)
			}
		})
	}
}

func createRepos(ctx context.Context, repoStore *database.RepoStore) ([]types.RepoName, error) {
	err := repoStore.Create(ctx, &types.Repo{Name: "github.com/example/a"}, &types.Repo{Name: "github.com/example/b"})
	if err != nil {
		return nil, err
	}
	repoA, err := repoStore.GetByName(ctx, "github.com/example/a")
	if err != nil {
		return nil, err
	}
	repoB, err := repoStore.GetByName(ctx, "github.com/example/b")
	if err != nil {
		return nil, err
	}
	return []types.RepoName{{ID: repoA.ID, Name: repoA.Name}, {ID: repoB.ID, Name: repoB.Name}}, nil
}

func TestConvertingVersionContextToSearchContext(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	internalCtx := actor.WithInternalActor(context.Background())
	db := dbtest.NewDB(t, "")
	u := database.Users(db)

	user, err := u.Create(internalCtx, database.NewUser{Username: "u", Password: "p"})
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}
	ctx := actor.WithActor(context.Background(), &actor.Actor{UID: user.ID})
	r := database.Repos(db)

	repos, err := createRepos(ctx, r)
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	versionContext := &schema.VersionContext{
		Name:        "vc1",
		Description: "vc1 description",
		Revisions: []*schema.VersionContextRevision{
			{Repo: "github.com/example/a", Rev: "branch-1"},
			{Repo: "github.com/example/a", Rev: "branch-3"},
			{Repo: "github.com/example/b", Rev: "branch-2"},
		},
	}

	wantRepositoryRevisions := []*types.SearchContextRepositoryRevisions{
		{Repo: repos[0], Revisions: []string{"branch-1", "branch-3"}},
		{Repo: repos[1], Revisions: []string{"branch-2"}},
	}
	wantSearchContext := &types.SearchContext{ID: 1, Name: "vc1", Description: "vc1 description", Public: true}

	gotSearchContext, err := ConvertVersionContextToSearchContext(ctx, db, versionContext)
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}
	// Ignore UpdateAt field
	gotSearchContext.UpdatedAt = wantSearchContext.UpdatedAt
	if !reflect.DeepEqual(wantSearchContext, gotSearchContext) {
		t.Fatalf("want search context %+v, got %+v", wantSearchContext, gotSearchContext)
	}

	gotRepositoryRevisions, err := database.SearchContexts(db).GetSearchContextRepositoryRevisions(ctx, gotSearchContext.ID)
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}
	if !reflect.DeepEqual(wantRepositoryRevisions, gotRepositoryRevisions) {
		t.Fatalf("wanted %+v repository revisions, got %+v", wantRepositoryRevisions, gotRepositoryRevisions)
	}
}

func TestResolvingSearchContextRepoNames(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	internalCtx := actor.WithInternalActor(context.Background())
	db := dbtest.NewDB(t, "")
	u := database.Users(db)
	r := database.Repos(db)

	user, err := u.Create(internalCtx, database.NewUser{Username: "u", Password: "p"})
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}
	repos, err := createRepos(internalCtx, r)

	ctx := actor.WithActor(context.Background(), &actor.Actor{UID: user.ID})
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}
	repositoryRevisions := []*types.SearchContextRepositoryRevisions{
		{Repo: repos[0], Revisions: []string{"branch-1"}},
		{Repo: repos[1], Revisions: []string{"branch-2"}},
	}

	searchContext, err := CreateSearchContextWithRepositoryRevisions(ctx, db, &types.SearchContext{Name: "searchcontext"}, repositoryRevisions)
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	gotRepos, err := r.ListRepoNames(ctx, database.ReposListOptions{SearchContextID: searchContext.ID})
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}
	if !reflect.DeepEqual(repos, gotRepos) {
		t.Fatalf("wanted %+v repositories, got %+v", repos, gotRepos)
	}
}

func TestSearchContextWriteAccessValidation(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	internalCtx := actor.WithInternalActor(context.Background())
	db := dbtest.NewDB(t, "")
	u := database.Users(db)

	org, err := database.Orgs(db).Create(internalCtx, "myorg", nil)
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}
	// First user is the site admin
	user1, err := u.Create(internalCtx, database.NewUser{Username: "u1", Password: "p"})
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}
	// Second user is not a site-admin and is a member of the org
	user2, err := u.Create(internalCtx, database.NewUser{Username: "u2", Password: "p"})
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}
	database.OrgMembers(db).Create(internalCtx, org.ID, user2.ID)
	// Third user is not a site-admin and is not a member of the org
	user3, err := u.Create(internalCtx, database.NewUser{Username: "u3", Password: "p"})
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	tests := []struct {
		name            string
		namespaceUserID int32
		namespaceOrgID  int32
		public          bool
		userID          int32
		wantErr         string
	}{
		{
			name:    "current user must be authenticated",
			userID:  0,
			wantErr: "current user not found",
		},
		{
			name:            "current user must match the user namespace",
			namespaceUserID: user2.ID,
			userID:          user3.ID,
			wantErr:         "search context user does not match current user",
		},
		{
			name:           "current user must be a member of the org namespace",
			namespaceOrgID: org.ID,
			userID:         user3.ID,
			wantErr:        "org member not found",
		},
		{
			name:    "non site-admin users are not valid for instance-level contexts",
			userID:  user2.ID,
			wantErr: "current user must be site-admin",
		},
		{
			name:            "site-admin is invalid for private user search context",
			namespaceUserID: user2.ID,
			userID:          user1.ID,
			wantErr:         "search context user does not match current user",
		},
		{
			name:           "site-admin is invalid for private org search context",
			namespaceOrgID: org.ID,
			userID:         user1.ID,
			wantErr:        "org member not found",
		},
		{
			name:   "site-admin is valid for private instance-level context",
			userID: user1.ID,
		},
		{
			name:            "site-admin is valid for any public user search context",
			namespaceUserID: user2.ID,
			public:          true,
			userID:          user1.ID,
		},
		{
			name:           "site-admin is valid for any public org search context",
			namespaceOrgID: org.ID,
			public:         true,
			userID:         user1.ID,
		},
		{
			name:            "current user is valid if matches the user namespace",
			namespaceUserID: user2.ID,
			userID:          user2.ID,
		},
		{
			name:           "current user is valid if a member of the org namespace",
			namespaceOrgID: org.ID,
			userID:         user2.ID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := actor.WithActor(context.Background(), &actor.Actor{UID: tt.userID})

			err := ValidateSearchContextWriteAccessForCurrentUser(ctx, db, tt.namespaceUserID, tt.namespaceOrgID, tt.public)

			expectErr := tt.wantErr != ""
			if !expectErr && err != nil {
				t.Fatalf("expected no error, got %s", err)
			}
			if expectErr && err == nil {
				t.Fatalf("wanted error, got none")
			}
			if expectErr && err != nil && !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("wanted error containing %s, got %s", tt.wantErr, err)
			}
		})
	}
}

func TestCreatingSearchContexts(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	internalCtx := actor.WithInternalActor(context.Background())
	db := dbtest.NewDB(t, "")
	u := database.Users(db)

	user1, err := u.Create(internalCtx, database.NewUser{Username: "u1", Password: "p"})
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}
	repos, err := createRepos(internalCtx, database.Repos(db))
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	existingSearchContext, err := database.SearchContexts(db).CreateSearchContextWithRepositoryRevisions(
		internalCtx,
		&types.SearchContext{Name: "existing"},
		[]*types.SearchContextRepositoryRevisions{},
	)
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	tooLongName := strings.Repeat("x", 33)
	tooLongRevision := strings.Repeat("x", 256)
	tests := []struct {
		name                string
		searchContext       *types.SearchContext
		userID              int32
		repositoryRevisions []*types.SearchContextRepositoryRevisions
		wantErr             string
	}{
		{
			name:          "cannot create search context with global name",
			searchContext: &types.SearchContext{Name: "global"},
			wantErr:       "cannot override global search context",
		},
		{
			name:          "cannot create search context with invalid name",
			searchContext: &types.SearchContext{Name: "invalid name"},
			userID:        user1.ID,
			wantErr:       "\"invalid name\" is not a valid search context name",
		},
		{
			name:          "can create search context with non-space separators",
			searchContext: &types.SearchContext{Name: "version_1.2-final/3"},
			userID:        user1.ID,
		},
		{
			name:          "cannot create search context with name too long",
			searchContext: &types.SearchContext{Name: tooLongName},
			userID:        user1.ID,
			wantErr:       fmt.Sprintf("search context name %q exceeds maximum allowed length (32)", tooLongName),
		},
		{
			name:          "cannot create search context with description too long",
			searchContext: &types.SearchContext{Name: "ctx", Description: strings.Repeat("x", 1025)},
			userID:        user1.ID,
			wantErr:       "search context description exceeds maximum allowed length (1024)",
		},
		{
			name:          "cannot create search context if it already exists",
			searchContext: existingSearchContext,
			userID:        user1.ID,
			wantErr:       "search context already exists",
		},
		{
			name:          "cannot create search context with revisions too long",
			searchContext: &types.SearchContext{Name: "ctx"},
			userID:        user1.ID,
			repositoryRevisions: []*types.SearchContextRepositoryRevisions{
				{Repo: repos[0], Revisions: []string{tooLongRevision}},
			},
			wantErr: fmt.Sprintf("revision %q exceeds maximum allowed length (255)", tooLongRevision),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := actor.WithActor(context.Background(), &actor.Actor{UID: tt.userID})

			_, err := CreateSearchContextWithRepositoryRevisions(ctx, db, tt.searchContext, tt.repositoryRevisions)

			expectErr := tt.wantErr != ""
			if !expectErr && err != nil {
				t.Fatalf("expected no error, got %s", err)
			}
			if expectErr && err == nil {
				t.Fatalf("wanted error, got none")
			}
			if expectErr && err != nil && !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("wanted error containing %s, got %s", tt.wantErr, err)
			}
		})
	}
}

func TestUpdatingSearchContexts(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	internalCtx := actor.WithInternalActor(context.Background())
	db := dbtest.NewDB(t, "")
	u := database.Users(db)

	user1, err := u.Create(internalCtx, database.NewUser{Username: "u1", Password: "p"})
	require.NoError(t, err)

	repos, err := createRepos(internalCtx, database.Repos(db))
	require.NoError(t, err)

	var scs []*types.SearchContext
	for i := 0; i < 6; i++ {
		sc, err := database.SearchContexts(db).CreateSearchContextWithRepositoryRevisions(
			internalCtx,
			&types.SearchContext{Name: strconv.Itoa(i)},
			[]*types.SearchContextRepositoryRevisions{},
		)
		require.NoError(t, err)
		scs = append(scs, sc)
	}

	set := func(sc *types.SearchContext, f func(*types.SearchContext)) *types.SearchContext {
		copied := *sc
		f(&copied)
		return &copied
	}

	tests := []struct {
		name                string
		update              *types.SearchContext
		repositoryRevisions []*types.SearchContextRepositoryRevisions
		userID              int32
		wantErr             string
	}{
		{
			name:    "cannot create search context with global name",
			update:  &types.SearchContext{Name: "global"},
			wantErr: "cannot update global search context",
		},
		{
			name:    "cannot update search context to use an invalid name",
			update:  set(scs[0], func(sc *types.SearchContext) { sc.Name = "invalid name" }),
			wantErr: "not a valid search context name",
		},
		{
			name:    "cannot update search context with name too long",
			update:  set(scs[1], func(sc *types.SearchContext) { sc.Name = strings.Repeat("x", 33) }),
			wantErr: "exceeds maximum allowed length (32)",
		},
		{
			name:    "cannot update search context with description too long",
			update:  set(scs[2], func(sc *types.SearchContext) { sc.Description = strings.Repeat("x", 1025) }),
			wantErr: "search context description exceeds maximum allowed length (1024)",
		},
		{
			name:   "cannot update search context with revisions too long",
			update: scs[3],
			repositoryRevisions: []*types.SearchContextRepositoryRevisions{
				{Repo: repos[0], Revisions: []string{strings.Repeat("x", 256)}},
			},
			wantErr: "exceeds maximum allowed length (255)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := actor.WithActor(context.Background(), &actor.Actor{UID: user1.ID})

			updated, err := UpdateSearchContextWithRepositoryRevisions(ctx, db, tt.update, tt.repositoryRevisions)
			if tt.wantErr != "" {
				require.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.update, updated)
		})
	}
}

func TestDeletingAutoDefinedSearchContext(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	internalCtx := actor.WithInternalActor(context.Background())
	db := dbtest.NewDB(t, "")
	u := database.Users(db)

	user1, err := u.Create(internalCtx, database.NewUser{Username: "u1", Password: "p"})
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	autoDefinedSearchContext := GetUserSearchContext(user1.Username, user1.ID)
	ctx := actor.WithActor(context.Background(), &actor.Actor{UID: user1.ID})
	err = DeleteSearchContext(ctx, db, autoDefinedSearchContext)

	wantErr := "cannot delete auto-defined search context"
	if err == nil {
		t.Fatalf("wanted error, got none")
	}
	if err != nil && !strings.Contains(err.Error(), wantErr) {
		t.Fatalf("wanted error containing %s, got %s", wantErr, err)
	}
}
