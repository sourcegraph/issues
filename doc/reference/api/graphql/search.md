# Sourcegraph search GraphQL API

This page adds some additional depth and context to Sourcegraph's search GraphQL API.

For general information about the GraphQL API and how to use it, see [this page instead](index.md).

## `src` CLI usage (easier than GraphQL)

Putting together a comprehensive GraphQL search query can be difficult. For this reason, we created the [`src` CLI tool](https://github.com/sourcegraph/src-cli) which allows you to simply run a search query and get the JSON results without constructing the GraphQL query:

```
export SRC_ENDPOINT=https://sourcegraph.com
export SRC_ACCESS_TOKEN=secret

src search -json 'repo:pallets/flask error'
```

You can then consume the JSON output directly, add `--get-curl` to get a `curl` execution line, and more. See [the `src` CLI tool](https://github.com/sourcegraph/src-cli) for more details.

## Sourcegraph 3.9+: Experimental paginated search

To enable better programmatic consumption of search results, Sourcegraph 3.9 introduces the ability to consume an entire search result set via multiple paginated search requests. The results will be returned with a stable order (defined below).

**The paginated search API is experimental and has some limitations. It is not yet ready for production use, but we are eager to hear feedback from early adopters as we work to further improve it.**

#### Cursor-based pagination & request flow

Sourcegraph's paginated search API is cursor-based. Each response contains a new cursor indicating where we left off when searching. It tells Sourcegraph where to continue when a future request with that cursor is made. The typical request flow looks something like:

- Fetch results 0-100: `search(query:"some query", first:100, after:null)`
- Fetch results 100-200: `search(query:"some query", first:100, after:$cursor)`
- Fetch results 200-300: `search(query:"some query", first:100, after:$cursor)`

Until `SearchResults.finished` is `true` - indicating no more results are available and with `$cursor` being the value from the previous requests `results.pageInfo.endCursor` field.

#### Choosing the right per-page value

When performing paginated requests, it is very important to choose a per-page value (`first`) based on your use case. Unlike regular paginated APIs which simply iterate over some existing data, Sourcegraph is performing nearly-live searches and compiling the data for you from our index and other sources in realtime.

For example, if you intend to display the results to a user like our web UI it is very important to choose a small `first` in order to always get quick responses. A request for `first:10` may be substantially faster than `first:11` in specific cases such as if the 11th result only exists in the last repository Sourcegraph would search based on your query.

In general, to get the best performance:

- User interfaces intending to display results should specify very conservative per-page values of around 2x as many results will fit on the screen at a given time.
- Programatic consumption of search results should specify generous per-page values around 1,000 to 5,000 (max), depending on their real-time nature.

#### Result ordering

The paginated search API produces search results with an eventually stable order. That is, in general results are in a stable enough order for programmatically consuming the entire result set, and repeated requests for the same search query generally see the same results. But if you for example intend to directly diff two complete result sets received via the paginated API there are some edge cases:

1. If new results are introduced (e.g. via a new commit) while a query or subsequent query is ongoing, Sourcegraph _MAY_ include those results.
2. If results are removed (e.g. via a new commit) while a query or subsequent query is ongoing, Sourcegraph _MAY_ skip over some results in the total set.
3. The order in which results are returned _MAY_ change between Sourcegraph versions.

In the event one of the above three exceptions do occur, Sourcegraph will be returning to you a subset of the overall result set. An easy way to visualize the behavior is as follows:

1. Sourcegraph has a complete ordered result set '[a, b, c, ..., z]'
2. At any point you may request a subset of that complete result set via a paginated request acting like array indexing / slicing.
3. At any point in time the addition/removal of results, or upgrade of Sourcegraph may alter Sourcegraph's complete search result set and add/remove or reorder elements in that array.
4. Subsequent requests for paginated results, hence, would observe that change which may be surprising.

It should be noted that while we do want to improve this behavior in the future, in most use cases it is fine because continued requests for the entire result set will result in eventual consistency.

#### Known limitations

There are a few known limitations with the current implementation:

1. You cannot query multiple result types yet. For example, you cannot ask for both text and symbol results in the same query.
2. The paginated search API currently only works with text results. If you try to include `type:symbol` in your query, for example, an error will be returned.
3. Cursor values given to you by Sourcegraph may change across Sourcegraph versions. In this case, once Sourcegraph is upgraded fetching more results for an ongoing paginated search may result in an error and retrying it from the start may be required.
