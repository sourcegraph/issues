package main

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// RenderTrackingIssue renders the work section of the given tracking issue.
func RenderTrackingIssue(context IssueContext) string {
	assignees := findAssignees(context.Match(NewMatcher(
		nonTrackingLabels(context.trackingIssue.Labels),
		context.trackingIssue.Milestone,
		"",
		false,
	)))

	var parts []string
	for _, assignee := range assignees {
		assigneeContext := context.Match(NewMatcher(
			nonTrackingLabels(context.trackingIssue.Labels),
			context.trackingIssue.Milestone,
			assignee,
			assignee == "",
		))

		parts = append(parts, NewAssigneeRenderer(assigneeContext, assignee).Render())
	}

	return strings.Join(parts, "\n")
}

// findAssignees returns the list of assignees for the given tracking issue. A user is an
// assignee for a tracking issue if there is a _leaf_ (non-tracking) issue or a pull request
// with that user as the assignee or author, respectively.
func findAssignees(context IssueContext) (assignees []string) {
	assigneeMap := map[string]struct{}{}
	for _, issue := range context.issues {
		for _, assignee := range issue.Assignees {
			assigneeMap[assignee] = struct{}{}
		}
		if len(issue.Assignees) == 0 {
			// Mark special empty assignee for the unassigned bucket
			assigneeMap[""] = struct{}{}
		}
	}
	for _, pullRequest := range context.pullRequests {
		assigneeMap[pullRequest.Author] = struct{}{}
	}

	for assignee := range assigneeMap {
		assignees = append(assignees, assignee)
	}
	sort.Strings(assignees)
	return assignees
}

type AssigneeRenderer struct {
	context              IssueContext
	assignee             string
	issueDisplayed       []bool
	pullRequestDisplayed []bool
}

func NewAssigneeRenderer(context IssueContext, assignee string) *AssigneeRenderer {
	return &AssigneeRenderer{
		context:              context,
		assignee:             assignee,
		issueDisplayed:       make([]bool, len(context.issues)),
		pullRequestDisplayed: make([]bool, len(context.pullRequests)),
	}
}

// Render returns the assignee section of the configured tracking issue for the
// configured assignee.
func (ar *AssigneeRenderer) Render() string {
	var labels [][]string
	for _, issue := range ar.context.issues {
		labels = append(labels, issue.Labels)
	}

	estimateFragment := ""
	if estimate := estimateFromLabelSets(labels); estimate != 0 {
		estimateFragment = fmt.Sprintf(": __%.2fd__", estimate)
	}

	assignee := ar.assignee
	if assignee == "" {
		assignee = "unassigned"
	}

	s := ""
	s += fmt.Sprintf(beginAssigneeMarkerFmt, ar.assignee)
	s += fmt.Sprintf("\n@%s%s\n\n", assignee, estimateFragment)
	s += ar.renderPendingWork()
	ar.resetDisplayFlags()
	s += ar.renderCompletedWork()
	s += endAssigneeMarker + "\n"
	return s
}

type MarkdownByStringKey struct {
	markdown string
	key      string
}

// renderPendingWork returns a list of pending work items rendered in markdown.
func (ar *AssigneeRenderer) renderPendingWork() string {
	var parts []MarkdownByStringKey
	parts = append(parts, ar.renderPendingTrackingIssues()...)
	parts = append(parts, ar.renderPendingIssues()...)
	parts = append(parts, ar.renderPendingPullRequests()...)

	if len(parts) == 0 {
		return ""
	}

	// Make a map of URLs to their rank with respect to the order
	// that each issue is currently referenced in the tracking issue.
	rankByURL := map[string]int{}
	for v, k := range ar.readTrackingIssueURLs() {
		rankByURL[k] = v
	}

	// Sort each rendered work item by:
	//
	//   - their order in the current tracking issue
	//   - pre-existing issue before new issues
	//   - their URL values

	sort.Slice(parts, func(i, j int) bool {
		rank1, ok1 := rankByURL[parts[i].key]
		rank2, ok2 := rankByURL[parts[j].key]

		if ok1 && ok2 {
			// compare explicit orders
			return rank1 < rank2
		}
		if !ok1 && !ok2 {
			// sort by URL if neither exists in previous order
			return parts[i].key < parts[j].key
		}

		// explicit ordering comes before implicit ordering
		return ok1
	})

	s := ""
	for _, part := range parts {
		s += part.markdown
	}

	return s
}

type MarkdownByIntegerKeyPair struct {
	markdown string
	key1     int64
	key2     int64
}

