# Refactor Go code using Comby

This campaign uses Sourcegraph's [structural search](../../search/structural.md) and [Comby](https://comby.dev/) to rewrite Go statements from

```go
fmt.Sprintf("%d", number)
```

to

```go
strconv.Itoa(number)
```

The statements are semantically equivalent, but the latter is clearer.

Since the replacements could require importing the `strconv` package, it uses [`goimports`](https://godoc.org/golang.org/x/tools/cmd/goimports) to update the list of imported packages in all `*.go` files.

## Campaign spec

```yaml
name: sprintf-to-itoa
description: |
  This campaign uses [Comby](https://comby.dev) to replace `fmt.Sprintf` calls
  in Go code with the equivalent but clearer `strconv.Iota` call.

on:
  - repositoriesMatchingQuery: lang:go fmt.Sprintf("%d", :[v]) patterntype:structural

steps:
  - run: comby -in-place 'fmt.Sprintf("%d", :[v])' 'strconv.Itoa(:[v])' .go -matcher .go -exclude-dir .,vendor
    container: comby/comby
  - run: goimports -w $(find . -type f -name '*.go' -not -path "./vendor/*")
    container: unibeautify/goimports

changesetTemplate:
  title: Replace fmt.Sprintf with equivalent strconv.Itoa
  body: This campaign replaces `fmt.Sprintf` with `strconv.Itoa`
  branch: campaigns/sprintf-to-itoa
  commit:
    message: Replacing fmt.Sprintf with strconv.Iota
  published: false
```

## Instructions

1. Save the campaign spec above as `YOUR_CAMPAIGN_SPEC.campaign.yaml`.
1. Create a campaign from the campaign spec by running the following [Sourcegraph CLI (`src`)](https://github.com/sourcegraph/src-cli) command:

    <pre><code>src campaign preview -f <em>YOUR_CAMPAIGN_SPEC.campaign.yaml</em> -namespace USERNAME_OR_ORG</code></pre>

1. Open the preview URL that the command printed out.
1. Examine the preview. Confirm that the changesets are the ones you intended to track. If not, edit the campaign spec and then rerun the command above.
1. Click the **Create campaign** button.
