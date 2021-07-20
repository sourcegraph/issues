package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/cockroachdb/errors"
)

// see https://api.slack.com/reference/block-kit/blocks
type slackMessage struct {
	Blocks []slackBlock `json:"blocks"`
}

type slackBlock map[string]interface{}

const slackTextMarkdown = "mrkdwn"

type slackText struct {
	Type string `json:"type"` // just use `slackTextMarkdown` for the most part
	Text string `json:"text"`
}

func newSlackButtonRun(runID string) slackBlock {
	return slackBlock{
		"type": "button",
		"text": slackText{
			Type: "plain_text",
			Text: "View run",
		},
		"url": fmt.Sprintf("https://github.com/sourcegraph/sourcegraph/actions/runs/%s", runID),
	}
}

func newSlackButtonDocs() slackBlock {
	return slackBlock{
		"type": "button",
		"text": slackText{
			Type: "plain_text",
			Text: "Docs",
		},
		"url": "https://about.sourcegraph.com/handbook/engineering/distribution/tools/resources_report",
	}
}

func newSlackButtonSheet(sheetID, sheetPage string) slackBlock {
	url := fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s", sheetID)
	if sheetPage != "" {
		url = fmt.Sprintf("%s#gid=%s", url, sheetPage)
	}
	return slackBlock{
		"type": "button",
		"text": slackText{
			Type: "plain_text",
			Text: "Report",
		},
		"style": "primary",
		"url":   url,
	}
}

func reportError(ctx context.Context, opts options, err error, scope string) {
	if *opts.slackWebhook != "" {
		blocks := []slackBlock{{
			"type": "section",
			"text": &slackText{
				Type: slackTextMarkdown,
				Text: fmt.Sprintf(":warning: Error encountered: %s: %v", scope, err),
			},
		}}
		if *opts.runID != "" {
			blocks = append(blocks, slackBlock{
				"type":     "actions",
				"elements": []slackBlock{newSlackButtonRun(*opts.runID), newSlackButtonDocs()},
			})
		}
		slackErr := sendSlackBlocks(ctx, *opts.slackWebhook, blocks)
		if slackErr != nil {
			log.Printf("slack: %v", err)
		}
	}
	if *opts.verbose {
		log.Printf("%s: %v", scope, err)
	}
}

func sendSlackBlocks(ctx context.Context, webhook string, blocks []slackBlock) error {
	b, err := json.Marshal(&slackMessage{blocks})
	if err != nil {
		return errors.Errorf("failed to post report to slack: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, webhook, bytes.NewReader(b))
	if err != nil {
		return errors.Errorf("failed to post report to slack: %w", err)
	}
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return errors.Errorf("failed to post report to slack: %w", err)
	}
	if resp.StatusCode != 200 {
		defer resp.Body.Close()
		message, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.Errorf("failed to post report to slack: %s", resp.Status)
		}
		return errors.Errorf("failed to post report to slack: %s", string(message))
	}
	return nil
}
