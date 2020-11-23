# Basic code intelligence

Sourcegraph comes with built-in code intelligence provided by search-based heuristics.

If you are interested in enabling precise code intelligence for your repository, see our [quickstart guide](../lsif_quickstart.md).

## How does it work?

[Basic code intelligence](https://github.com/sourcegraph/sourcegraph-basic-code-intel) is able to provide 3 core code intelligence features:

- Jump to definition: it performs a [symbol search](../../code_search/explanations/features.md#symbol-search)
- Hover documentation: it first finds the definition then extracts documentation from comments near the definition
- Find references: it performs a case-sensitive word-boundary cross-repository [plain text search](../../code_search/explanations/features.md#powerful-flexible-queries) for the given symbol

Basic code intelligence also filters results by file extension and by imports at the top of the file for some languages.

## What languages are supported?

Basic code intelligence supports all of [the most popular programming languages](https://sourcegraph.com/extensions).

Are you using a language we don't support? [File a GitHub issue](https://github.com/sourcegraph/sourcegraph/issues/new/choose) or [submit a PR](https://github.com/sourcegraph/sourcegraph-basic-code-intel#adding-a-new-sourcegraphsourcegraph-lang-extension).

## Why are my results sometimes incorrect?

Basic code intelligence uses search-based heuristics, rather than parsing the code into an [abstract syntax tree](https://en.wikipedia.org/wiki/Abstract_syntax_tree) (AST). Incorrect results occur more often for tokens with common names (such as `Get`) than for tokens with more unique names simply because those tokens appear more often in the search index.

If you require 100% confidence in accuracy for a definition or reference results for a symbol you hovered over we recommend utilizing precise code intelligence. Scenarios where you may still get basic code intelligence results even with precision on are described in more detail in the [precise code intelligence docs](./precise_code_intelligence.md).
