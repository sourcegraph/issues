package graphqlbackend

import (
	"context"
	"errors"
	"testing"

	gqlerrors "github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/gqltesting"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/db"
)

func TestRepositories(t *testing.T) {
	resetMocks()
	repos := []*types.Repo{
		{Name: "repo1"},
		{Name: "repo2"},
		{
			Name: "repo3",
			RepoFields: &types.RepoFields{
				Cloned: true,
			},
		},
	}
	db.Mocks.Repos.List = func(ctx context.Context, opt db.ReposListOptions) ([]*types.Repo, error) {
		if opt.NoCloned {
			return repos[0:2], nil
		}
		if opt.OnlyCloned {
			return repos[2:], nil
		}

		return repos, nil
	}

	db.Mocks.Repos.Count = func(context.Context, db.ReposListOptions) (int, error) { return 3, nil }
	gqltesting.RunTests(t, []*gqltesting.Test{
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
						"totalCount": null,
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
		db.Mocks.Repos.List = func(ctx context.Context, opt db.ReposListOptions) ([]*types.Repo, error) {
			return repos[0:2], nil
		}
		defer func() { db.Mocks.Repos.List = nil }()

		gqltesting.RunTests(t, []*gqltesting.Test{
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
		db.Mocks.Repos.List = func(ctx context.Context, opt db.ReposListOptions) ([]*types.Repo, error) {
			return repos[1:], nil
		}
		defer func() { db.Mocks.Repos.List = nil }()

		gqltesting.RunTests(t, []*gqltesting.Test{
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
		db.Mocks.Repos.List = func(ctx context.Context, opt db.ReposListOptions) ([]*types.Repo, error) {
			return repos[1:], nil
		}
		defer func() { db.Mocks.Repos.List = nil }()

		gqltesting.RunTests(t, []*gqltesting.Test{
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
		db.Mocks.Repos.List = func(ctx context.Context, opt db.ReposListOptions) ([]*types.Repo, error) {
			return repos, nil
		}
		defer func() { db.Mocks.Repos.List = nil }()

		gqltesting.RunTests(t, []*gqltesting.Test{
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
		db.Mocks.Repos.List = func(ctx context.Context, opt db.ReposListOptions) ([]*types.Repo, error) {
			return nil, nil
		}
		defer func() { db.Mocks.Repos.List = nil }()

		gqltesting.RunTests(t, []*gqltesting.Test{
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
		db.Mocks.Repos.List = func(ctx context.Context, opt db.ReposListOptions) ([]*types.Repo, error) {
			return nil, nil
		}
		defer func() { db.Mocks.Repos.List = nil }()

		gqltesting.RunTests(t, []*gqltesting.Test{
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
						ResolverError: errors.New(`cannot unmarshal repository cursor type: ""`),
						Message:       `cannot unmarshal repository cursor type: ""`,
						Path:          []interface{}{"repositories"},
					},
				},
			},
		})
	})
}
