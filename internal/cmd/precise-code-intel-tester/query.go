package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/go-cmp/cmp"

	"github.com/sourcegraph/sourcegraph/internal/cmd/precise-code-intel-tester/util"
)

// queryCommand runs the "query" command.
func queryCommand() error {
	var fns []util.ParallelFn
	for _, f := range queryGenerators {
		fns = append(fns, f()...)
	}

	start := time.Now()

	ctx, cleanup := util.SignalSensitiveContext()
	defer cleanup()

	if err := util.RunParallel(ctx, numConcurrentRequests, fns); err != nil {
		return err
	}

	fmt.Printf("All queries completed in %s\n", time.Since(start))
	return nil
}

// queryGenerators is the list of functions that create query test functions.
var queryGenerators = []func() []util.ParallelFn{
	referencesFromDefinitionsQueries,
	definitionsFromReferencesQueries,
	referencesFromReferencesQueries,
}

// referencesFromDefinitionsQueries returns a list of test functions that queries the references of all the test cases definitions.
func referencesFromDefinitionsQueries() []util.ParallelFn {
	var fns []util.ParallelFn
	for _, testCase := range testCases {
		fns = append(fns, makeTestQueryFunction("references", testCase.Definition, testCase.References, queryReferences))
	}

	return fns
}

// definitionsFromReferencesQueries returns a list of test functions that queries the definitions of all the test cases references.
func definitionsFromReferencesQueries() []util.ParallelFn {
	var fns []util.ParallelFn
	for _, testCase := range testCases {
		for _, reference := range testCase.References {
			fns = append(fns, makeTestQueryFunction("definitions", reference, []Location{testCase.Definition}, queryDefinitions))
		}
	}

	return fns
}

// referencesFromReferencesQueries returns a list of test functions that queries the references of all the test cases references.
func referencesFromReferencesQueries() []util.ParallelFn {
	if !queryReferencesOfReferences {
		return nil
	}

	var fns []util.ParallelFn
	for _, testCase := range testCases {
		for _, reference := range testCase.References {
			fns = append(fns, makeTestQueryFunction("references", reference, testCase.References, queryReferences))
		}
	}

	return fns
}

// makeTestQueryFunction constructs a function for RunParallel that invokes the given query function and
// checks the returned locations against the given expected locations.
func makeTestQueryFunction(name string, location Location, expectedLocations []Location, f QueryFunc) util.ParallelFn {
	var numFinished int32

	fn := func(ctx context.Context) error {
		locations, err := f(ctx, location)
		if err != nil {
			return err
		}

		if checkQueryResult {
			sortLocations(locations)

			if diff := cmp.Diff(expectedLocations, locations); diff != "" {
				return errors.Errorf("unexpected locations (-want +got):\n%s", diff)
			}
		}

		atomic.AddInt32(&numFinished, 1)
		return nil
	}

	description := fmt.Sprintf(
		"Checking %s of %s@%s %s %d:%d",
		name,
		strings.TrimPrefix(location.Repo, "github.com/sourcegraph-testing/"),
		location.Rev[:6],
		location.Path,
		location.Line,
		location.Character,
	)

	return util.ParallelFn{
		Fn:          fn,
		Description: func() string { return description },
		Total:       func() int { return 1 },
		Finished:    func() int { return int(atomic.LoadInt32(&numFinished)) },
	}
}

// QueryFunc performs a GraphQL query (definition or references) given the source location.
type QueryFunc func(context.Context, Location) ([]Location, error)

// queryDefinitions returns all of the LSIF definitions for the given location.
func queryDefinitions(ctx context.Context, location Location) (locations []Location, err error) {
	var query = `
		query Definitions($repository: String!, $commit: String!, $path: String!, $line: Int!, $character: Int!) {
			repository(name: $repository) {
				commit(rev: $commit) {
					blob(path: $path) {
						lsif {
							definitions(line: $line, character: $character) {
								nodes {
									resource {
										path
										repository {
											name
										}
										commit {
											oid
										}
									}
									range {
										start {
											line
											character
										}
										end {
											line
											character
										}
									}
								}
								pageInfo {
									endCursor
								}
							}
						}
					}
				}
			}
		}
	`

	variables := map[string]interface{}{
		"repository": location.Repo,
		"commit":     location.Rev,
		"path":       location.Path,
		"line":       location.Line,
		"character":  location.Character,
	}

	payload := QueryResponse{}
	if err := util.QueryGraphQL(ctx, endpoint, "CodeIntelTesterDefinitions", token, query, variables, &payload); err != nil {
		return nil, err
	}

	lsifPayload := payload.Data.Repository.Commit.Blob.LSIF

	for _, node := range lsifPayload.Definitions.Nodes {
		locations = append(locations, Location{
			Repo:      node.Resource.Repository.Name,
			Rev:       node.Resource.Commit.Oid,
			Path:      node.Resource.Path,
			Line:      node.Range.Start.Line,
			Character: node.Range.Start.Character,
		})
	}

	return locations, nil
}

// queryReferences returns all of the LSIF references for the given location.
func queryReferences(ctx context.Context, location Location) (locations []Location, err error) {
	var query = `
		query References($repository: String!, $commit: String!, $path: String!, $line: Int!, $character: Int!, $after: String) {
			repository(name: $repository) {
				commit(rev: $commit) {
					blob(path: $path) {
						lsif {
							references(line: $line, character: $character, after: $after) {
								nodes {
									resource {
										path
										repository {
											name
										}
										commit {
											oid
										}
									}
									range {
										start {
											line
											character
										}
										end {
											line
											character
										}
									}
								}
								pageInfo {
									endCursor
								}
							}
						}
					}
				}
			}
		}
	`

	endCursor := ""
	for {
		variables := map[string]interface{}{
			"repository": location.Repo,
			"commit":     location.Rev,
			"path":       location.Path,
			"line":       location.Line,
			"character":  location.Character,
		}
		if endCursor != "" {
			variables["after"] = endCursor
		}

		payload := QueryResponse{}
		if err := util.QueryGraphQL(ctx, endpoint, "CodeIntelTesterReferences", token, query, variables, &payload); err != nil {
			return nil, err
		}

		lsifPayload := payload.Data.Repository.Commit.Blob.LSIF

		for _, node := range lsifPayload.References.Nodes {
			locations = append(locations, Location{
				Repo:      node.Resource.Repository.Name,
				Rev:       node.Resource.Commit.Oid,
				Path:      node.Resource.Path,
				Line:      node.Range.Start.Line,
				Character: node.Range.Start.Character,
			})
		}

		if endCursor = lsifPayload.References.PageInfo.EndCursor; endCursor == "" {
			break
		}
	}

	return locations, nil
}

// sortLocations sorts a slice of Locations by repo, rev, path, line, then character.
func sortLocations(locations []Location) {
	sort.Slice(locations, func(i, j int) bool { return compareLocations(locations[i], locations[j]) < 0 })
}

// Compare returns an integer comparing two locations. The result will be 0 if a == b,
// -1 if a < b, and +1 if a > b.
func compareLocations(a, b Location) int {
	fieldComparison := []int{
		strings.Compare(a.Repo, b.Repo),
		strings.Compare(a.Rev, b.Rev),
		strings.Compare(a.Path, b.Path),
		a.Line - b.Line,
		a.Character - b.Character,
	}

	for _, cmp := range fieldComparison {
		if cmp != 0 {
			return cmp
		}
	}
	return 0
}
