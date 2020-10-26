package gqltestutil

import (
	"github.com/pkg/errors"
)

type SearchRepositoryResult struct {
	Name string `json:"name"`
}

type SearchRepositoryResults []*SearchRepositoryResult

// Exists returns the list of missing repositories from given names that do not exist
// in search results. If all of given names are found, it returns empty list.
func (rs SearchRepositoryResults) Exists(names ...string) []string {
	set := make(map[string]struct{}, len(names))
	for _, name := range names {
		set[name] = struct{}{}
	}
	for _, r := range rs {
		delete(set, r.Name)
	}

	missing := make([]string, 0, len(set))
	for name := range set {
		missing = append(missing, name)
	}
	return missing
}

// SearchRepositories search repositories with given query.
func (c *Client) SearchRepositories(query string) (SearchRepositoryResults, error) {
	const gqlQuery = `
query Search($query: String!) {
	search(query: $query) {
		results {
			results {
				... on Repository {
					name
				}
			}
		}
	}
}
`
	variables := map[string]interface{}{
		"query": query,
	}
	var resp struct {
		Data struct {
			Search struct {
				Results struct {
					Results []*SearchRepositoryResult `json:"results"`
				} `json:"results"`
			} `json:"search"`
		} `json:"data"`
	}
	err := c.GraphQL("", "", gqlQuery, variables, &resp)
	if err != nil {
		return nil, errors.Wrap(err, "request GraphQL")
	}

	return resp.Data.Search.Results.Results, nil
}

type SearchFileResults struct {
	MatchCount int64        `json:"matchCount"`
	Alert      *SearchAlert `json:"alert"`
	Results    []*struct {
		File struct {
			Name string `json:"name"`
		} `json:"file"`
		Repository struct {
			Name string `json:"name"`
		} `json:"repository"`
		RevSpec struct {
			Expr string `json:"expr"`
		} `json:"revSpec"`
	} `json:"results"`
}

type ProposedQuery struct {
	Description string `json:"description"`
	Query       string `json:"query"`
}

// SearchAlert is an alert specific to searches (i.e. not site alert).
type SearchAlert struct {
	Title           string          `json:"title"`
	Description     string          `json:"description"`
	ProposedQueries []ProposedQuery `json:"proposedQueries"`
}

// SearchFiles searches files with given query. It returns the match count and
// corresponding file matches. Search alert is also included if any.
func (c *Client) SearchFiles(query string) (*SearchFileResults, error) {
	const gqlQuery = `
query Search($query: String!) {
	search(query: $query) {
		results {
			matchCount
			alert {
				title
				description
				proposedQueries {
					description
					query
				}
			}
			results {
				... on FileMatch {
					file {
						name
					}
					repository {
						name
					}
					revSpec {
						... on GitRevSpecExpr {
							expr
						}
					}
				}
			}
		}
	}
}
`
	variables := map[string]interface{}{
		"query": query,
	}
	var resp struct {
		Data struct {
			Search struct {
				Results struct {
					*SearchFileResults
				} `json:"results"`
			} `json:"search"`
		} `json:"data"`
	}
	err := c.GraphQL("", "", gqlQuery, variables, &resp)
	if err != nil {
		return nil, errors.Wrap(err, "request GraphQL")
	}

	return resp.Data.Search.Results.SearchFileResults, nil
}

type SearchCommitResults struct {
	MatchCount int64 `json:"matchCount"`
	Results    []*struct {
		URL string `json:"url"`
	} `json:"results"`
}

// SearchCommits searches commits with given query. It returns the match count and
// corresponding file matches.
func (c *Client) SearchCommits(query string) (*SearchCommitResults, error) {
	const gqlQuery = `
query Search($query: String!) {
	search(query: $query) {
		results {
			matchCount
			results {
				... on CommitSearchResult {
					url
				}
			}
		}
	}
}
`
	variables := map[string]interface{}{
		"query": query,
	}
	var resp struct {
		Data struct {
			Search struct {
				Results struct {
					*SearchCommitResults
				} `json:"results"`
			} `json:"search"`
		} `json:"data"`
	}
	err := c.GraphQL("", "", gqlQuery, variables, &resp)
	if err != nil {
		return nil, errors.Wrap(err, "request GraphQL")
	}

	return resp.Data.Search.Results.SearchCommitResults, nil
}

type SearchStatsResult struct {
	Languages []struct {
		Name       string `json:"name"`
		TotalLines int    `json:"totalLines"`
	} `json:"languages"`
}

// SearchStats returns statistics of given query.
func (c *Client) SearchStats(query string) (*SearchStatsResult, error) {
	const gqlQuery = `
query SearchResultsStats($query: String!) {
	search(query: $query) {
		stats {
			languages {
				name
				totalLines
			}
		}
	}
}
`
	variables := map[string]interface{}{
		"query": query,
	}
	var resp struct {
		Data struct {
			Search struct {
				Stats *SearchStatsResult `json:"stats"`
			} `json:"search"`
		} `json:"data"`
	}
	err := c.GraphQL("", "", gqlQuery, variables, &resp)
	if err != nil {
		return nil, errors.Wrap(err, "request GraphQL")
	}

	return resp.Data.Search.Stats, nil
}
