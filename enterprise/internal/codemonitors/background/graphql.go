package background

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/url"
	"runtime"
	"time"

	"github.com/sourcegraph/sourcegraph/internal/api"

	"golang.org/x/net/context/ctxhttp"

	"github.com/cockroachdb/errors"
)

type graphQLQuery struct {
	Query     string      `json:"query"`
	Variables interface{} `json:"variables"`
}

const gqlSearchQuery = `query Search(
	$query: String!,
) {
	search(query: $query) {
		results {
			approximateResultCount
			limitHit
			cloning { name }
			timedout { name }
			results {
				__typename
				... on FileMatch {
					limitHit
					lineMatches {
						preview
						lineNumber
						offsetAndLengths
					}
				}
				... on CommitSearchResult {
					refs {
						name
						displayName
						prefix
						repository {
							name
						}
					}
					sourceRefs {
						name
						displayName
						prefix
						repository {
							name
						}
					}
					messagePreview {
						value
						highlights {
							line
							character
							length
						}
					}
					diffPreview {
						value
						highlights {
							line
							character
							length
						}
					}
					commit {
						repository {
							name
						}
						oid
						abbreviatedOID
						author {
							person {
								displayName
								avatarURL
							}
							date
						}
						message
					}
				}
			}
			alert {
				title
				description
				proposedQueries {
					description
					query
				}
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
				ApproximateResultCount string
				Cloning                []*api.Repo
				Timedout               []*api.Repo
				Results                []interface{}
			}
		}
	}
	Errors []interface{}
}

func search(ctx context.Context, query string) (*gqlSearchResponse, error) {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(graphQLQuery{
		Query:     gqlSearchQuery,
		Variables: gqlSearchVars{Query: query},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Encode")
	}

	url, err := gqlURL("Search")
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

func gqlURL(queryName string) (string, error) {
	u, err := url.Parse(api.InternalClient.URL)
	if err != nil {
		return "", err
	}
	u.Path = "/.internal/graphql"
	u.RawQuery = queryName
	return u.String(), nil
}

// extractTime extracts the time from the given search result.
func extractTime(result interface{}) (t *time.Time, err error) {
	// Use recover because we assume the data structure here a lot, for less
	// error checking.
	defer func() {
		if r := recover(); r != nil {
			// Same as net/http
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			log.Printf("failed to extract time from search result: %v\n%s", r, buf)
			err = errors.Errorf("failed to extract time from search result")
		}
	}()

	m := result.(map[string]interface{})
	typeName := m["__typename"].(string)
	switch typeName {
	case "CommitSearchResult":
		commit := m["commit"].(map[string]interface{})
		author := commit["author"].(map[string]interface{})
		date := author["date"].(string)

		// This relies on the date format that our API returns. It was previously broken
		// and should be checked first in case date extraction stops working.
		t, err := time.Parse(time.RFC3339, date)
		if err != nil {
			return nil, err
		}
		return &t, nil
	default:
		return nil, errors.Errorf("unexpected result __typename %q", typeName)
	}
}
