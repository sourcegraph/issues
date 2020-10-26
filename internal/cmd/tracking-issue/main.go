// Command tracking-issue uses the GitHub API to maintain open tracking issues.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/machinebox/graphql"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

const (
	beginWorkMarker        = "<!-- BEGIN WORK -->"
	endWorkMarker          = "<!-- END WORK -->"
	beginAssigneeMarkerFmt = "<!-- BEGIN ASSIGNEE: %s -->"
	endAssigneeMarker      = "<!-- END ASSIGNEE -->"
)

func main() {
	token := flag.String("token", os.Getenv("GITHUB_TOKEN"), "GitHub personal access token")
	org := flag.String("org", "sourcegraph", "GitHub organization to list issues from")
	dry := flag.Bool("dry", false, "If true, do not update GitHub tracking issues in-place, but print them to stdout")
	verbose := flag.Bool("verbose", false, "If true, print the resulting tracking issue bodies to stdout")

	flag.Parse()

	if err := run(*token, *org, *dry, *verbose); err != nil {
		log.Fatal(err)
	}
}

func run(token, org string, dry, verbose bool) (err error) {
	if token == "" {
		return fmt.Errorf("no -token given")
	}

	if org == "" {
		return fmt.Errorf("no -org given")
	}

	ctx := context.Background()
	cli := graphql.NewClient("https://api.github.com/graphql", graphql.WithHTTPClient(
		oauth2.NewClient(ctx, oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		))),
	)

	trackingIssues, err := ListTrackingIssues(ctx, cli, org)
	if err != nil {
		return errors.Wrap(err, "ListTrackingIssues")
	}

	var openTrackingIssues []*Issue
	for _, trackingIssue := range trackingIssues {
		if strings.EqualFold(trackingIssue.State, "open") {
			openTrackingIssues = append(openTrackingIssues, trackingIssue)
		}
	}

	if len(openTrackingIssues) == 0 {
		log.Printf("No open tracking issues found. Exiting.")
		return nil
	}

	issues, pullRequests, err := LoadTrackingIssues(ctx, cli, org, openTrackingIssues)
	if err != nil {
		return errors.Wrap(err, "LoadTrackingIssues")
	}

	if err := Resolve(trackingIssues, issues, pullRequests); err != nil {
		return err
	}

	var updatedTrackingIssues []*Issue
	for _, trackingIssue := range openTrackingIssues {
		context := NewIssueContext(trackingIssue, trackingIssues, issues, pullRequests)

		updated, ok := trackingIssue.UpdateBody(RenderTrackingIssue(context))
		if !ok {
			log.Printf("failed to patch work section in %q %s", trackingIssue.Title, trackingIssue.URL)
			continue
		}
		if !updated {
			log.Printf("%q %s not modified.", trackingIssue.Title, trackingIssue.URL)
			continue
		}

		if !dry {
			log.Printf("%q %s modified", trackingIssue.Title, trackingIssue.URL)
			updatedTrackingIssues = append(updatedTrackingIssues, trackingIssue)
		} else {
			log.Printf("%q %s modified, but not updated due to -dry=true.", trackingIssue.Title, trackingIssue.URL)
		}

		if verbose {
			log.Printf("%q %s body\n%s\n\n", trackingIssue.Title, trackingIssue.URL, trackingIssue.Body)
		}
	}

	if err := updateIssues(ctx, cli, updatedTrackingIssues); err != nil {
		return err
	}

	return nil
}
