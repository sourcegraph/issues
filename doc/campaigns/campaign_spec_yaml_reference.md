# Campaign spec YAML reference

<style>.markdown-body h2 { margin-top: 50px; }</style>

[Sourcegraph campaigns](../index.md) use [campaign specs](../index.md#campaign-specs) to define campaigns.

This page is a reference guide to the campaign spec YAML format in which campaign specs are defined. If you're new to YAML and want a short introduction, see "[Learn YAML in five minutes](https://learnxinyminutes.com/docs/yaml/)."

## [`name`](#name)

The name of the campaign, which is unique among all campaigns in the namespace. A campaign's name is case-preserving.

### Examples

```yaml
name: update-go-import-statements
```

```yaml
name: update-node.js
```

## [`description`](#description)

The description of the campaign. It's rendered as Markdown.

### Examples

```yaml
description: This campaign changes all `fmt.Sprintf` calls to `strconv.Iota`.
```

```yaml
description: |
  This campaign changes all imports from
  
  `gopkg.in/sourcegraph/sourcegraph-in-x86-asm`
  
  to
  
  `github.com/sourcegraph/sourcegraph-in-x86-asm`
```

## [`on`](#on)

The set of repositories (and branches) to run the campaign on, specified as a list of search queries (that match repositories) and/or specific repositories.

### Examples

```yaml
on:
  - repositoriesMatchingQuery: lang:go fmt.Sprintf("%d", :[v]) patterntype:structural
  - repository: github.com/sourcegraph/sourcegraph
```

## [`on.repositoriesMatchingQuery`](#on-repositoriesmatchingquery)

A Sourcegraph search query that matches a set of repositories (and branches). Each matched repository branch is added to the list of repositories that the campaign will be run on.

See "[Code search](../code_search/index.md)" for more information on Sourcegraph search queries.

### Examples

```yaml
on:
  - repositoriesMatchingQuery: file:README.md -repo:github.com/sourcegraph/src-cli
```

```yaml
on:
  - repositoriesMatchingQuery: lang:typescript file:web const changesetStatsFragment
```

## [`on.repository`](#on-repository)

A specific repository (and branch) that is added to the list of repositories that the campaign will be run on.

A `branch` attribute specifies the branch on the repository to propose changes to. If unset, the repository's default branch is used.

### Examples

```yaml
on:
  - repository: github.com/sourcegraph/src-cli
```

```yaml
on:
  - repository: github.com/sourcegraph/sourcegraph
    branch: 3.19-beta
  - repository: github.com/sourcegraph/src-cli
```

## [`steps`](#steps)

The sequence of commands to run (for each repository branch matched in the `on` property) to produce the campaign's changes.

### Examples

```yaml
steps:
  - run: echo "Hello World!" >> README.md
    container: alpine:3
```

```yaml
steps:
  - run: comby -in-place 'fmt.Sprintf("%d", :[v])' 'strconv.Itoa(:[v])' .go -matcher .go -exclude-dir .,vendor
    container: comby/comby
  - run: gofmt -w ./
    container: golang:1.15-alpine
```

```yaml
steps:
  - run: ./update_dependency.sh
    container: our-custom-image
    env:
      OLD_VERSION: 1.31.7
      NEW_VERSION: 1.33.0
```

## [`steps.run`](#steps-run)

The shell command to run in the container. It can also be a multi-line shell script. The working directory is the root directory of the repository checkout.

## [`steps.container`](#steps-run)

The Docker image used to launch the Docker container in which the shell command is run.

The image has to have either the `/bin/sh` or the `/bin/bash` shell.

It is executed using `docker` on the machine on which the [Sourcegraph CLI (`src`)](https://github.com/sourcegraph/src-cli) is executed. If the image exists locally, that is used. Otherwise it's pulled using `docker pull`.

## [`steps.env`](#steps-env)

Environment variables to set in the environment when running this command.

## [`importChangesets`](#importchangesets)

An array describing which already-existing changesets should be imported from the code host into the campaign.

### Examples

```yaml
importChangesets:
  - repository: github.com/sourcegraph/sourcegraph
    externalIDs: [13323, "13343", 13342, 13380]
  - repository: github.com/sourcegraph/src-cli
    externalIDs: [260, 271]
```

## [`importChangesets.repository`](#importchangesets-repository)

The repository name as configured on your Sourcegraph instance.

## [`importChangesets.externalIDs`](#importchangesets-externalids)

The changesets to import from the code host. For GitHub this is the pull request number, for GitLab this is the merge request number, for Bitbucket Server this is the pull request number.

## [`changesetTemplate`](#changesettemplate)

A template describing how to create (and update) changesets with the file changes produced by the command steps.

This defines what the changesets on the code hosts (pull requests on GitHub, merge requests on Gitlab, ...) will look like.

### Examples

```yaml
changesetTemplate:
  title: Replace equivalent fmt.Sprintf calls with strconv.Itoa
  body: This campaign replaces `fmt.Sprintf("%d", integer)` calls with semantically equivalent `strconv.Itoa` calls
  branch: campaigns/sprintf-to-itoa
  commit:
    message: Replacing fmt.Sprintf with strconv.Iota
    author:
      name: Lisa Coder
      email: lisa@example.com
  published: false
```

```yaml
changesetTemplate:
  title: Update rxjs in package.json to newest version
  body: This pull request updates rxjs to the newest version, `6.6.2`.
  branch: campaigns/update-rxjs
  commit:
    message: Update rxjs to 6.6.2
  published: true
```

```yaml
changesetTemplate:
  title: Run go fmt over all Go files
  body: Regular `go fmt` run over all our Go files.
  branch: go-fmt
  commit:
    message: Run go fmt
    author:
      name: Anna Wizard
      email: anna@example.com
  published:
    # Do not meddle in the affairs of wizards, for they are subtle and quick to anger.
    - git.istari.example/*: false
    - git.istari.example/anna/*: true
```

## [`changesetTemplate.title`](#changesettemplate-title)

The title of the changeset on the code host.

## [`changesetTemplate.body`](#changesettemplate-body)

The body (description) of the changeset on the code host. If the code supports Markdown you can use it here.

## [`changesetTemplate.branch`](#changesettemplate-branch)

The name of the Git branch to create or update on each repository with the changes.

## [`changesetTemplate.commit`](#changesettemplate-commit)

The Git commit to create with the changes.

## [`changesetTemplate.commit.message`](#changesettemplate-commit-message)

The Git commit message.

## [`changesetTemplate.commit.author`](#changesettemplate-commit-author)

The `name` and `email` of the Git commit author.

### Examples

```yaml
changesetTemplate:
  commit:
    author:
      name: Alan Turing
      email: alan.turing@example.com
```

## [`changesetTemplate.published`](#changesettemplate-published)

Whether to publish the changeset. This may be a boolean value (ie `true` or `false`), `'draft'`, or [an array to only publish some changesets within the campaign](#publishing-only-specific-changesets).

An unpublished changeset can be previewed on Sourcegraph by any person who can view the campaign, but its commit, branch, and pull request aren't created on the code host.

When `published` is set to `draft` a commit, branch, and pull request / merge request are being created on the code host **in draft mode**. This means:

- On GitHub the changeset will be a [draft pull request](https://docs.github.com/en/free-pro-team@latest/github/collaborating-with-issues-and-pull-requests/about-pull-requests#draft-pull-requests).
- On GitLab the changeset will be a merge request whose title is be prefixed with `'WIP: '` to [flag it as a draft merge request](https://docs.gitlab.com/ee/user/project/merge_requests/work_in_progress_merge_requests.html#adding-the-draft-flag-to-a-merge-request).
- On BitBucket Server draft pull requests are not supported and changesets published as `draft` won't be created.

> NOTE: Changesets that have already been published on a code host as a non-draft (`published: true`) cannot be converted into drafts. Changesets can only go from unpublished to draft to published, but not from published to draft. That also allows you to take it out of draft mode on your code host, without risking Sourcegraph to revert to draft mode.

A published changeset results in a commit, branch, and pull request being created on the code host.

### [Publishing only specific changesets](#publishing-only-specific-changesets)

To publish only specific changesets within a campaign, an array of single-element objects can be provided. For example:

```yaml
published:
  - github.com/sourcegraph/sourcegraph: true
  - github.com/sourcegraph/src-cli: false
  - github.com/sourcegraph/campaignutils: draft
```

Each key will be matched against the repository name using [glob](https://godoc.org/github.com/gobwas/glob#Compile) syntax. The [gobwas/glob library](https://godoc.org/github.com/gobwas/glob#Compile) is used for matching, with the key operators being:

| Term | Meaning |
|------|---------|
| `*`  | Match any sequence of characters |
| `?`  | Match any single character |
| `[ab]` | Match either `a` or `b` |
| `[a-z]` | Match any character between `a` and `z`, inclusive |
| `{abc,def}` | Match either `abc` or `def` |

If multiple entries match a repository, then the last entry will be used. For example, `github.com/a/b` will _not_ be published given this configuration:

```yaml
published:
  - github.com/a/*: true
  - github.com/*: false
```

If no entries match, then the repository will not be published. To make the default true, add a wildcard entry as the first item in the array:

```yaml
published:
  - "*": true
  - github.com/*: false
```

> NOTE: The standalone `"*"` is quoted in the key to avoid ambiguity in the YAML document.

### Examples

To publish all changesets created by a campaign:

```yaml
changesetTemplate:
  published: true
```

To publish all changesets created by a campaign as drafts:

```yaml
changesetTemplate:
  published: draft
```

To only publish changesets within the `sourcegraph` GitHub organization:

```yaml
changesetTemplate:
  published:
    - github.com/sourcegraph/*: true
```

To publish all changesets that are not on GitLab:

```yaml
changesetTemplate:
  published:
    - "*": true
    - gitlab.com/*: false
```

To publish all changesets on GitHub as draft:

```yaml
changesetTemplate:
  published:
    - "*": true
    - github.com/*: draft
```
