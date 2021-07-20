package graphqlbackend

import (
	"context"
	"testing"

	"github.com/cockroachdb/errors"
	gqlerrors "github.com/graph-gophers/graphql-go/errors"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

func TestRepositories(t *testing.T) {
	resetMocks()
	repos := []*types.Repo{
		{Name: "repo1"},
		{Name: "repo2"},
		{
			Name: "repo3",
		},
	}
	database.Mocks.Repos.List = func(ctx context.Context, opt database.ReposListOptions) ([]*types.Repo, error) {
		if opt.NoCloned {
			return repos[0:2], nil
		}
		if opt.OnlyCloned {
			return repos[2:], nil
		}

		return repos, nil
	}

	database.Mocks.Repos.Count = func(context.Context, database.ReposListOptions) (int, error) { return 3, nil }

	// Test as non site admin first
	database.Mocks.Users.GetByCurrentAuthUser = func(ctx context.Context) (*types.User, error) {
		return &types.User{
			ID:        1,
			SiteAdmin: false,
		}, nil
	}

	RunTests(t, []*Test{
		{
			Schema: mustParseGraphQLSchema(t),
			Query: `
				{
					repositories {
						nodes { name }
						totalCount
						pageInfo { hasNextPage }
					}
				}
			`,
			ExpectedResult: `
				{
					"repositories": {
						"nodes": [
							{ "name": "repo1" },
							{ "name": "repo2" },
							{ "name": "repo3" }
						],
						"totalCount": null,
						"pageInfo": {"hasNextPage": false}
					}
				}
			`,
			ExpectedErrors: []*gqlerrors.QueryError{
				{
					Path:          []interface{}{"repositories", "totalCount"},
					Message:       backend.ErrMustBeSiteAdmin.Error(),
					ResolverError: backend.ErrMustBeSiteAdmin,
				},
			},
		},
	})

	// Then test as site admin
	database.Mocks.Users.GetByCurrentAuthUser = func(ctx context.Context) (*types.User, error) {
		return &types.User{
			ID:        1,
			SiteAdmin: true,
		}, nil
	}

	RunTests(t, []*Test{
		{
			Schema: mustParseGraphQLSchema(t),
			Query: `
				{
					repositories {
						nodes { name }
						totalCount
						pageInfo { hasNextPage }
					}
				}
			`,
			ExpectedResult: `
				{
					"repositories": {
						"nodes": [
							{ "name": "repo1" },
							{ "name": "repo2" },
							{ "name": "repo3" }
						],
						"totalCount": 3,
						"pageInfo": {"hasNextPage": false}
					}
				}
			`,
		},
		{
			Schema: mustParseGraphQLSchema(t),
			// cloned and notCloned are true by default
			// this test ensures the behavior is the same
			// when setting them explicitly
			Query: `
				{
					repositories(cloned: true, notCloned: true) {
						nodes { name }
						totalCount
						pageInfo { hasNextPage }
					}
				}
			`,
			ExpectedResult: `
				{
					"repositories": {
						"nodes": [
							{ "name": "repo1" },
							{ "name": "repo2" },
							{ "name": "repo3" }
						],
						"totalCount": 3,
						"pageInfo": {"hasNextPage": false}
					}
				}
			`,
		},
		{
			Schema: mustParseGraphQLSchema(t),
			Query: `
				{
					repositories(first: 2) {
						nodes { name }
						pageInfo { hasNextPage }
					}
				}
			`,
			ExpectedResult: `
				{
					"repositories": {
						"nodes": [
							{ "name": "repo1" },
							{ "name": "repo2" }
						],
						"pageInfo": {"hasNextPage": true}
					}
				}
			`,
		},
		{
			Schema: mustParseGraphQLSchema(t),
			Query: `
				{
					repositories(cloned: false) {
						nodes { name }
						pageInfo { hasNextPage }
					}
				}
			`,
			ExpectedResult: `
				{
					"repositories": {
						"nodes": [
							{ "name": "repo1" },
							{ "name": "repo2" }
						],
						"pageInfo": {"hasNextPage": false}
					}
				}
			`,
		},
		{
			Schema: mustParseGraphQLSchema(t),
			Query: `
				{
					repositories(notCloned: false) {
						nodes { name }
						pageInfo { hasNextPage }
					}
				}
			`,
			ExpectedResult: `
				{
					"repositories": {
						"nodes": [
							{ "name": "repo3" }
						],
						"pageInfo": {"hasNextPage": false}
					}
				}
			`,
		},
		{
			Schema: mustParseGraphQLSchema(t),
			Query: `
				{
					repositories(notCloned: false, cloned: false) {
						nodes { name }
						pageInfo { hasNextPage }
					}
				}
			`,
			ExpectedResult: `
				{
					"repositories": {
						"nodes": [
							{ "name": "repo1" },
							{ "name": "repo2" }
						],
						"pageInfo": {"hasNextPage": false}
					}
				}
			`,
		},
	})
}

func TestRepositories_CursorPagination(t *testing.T) {
	resetMocks()

	repos := []*types.Repo{
		{ID: 0, Name: "repo1"},
		{ID: 1, Name: "repo2"},
		{ID: 2, Name: "repo3"},
	}

	t.Run("Initial page without a cursor present", func(t *testing.T) {
		database.Mocks.Repos.List = func(ctx context.Context, opt database.ReposListOptions) ([]*types.Repo, error) {
			return repos[0:2], nil
		}
		defer func() { database.Mocks.Repos.List = nil }()

		RunTests(t, []*Test{
			{
				Schema: mustParseGraphQLSchema(t),
				Query: `
				{
					repositories(first: 1) {
						nodes {
							name
						}
						pageInfo {
							endCursor
						}
					}
				}
			`,
				ExpectedResult: `
				{
					"repositories": {
						"nodes": [{
							"name": "repo1"
						}],
						"pageInfo": {
						  "endCursor": "UmVwb3NpdG9yeUN1cnNvcjp7IkNvbHVtbiI6Im5hbWUiLCJWYWx1ZSI6InJlcG8yIiwiRGlyZWN0aW9uIjoibmV4dCJ9"
						}
					}
				}
			`,
			},
		})
	})

	t.Run("Second page in ascending order", func(t *testing.T) {
		database.Mocks.Repos.List = func(ctx context.Context, opt database.ReposListOptions) ([]*types.Repo, error) {
			return repos[1:], nil
		}
		defer func() { database.Mocks.Repos.List = nil }()

		RunTests(t, []*Test{
			{
				Schema: mustParseGraphQLSchema(t),
				Query: `
				{
					repositories(first: 1, after: "UmVwb3NpdG9yeUN1cnNvcjp7IkNvbHVtbiI6Im5hbWUiLCJWYWx1ZSI6InJlcG8yIiwiRGlyZWN0aW9uIjoibmV4dCJ9") {
						nodes {
							name
						}
						pageInfo {
							endCursor
						}
					}
				}
			`,
				ExpectedResult: `
				{
					"repositories": {
						"nodes": [{
							"name": "repo2"
						}],
						"pageInfo": {
						  "endCursor": "UmVwb3NpdG9yeUN1cnNvcjp7IkNvbHVtbiI6Im5hbWUiLCJWYWx1ZSI6InJlcG8zIiwiRGlyZWN0aW9uIjoibmV4dCJ9"
						}
					}
				}
			`,
			},
		})
	})

	t.Run("Second page in descending order", func(t *testing.T) {
		database.Mocks.Repos.List = func(ctx context.Context, opt database.ReposListOptions) ([]*types.Repo, error) {
			return repos[1:], nil
		}
		defer func() { database.Mocks.Repos.List = nil }()

		RunTests(t, []*Test{
			{
				Schema: mustParseGraphQLSchema(t),
				Query: `
				{
					repositories(first: 1, after: "UmVwb3NpdG9yeUN1cnNvcjp7IkNvbHVtbiI6Im5hbWUiLCJWYWx1ZSI6InJlcG8yIiwiRGlyZWN0aW9uIjoicHJldiJ9", descending: true) {
						nodes {
							name
						}
						pageInfo {
							endCursor
						}
					}
				}
			`,
				ExpectedResult: `
				{
					"repositories": {
						"nodes": [{
							"name": "repo2"
						}],
						"pageInfo": {
						  "endCursor": "UmVwb3NpdG9yeUN1cnNvcjp7IkNvbHVtbiI6Im5hbWUiLCJWYWx1ZSI6InJlcG8zIiwiRGlyZWN0aW9uIjoicHJldiJ9"
						}
					}
				}
			`,
			},
		})
	})

	t.Run("Initial page with no further rows to fetch", func(t *testing.T) {
		database.Mocks.Repos.List = func(ctx context.Context, opt database.ReposListOptions) ([]*types.Repo, error) {
			return repos, nil
		}
		defer func() { database.Mocks.Repos.List = nil }()

		RunTests(t, []*Test{
			{
				Schema: mustParseGraphQLSchema(t),
				Query: `
				{
					repositories(first: 3) {
						nodes {
							name
						}
						pageInfo {
							endCursor
						}
					}
				}
			`,
				ExpectedResult: `
				{
					"repositories": {
						"nodes": [{
							"name": "repo1"
						}, {
							"name": "repo2"
						}, {
							"name": "repo3"
						}],
						"pageInfo": {
						  "endCursor": null
						}
					}
				}
			`,
			},
		})
	})

	t.Run("With no repositories present", func(t *testing.T) {
		database.Mocks.Repos.List = func(ctx context.Context, opt database.ReposListOptions) ([]*types.Repo, error) {
			return nil, nil
		}
		defer func() { database.Mocks.Repos.List = nil }()

		RunTests(t, []*Test{
			{
				Schema: mustParseGraphQLSchema(t),
				Query: `
				{
					repositories(first: 1) {
						nodes {
							name
						}
						pageInfo {
							endCursor
						}
					}
				}
			`,
				ExpectedResult: `
				{
					"repositories": {
						"nodes": [],
						"pageInfo": {
						  "endCursor": null
						}
					}
				}
			`,
			},
		})
	})

	t.Run("With an invalid cursor provided", func(t *testing.T) {
		database.Mocks.Repos.List = func(ctx context.Context, opt database.ReposListOptions) ([]*types.Repo, error) {
			return nil, nil
		}
		defer func() { database.Mocks.Repos.List = nil }()

		RunTests(t, []*Test{
			{
				Schema: mustParseGraphQLSchema(t),
				Query: `
				{
					repositories(first: 1, after: "invalid-cursor-value") {
						nodes {
							name
						}
						pageInfo {
							endCursor
						}
					}
				}
			`,
				ExpectedResult: "null",
				ExpectedErrors: []*gqlerrors.QueryError{
					{
						Path:          []interface{}{"repositories"},
						Message:       `cannot unmarshal repository cursor type: ""`,
						ResolverError: errors.Errorf(`cannot unmarshal repository cursor type: ""`),
					},
				},
			},
		})
	})
}
