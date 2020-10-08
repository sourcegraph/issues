# Code intelligence

Code intelligence provides advanced code navigation features that let developers explore source code. It displays rich metadata about functions, variables, and cross-references in the code. Visit the [features](./features.md) page to learn more or jump to a section for a specific feature:

- [Hover tooltips](./features.md#hover-tooltips-with-documentation-and-type-signatures)
- [Go to definition](./features.md#go-to-definition)
- [Find references](./features.md#find-references)
- [Symbol search](./features.md#symbol-search)
 
Code intelligence is enabled by [Sourcegraph extensions](../../extensions/index.md) and provides users with two different types of code intelligence; basic and precise. 

**Basic** is [search-based code intelligence](./basic_code_intelligence.md) that works out of the box with all of the most popular programming languages via [extensions](https://sourcegraph.com/extensions?query=category%3A%22Programming+languages%22). 

**Precise** code intelligence is an opt-in feature that is enabled in your admin settings and requires you to upload [LSIF data](./lsif_quickstart.md) for each repository to your Sourcegraph instance. Once setup is complete on Sourcegraph, precise code intelligence is available for use across popular development tools:

- On the Sourcegraph web UI
- On code files on your code host, via [integrations](../../integration/index.md)
- On diffs in your code review tool, via integrations
- Via the [Sourcegraph API](https://docs.sourcegraph.com/api/graphql)

## Basic vs Precise

Basic code intelligence is available by default on all Sourcegraph instances and provides fuzzy code intelligence using a combination of ctags and search. It is great for immediate access to code intelligence features, but due to its dependence on text-based search its results are considered imprecise. 

Precise code intelligence returns metadata from a knowledge graph that is generated through code analysis. The precomputation step is used to generate the graph results in lookups that are fast and have a high degree of accuracy. 

To learn more about how to work with each type of code intelligence visit the [basic](./basic_code_intelligence.md) and [precise](./precise_code_intelligence.md) sections.

## Getting started

- Setup your [Sourcegraph instance](../../admin/install/index.md), then enable the [Sourcegraph extension](https://docs.sourcegraph.com/extensions) for each language you want to use. The language extensions should be on by default for a new instance.
- To add code intelligence to your code host and/or code review tool, see the [browser extension](../../integration/browser_extension.md) documentation.
- Interested in trying it out on public code? See [this sample file](https://sourcegraph.com/github.com/dgrijalva/jwt-go/-/blob/token.go#L37:6$references) on Sourcegraph Cloud.
