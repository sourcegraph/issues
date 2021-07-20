package queryrunner

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"

	"github.com/sourcegraph/sourcegraph/internal/api"

	"golang.org/x/net/context/ctxhttp"

	"github.com/cockroachdb/errors"
)

// This file contains all the methods required to execute Sourcegraph searches using our GraphQL
// API and get results back.

// graphQLQuery describes a general GraphQL query and its variables.
type graphQLQuery struct {
	Query     string      `json:"query"`
	Variables interface{} `json:"variables"`
}

const gqlSearchQuery = `query Search(
	$query: String!,
) {
	search(query: $query, version: V2, patternType:literal) {
		results {
			limitHit
			cloning { name }
			missing { name }
			timedout { name }
			matchCount
			results {
				__typename
				... on FileMatch {
					repository {
						id
					}
					lineMatches {
						offsetAndLengths
					}
					symbols {
						name
					}
				}
				... on CommitSearchResult {
					matches {
						highlights {
							line
						}
					}
					commit {
						repository {
							id
						}
					}
				}
				... on Repository {
					id
				}
			}
			alert {
				title
				description
			}
		}
	}
}`

type gqlSearchVars struct {
	Query string `json:"query"`
}

type gqlSearchResponse struct {
	Data struct {
		Search struct {
			Results struct {
				LimitHit   bool
				Cloning    []*api.Repo
				Missing    []*api.Repo
				Timedout   []*api.Repo
				MatchCount int
				Results    []json.RawMessage
				Alert      *struct {
					Title       string
					Description string
				}
			}
		}
	}
	Errors []interface{}
}

// search executes the given search query.
func search(ctx context.Context, query string) (*gqlSearchResponse, error) {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(graphQLQuery{
		Query:     gqlSearchQuery,
		Variables: gqlSearchVars{Query: query},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Encode")
	}

	url, err := gqlURL("InsightsSearch")
	if err != nil {
		return nil, errors.Wrap(err, "constructing frontend URL")
	}

	resp, err := ctxhttp.Post(ctx, nil, url, "application/json", &buf)
	if err != nil {
		return nil, errors.Wrap(err, "Post")
	}
	defer resp.Body.Close()

	var res *gqlSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, errors.Wrap(err, "Decode")
	}
	if len(res.Errors) > 0 {
		return res, errors.Errorf("graphql: errors: %v", res.Errors)
	}
	return res, nil
}

// gqlURL returns the frontend's internal GraphQL API URL, with the given ?queryName parameter
// which is used to keep track of the source and type of GraphQL queries.
func gqlURL(queryName string) (string, error) {
	u, err := url.Parse(api.InternalClient.URL)
	if err != nil {
		return "", err
	}
	u.Path = "/.internal/graphql"
	u.RawQuery = queryName
	return u.String(), nil
}
