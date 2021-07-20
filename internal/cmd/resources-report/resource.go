package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
)

type Platform string

const (
	PlatformGCP Platform = "gcp"
	PlatformAWS Platform = "aws"
)

type Resource struct {
	Platform   Platform
	Identifier string
	Type       string
	Location   string
	Owner      string
	Created    time.Time
	Meta       map[string]interface{}

	Allowed bool
}

type Resources []Resource

func (r Resources) Less(i, j int) bool { return r[i].Created.After(r[j].Created) }
func (r Resources) Len() int           { return len(r) }
func (r Resources) Swap(i, j int) {
	tmp := r[i]
	r[i] = r[j]
	r[j] = tmp
}

// NonAllowed returns only resources that are not allowed
func (r Resources) NonAllowed() (filtered Resources, allowed int) {
	for _, resource := range r {
		if resource.Allowed {
			allowed++
		} else {
			filtered = append(filtered, resource)
		}
	}
	return
}

func hasPrefix(value string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func generateReport(ctx context.Context, opts options, resources Resources) error {
	// count and drop allowed resources
	filteredResources, allowed := resources.NonAllowed()

	// resources are sorted by creation beforehand
	highlightSince := time.Now().Add(-*opts.highlightWindow).UTC()
	var highlighted int
	for i, r := range filteredResources {
		if r.Created.UTC().After(highlightSince) {
			highlighted = i + 1
		} else {
			break
		}
	}

	// populate google sheet with data
	var reportPage string
	if *opts.sheetID != "" {
		var err error
		if reportPage, err = updateSheet(ctx, *opts.sheetID, filteredResources, updateSheetOptions{
			HighlightedRows: highlighted,
			PruneOlderThan:  *opts.sheetPruneOlderThan,
			Verbose:         *opts.verbose,
		}); err != nil {
			return errors.Errorf("sheets: %w", err)
		}
	}

	// generate message to deliver
	if *opts.slackWebhook != "" {
		buttons := []slackBlock{
			newSlackButtonSheet(*opts.sheetID, reportPage),
			newSlackButtonDocs(),
		}
		if *opts.runID != "" {
			buttons = append(buttons, newSlackButtonRun(*opts.runID))
		}
		blocks := []slackBlock{
			{
				"type": "section",
				"text": &slackText{
					Type: slackTextMarkdown,
					Text: fmt.Sprintf(`:package: I've found:
- %d resources created in the past %s
- %d resources created in the past %s
- %d resources were allowed`,
						highlighted, opts.highlightWindow, len(filteredResources), opts.window, allowed),
				},
			},
			{
				"type":     "actions",
				"elements": buttons,
			},
		}
		if err := sendSlackBlocks(ctx, *opts.slackWebhook, blocks); err != nil {
			return errors.Errorf("slack: %w", err)
		}
	}

	return nil
}

// hasKeyValue returns true if the given data has a key-value entry matching one in kvs
func hasKeyValue(data, kvs map[string]string) bool {
	for key, value := range kvs {
		if data[key] == value {
			return true
		}
	}
	return false
}
