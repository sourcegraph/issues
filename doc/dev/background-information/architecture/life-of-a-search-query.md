# Life of a search query

This document describes how our backend systems serve search results to clients. There are multiple kinds of searches (e.g. text, repository, file, symbol, diff, commit), but this document will focus on text searches.

## Clients

There are a few ways to perform a search with Sourcegraph:

1. Typing a query into the search bar of the Sourcegraph web application.
2. Typing a query into your browser's location bar after configuring a [browser search engine shortcut](https://docs.sourcegraph.com/integration/browser_search_engine).
3. Using the [src CLI command](https://github.com/sourcegraph/src-cli).

In all cases, clients use the [search query](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph%24+%5Cbsearch%5C%28+file:schema.graphql) in our GraphQL API that is exposed by our [frontend](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/tree/cmd/frontend) service.

## Frontend

The frontend implements the GraphQL search resolver [here](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph%24+"func+%28r+*schemaResolver%29+Search%28").

First, the frontend [resolves which repositories need to be searched](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph%24+%22func+%28r+*searchResolver%29+resolveRepositories%28%22). It parses the query for any repository filters and then queries the database for the list of repositories that match those filters. If no filters are provided then all repositories are searched, as long as the number of repositories doesn't exceed the configured limit. Private instances default to an unlimited number of repositories, but sourcegraph.com has smaller configured limit (`"maxReposToSearch": 400` at the time of writing, but you can check the [site config for the current value](https://sourcegraph.com/site-admin/configuration)) because it isn't cost effective for us to to search/index all open source code on GitHub.

Next, the frontend [determines which repository@revision combinations are indexed by zoekt](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph%24+"zoektIndexedRepos%28"+file:textsearch%5C.go) by [consulting an in-memory cache that is kept up-to-date with regular asynchronous polling](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph%24+"%29+start%28"+file:text.go). It concurrently [queries zoekt](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph%24+%22zoektSearchHEAD%28%22+file:textsearch%5C.go) for indexed repositories and [queries searcher](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph%24+"+searchFilesInRepo%28"+file:textsearch%5C.go) for non-indexed repositories.

## Zoekt (indexed search)

zoekt-webserver [serves search requests](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/zoekt%24+"serveSearchErr%28") by [iterating through matches in the index](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/zoekt%24+"func+%28d+*indexData%29+Search"). It watches the index directory and loads/unloads index files as they come and go.

To decide what to index [zoekt-sourcegraph-indexserver](https://sourcegraph.com/github.com/sourcegraph/zoekt/-/tree/cmd/zoekt-sourcegraph-indexserver) sends an [HTTP Get request to the frontend internal API](https://sourcegraph.com/search?q=r:github.com/sourcegraph/+-file:%28test%7Cspec%29+%22/repos/index%22+fork:yes&patternType=regexp) at most once per minute to fetch the list of repository names to index. For each repository the indexserver will compare what Sourcegraph wants indexed (commit, configuration, etc.) to what is already indexed on disk and will start an index job for anything that is missing.

What we index in a repository is affected by admin configuration (branches, large file allow list). For each repository the indexserver [asks the frontend for configuration](https://sourcegraph.com/search?q=r:github.com/sourcegraph/+-file:%28test%7Cspec%29+%22/search/configuration%22+fork:yes&patternType=regexp).
It fetches git data by calling [another internal frontend API](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/zoekt%24+"func+tarballURL") which [redirects to the archive on gitserver](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph%24+"func+serveGitTar%28"+). If indexing multiple branches, it instead relies on [git shallow clones](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph%24+GitUploadPack&patternType=literal).

## Searcher (non-indexed search)

Searcher is a horizontally scalable stateless service that performs non-indexed code search. Each request is a search on a single repository (the frontend searches multiple repositories by sending one concurrent request per repository). To serve a [search request](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph%24+file:search/search.go+"s.search"), it first [fetches a zip archive of the repo at the desired commit](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/cmd/searcher/search/search.go#L190-199) [from gitserver](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph%24+%22FetchTar:%22+file:searcher/main.go) and then [iterates through the files in the archive](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph%24+%22func+concurrentFind%28%22) to perform the actual search.
