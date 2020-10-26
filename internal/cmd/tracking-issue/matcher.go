package main

import "fmt"

type Matcher struct {
	labels     []string
	milestone  string
	assignee   string
	noAssignee bool
}

// NewMatcher returns a matcher with the given expected properties.
func NewMatcher(labels []string, milestone string, assignee string, noAssignee bool) *Matcher {
	return &Matcher{
		labels:     labels,
		milestone:  milestone,
		assignee:   assignee,
		noAssignee: noAssignee,
	}
}

// Issue returns true if the given issue matches the expected properties. An issue
// with the tracking issue will never be matched.
func (m *Matcher) Issue(issue *Issue) bool {
	return testAll(
		!contains(issue.Labels, "tracking"),
		m.testAssignee(issue.Assignees...),
		m.testLabels(issue.Labels),
		m.testMilestone(issue.Milestone, issue.Labels),
	)
}

// PullRequest returns true if the given pull request matches the expected properties.
func (m *Matcher) PullRequest(pullRequest *PullRequest) bool {
	return testAll(
		m.testAssignee(pullRequest.Author),
		m.testLabels(pullRequest.Labels),
		m.testMilestone(pullRequest.Milestone, pullRequest.Labels),
	)
}

// testAssignee returns true if this matcher was configured with a non-empty assignee
// that is present in the given list of assignees.
func (m *Matcher) testAssignee(assignees ...string) bool {
	if m.noAssignee {
		return len(assignees) == 0
	}

	if m.assignee == "" {
		return true
	}

	return contains(assignees, m.assignee)
}

// testLabels returns true if every label that this matcher was configured with exists
// in the given label list.
func (m *Matcher) testLabels(labels []string) bool {
	for _, label := range m.labels {
		if !contains(labels, label) {
			return false
		}
	}

	return true
}

// testMilestone returns true if the given milestone matches the milestone the matcher
// was configured with, if the given labels contains a planned/{milestone} label, or
// the milestone on the tracking issue is not restricted.
func (m *Matcher) testMilestone(milestone string, labels []string) bool {
	return m.milestone == "" || milestone == m.milestone || contains(labels, fmt.Sprintf("planned/%s", m.milestone))
}

// testAll returns true if all of the given values are true.
func testAll(tests ...bool) bool {
	for _, test := range tests {
		if !test {
			return false
		}
	}

	return true
}
