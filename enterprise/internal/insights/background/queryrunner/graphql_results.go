package queryrunner

import (
	"encoding/json"

	"github.com/cockroachdb/errors"
)

// TODO(slimsag): future: It's really nasty that our GraphQL search API is like this:
//
// 1. GraphQL makes it really annoying for us to extract the individual result values out
//    due to the `... on Type` switches.
// 2. More annoying, the GraphQL API has "results" which contain multiple matches. The web UI
//    reported the aggregate "match" count, not the number of "results" (and misleadingly
//    _calls these_ "results") - since we cannot use the global/aggregate count returned by the
//    API (we need per-repository match counts) we have to calculate this information ourselves
//    in the same way the search backend does (see the matchCount() methods) in order to get the
//    correct value. Yuck.
//
// Consider this GraphQL query snippet, which is *required* and *the only way* to count the total
// number of matches (or as our UI calls them, "results"):
//
// 	results {
// 		__typename
// 		... on FileMatch {
// 			repository {
// 				id
// 			}
// 			lineMatches {
// 				offsetAndLengths
// 			}
// 			symbols {
// 				name
// 			}
// 		}
// 		... on CommitSearchResult {
// 			matches {
// 				highlights {
// 					line
// 				}
// 			}
// 			commit {
// 				repository {
// 					id
// 				}
// 			}
// 		}
// 		... on Repository {
// 			id
// 		}
// 	}
//
// In this case:
//
// * A `FileMatch` result (I can't believe it's called a _match_ and yet it's a _result_) actually
//   can contain _several matches_ described by `lineMatches`. `lineMatches` describes the individual
//   lines matched within the file, and `offsetAndLengths` describes the matches _within a single line_.
//   The frontend tallies all of these together to say "we found this many results."
//   This is the result type for regular text searches.
// * A `FileMatch` result can ALSO contain `symbols` matches, in the case of a `type:symbol` search. In
//   this case, `lineMatches` will be empty.
// * A `CommitSearchResult` is the result type returned for `type:commit` and `type:diff` searches. It
//   can be either a single commit result (in which case `matches` will be empty..) or it can contain
//   `matches` indicating it found several matching lines within the contents of a diff.
// * A `type:repository` search will result in just a `Repository` result type.
//
// And, to be clear, ALL of these at the same time can be returned - since it is possible to combine multiple
// `type:` parameters in a search query and AFAIK that is also the default (some get mixed in _sometimes_.)
//
// If you are disgusted/shocked by this, astonished at how hard this makes it to count the number of individual
// matches, about the extremely poor usage of the terms "matches" and "results" in our codebase - just know
// that so am I and this is some of the oldest "tech debt" in all of Sourcegraph :)

type result interface {
	repoID() string
	matchCount() int
}

func decodeResult(result json.RawMessage) (result, error) {
	typeName := struct {
		TypeName string `json:"__typeName"`
	}{}
	if err := json.Unmarshal(result, &typeName); err != nil {
		return nil, err
	}
	switch typeName.TypeName {
	case "FileMatch":
		var v fileMatch
		if err := json.Unmarshal(result, &v); err != nil {
			return nil, err
		}
		return &v, nil
	case "CommitSearchResult":
		var v commitSearchResult
		if err := json.Unmarshal(result, &v); err != nil {
			return nil, err
		}
		return &v, nil
	case "Repository":
		var v repository
		if err := json.Unmarshal(result, &v); err != nil {
			return nil, err
		}
		return &v, nil
	default:
		return nil, errors.Errorf("cannot decode search result: unexpected TypeName: %s", string(result))
	}
}

type fileMatch struct {
	Repository struct {
		ID string
	}
	LineMatches []struct {
		OffsetAndLengths [][]int
	}
	Symbols []struct {
		Name string
	}
}

func (r *fileMatch) matchCount() int {
	matches := len(r.Symbols)
	for _, lineMatch := range r.LineMatches {
		matches += len(lineMatch.OffsetAndLengths)
	}
	if matches == 0 {
		matches = 1 // 1 to count "empty" results like type:path results
	}
	return matches
}

func (r *fileMatch) repoID() string {
	return r.Repository.ID
}

type commitSearchResult struct {
	Matches struct {
		Highlights []struct {
			Line int
		}
	}
	Commit struct {
		Repository struct {
			ID string
		}
	}
}

func (r *commitSearchResult) repoID() string {
	return r.Commit.Repository.ID
}

func (r *commitSearchResult) matchCount() int {
	matches := 1
	if len(r.Matches.Highlights) > 0 {
		matches = len(r.Matches.Highlights)
	}
	return matches
}

type repository struct {
	ID string
}

func (r *repository) repoID() string {
	return r.ID
}

func (r *repository) matchCount() int {
	return 1
}
