# Resources report tool [![Resources Report](https://github.com/sourcegraph/sourcegraph/workflows/Resources%20Report/badge.svg)](https://github.com/sourcegraph/sourcegraph/actions?query=workflow%3A%22Resources+Report%22)

This tool reports on the status of various resources in AWS and GCP accounts. Credentials are expected to be set up beforehand, and leverage default credentials of each supported platform. Basic usage:

```sh
go build && ./resources-report --aws --gcp \
  --slack.webhook="https://hooks.slack.com/services/xxxxxxxxx/xxxxxxxxxxx/xxxxxxxxxxxxxxxxxxxxxxxx"
  --sheets.id="xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

This document primarily outlines how to set up this tool and use it directly - for more details, please [refer to the handbook entry](https://about.sourcegraph.com/handbook/engineering/distribution/tools/resources_report).

## Authentication

### Google Cloud Platform

Credentials should be a GCP service account with access to the following permissions in all relevant projects:

- `Viewer`

The path to the key should be set to `GOOGLE_APPLICATION_CREDENTIALS`. For the GitHub Action, set the key to `RR_GCP_ACCOUNT_KEY` in the Secrets tab by encoding it in base64, e.g. `cat $GOOGLE_APPLICATION_CREDENTIALS | base64` (see [`resources-report.yml`](../../../.github/workflows/resources-report.yml)).

#### Google Sheet

This tool dumps the output to a Google Sheet. The sheet ID should be provided to `--sheet.id`. For the GitHub action, set the sheet ID to `RR_SHEET_ID` in the Secrets tab.

The service account used should also have access to the spreadsheet of your choice - to enable this, share the spreadsheet with the service account's email address.

### Amazon Web Services

Credentials should be an AWS IAM with the following permissions:

- `ReadOnlyAccess`

Credentials should be set in `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`. For the GitHub Action, set these variables to `RR_ACCESS_KEY_ID` etc. in the Secrets tab (see [`resources-report.yml`](../../../.github/workflows/resources-report.yml)).

### Slack

This bot can be [configured in Slack](https://api.slack.com/apps/A013EETK25V) with various channel webhooks under "Incoming Webhooks". For the GitHub Action, set the webhook to `RR_SLACK_WEBHOOK` in the Secrets tab (see [`resources-report.yml`](../../../.github/workflows/resources-report.yml)).

## Resources

- GCP: Declare resource types to query in [`gcp.go`](./gcp.go)'s `gcpResources` variable.
- AWS: Declare queries for resources as functions in [`aws.go`](./aws.go)'s `awsResources` variable.

(c) Copyright Sourcegraph 2012-2023.