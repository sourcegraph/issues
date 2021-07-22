/**
 * The arguments for the connection query
 */
export interface ConnectionQueryArguments {
    first?: number
    after?: string
    query?: string
}

/**
 * See https://facebook.github.io/relay/graphql/connections.htm.
 */
export interface Connection<N> {
    /**
     * The list of items (nodes) in this connection's current page.
     */
    nodes: N[]

    /**
     * The total count of items in the connection (not subject to pagination). The type accounts
     * for all known GraphQL XyzConnection types.
     *
     * If the value is a number, then the precise total count is known. If null, then the total
     * count was not precisely computable for this particular query (but might be for other queries).
     * If undefined, then the resolver never supports producing a total count.
     *
     * In the future, the UI might show `null` differently from `undefined`, but for now, the
     * distinction is maintained solely for typechecking to pass.
     */
    totalCount?: number | null

    /**
     * If set, indicates whether there is a next page. Not all GraphQL XyzConnection types return
     * pageInfo (if not, then they generally all do return totalCount). If there is a cursor to use
     * on a subsequent request it is also provided here.
     */
    pageInfo?: { hasNextPage: boolean; endCursor?: string | null }

    /**
     * If set, this error is displayed. Even when there is an error, the results are still displayed.
     */
    error?: string | null
}
