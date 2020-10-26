# LSIF quickstart guide

> NOTE: We are working on creating guides by language, so first check out [language documentation](how-to/index.md#language-specific-guides)! This general guide can be used when a language specific guide is not available.

We'll walk you through installing and generating LSIF data locally on your machine, and then manually uploading the LSIF data to your Sourcegraph instance for your repository. This will let you experiment with the process locally, and test your generated LSIF data on your repository before you update your CI process. The steps for enabling precise code intelligence are as follows:

1. Install Sourcegraph CLI
2. Install LSIF Indexer
3. Generate LSIF data
4. Upload LSIF data

### 1. Install Sourcegraph CLI

The [Sourcegraph CLI (`src`)](https://github.com/sourcegraph/src-cli) is used for uploading LSIF data to your Sourcegraph instance (replace `linux` with `darwin` for macOS):

```
$ curl -L https://sourcegraph.com/.api/src-cli/src_linux_amd64 -o /usr/local/bin/src
$ chmod +x /usr/local/bin/src
```

### 2. Install LSIF indexer

An LSIF indexer is a command line tool that performs code analysis of source code and outputs a file with metadata containing all the definitions, references, and hover documentation in a project in LSIF. The LSIF file is then uploaded to Sourcegraph to power code intelligence features.

Install the indexer for the required programming language of your repository by following the instructions in the indexer's README:

  1. [C++](https://github.com/sourcegraph/lsif-cpp)
  1. [Dart](https://github.com/sourcegraph/lsif-dart)
  1. [Go](https://github.com/sourcegraph/lsif-go)
  1. [Haskell](https://github.com/mpickering/hie-lsif)
  1. [Java](https://github.com/sourcegraph/lsif-java)
  1. [Javascript/Typescript](https://github.com/sourcegraph/lsif-node)
  1. [Jsonnet](https://github.com/sourcegraph/lsif-jsonnet)
  1. [Python](https://github.com/sourcegraph/lsif-py)
  1. [OCaml](https://github.com/rvantonder/lsif-ocaml)
  1. [Scala](https://github.com/sourcegraph/lsif-semanticdb)

### 3. Generate LSIF data

To generate the LSIF data for your repository run the command in the _generate LSIF data_ step found in the README of the installed indexer.

### 4. Upload LSIF data

The upload step is the same for all languages. Make sure the current working directory is a path inside your repository, then use the Sourcegraph CLI to upload the LSIF file:

#### To a private Sourcegraph instance (on prem)
```
$ src -endpoint=<your sourcegraph endpoint> lsif upload -file=<LSIF file (e.g. dump.lsif)>
```

#### To cloud based Sourcegraph.com
```
$ src lsif upload -github-token=<your github token> -file=<LSIF file (e.g. dump.lsif)>
```

The `src-cli` upload command will try to infer the repository and git commit by invoking git commands on your local clone. If git is not installed, is older than version 2.7.0 or you are running on code outside of a git clone, you will need to also specify the `-repo` and `-commit` flags explicitly.

> NOTE: If you're using Sourcegraph.com or have enabled [`lsifEnforceAuth`](https://docs.sourcegraph.com/admin/config/site_config#lsifEnforceAuth) you need to [supply a GitHub token](#proving-ownership-of-a-github-repository) supplied via the `-github-token` flag in the command above.

On successful upload you'll see the following message:

```
Repository: <location of repository>
Commit: <40-character commit associated with this LSIF upload>
File: <LSIF data file>
Root: <subdirectory in the repository where this LSIF dump was generated>

LSIF dump successfully uploaded for processing.
View processing status at <link to your Sourcegraph instance LSIF status>.
```

## Automate code indexing

Now that you have successfully enabled code intelligence for your repository, you can automate source code indexing to ensure precise code intelligence stays up to date with the most recent code changes in the repository. See our [continuous integration guide](how-to/adding_lsif_to_workflows.md) to setup automation.

## Troubleshooting

### Testing

> NOTE: Make sure you have configured your Sourcegraph instance and [enabled precise code intelligence](explanations/precise_code_intelligence.md#enabling-lsif-on-your-sourcegraph-instance).

Once LSIF data has uploaded, open the Sourcegraph UI or your code host (i.e. GitHub) and navigate to any code file that was part of the repository that was analyzed by the LSIF indexer. Hover over a symbol, variable or function name in the file, you should now see rich LSIF metadata as the source for hover-tooltips, definitions, and references.

If precise code intelligence has been correctly enabled hover text should not be decorated with a ![tooltip](img/basic-code-intel-tooltip.svg) icon. This icon indicates the results are from search-based [basic code intelligence](explanations/basic_code_intelligence.md). This tooltip icon will be absent when results are precise!

### Error Logs

To view LSIF indexer processing failures go to **Repository settings > Code intelligence > Activity for this repository** in your Sourcegraph instance. Failures can occur if the LSIF data is invalid (e.g., malformed indexer output), or problems were encountered during processing (e.g., system-level bug, flaky connections, etc). Try again or [file an issue](https://github.com/sourcegraph/sourcegraph/issues/new) if the problem persists.

### Common Errors

Possible errors that can happen during upload include:

- Clone in progress: the instance doesn't have the necessary data to process your upload yet, retry in a few minutes
- Unknown repository (404): check your `-endpoint` and make sure you can view the repository on your Sourcegraph instance
- Invalid commit (404): try visiting the repository at that commit on your Sourcegraph instance to trigger an update
- Invalid auth when using Sourcegraph.com or when [`lsifEnforceAuth`](https://docs.sourcegraph.com/admin/config/site_config#lsifEnforceAuth) is `true` (401 for an invalid token or 404 if the repository cannot be found on GitHub.com): make sure your GitHub token is valid and that the repository is correct
- Unexpected errors (500s): [file an issue](https://github.com/sourcegraph/sourcegraph/issues/new)
