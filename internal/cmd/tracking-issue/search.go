package main

import (
	"fmt"
	"time"
)

const issueFields = `
	__typename
	id, title, body, state, number, url
	createdAt, closedAt
	repository { nameWithOwner, isPrivate }
	author { login }
	assignees(first: 25) { nodes { login } }
	labels(first: 25) { nodes { name } }
	milestone { title }
`

const pullRequestFields = issueFields + `
	commits(first: 1) { nodes { commit { authoredDate } } }
`

// makeSearchQuery creates a GraphQL `search` fragment that captures the fields
// of issue and pull request types. This fragment expects that the outer request
// defines the variables `query${alias}`, `count${alias}`, and `cursor${alias}`.
func makeSearchQuery(alias string) string {
	return fmt.Sprintf(`
		search%[1]s: search(query: $query%[1]s, type: ISSUE, first: $count%[1]s, after: $cursor%[1]s) {
			nodes {
				... on Issue {
					%s
				}
				... on PullRequest {
					%s
				}
			}
			pageInfo {
				endCursor
				hasNextPage
			}
		}
	`, alias, issueFields, pullRequestFields)
}

type SearchResult struct {
	Nodes    []SearchNode
	PageInfo struct {
		EndCursor   string
		HasNextPage bool
	}
}

type SearchNode struct {
	Typename   string `json:"__typename"`
	ID         string
	Title      string
	Body       string
	State      string
	Number     int
	URL        string
	Repository struct {
		NameWithOwner string
		IsPrivate     bool
	}
	Author    struct{ Login string }
	Assignees struct{ Nodes []struct{ Login string } }
	Labels    struct{ Nodes []struct{ Name string } }
	Milestone struct{ Title string }
	Commits   struct {
		Nodes []struct {
			Commit struct{ AuthoredDate time.Time }
		}
	}
	CreatedAt time.Time
	UpdatedAt time.Time
	ClosedAt  time.Time
}

// unmarshalSearchNodes unmarshals the given nodes into a list of issues and
// a list of pull requests.
func unmarshalSearchNodes(nodes []SearchNode) (issues []*Issue, prs []*PullRequest) {
	for _, node := range nodes {
		switch node.Typename {
		case "Issue":
			issues = append(issues, unmarshalIssue(node))
		case "PullRequest":
			prs = append(prs, unmarshalPullRequest(node))
		}
	}

	return issues, prs
}

// unmarshalIssue unmarshals the given node into an issue object.
func unmarshalIssue(n SearchNode) *Issue {
	issue := &Issue{
		ID:         n.ID,
		Title:      n.Title,
		Body:       n.Body,
		State:      n.State,
		Number:     n.Number,
		URL:        n.URL,
		Repository: n.Repository.NameWithOwner,
		Private:    n.Repository.IsPrivate,
		Milestone:  n.Milestone.Title,
		Author:     n.Author.Login,
		CreatedAt:  n.CreatedAt,
		UpdatedAt:  n.UpdatedAt,
		ClosedAt:   n.ClosedAt,
	}

	for _, assignee := range n.Assignees.Nodes {
		issue.Assignees = append(issue.Assignees, assignee.Login)
	}

	for _, label := range n.Labels.Nodes {
		issue.Labels = append(issue.Labels, label.Name)
	}

	return issue
}

// unmarshalPullRequest unmarshals the given node into an pull request object.
func unmarshalPullRequest(n SearchNode) *PullRequest {
	pr := &PullRequest{
		ID:         n.ID,
		Title:      n.Title,
		Body:       n.Body,
		State:      n.State,
		Number:     n.Number,
		URL:        n.URL,
		Repository: n.Repository.NameWithOwner,
		Private:    n.Repository.IsPrivate,
		Milestone:  n.Milestone.Title,
		Author:     n.Author.Login,
		CreatedAt:  n.CreatedAt,
		UpdatedAt:  n.UpdatedAt,
		ClosedAt:   n.ClosedAt,
		BeganAt:    n.Commits.Nodes[0].Commit.AuthoredDate,
	}

	for _, assignee := range n.Assignees.Nodes {
		pr.Assignees = append(pr.Assignees, assignee.Login)
	}

	for _, label := range n.Labels.Nodes {
		pr.Labels = append(pr.Labels, label.Name)
	}

	return pr
}
