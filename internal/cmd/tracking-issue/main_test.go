package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/machinebox/graphql"
	"github.com/sourcegraph/sourcegraph/internal/testutil"
	"golang.org/x/oauth2"
)

var (
	testUpdate        = flag.Bool("update", false, "update testdata golden")
	testUpdateFixture = flag.Bool("update.fixture", false, "update testdata API response")

	testIssues = []int{
		13675, // Distribution 3.21 Tracking issue
		13987, // Code Intelligence 3.21 Tracking issue
		13988, // Cloud 2020-09-23 Tracking issue
		14166, // RFC-214: Tracking issue
	}
)

func TestIntegration(t *testing.T) {
	mockLastUpdate(t)

	trackingIssues, issues, pullRequests, err := testFixtures()
	if err != nil {
		t.Fatal(err)
	}

	if err := Resolve(trackingIssues, issues, pullRequests); err != nil {
		t.Fatal(err)
	}

	for _, number := range testIssues {
		t.Run(fmt.Sprintf("#%d", number), func(t *testing.T) {
			for _, trackingIssue := range trackingIssues {
				if trackingIssue.Number != number {
					continue
				}

				context := NewIssueContext(trackingIssue, trackingIssues, issues, pullRequests)
				if _, ok := trackingIssue.UpdateBody(RenderTrackingIssue(context)); !ok {
					t.Fatal("failed to patch issue")
				}

				goldenPath := filepath.Join("testdata", fmt.Sprintf("issue-%d.md", number))
				testutil.AssertGolden(t, goldenPath, *testUpdate, trackingIssue.Body)
				return
			}

			t.Fatalf(`Could not find golden file for #%d. Please run go test -update.fixture".`, number)
		})
	}
}

func mockLastUpdate(t *testing.T) {
	lastUpdate, err := getOrUpdateLastUpdateTime(*testUpdate)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}
	now = func() time.Time { return lastUpdate }
}

func getOrUpdateLastUpdateTime(update bool) (time.Time, error) {
	lastUpdateFile := filepath.Join("testdata", "last-update.txt")

	if update {
		now := time.Now().UTC()
		if err := ioutil.WriteFile(lastUpdateFile, []byte(now.Format(time.RFC3339)), os.ModePerm); err != nil {
			return time.Time{}, err
		}

		return now, nil
	}

	content, err := ioutil.ReadFile(lastUpdateFile)
	if err != nil {
		return time.Time{}, err
	}

	return time.Parse(time.RFC3339, string(content))
}

type FixturePayload struct {
	TrackingIssues []*Issue
	Issues         []*Issue
	PullRequests   []*PullRequest
}

func testFixtures() (trackingIssues []*Issue, issues []*Issue, pullRequests []*PullRequest, _ error) {
	if *testUpdateFixture {
		return updateTestFixtures()
	}

	return readFixturesFile()
}

func updateTestFixtures() (trackingIssues []*Issue, issues []*Issue, pullRequests []*PullRequest, _ error) {
	token := flag.String("token", os.Getenv("GITHUB_TOKEN"), "GitHub personal access token")

	ctx := context.Background()
	cli := graphql.NewClient("https://api.github.com/graphql", graphql.WithHTTPClient(
		oauth2.NewClient(ctx, oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: *token},
		))),
	)

	trackingIssues, err := ListTrackingIssues(ctx, cli, "sourcegraph")
	if err != nil {
		return nil, nil, nil, err
	}

	var matchingIssues []*Issue
	for _, issues := range trackingIssues {
		for _, n := range testIssues {
			if issues.Number == n {
				matchingIssues = append(matchingIssues, issues)
				break
			}
		}
	}

	issues, pullRequests, err = LoadTrackingIssues(ctx, cli, "sourcegraph", matchingIssues)
	if err != nil {
		return nil, nil, nil, err
	}

	// Redact any private data from the response
	for _, issue := range trackingIssues {
		if issue.Private {
			issue.Title = issue.Repository
			issue.Labels = redactLabels(issue.Labels)
			issue.Body = "REDACTED"
		}
	}
	for _, issue := range issues {
		if issue.Private {
			issue.Title = issue.Repository
			issue.Labels = redactLabels(issue.Labels)
			issue.Body = "REDACTED"
		}
	}
	for _, pullRequest := range pullRequests {
		if pullRequest.Private {
			pullRequest.Title = pullRequest.Repository
			pullRequest.Labels = redactLabels(pullRequest.Labels)
			pullRequest.Body = "REDACTED"
		}
	}

	if err := writeFixturesFile(trackingIssues, issues, pullRequests); err != nil {
		return nil, nil, nil, err
	}

	return trackingIssues, issues, pullRequests, nil
}

func readFixturesFile() (trackingIssues []*Issue, issues []*Issue, pullRequests []*PullRequest, _ error) {
	contents, err := ioutil.ReadFile(filepath.Join("testdata", "fixtures.json"))
	if err != nil {
		return nil, nil, nil, err
	}

	var payload FixturePayload
	if err := json.Unmarshal(contents, &payload); err != nil {
		return nil, nil, nil, err
	}

	return payload.TrackingIssues, payload.Issues, payload.PullRequests, nil
}

func writeFixturesFile(trackingIssues []*Issue, issues []*Issue, pullRequests []*PullRequest) error {
	contents, err := json.Marshal(FixturePayload{
		TrackingIssues: trackingIssues,
		Issues:         issues,
		PullRequests:   pullRequests,
	})
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join("testdata", "fixtures.json"), contents, os.ModePerm)
}
