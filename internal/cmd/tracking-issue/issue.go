package main

import (
	"strings"
	"time"
)

// Issue represents an existing GitHub Issue.
type Issue struct {
	ID                  string
	Title               string
	Body                string
	Number              int
	URL                 string
	State               string
	Repository          string
	Private             bool
	Labels              []string
	Assignees           []string
	Milestone           string
	Author              string
	CreatedAt           time.Time
	UpdatedAt           time.Time
	ClosedAt            time.Time
	TrackedBy           []*Issue       `json:"-"`
	TrackedIssues       []*Issue       `json:"-"`
	TrackedPullRequests []*PullRequest `json:"-"`
	LinkedPullRequests  []*PullRequest `json:"-"`
}

func (issue *Issue) Closed() bool {
	return strings.EqualFold(issue.State, "closed")
}

func (issue *Issue) SafeTitle() string {
	if issue.Private {
		return issue.Repository
	}

	return issue.Title
}

func (issue *Issue) SafeLabels() []string {
	if issue.Private {
		return redactLabels(issue.Labels)
	}

	return issue.Labels
}

func (issue *Issue) UpdateBody(markdown string) (updated bool, ok bool) {
	prefix, _, suffix, ok := partition(issue.Body, beginWorkMarker, endWorkMarker)
	if !ok {
		return false, false
	}

	newBody := prefix + "\n" + markdown + suffix
	if newBody == issue.Body {
		return false, true
	}

	issue.Body = newBody
	return true, true
}
