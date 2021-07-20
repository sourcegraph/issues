package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
)

const resultsBuffer = 5

// options declares configuration available to users of the resourcs-report
//
// See flag usage strings in `main()` or `resources-report -help`
type options struct {
	slackWebhook        *string
	sheetID             *string
	sheetPruneOlderThan *time.Duration

	window          *time.Duration
	highlightWindow *time.Duration

	gcp                *bool
	gcpLabelsAllowlist map[string]string

	aws              *bool
	awsTagsAllowlist map[string]string

	runID   *string
	dry     *bool
	verbose *bool
	timeout *time.Duration
}

func main() {
	help := flag.Bool("help", false, "Show help text")
	gcpAllowlistLabelsStr := flag.String("gcp.allowlist", "", "GCP labels to allowlist (comma-separated key:value pairs)")
	awsAllowlistTagsStr := flag.String("aws.allowlist", "", "AWS tags to allowlist (comma-separated key:value pairs)")
	opts := options{
		slackWebhook:        flag.String("slack.webhook", os.Getenv("SLACK_WEBHOOK"), "Slack webhook to post updates to"),
		sheetID:             flag.String("sheet.id", os.Getenv("SHEET_ID"), "ID of Google Sheet to write to"),
		sheetPruneOlderThan: flag.Duration("sheet.pruneolderthan", 30*(24*time.Hour), "Prune report pages older than the given duration"),

		gcp:             flag.Bool("gcp", false, "Report on Google Cloud resources"),
		aws:             flag.Bool("aws", false, "Report on Amazon Web Services resources"),
		window:          flag.Duration("window", 48*time.Hour, "Restrict results to resources created within a period"),
		highlightWindow: flag.Duration("window.highlight", 24*time.Hour, "Highlight resources created within a period"),

		runID:   flag.String("run.id", os.Getenv("GITHUB_RUN_ID"), "ID of workflow run (for usage in GitHub Actions)"),
		dry:     flag.Bool("dry", false, "Do not post updates to slack, but print them to stdout"),
		verbose: flag.Bool("verbose", false, "Print debug output to stdout (WARNING: might include sensitive information)"),
		timeout: flag.Duration("timeout", time.Minute, "Set a timeout for report generation"),
	}
	flag.Parse()
	if *help {
		flag.CommandLine.Usage()
		return
	}
	if *gcpAllowlistLabelsStr != "" {
		opts.gcpLabelsAllowlist = csvToMap(*gcpAllowlistLabelsStr)
	}
	if *awsAllowlistTagsStr != "" {
		opts.awsTagsAllowlist = csvToMap(*awsAllowlistTagsStr)
	}
	if err := run(opts); err != nil {
		log.Fatal(err)
	}
	log.Println("done")
}

func run(opts options) error {
	ctx, cancel := context.WithTimeout(context.Background(), *opts.timeout)
	defer cancel()

	// Collect resources - let detailed errors be handled by reportErr, which
	// will attempt to send it to Slack. This hopefully prevents errors from
	// revealing too much in our public build logs. If Slack fails, just log it
	// and hope Slack doesn't spit out anything sensitive.
	var resources Resources
	since := time.Now().UTC().Add(-*opts.window)
	if *opts.gcp {
		rs, err := collectGCPResources(ctx, since, *opts.verbose, opts.gcpLabelsAllowlist)
		if err != nil {
			reportError(ctx, opts, err, "gcp")
			return errors.Errorf("gcp: failed to collect resources")
		}
		resources = append(resources, rs...)
	}
	if *opts.aws {
		rs, err := collectAWSResources(ctx, since, *opts.verbose, opts.awsTagsAllowlist)
		if err != nil {
			reportError(ctx, opts, err, "aws")
			return errors.Errorf("aws: failed to collect resources")
		}
		resources = append(resources, rs...)
	}
	sort.Sort(resources)

	// report results
	if *opts.verbose {
		log.Println("collected resources:\n", reportString(resources))
		log.Printf("found a total of %d resources created since %s", len(resources), since.String())
	}
	if !*opts.dry {
		if err := generateReport(ctx, opts, resources); err != nil {
			reportError(ctx, opts, err, "report")
			return errors.Errorf("report: %w", err)
		}
	}

	return nil
}

func reportString(resources Resources) string {
	if len(resources) == 0 {
		return "none\n"
	}
	var output string
	for _, r := range resources {
		output += fmt.Sprintf(" * %+v\n", r)
	}
	return output
}

// csvToMap accepts a comma-delimited set of key:pair values (e.g. `key1:value1,key2:value2`)
// and converts it to a map.
func csvToMap(str string) map[string]string {
	m := map[string]string{}
	for _, pair := range strings.Split(str, ",") {
		keyValue := strings.Split(pair, ":")
		m[keyValue[0]] = keyValue[1]
	}
	return m
}