func SortByIntegerKeyPair(parts []MarkdownByIntegerKeyPair) (markdown []string) {
	sort.Slice(parts, func(i, j int) bool {
		if parts[i].key1 != parts[j].key1 {
			return parts[i].key1 < parts[j].key1
		}
		return parts[i].key2 < parts[j].key2
	})

	for _, part := range parts {
		markdown = append(markdown, part.markdown)
	}

	return markdown
}

// renderPendingTrackingIssues returns a rendered list of tracking issues (with rendered children)
// along with that tracking issue's URL for later reordering of the resulting list.
func (ar *AssigneeRenderer) renderPendingTrackingIssues() (parts []MarkdownByStringKey) {
	for _, issue := range ar.context.trackingIssues {
		if issue == ar.context.trackingIssue {
			continue
		}

		if !issue.Closed() {
			var pendingParts []MarkdownByIntegerKeyPair

			for _, issue := range issue.TrackedIssues {
				if _, ok := ar.findIssue(issue); ok {
					key := int64(0)
					if issue.Closed() {
						key = 1
					}

					pendingParts = append(pendingParts, MarkdownByIntegerKeyPair{
						markdown: "  " + ar.renderIssue(issue),
						key1:     key,
						key2:     int64(issue.Number),
					})
				}
			}
			for _, pullRequest := range issue.TrackedPullRequests {
				if i, ok := ar.findPullRequest(pullRequest); ok && !ar.pullRequestDisplayed[i] {
					key := int64(0)
					if pullRequest.Done() {
						key = 1
					}

					pendingParts = append(pendingParts, MarkdownByIntegerKeyPair{
						markdown: "  " + ar.renderPullRequest(pullRequest),
						key1:     key,
						key2:     int64(pullRequest.Number),
					})
				}
			}

			if len(pendingParts) > 0 {
				parts = append(parts, MarkdownByStringKey{
					markdown: ar.renderIssue(issue) + strings.Join(SortByIntegerKeyPair(pendingParts), ""),
					key:      issue.URL,
				})
			}
		}
	}

	return parts
}

// renderPendingIssues returns a rendered list of unclosed issues along with that issue's URL for later
// reordering of the resulting list. The resulting list does not include any issue that was already
// rendered by renderPendingTrackingIssues.
func (ar *AssigneeRenderer) renderPendingIssues() (parts []MarkdownByStringKey) {
	for i, issue := range ar.context.issues {
		if !ar.issueDisplayed[i] && !issue.Closed() {
			parts = append(parts, MarkdownByStringKey{
				markdown: ar.renderIssue(issue),
				key:      issue.URL,
			})
		}
	}

	return parts
}

// renderPendingPullRequests returns a rendered list of unclosed pull requests along with that issue's
// URL for later reordering of the resulting list. The resulting list does not include any pull request
// that was already rendered by renderPendingTrackingIssues or renderPendingIssues.
func (ar *AssigneeRenderer) renderPendingPullRequests() (parts []MarkdownByStringKey) {
	for i, pullRequest := range ar.context.pullRequests {
		if !ar.pullRequestDisplayed[i] && !pullRequest.Done() {
			parts = append(parts, MarkdownByStringKey{
				markdown: ar.renderPullRequest(pullRequest),
				key:      pullRequest.URL,
			})
		}
	}

	return parts
}

// renderCompletedWork returns a list of completed work items rendered in markdown.
func (ar *AssigneeRenderer) renderCompletedWork() string {
	var completedParts []MarkdownByIntegerKeyPair
	completedParts = append(completedParts, ar.renderCompletedTrackingIssues()...)
	completedParts = append(completedParts, ar.renderCompletedIssues()...)
	completedParts = append(completedParts, ar.renderCompletedPullRequests()...)

	if len(completedParts) == 0 {
		return ""
	}

	var labels [][]string
	for _, issue := range ar.context.issues {
		if issue.Closed() {
			labels = append(labels, issue.Labels)
		}
	}

	estimateFragment := ""
	if estimate := estimateFromLabelSets(labels); estimate != 0 {
		estimateFragment = fmt.Sprintf(": __%.2fd__", estimate)
	}

	return fmt.Sprintf("\nCompleted%s\n%s", estimateFragment, strings.Join(SortByIntegerKeyPair(completedParts), ""))
}

