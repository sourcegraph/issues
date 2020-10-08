# Updating Go import statements using Comby

This campaign rewrites Go import paths for the `log15` package from `gopkg.in/inconshreveable/log15.v2` to `github.com/inconshreveable/log15` using [Comby](https://comby.dev/).

It can handle single-package import statements like these

```go
import "gopkg.in/inconshreveable/log15.v2"
```

and multi-package import statements like these:

```go
import (
	"io"

	"github.com/pkg/errors"
	"gopkg.in/inconshreveable/log15.v2"
)
```

## Campaign spec

```yaml
name: update-log15-import
description: This campaign updates Go import paths for the `log15` package from `gopkg.in/inconshreveable/log15.v2` to `github.com/inconshreveable/log15` using [Comby](https://comby.dev/)

# Find all repositories that contain the import we want to change.
on:
  - repositoriesMatchingQuery: lang:go gopkg.in/inconshreveable/log15.v2

# In each repository
steps:
  # we first replace the import when it's part of a multi-package import statement
  - run: comby -in-place 'import (:[before]"gopkg.in/inconshreveable/log15.v2":[after])' 'import (:[before]"github.com/inconshreveable/log15":[after])' .go -matcher .go -exclude-dir .,vendor
    container: comby/comby
  # ... and when it's a single import line.
  - run: comby -in-place 'import "gopkg.in/inconshreveable/log15.v2"' 'import "github.com/inconshreveable/log15"' .go -matcher .go -exclude-dir .,vendor
    container: comby/comby

# Describe the changeset (e.g., GitHub pull request) you want for each repository.
changesetTemplate:
  title: Update import path for log15 package to use GitHub
  body: Updates Go import paths for the `log15` package from `gopkg.in/inconshreveable/log15.v2` to `github.com/inconshreveable/log15` using [Comby](https://comby.dev/)
  branch: campaigns/update-log15-import # Push the commit to this branch.
  commit:
    message: Fix import path for log15 package
  published: false
```

## Instructions

1. Save the campaign spec above as `YOUR_CAMPAIGN_SPEC.campaign.yaml`.
1. Create a campaign from the campaign spec by running the following [Sourcegraph CLI (`src`)](https://github.com/sourcegraph/src-cli) command:

    <pre><code>src campaign preview -f <em>YOUR_CAMPAIGN_SPEC.campaign.yaml</em> -namespace USERNAME_OR_ORG</code></pre>

1. Open the preview URL that the command printed out.
1. Examine the preview. Confirm that the changesets are the ones you intended to track. If not, edit the campaign spec and then rerun the command above.
1. Click the **Create campaign** button.
