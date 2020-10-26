package main

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/machinebox/graphql"
)

// ListTrackingIssues returns all issues with the `tracking` label (and at least one other label)
// in the given organization.
func ListTrackingIssues(ctx context.Context, cli *graphql.Client, org string) ([]*Issue, error) {
	issues, _, err := LoadIssues(ctx, cli, []string{fmt.Sprintf("org:%q label:tracking", org)})
	if err != nil {
		return nil, err
	}

	var trackingIssues []*Issue
	for _, issue := range issues {
		if len(issue.Labels) > 1 {
			// Only care about non-empty tracking issues
			trackingIssues = append(trackingIssues, issue)
		}
	}

	return trackingIssues, nil
}

// LoadTrackingIssues returns all issues and pull requests which are relevant to the given set
// of tracking issues in the given organization. The result of this function may be a superset
// of objects that should be rendered for the tracking issue.
func LoadTrackingIssues(ctx context.Context, cli *graphql.Client, org string, trackingIssues []*Issue) ([]*Issue, []*PullRequest, error) {
	issues, pullRequests, err := LoadIssues(ctx, cli, makeQueries(org, trackingIssues))
	if err != nil {
		return nil, nil, err
	}

	issuesMap := map[string]*Issue{}
	for _, v := range issues {
		if !contains(v.Labels, "tracking") {
			issuesMap[v.ID] = v
		}
	}

	var nonTrackingIssues []*Issue
	for _, v := range issuesMap {
		nonTrackingIssues = append(nonTrackingIssues, v)
	}

	return nonTrackingIssues, pullRequests, err
}

// makeQueries returns a set of search queries that, when queried together, should return all of
// the relevant issue and pull requests for the given tracking issues.
func makeQueries(org string, trackingIssues []*Issue) (queries []string) {
	var rawTerms [][]string
	for _, trackingIssue := range trackingIssues {
		var labelTerms []string
		for _, label := range nonTrackingLabels(trackingIssue.Labels) {
			labelTerms = append(labelTerms, fmt.Sprintf("label:%q", label))
		}

		if trackingIssue.Milestone == "" {
			rawTerms = append(rawTerms, labelTerms)
		} else {
			rawTerms = append(rawTerms, [][]string{
				append(labelTerms, fmt.Sprintf("milestone:%q", trackingIssue.Milestone)),
				append(labelTerms, fmt.Sprintf("-milestone:%q", trackingIssue.Milestone), fmt.Sprintf(`label:"planned/%s"`, trackingIssue.Milestone)),
			}...)
		}
	}

	for i, terms := range rawTerms {
		// Add org term to every set of terms
		rawTerms[i] = append(terms, fmt.Sprintf("org:%q", org))
	}

	properSuperset := func(a, b []string) bool {
		for _, term := range b {
			if !contains(a, term) {
				return false
			}
		}

		return len(a) != len(b)
	}

	hasProperSuperset := func(terms []string) bool {
		for _, other := range rawTerms {
			if properSuperset(terms, other) {
				return true
			}
		}

		return false
	}

	// If there are two sets of terms such that one subsumes the other, then the more specific one will
	// be omitted from the result set. This is because a more general query will already return all of
	// the same results as the more specific one, and omitting it from the query should not affect the
	// set of objects that are returned from the API.

	for _, terms := range rawTerms {
		if hasProperSuperset(terms) {
			continue
		}

		sort.Strings(terms)
		queries = append(queries, strings.Join(terms, " "))
	}

	return queries
}
