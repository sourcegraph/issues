# Precise code intelligence

Precise code intelligence relies on [LSIF](https://github.com/Microsoft/language-server-protocol/blob/master/indexFormat/specification.md) 
(Language Server Index Format) data to deliver precomputed code intelligence. It provides fast and highly accurate code intelligence but needs to be periodically generated and uploaded to your Sourcegraph instance. Precise code intelligence is an opt-in feature: repositories for which you have not uploaded LSIF data will continue to use the basic search based code intelligence.

> NOTE: Precise code intelligence using LSIF is supported in Sourcegraph 3.8 and up.

## Getting started

First navigate to your [global settings](https://sourcegraph.example.com/site-admin/global-settings) on your Sourcegraph instance and enable LSIF:

```json
  "codeIntel.lsif": true
```

Then select a language specific guide from the list below to generate and upload LSIF files for your repository. The LSIF data is used by Sourcegraph instances to power code intelligence requests (such as hovers, definitions, and references). If you don't see a guide for the language you need below, follow our general [quickstart guide](../lsif_quickstart.md) to setup precise code intelligence.

- [Go](../how-to/index_a_go_repository.md)
- [Javascript / Typescript](../how-to/index_a_typescript_and_javascript_repository.md)

After completing the initial setup, follow the [continuous integration guide](../how-to/adding_lsif_to_workflows.md#lsif-in-continuous-integration) to automate indexing of code changes as part of your CI/CD practice.

> NOTE: LSIF support is still a relatively new feature. We are currently validating that the feature remains responsive even with tens of thousands of repositories. We recommend you start by uploading a smaller number of key repositories. Once you reach 50 to 100 repositories we will be able to provide specific recommendations for ensuring stability and performance of your Sourcegraph instance.

## Cross-repository code intelligence

Cross-repository code intelligence will only be powered by LSIF when **both** repositories have LSIF data. When the current file has LSIF data and the other repository doesn't, the missing precise results will be supplemented with imprecise search-based code intelligence.

## Why are my results sometimes incorrect?

If LSIF data is not found for a particular file in a repository, Sourcegraph will fall back to basic code intelligence. You may occasionally see results from [basic code intelligence](basic_code_intelligence.md) even when you have uploaded LSIF data. Such results are indicated with a ![tooltip](../img/basic-code-intel-tooltip.svg) tooltip. This can happen in the following scenarios:

- The symbol has LSIF data, but it is defined in a repository which does not have LSIF data.
- The nearest commit that has LSIF data is too far away from your browsing commit. [The limit is 100 commits](https://github.com/sourcegraph/sourcegraph/blob/e7803474dbac8021e93ae2af930269045aece079/lsif/src/shared/constants.ts#L25) ahead/behind.
- The line containing the symbol was created or edited between the nearest indexed commit and the commit being browsed.
- The _Find references_ panel will always include search-based results, but only after all of the precise results have been displayed. This ensures every symbol has code intelligence.

## Size of upload data

The following table gives a rough estimate for the space and time requirements for indexing and conversion. These repositories are a representative sample of public Go repositories available on GitHub. The working tree size is the size of the clone at the given commit (without git history), the number of files indexed, and the number of lines of Go code in the repository. The index size gives the size of the uncompressed LSIF output of the indexer. The conversion size gives the total amount of disk space occupied after uploading the dump to a Sourcegraph instance.

| Repository | Working tree size | Index time | Index size | Processing time | Post-processing size |
| ------------------------------------------------------------------- | ------------------------------- | ------ | ----- | ------- | ----- |
| [bigcache](https://github.com/allegro/bigcache/tree/b7689f7)        | 216KB,   32 files,   2.585k loc |  1.18s | 3.5MB |   0.45s | 0.6MB |
| [sqlc](https://github.com/kyleconroy/sqlc/tree/16cc4e9)             | 396KB,   24 files,   7.041k loc |  1.53s | 7.2MB |   1.62s | 1.6MB |
| [nebula](https://github.com/slackhq/nebula/tree/a680ac2)            | 700KB,   71 files,  10.704k loc |  2.48s |  16MB |   1.63s | 2.9MB |
| [cayley](https://github.com/cayleygraph/cayley/tree/4d89b8a)        | 5.6MB,  226 files,  36.346k loc |  5.58s |  51MB |   4.68s |  11MB |
| [go-ethereum](https://github.com/ethereum/go-ethereum/tree/275cd49) |  27MB,  945 files, 317.664k loc | 20.53s | 255MB |  77.40s |  50MB |
| [kubernetes](https://github.com/kubernetes/kubernetes/tree/e680ad7) | 301MB, 4577 files,   1.550m loc |  1.21m | 910MB |  80.06s | 162MB |
| [aws-sdk-go](https://github.com/aws/aws-sdk-go/tree/18a2d30)        | 119MB, 1759 files,   1.067m loc |  8.20m | 1.3GB | 155.82s | 358MB |

## Data retention policy

The bulk of LSIF data is stored on-disk, and as code intelligence data for a commit ages it becomes less useful. Sourcegraph will automatically remove the least recently uploaded data if the amount of used disk space exceeds a configurable threshold. This value defaults to 10 GiB (10⨉2^30 = 10737418240  bytes), and can be changed via the `DBS_DIR_MAXIMUM_SIZE_BYTES` environment variable.

## More about LSIF

- [Writing an LSIF indexer](writing_an_indexer.md)
- [Adding LSIF to many repositories](../how-to/adding_lsif_to_many_repos.md)

To learn more, check out our lightning talk about LSIF from GopherCon 2019 or the [introductory blog post](https://about.sourcegraph.com/go/code-intelligence-with-lsif):

<iframe width="560" height="315" src="https://www.youtube.com/embed/fMIRKRj_A88" frameborder="0" allow="accelerometer; autoplay; encrypted-media; gyroscope; picture-in-picture" allowfullscreen></iframe>