// renderCompletedTrackingIsssues returns a rendered list of closed tracking issues along with that
// issue's closed-at time and that issue's number for later reordering of the resulting list. This
// will also set the completed flag on all tracked issues and pull requests.
func (ar *AssigneeRenderer) renderCompletedTrackingIssues() (completedParts []MarkdownByIntegerKeyPair) {
	for _, issue := range ar.context.trackingIssues {
		if issue.Closed() {
			children := 0
			for _, issue := range issue.TrackedIssues {
				if i, ok := ar.findIssue(issue); ok {
					ar.issueDisplayed[i] = true
					children++
				}
			}
			for _, pullRequest := range issue.TrackedPullRequests {
				if i, ok := ar.findPullRequest(pullRequest); ok {
					ar.pullRequestDisplayed[i] = true
					children++
				}
			}

			if children > 0 {
				completedParts = append(completedParts, MarkdownByIntegerKeyPair{
					markdown: ar.renderIssue(issue),
					key1:     issue.ClosedAt.Unix(),
					key2:     int64(issue.Number),
				})
			}
		}
	}

	return completedParts
}

// renderCompletedIssues returns a rendered list of closed issues along with that issue's closed-at
// time and that issue's number for later reordering of the resulting list.
func (ar *AssigneeRenderer) renderCompletedIssues() (completedParts []MarkdownByIntegerKeyPair) {
	for i, issue := range ar.context.issues {
		if !ar.issueDisplayed[i] && issue.Closed() {
			completedParts = append(completedParts, MarkdownByIntegerKeyPair{
				markdown: ar.renderIssue(issue),
				key1:     issue.ClosedAt.Unix(),
				key2:     int64(issue.Number),
			})
		}
	}

	return completedParts
}

// renderCompletedPullRequests returns a rendered list of closed pull request along with that pull
// request's closed-at time and that pull request's number for later reordering of the resulting list.
func (ar *AssigneeRenderer) renderCompletedPullRequests() (completedParts []MarkdownByIntegerKeyPair) {
	for i, pullRequest := range ar.context.pullRequests {
		if !ar.pullRequestDisplayed[i] && pullRequest.Done() {
			completedParts = append(completedParts, MarkdownByIntegerKeyPair{
				markdown: ar.renderPullRequest(pullRequest),
				key1:     pullRequest.ClosedAt.Unix(),
				key2:     int64(pullRequest.Number),
			})
		}
	}

	return completedParts
}

// renderIssue returns the given issue rendered as markdown. This will also set the
// displayed flag on this issue as well as all linked pull requests.
func (ar *AssigneeRenderer) renderIssue(issue *Issue) string {
	if i, ok := ar.findIssue(issue); ok {
		ar.issueDisplayed[i] = true
	}

	for _, pullRequest := range issue.LinkedPullRequests {
		if i, ok := ar.findPullRequest(pullRequest); ok {
			ar.pullRequestDisplayed[i] = true
		}
	}

	return ar.doRenderIssue(issue, ar.context.trackingIssue.Milestone)
}

// renderPullRequest returns the given pull request rendered as markdown. This will also
// set the displayed flag on the pull request.
func (ar *AssigneeRenderer) renderPullRequest(pullRequest *PullRequest) string {
	if i, ok := ar.findPullRequest(pullRequest); ok {
		ar.pullRequestDisplayed[i] = true
	}

	return renderPullRequest(pullRequest)
}

// findIssue returns the index of the given issue in the current context. If the issue does not
// exist then a false-valued flag is returned.
func (ar *AssigneeRenderer) findIssue(v *Issue) (int, bool) {
	for i, x := range ar.context.issues {
		if v == x {
			return i, true
		}
	}

	return 0, false
}

// findPullRequest returns the index of the given pull request in the current context. If the pull
// request does not exist then a false-valued flag is returned.
func (ar *AssigneeRenderer) findPullRequest(v *PullRequest) (int, bool) {
	for i, x := range ar.context.pullRequests {
		if v == x {
			return i, true
		}
	}

	return 0, false
}

var issueOrPullRequestMatcher = regexp.MustCompile(`https://github\.com/[^/]+/[^/]+/(issues|pull)/\d+`)

// readTrackingIssueURLs reads each line of the current tracking issue body and extracts issue and
// pull request references. The order of the output slice is the order in which each URL is first
// referenced and is used to maintain a stable ordering in the GitHub UI.
//
// Note: We use the fact that rendered work items always reference themselves first, and any additional
// issue or pull request URLs on that line are only references. By parsing line-by-line and pulling the
// first URL we see, we should get an accurate ordering.
func (ar *AssigneeRenderer) readTrackingIssueURLs() (urls []string) {
	_, body, _, ok := partition(ar.context.trackingIssue.Body, fmt.Sprintf(beginAssigneeMarkerFmt, ar.assignee), endAssigneeMarker)
	if !ok {
		return
	}

	for _, line := range strings.Split(body, "\n") {
		if url := issueOrPullRequestMatcher.FindString(line); url != "" {
			urls = append(urls, url)
		}
	}

	return urls
}

