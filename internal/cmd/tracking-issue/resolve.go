package main

import (
	"fmt"
	"regexp"
)

// Resolve will populate the relationship fields of the registered issues and pull
// requests objects.
func Resolve(trackingIssues, issues []*Issue, pullRequests []*PullRequest) error {
	linkPullRequestsAndIssues(trackingIssues, issues, pullRequests)
	linkTrackingIssues(trackingIssues, issues, pullRequests)
	return checkForCycles(trackingIssues, issues, pullRequests)
}

// linkPullRequestsAndIssues populates the LinkedPullRequests and LinkedIssues fields of
// each resolved issue and pull request value. A pull request and an issue are linked if
// the pull request body contains a reference to the issue number.
func linkPullRequestsAndIssues(trackingIssues, issues []*Issue, pullRequests []*PullRequest) {
	for _, issue := range issues {
		patterns := []*regexp.Regexp{
			// TODO(efritz) - should probably match repository as well
			regexp.MustCompile(fmt.Sprintf(`#%d([^\d]|$)`, issue.Number)),
			regexp.MustCompile(fmt.Sprintf(`https://github.com/[^/]+/[^/]+/issues/%d([^\d]|$)`, issue.Number)),
		}

		for _, pullRequest := range pullRequests {
			for _, pattern := range patterns {
				if pattern.MatchString(pullRequest.Body) {
					issue.LinkedPullRequests = append(issue.LinkedPullRequests, pullRequest)
					pullRequest.LinkedIssues = append(pullRequest.LinkedIssues, issue)
				}
			}
		}
	}
}

// linkTrackingIssues populates the TrackedIssues, TrackedPullRequests, and TrackedBy
// fields of each resolved issue and pull request value. An issue or pull request is
// tracked by a tracking issue if the labels, milestone, and assignees all match the
// tracking issue properties (if supplied).
func linkTrackingIssues(trackingIssues, issues []*Issue, pullRequests []*PullRequest) {
	for _, trackingIssue := range trackingIssues {
		matcher := NewMatcher(
			nonTrackingLabels(trackingIssue.Labels),
			trackingIssue.Milestone,
			"",
			false,
		)

		for _, issue := range issues {
			if matcher.Issue(issue) {
				trackingIssue.TrackedIssues = append(trackingIssue.TrackedIssues, issue)
				issue.TrackedBy = append(issue.TrackedBy, trackingIssue)
			}
		}

		for _, pullRequest := range pullRequests {
			if matcher.PullRequest(pullRequest) {
				trackingIssue.TrackedPullRequests = append(trackingIssue.TrackedPullRequests, pullRequest)
				pullRequest.TrackedBy = append(pullRequest.TrackedBy, trackingIssue)
			}
		}
	}
}

// checkForCycles checks for a cycle over the tracked issues relationship in the set of resolved
// issues. We currently check this condition because the rendering pass does not check for cycles
// and can create an infinite loop.
func checkForCycles(trackingIssues, issues []*Issue, pullRequests []*PullRequest) error {
	for _, issue := range issues {
		if !visitNode(issue, map[string]struct{}{}) {
			// TODO(efritz) - we should try to proactively cut cycles
			return fmt.Errorf("Tracking issues contain cycles")
		}
	}

	return nil
}

// visitNode performs a depth-first-search over tracked issues relationships. This
// function will return false if the traversal encounters a node that has already
// been visited.
func visitNode(issue *Issue, visited map[string]struct{}) bool {
	if _, ok := visited[issue.ID]; ok {
		return false
	}
	visited[issue.ID] = struct{}{}

	for _, c := range issue.TrackedIssues {
		if !visitNode(c, visited) {
			return false
		}
	}

	return true
}
