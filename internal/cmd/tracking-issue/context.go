package main

import "strings"

// IssueContext tracks a visible set of issues, tracking issues, and pull requests
// with respect to a given tracking issue. The visible set of issues and pull requests
// can be refined with additional restrictions.
type IssueContext struct {
	trackingIssue  *Issue
	trackingIssues []*Issue
	issues         []*Issue
	pullRequests   []*PullRequest
}

// NewIssueContext creates  anew issue context with the given visible issues, tracking
// issues, and pull requests.
func NewIssueContext(trackingIssue *Issue, trackingIssues []*Issue, issues []*Issue, pullRequests []*PullRequest) IssueContext {
	return IssueContext{
		trackingIssue:  trackingIssue,
		trackingIssues: trackingIssues,
		issues:         issues,
		pullRequests:   pullRequests,
	}
}

// Match will return a new issue context where all visible issues and pull requests match
// the given matcher.
func (context IssueContext) Match(matcher *Matcher) IssueContext {
	return IssueContext{
		trackingIssue:  context.trackingIssue,
		trackingIssues: matchingTrackingIssues(context.trackingIssue, context.issues, context.pullRequests, matcher),
		issues:         matchingIssues(context.trackingIssue, context.issues, matcher),
		pullRequests:   matchingPullRequests(context.pullRequests, matcher),
	}
}

// matchingIssues returns the given issues that match the given matcher.
func matchingIssues(trackingIssue *Issue, issues []*Issue, matcher *Matcher) (matchingIssues []*Issue) {
	for _, issue := range issues {
		if issue != trackingIssue && matcher.Issue(issue) {
			matchingIssues = append(matchingIssues, issue)
		}
	}

	return deduplicateIssues(matchingIssues)
}

// matchingPullRequests returns the given pull requests that match the given matcher.
func matchingPullRequests(pullRequests []*PullRequest, matcher *Matcher) (matchingPullRequests []*PullRequest) {
	for _, pullRequest := range pullRequests {
		if matcher.PullRequest(pullRequest) {
			matchingPullRequests = append(matchingPullRequests, pullRequest)
		}
	}

	return deduplicatePullRequests(matchingPullRequests)
}

// matchingTrackingIssues returns the given tracking issues that match the matcher and do not track
// only a `team/*` label.
func matchingTrackingIssues(trackingIssue *Issue, issues []*Issue, pullRequests []*PullRequest, matcher *Matcher) (matchingTrackingIssues []*Issue) {
	var stack []*Issue
	for _, issue := range matchingIssues(trackingIssue, issues, matcher) {
		stack = append(stack, issue.TrackedBy...)
	}
	for _, pullRequest := range matchingPullRequests(pullRequests, matcher) {
		for _, issue := range pullRequest.TrackedBy {
			if contains(issue.Labels, "tracking") {
				stack = append(stack, issue)
			} else {
				stack = append(stack, issue.TrackedBy...)
			}
		}
	}

	for len(stack) > 0 {
		var top *Issue
		top, stack = stack[0], stack[1:]

		if len(top.Labels) != 2 || !strings.HasPrefix(nonTrackingLabels(top.Labels)[0], "team/") {
			matchingTrackingIssues = append(matchingTrackingIssues, top)
		}

		stack = append(stack, top.TrackedBy...)
	}

	return deduplicateIssues(matchingTrackingIssues)
}