// resetDisplayFlags unsets the displayed flag for all issues and pull requests.
func (ar *AssigneeRenderer) resetDisplayFlags() {
	for i := range ar.context.issues {
		ar.issueDisplayed[i] = false
	}
	for i := range ar.context.pullRequests {
		ar.pullRequestDisplayed[i] = false
	}
}

// doRenderIssue returns the given issue rendered in markdown.
func (ar *AssigneeRenderer) doRenderIssue(issue *Issue, milestone string) string {
	title := issue.SafeTitle()
	if issue.Milestone != milestone && contains(issue.Labels, fmt.Sprintf("planned/%s", milestone)) {
		// deprioritized
		title = fmt.Sprintf("~%s~", title)
	}

	pullRequestFragment := ""
	if len(issue.LinkedPullRequests) > 0 {
		var parts []MarkdownByIntegerKeyPair
		for _, pullRequest := range issue.LinkedPullRequests {
			summary := fmt.Sprintf("[#%d](%s)", pullRequest.Number, pullRequest.URL)
			if pullRequest.Done() {
				summary = fmt.Sprintf("~%s~", summary)
			}

			parts = append(parts, MarkdownByIntegerKeyPair{
				markdown: summary,
				key1:     pullRequest.CreatedAt.Unix(),
				key2:     int64(pullRequest.Number),
			})
		}

		pullRequestFragment = fmt.Sprintf("; PRs: %s", strings.Join(SortByIntegerKeyPair(parts), ", "))
	}

	estimate := estimateFromLabelSet(issue.Labels)
	if estimate == 0 {
		var labels [][]string
		for _, child := range issue.TrackedIssues {
			if _, ok := ar.findIssue(child); ok {
				labels = append(labels, child.Labels)
			}
		}

		estimate = estimateFromLabelSets(labels)
	}
	estimateFragment := ""
	if estimate != 0 {
		estimateFragment = fmt.Sprintf(" __%.2fd__", estimate)
	}

	emojis := Emojis(issue.SafeLabels(), issue.Repository, issue.Body, nil)
	if emojis != "" {
		emojis = " " + emojis
	}

	if issue.Closed() {
		return fmt.Sprintf(
			"- [x] (🏁 %s) %s ([~#%d~](%s)%s)%s%s\n",
			formatTimeSince(issue.ClosedAt),
			title,
			issue.Number,
			issue.URL,
			pullRequestFragment,
			estimateFragment,
			emojis,
		)
	}

	return fmt.Sprintf(
		"- [ ] %s ([#%d](%s)%s)%s%s\n",
		title,
		issue.Number,
		issue.URL,
		pullRequestFragment,
		estimateFragment,
		emojis,
	)
}

// renderPullRequest returns the given pull request rendered in markdown.
func renderPullRequest(pullRequest *PullRequest) string {
	emojis := Emojis(pullRequest.SafeLabels(), pullRequest.Repository, pullRequest.Body, map[string]string{
		"pull-request": ":shipit:",
	})

	if pullRequest.Done() {
		title := pullRequest.SafeTitle()
		if pullRequest.Closed() {
			title = fmt.Sprintf("~%s~", title)
		}

		return fmt.Sprintf(
			"- [x] (🏁 %s) %s ([~#%d~](%s)) %s\n",
			formatTimeSince(pullRequest.ClosedAt),
			title,
			pullRequest.Number,
			pullRequest.URL,
			emojis,
		)
	}

	return fmt.Sprintf(
		"- [ ] %s ([#%d](%s)) %s\n",
		pullRequest.SafeTitle(),
		pullRequest.Number,
		pullRequest.URL,
		emojis,
	)
}

// estimateFromLabelSets returns the sum of `estimate/` labels in the given label sets.
func estimateFromLabelSets(labels [][]string) (days float64) {
	for _, labels := range labels {
		days += estimateFromLabelSet(labels)
	}

	return days
}

// estimateFromLabelSet returns the value of a `estimate/` lables in the given label set.
func estimateFromLabelSet(labels []string) float64 {
	for _, label := range labels {
		if strings.HasPrefix(label, "estimate/") {
			d, _ := strconv.ParseFloat(strings.TrimSuffix(strings.TrimPrefix(label, "estimate/"), "d"), 64)
			return d
		}
	}

	return 0
}
