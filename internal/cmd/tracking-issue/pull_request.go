package main

import (
	"strings"
	"time"
)

// PullRequest represents an existing GitHub PullRequest.
type PullRequest struct {
	ID           string
	Title        string
	Body         string
	Number       int
	URL          string
	State        string
	Repository   string
	Private      bool
	Labels       []string
	Assignees    []string
	Milestone    string
	Author       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ClosedAt     time.Time
	BeganAt      time.Time // Time of the first authored commit
	TrackedBy    []*Issue  `json:"-"`
	LinkedIssues []*Issue  `json:"-"`
}

func (pullRequest *PullRequest) Closed() bool {
	return strings.EqualFold(pullRequest.State, "closed")
}

func (pullRequest *PullRequest) Merged() bool {
	return strings.EqualFold(pullRequest.State, "merged")
}

func (pullRequest *PullRequest) Done() bool {
	return pullRequest.Merged() || pullRequest.Closed()
}

func (pullRequest *PullRequest) SafeTitle() string {
	if pullRequest.Private {
		return pullRequest.Repository
	}

	return pullRequest.Title
}

func (pullRequest *PullRequest) SafeLabels() []string {
	if pullRequest.Private {
		return redactLabels(pullRequest.Labels)
	}

	return pullRequest.Labels
}
