# Search pagination

This document describes at a high-level how our backend implements search pagination.

Search pagination is an experimental API [we added as part of Sourcegraph 3.9](https://github.com/sourcegraph/sourcegraph/pull/4796) and is currently only used for programmatic consumption of search results, not in the web UI for example.

## Prerequisites

Read the [life of a search query](life-of-a-search-query.md) document first, as search pagination effectively acts as a layer _on top of_ our search architecture. We will only talk broadly about the concepts applied to perform pagination, while the previously mentioned document goes into more depth about the actual codepaths we reference here.

Additionally, you should first read and understand how an end-user would make use of our pagination API by reading [the search API documentation](https://docs.sourcegraph.com/api/graphql/search).

## Entrypoints

First, we detect a search query as being paginated at the primary GraphQL entrypoint and initialize a pagination struct here: https://sourcegraph.com/github.com/sourcegraph/sourcegraph@sg/paginated-search/-/blob/cmd/frontend/graphqlbackend/search.go#L76-92

Next, the actual pagination begins when `paginatedResults` is called: https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph%24%40sg/paginated-search+paginatedResults

## Terminology

For the purposes of clarity in this document, we will use the following terms:

### "search backend"

We use the term "search backend" to describe a function which performs a search for a specific result `type:`. The function can perform an indexed (e.g. zoekt text search or symbol search) or unindexed (e.g. commit/diff search). The list of these at the time of writing are:

- `searchRepositories` for finding `type:repo` results
- `searchSymbols` for finding `type:symbol` results
- `searchFilesInRepos` for finding `type:file` (text) and `type:path` (filepath) results.
- `searchCommitDiffsInRepos` for finding `type:diff` results.
- `searchCommitLogInRepos` for finding `type:commit` results.

### "cursor"

The cursor is a base64 opaque string from a clients point of view. It contains metadata that is passed back and forth between the client and the server in order for the server to know where the client left off and where the server should begin looking for more results. Its actual definition in code is [here](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph+type+searchCursor+struct).

Our cursors are considered to be:

- Usable at any point in the future (exception: we document this is not true across upgrades _currently_).
- Loosely associated with a user. For example, we may cache results for for `Cursor123:UserA` and if `UserB` tries to make use of `Cursor123` it may produce a different result set (e.g. due to repository permissions).

Note: "cursor-based pagination" is an established concept and you can find resources describing different cursor-based pagination approaches elsewhere online.

## How search pagination works

### Shared backends

Shared between both paginated and non-paginated search are the various _search backends_. These backends take a list of repositories to search through and produce a set of results (`[]searchResultResolver`) and a metadata structure about those results (`searchResultsCommon`) describing some properties like if any repositories timed out during the search, if we hit a limit during the search, etc.

### Non-paginated search process

In the case of non-paginated search, we perform a basic process:

1. [Identify which repositories to search](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph+file:search_results.go+determineRepos).
2. For every desired result `type:`, [concurrently invoke the search backend](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph+file:search_results.go+goroutine.Go) to search across all repositories.
3. [Wait for the results to come back](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/cmd/frontend/graphqlbackend/search_results.go#L1045-1055), being more aggressive with timeouts of slower search backends deemed to be optional.

If you want a large number of results with the above, you must specify a larger enough `count:` or `timeout:` parameter in your search query or else you may not get back enough. In some cases, it is not possible to specify a large enough `timeout` which has a maximum allowed value of 2 minutes.

### Paginated search process

Paginated search effectively reimplements the same process described above, just with alterations needed to make the process one that is done between the client and server via multiple requests and in a determistic way.

For a paginated request, we perform the following process:

1. [Identify which repositories to search](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph+file:search_pagination.go+determineRepos) and [sort them globally](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph+file:search_pagination.go+%22we+must+sort+the+repositories+deterministically.%22).
2. [Create a wrapper around each search backend](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph+file:search_pagination.go+%22%26repoPaginationPlan%22) which handles searching through smaller batches of repositories (in contrast to all repositories as non-paginated search does) to get enough results for the single paginated request and produce a cursor describing where a subsequent request can "pick up" searching again for the next page of results.
3. [Execute the wrapped search backends](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/cmd/frontend/graphqlbackend/search_pagination.go#L176-182) and return the relevant cursor.

#2 above, while sounding simple in practice, is where the vast majority of the complexity lies:

- We don't know _how many results_ there are for a query in a repository without actually executing the search backends for the repository.
- If we search _too many repositories_, we may get back hundreds of thousands of results when we only need e.g. 1,000 to fulfill the request and the pagination API will be slow.
- If we search _too few repositories_, we may take a very long time to find sufficient results and paginated search would end up being _slower_ than non-paginated search.
- Some search backends are substantially faster/slower than others. For example, zoekt text and zoekt symbol search are indexed and very fast -- while commit and diff search are unindexed live searches and very slow. How do we account for this?
- In the future, Zoekt search backends could potentially have actual pagination support. How would it integrate?

To address the above, we use a relatively simple solution which is applicable to all search backends today with tuning parameters we can apply to each search backend to handle their different performance characteristics -- while still allowing us to make use of search backends in the future with true pagination support.

### The repository pagination planner

The repository pagination planner is what wraps existing search backends to provide result-level pagination.

Consider that you have 10 repositories added to Sourcegraph that are searchable by you (in practice, this would be a much larger number like say 10,000 to 80,000):

```
[A, B, C, D, E, F, G, H, I, J]
```

Each search backend is capable of taking a subset of this repository list and searching over it for results. The question is, how many do we search at a time?

- If we searched over _all_ of them at once, we would effectively be performing a non-paginated search.
- If we searched over _one_ at a time, it could be quite a while before we find any results if the only repositories with results are e.g. H/I/J as we would need to search all prior ones first.

Ideally, we would search over N repositories determined by:

1. How fast/slow the search backend is
2. How many resources the Sourcegraph instance has
3. An estimation of the worst-case scenario performance

And this is exactly what `repoPaginationPlan` describes: a plan for executing a search function that searches only over a set of repositories (i.e. the search function offers no pagination or result-level pagination capabilities) to ultimately provide result-level pagination. That is, if you have a function which can provide a complete list of results for a given repository, then the planner can be used to implement result-level pagination on top of that function.

To determine how many of the repositories to search, it uses the following tunable factors:

- `searchBucketDivisor` => search `numTotalReposOnSourcegraph() / searchBucketDivisor` repositories at a given time
  - This acts as an approximation for how many resources the Sourcegraph instance has. The more repositories, the more resources are expected.
- `searchBucketMin` => search no less than this many at a given time
  - For example, this ensures that a small brand new Sourcegraph instance with just 10 repositories does not needlessly search in series when all could be safely searched in parallel.
- `searchBucketMax` => search no more than this many repository at a given time
  - For example, this ensures that a very large instance (say with 80k repositories) does not try to search too many repositories at once and take down the network temporarily in doing so.

#### Cursor generation

The repository pagination planner is also responsible for producing a cursor for where a subsequent request could begin searching again in the globally-sorted list of repositories. This is tricky because our client is asking for result-level pagination but our search backends effectively only perform repository-level pagination. For example, consider that the planner searches a batch of repositories and finds the following results for repositories (A-Z) and results (0-9):

```
[A1, A2, A3, B1, B2, B3, C1, C2, C3]
```

If the client makes an initial paginated request and asks for `first:5` then we need to send them results `A1` through `B2` and a cursor that indicates we will continue searching again at `B3` so they get the last result in repository `B`.

For this reason, the cursor that we give to users in responses has both a _global repository offset_ and _result offset_ within the first repository. In the above example, we would return a cursor with `RepositoryOffset: 1` to indicate that we should resume searching in repository `B` and `ResultOffset: 2` to indicate that we should resume searching starting at result 3.
