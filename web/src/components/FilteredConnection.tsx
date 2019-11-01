import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import * as H from 'history'
import { uniq, upperFirst } from 'lodash'
import * as React from 'react'
import { combineLatest, merge, Observable, of, Subject, Subscription } from 'rxjs'
import {
    catchError,
    debounceTime,
    delay,
    distinctUntilChanged,
    filter,
    map,
    publishReplay,
    refCount,
    skip,
    startWith,
    switchMap,
    takeUntil,
    tap,
} from 'rxjs/operators'
import * as GQL from '../../../shared/src/graphql/schema'
import { asError, ErrorLike, isErrorLike } from '../../../shared/src/util/errors'
import { pluralize } from '../../../shared/src/util/strings'
import { Form } from './Form'
import { RadioButtons } from './RadioButtons'

/** Checks if the passed value satisfies the GraphQL Node interface */
const hasID = (obj: any): obj is { id: GQL.ID } => obj && typeof obj.id === 'string'

interface FilterProps {
    /** All filters. */
    filters: FilteredConnectionFilter[]

    /** Called when a filter is selected. */
    onDidSelectFilter: (filter: FilteredConnectionFilter) => void

    /** The ID of the active filter. */
    value: string
}

interface FilterState {}

class FilteredConnectionFilterControl extends React.PureComponent<FilterProps, FilterState> {
    public render(): React.ReactFragment {
        return <RadioButtons nodes={this.props.filters} selected={this.props.value} onChange={this.onChange} />
    }

    private onChange: React.ChangeEventHandler<HTMLInputElement> = e => {
        const id = e.currentTarget.value
        const filter = this.props.filters.find(f => f.id === id)!
        this.props.onDidSelectFilter(filter)
    }
}

/**
 * Fields that belong in ConnectionPropsCommon and that don't depend on the type parameters. These are the fields
 * that are most likely to be needed by callers, and it's simpler for them if they are in a parameter-less type.
 */
interface ConnectionDisplayProps {
    /** list HTML element type. Default is <ul>. */
    listComponent?: 'ul' | 'table' | 'div'

    /** CSS class name for the list element (<ul>, <table>, or <div>). */
    listClassName?: string

    /** CSS class name for the "Show more" button. */
    showMoreClassName?: string

    /** The English noun (in singular form) describing what this connection contains. */
    noun: string

    /** The English noun (in plural form) describing what this connection contains. */
    pluralNoun: string

    /** Do not show a "Show more" button. */
    noShowMore?: boolean

    /** Do not show a count summary if all nodes are visible in the list's first page. */
    noSummaryIfAllNodesVisible?: boolean

    /** The component displayed when the list of nodes is empty. */
    emptyElement?: JSX.Element

    /** Append new results onto existing list. This should always be true when paging with cursors. */
    appendResults?: boolean
}

/**
 * Props for the FilteredConnection component's result nodes and associated summary/pagination controls.
 *
 * @template C The GraphQL connection type, such as GQL.IRepositoryConnection.
 * @template N The node type of the GraphQL connection, such as GQL.IRepository (if C is GQL.IRepositoryConnection)
 * @template NP Props passed to `nodeComponent` in addition to `{ node: N }`
 */
interface ConnectionPropsCommon<N, NP = {}> extends ConnectionDisplayProps {
    /** Header row to appear above all nodes. */
    headComponent?: React.ComponentType<{ nodes: N[] }>

    /** Footer row to appear below all nodes. */
    footComponent?: React.ComponentType<{ nodes: N[] }>

    /** The component type to use to display each node. */
    nodeComponent: React.ComponentType<{ node: N } & NP>

    /** Props to pass to each nodeComponent in addition to `{ node: N }`. */
    nodeComponentProps?: NP
}

/** State related to the ConnectionNodes component. */
interface ConnectionStateCommon {
    query: string
    first: number

    connectionQuery?: string

    /**
     * Whether the connection is loading. It is not equivalent to connection === undefined because we preserve the
     * old data for ~250msec while loading to reduce jitter.
     */
    loading: boolean
}

interface ConnectionNodesProps<C extends Connection<N>, N, NP = {}>
    extends ConnectionPropsCommon<N, NP>,
        ConnectionStateCommon {
    /** The fetched connection data or an error (if an error occurred). */
    connection: C

    location: H.Location

    onShowMore: () => void
}

class ConnectionNodes<C extends Connection<N>, N, NP = {}> extends React.PureComponent<ConnectionNodesProps<C, N, NP>> {
    public render(): JSX.Element | null {
        const NodeComponent = this.props.nodeComponent
        const ListComponent: any = this.props.listComponent || 'ul' // TODO: remove cast when https://github.com/Microsoft/TypeScript/issues/28768 is fixed
        const HeadComponent = this.props.headComponent
        const FootComponent = this.props.footComponent

        const hasNextPage = this.props.connection
            ? this.props.connection.pageInfo
                ? this.props.connection.pageInfo.hasNextPage
                : typeof this.props.connection.totalCount === 'number' &&
                  this.props.connection.nodes.length < this.props.connection.totalCount
            : false

        let totalCount: number | null = null
        if (this.props.connection) {
            if (typeof this.props.connection.totalCount === 'number') {
                totalCount = this.props.connection.totalCount
            } else if (
                // TODO(sqs): this line below is wrong because this.props.first might've just been changed and
                // this.props.connection.nodes is still the data fetched from before this.props.first was changed.
                // this causes the UI to incorrectly show "N items total" even when the count is indeterminate right
                // after the user clicks "Show more" but before the new data is loaded.
                this.props.connection.nodes.length < this.props.first ||
                (this.props.connection.nodes.length === this.props.first &&
                    this.props.connection.pageInfo &&
                    typeof this.props.connection.pageInfo.hasNextPage === 'boolean' &&
                    !this.props.connection.pageInfo.hasNextPage)
            ) {
                totalCount = this.props.connection.nodes.length
            }
        }

        let summary: React.ReactFragment | undefined
        if (
            !this.props.loading &&
            this.props.connection &&
            (!this.props.noSummaryIfAllNodesVisible || this.props.connection.nodes.length === 0 || hasNextPage)
        ) {
            if (totalCount !== null && totalCount > 0) {
                summary = (
                    <p className="filtered-connection__summary">
                        <small>
                            <span>
                                {totalCount} {pluralize(this.props.noun, totalCount, this.props.pluralNoun)}{' '}
                                {this.props.connectionQuery ? (
                                    <span>
                                        {' '}
                                        matching <strong>{this.props.connectionQuery}</strong>
                                    </span>
                                ) : (
                                    'total'
                                )}
                            </span>{' '}
                            {this.props.connection.nodes.length < totalCount &&
                                `(showing first ${this.props.connection.nodes.length})`}
                        </small>
                    </p>
                )
            } else if (this.props.connection.pageInfo && this.props.connection.pageInfo.hasNextPage) {
                // No total count to show, but it will show a 'Show more' button.
            } else if (totalCount === 0) {
                summary = this.props.emptyElement || (
                    <p className="filtered-connection__summary">
                        <small>
                            No {this.props.pluralNoun}{' '}
                            {this.props.connectionQuery && (
                                <span>
                                    matching <strong>{this.props.connectionQuery}</strong>
                                </span>
                            )}
                        </small>
                    </p>
                )
            }
        }

        const nodes = this.props.connection.nodes.map((node, i) => (
            <NodeComponent key={hasID(node) ? node.id : i} node={node} {...this.props.nodeComponentProps!} />
        ))

        return (
            <>
                {this.props.connectionQuery && summary}
                {this.props.connection && this.props.connection.nodes.length > 0 && (
                    <ListComponent className={`filtered-connection__nodes ${this.props.listClassName || ''}`}>
                        {HeadComponent && <HeadComponent nodes={this.props.connection.nodes} />}
                        {ListComponent === 'table' ? <tbody>{nodes}</tbody> : nodes}
                        {FootComponent && <FootComponent nodes={this.props.connection.nodes} />}
                    </ListComponent>
                )}
                {!this.props.connectionQuery && summary}
                {!this.props.loading && !this.props.noShowMore && this.props.connection && hasNextPage && (
                    <button
                        type="button"
                        className={`btn btn-secondary btn-sm filtered-connection__show-more ${this.props
                            .showMoreClassName || ''}`}
                        onClick={this.onClickShowMore}
                    >
                        Show more
                    </button>
                )}
            </>
        )
    }

    private onClickShowMore = () => this.props.onShowMore()
}

/**
 * Fields that belong in FilteredConnectionProps and that don't depend on the type parameters. These are the fields
 * that are most likely to be needed by callers, and it's simpler for them if they are in a parameter-less type.
 */
interface FilteredConnectionDisplayProps extends ConnectionDisplayProps {
    history: H.History
    location: H.Location

    /** CSS class name for the root element. */
    className?: string

    /** Whether to display it more compactly. */
    compact?: boolean

    /**
     * An observable that upon emission causes the connection to refresh the data (by calling queryConnection).
     *
     * In most cases, it's simpler to use updateOnChange.
     */
    updates?: Observable<void>

    /**
     * Refresh the data when this value changes. It is typically constructed as a key from the query args.
     */
    updateOnChange?: string

    /** The number of items to fetch, by default. */
    defaultFirst?: number

    /** Hides the filter input field. */
    hideSearch?: boolean

    /** Autofocuses the filter input field. */
    autoFocus?: boolean

    /** Whether we will update the URL query string to reflect the filter and pagination state or not. */
    shouldUpdateURLQuery?: boolean

    /**
     * Filters to display next to the filter input field.
     *
     * Filters are mutually exclusive.
     */
    filters?: FilteredConnectionFilter[]

    /** Called when a filter is selected and on initial render. */
    onFilterSelect?: (filterID: string | undefined) => void
}

/**
 * Props for the FilteredConnection component.
 *
 * @template C The GraphQL connection type, such as GQL.IRepositoryConnection.
 * @template N The node type of the GraphQL connection, such as GQL.IRepository (if C is GQL.IRepositoryConnection)
 * @template NP Props passed to `nodeComponent` in addition to `{ node: N }`
 */
interface FilteredConnectionProps<C extends Connection<N>, N, NP = {}>
    extends ConnectionPropsCommon<N, NP>,
        FilteredConnectionDisplayProps {
    /** Called to fetch the connection data to populate this component. */
    queryConnection: (args: FilteredConnectionQueryArgs) => Observable<C>

    /** Called when the queryConnection Observable emits. */
    onUpdate?: (value: C | ErrorLike | undefined) => void
}

/**
 * The arguments for the Props.queryConnection function.
 */
export interface FilteredConnectionQueryArgs {
    first?: number
    after?: string
    query?: string
}

/**
 * A filter to display next to the filter input field.
 */
export interface FilteredConnectionFilter {
    /** The UI label for the filter. */
    label: string

    /**
     * The URL string for this filter (conventionally the label, lowercased and without spaces and punctuation).
     */
    id: string

    /** An optional tooltip to display for this filter. */
    tooltip?: string

    /** Additional query args to pass to the queryConnection function when this filter is enabled. */
    args: { [name: string]: string | number | boolean }
}

interface FilteredConnectionState<C extends Connection<N>, N> extends ConnectionStateCommon {
    /** The active filter's ID (FilteredConnectionFilter.id), if any. */
    activeFilter: FilteredConnectionFilter | undefined

    /** The fetched connection data or an error (if an error occurred). */
    connectionOrError?: C | ErrorLike
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
     * If set, indicates whether there is a next page and provides an optional cursor to use on a
     * subsequent request. Not all GraphQL XyzConnection types return pageInfo (if not, then they
     * generally all do return totalCount).
     */
    pageInfo?: { endCursor?: string | null; hasNextPage: boolean }

    /**
     * If set, this error is displayed. Even when there is an error, the results are still displayed.
     */
    error?: string | null
}

/** The URL query parameter where the search query for FilteredConnection is stored. */
const QUERY_KEY = 'query'

/**
 * Displays a collection of items with filtering and pagination. It is called
 * "connection" because it is intended for use with GraphQL, which calls it that
 * (see http://graphql.org/learn/pagination/).
 *
 * @template N The node type of the GraphQL connection, such as `GQL.IRepository` (if `C` is `GQL.IRepositoryConnection`)
 * @template NP Props passed to `nodeComponent` in addition to `{ node: N }`
 * @template C The GraphQL connection type, such as `GQL.IRepositoryConnection`.
 */
export class FilteredConnection<N, NP = {}, C extends Connection<N> = Connection<N>> extends React.PureComponent<
    FilteredConnectionProps<C, N, NP>,
    FilteredConnectionState<C, N>
> {
    public static defaultProps: Partial<FilteredConnectionProps<any, any>> = {
        defaultFirst: 20,
        shouldUpdateURLQuery: true,
    }

    private queryInputChanges = new Subject<string>()
    private activeFilterChanges = new Subject<FilteredConnectionFilter>()
    private showMoreClicks = new Subject<void>()
    private componentUpdates = new Subject<FilteredConnectionProps<C, N, NP>>()
    private subscriptions = new Subscription()

    private filterRef: HTMLInputElement | null = null

    constructor(props: FilteredConnectionProps<C, N, NP>) {
        super(props)

        const q = new URLSearchParams(this.props.location.search)
        this.state = {
            loading: true,
            query: (!this.props.hideSearch && q.get(QUERY_KEY)) || '',
            activeFilter: getFilterFromURL(q, this.props.filters),
            first: parseQueryInt(q, 'first') || this.props.defaultFirst!,

            // Note: Do not set after from the URL, as this doesn't track the number
            // of results on the previous page. This makes the count look broken when
            // coming to a page in the middle of a set of results.
            //
            // For example:
            //   (1) come to page with first = 20
            //   (2) set first and after cursor in URL
            //   (3) reload page; will skip 20 results but will display (first 20 of X)
            //
            // Instead, we use the `previousPagesCount` parameter as a marker for the
            // number of results we have seen on previous pages, tracked in the
            // implementation of the `componentDidMount` method below.
        }
    }

    public componentDidMount(): void {
        const activeFilterChanges = this.activeFilterChanges.pipe(
            startWith(this.state.activeFilter),
            distinctUntilChanged()
        )
        const queryChanges = this.queryInputChanges.pipe(
            distinctUntilChanged(),
            tap(query => !this.props.hideSearch && this.setState({ query })),
            debounceTime(200),
            startWith(this.state.query)
        )
        const refreshRequests = new Subject<void>()

        this.subscriptions.add(
            activeFilterChanges
                .pipe(
                    tap(filter => {
                        if (this.props.onFilterSelect) {
                            this.props.onFilterSelect(filter ? filter.id : undefined)
                        }
                    })
                )
                .subscribe()
        )

        this.subscriptions.add(
            // Use this.activeFilterChanges not activeFilterChanges so that it doesn't trigger on the initial mount
            // (it doesn't need to).
            this.activeFilterChanges.subscribe(filter => this.setState({ activeFilter: filter }))
        )

        const q = new URLSearchParams(this.props.location.search)

        // The number of additional results we want to load on the first query. This is used instead of encoding
        // the `after` parameter so that we load the entire visible result set again when a user has loaded multiple
        // pages of results and copied the URL.
        const previousPagesCount = parseQueryInt(q, 'previousPagesCount') || 0

        // Whether or not a query has been made. We track this so that we only change the `first` parameter of
        // the very first request of this component (see `previousPagesCount` defined above).
        let queried = false

        // The endCursor of the previous request. This is defined here and injected into the filter state
        // as React's setState is asynchronous and we can't guarantee the state is updated before the observer
        // of a query change, active filter change, or refresh request is handled.
        let after: string | undefined

        // The list of nodes that should be prepended to the next page of results (when `appendResults` is true).
        // This is cleared when the query or active filters change, and updated on each successful request of
        // a new page of results.
        let previousPage: N[] = []

        // Called on query/active filter change to reset pagination state
        const resetCursor = (): void => {
            after = undefined
            previousPage = []
        }

        // Track the last query and filter we used. We only want to show the loader if these change,
        // not when a refresh is requested for the same query/filter (or else there would be jitter).
        let lastQuery: string | undefined
        let lastFilter: FilteredConnectionFilter | undefined
        this.subscriptions.add(
            combineLatest([
                queryChanges.pipe(tap(resetCursor)),
                activeFilterChanges.pipe(tap(resetCursor)),
                refreshRequests.pipe(startWith<void>(undefined)),
            ])
                .pipe(
                    tap(([query, filter]) => {
                        if (this.props.shouldUpdateURLQuery) {
                            this.props.history.replace({
                                search: this.urlQuery({ query, filter, previousPagesCount: previousPage.length }),
                                hash: this.props.location.hash,
                            })
                        }
                    }),
                    switchMap(([query, filter]) => {
                        type PartialStateUpdate = Pick<
                            FilteredConnectionState<C, N>,
                            'connectionOrError' | 'loading' | 'connectionQuery'
                        >

                        // Prepend the nodes of the current page to the nodes of the new
                        // connection, if appendResults is set, the previous page is not
                        // stale, and neither the connection nor the current page are an
                        // error-like value.
                        const prependPreviousPage = (c: C | ErrorLike | undefined): typeof c => {
                            if (c && !isErrorLike(c) && this.props.appendResults) {
                                c.nodes = previousPage.concat(c.nodes)
                                previousPage = c.nodes
                            }

                            return c
                        }

                        const result = this.props
                            .queryConnection({
                                // If this is our first query, load `previousPagesCount` additional results
                                first: this.state.first + ((!queried && previousPagesCount) || 0),
                                after,
                                query,
                                ...(filter ? filter.args : {}),
                            })
                            .pipe(
                                tap(() => {
                                    // Do not modify the first parameter for subsequent requests
                                    queried = true
                                }),
                                catchError(error => [asError(error)]),
                                map(
                                    (c): PartialStateUpdate => ({
                                        connectionOrError: prependPreviousPage(c),
                                        connectionQuery: query,
                                        loading: false,
                                    })
                                ),
                                publishReplay<PartialStateUpdate>(),
                                refCount()
                            )

                        const showLoading = query !== lastQuery || filter !== lastFilter
                        lastQuery = query
                        lastFilter = filter
                        return showLoading
                            ? merge(
                                  result,
                                  of({ connectionOrError: undefined, loading: true }).pipe(
                                      delay(250),
                                      takeUntil(result)
                                  )
                              )
                            : result
                    }),
                    tap(({ connectionOrError }) => {
                        if (this.props.onUpdate) {
                            this.props.onUpdate(connectionOrError)
                        }

                        // If our response has a page info object, try to pull the end cursor out of
                        // it so we can pass it to the subsequent request. We'll either update or
                        // unset the after cursor here so it is not used twice.
                        if (connectionOrError && !isErrorLike(connectionOrError) && connectionOrError.pageInfo) {
                            after = connectionOrError.pageInfo.endCursor || undefined
                        } else {
                            resetCursor()
                        }
                    })
                )
                .subscribe(stateUpdate => this.setState(stateUpdate), err => console.error(err))
        )

        this.subscriptions.add(
            this.showMoreClicks
                .pipe(
                    map(() =>
                        // If after is set, then it's the endCursor from the previous request.
                        // Use this and do not change the first (page size) parameter. Otherwise,
                        // we'll fallback to our legacy 'request-more' paging technique and not
                        // supply a cursor to the subsequent request.
                        ({ first: after ? this.state.first : this.state.first * 2 })
                    )
                )
                .subscribe(({ first }) => this.setState({ first, loading: true }, () => refreshRequests.next()))
        )

        if (this.props.updates) {
            this.subscriptions.add(
                this.props.updates.subscribe(c => {
                    this.setState({ loading: true }, () => refreshRequests.next())
                })
            )
        }

        this.subscriptions.add(
            this.componentUpdates
                .pipe(
                    distinctUntilChanged((a, b) => a.updateOnChange === b.updateOnChange),
                    filter(({ updateOnChange }) => updateOnChange !== undefined)
                )
                .subscribe(() => {
                    this.setState({ loading: true, connectionOrError: undefined }, () => refreshRequests.next())
                })
        )

        // Reload collection when the query callback changes.
        this.subscriptions.add(
            this.componentUpdates
                .pipe(
                    map(({ queryConnection }) => queryConnection),
                    distinctUntilChanged(),
                    skip(1), // prevent from triggering on initial mount
                    tap(() => this.focusFilter())
                )
                .subscribe(() =>
                    this.setState({ loading: true, connectionOrError: undefined }, () => refreshRequests.next())
                )
        )
        this.componentUpdates.next(this.props)
    }

    private urlQuery(arg: {
        first?: number
        query?: string
        filter?: FilteredConnectionFilter
        previousPagesCount?: number
    }): string {
        if (!arg.first) {
            arg.first = this.state.first
        }
        if (!arg.query) {
            arg.query = this.state.query
        }
        if (!arg.filter) {
            arg.filter = this.state.activeFilter
        }
        const q = new URLSearchParams()
        if (arg.query) {
            q.set(QUERY_KEY, arg.query)
        }

        if (arg.first !== this.props.defaultFirst) {
            q.set('first', String(arg.first))
        }
        if (arg.filter && this.props.filters && arg.filter !== this.props.filters[0]) {
            q.set('filter', arg.filter.id)
        }
        if (arg.previousPagesCount !== 0) {
            q.set('previousPagesCount', String(arg.previousPagesCount))
        }
        return q.toString()
    }

    public componentDidUpdate(): void {
        this.componentUpdates.next(this.props)
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        const errors: string[] = []
        if (isErrorLike(this.state.connectionOrError)) {
            errors.push(...uniq(this.state.connectionOrError.message.split('\n')))
        }
        if (
            this.state.connectionOrError &&
            !isErrorLike(this.state.connectionOrError) &&
            this.state.connectionOrError.error
        ) {
            errors.push(this.state.connectionOrError.error)
        }

        const compactnessClass = `filtered-connection--${this.props.compact ? 'compact' : 'noncompact'}`
        return (
            <div
                className={`filtered-connection e2e-filtered-connection ${compactnessClass} ${this.props.className ||
                    ''}`}
            >
                {(!this.props.hideSearch || this.props.filters) && (
                    <Form className="filtered-connection__form" onSubmit={this.onSubmit}>
                        {!this.props.hideSearch && (
                            <input
                                className="form-control filtered-connection__filter"
                                type="search"
                                placeholder={`Search ${this.props.pluralNoun}...`}
                                name="query"
                                value={this.state.query}
                                onChange={this.onChange}
                                autoFocus={this.props.autoFocus}
                                autoComplete="off"
                                autoCorrect="off"
                                autoCapitalize="off"
                                ref={this.setFilterRef}
                                spellCheck={false}
                            />
                        )}
                        {this.props.filters && this.state.activeFilter && (
                            <FilteredConnectionFilterControl
                                filters={this.props.filters}
                                onDidSelectFilter={this.onDidSelectFilter}
                                value={this.state.activeFilter.id}
                            />
                        )}
                    </Form>
                )}
                {errors.length > 0 && (
                    <div className="alert alert-danger filtered-connection__error">
                        {errors.map((m, i) => (
                            <React.Fragment key={i}>
                                {upperFirst(m)}
                                <br />
                            </React.Fragment>
                        ))}
                    </div>
                )}
                {this.state.connectionOrError && !isErrorLike(this.state.connectionOrError) && (
                    <ConnectionNodes
                        connection={this.state.connectionOrError}
                        loading={this.state.loading}
                        connectionQuery={this.state.connectionQuery}
                        first={this.state.first}
                        query={this.state.query}
                        noun={this.props.noun}
                        pluralNoun={this.props.pluralNoun}
                        listComponent={this.props.listComponent}
                        listClassName={this.props.listClassName}
                        headComponent={this.props.headComponent}
                        footComponent={this.props.footComponent}
                        showMoreClassName={this.props.showMoreClassName}
                        nodeComponent={this.props.nodeComponent}
                        nodeComponentProps={this.props.nodeComponentProps}
                        noShowMore={this.props.noShowMore}
                        noSummaryIfAllNodesVisible={this.props.noSummaryIfAllNodesVisible}
                        onShowMore={this.onClickShowMore}
                        location={this.props.location}
                        emptyElement={this.props.emptyElement}
                    />
                )}
                {this.state.loading && (
                    <span className="filtered-connection__loader e2e-filtered-connection__loader">
                        <LoadingSpinner className="icon-inline" />
                    </span>
                )}
            </div>
        )
    }

    private setFilterRef = (e: HTMLInputElement | null) => {
        this.filterRef = e
        if (e && this.props.autoFocus) {
            // TODO(sqs): The 30 msec delay is needed, or else the input is not
            // reliably focused. Find out why.
            setTimeout(() => e.focus(), 30)
        }
    }

    private focusFilter = () => {
        if (this.filterRef) {
            this.filterRef.focus()
        }
    }

    private onSubmit: React.FormEventHandler<HTMLFormElement> = e => {
        // Do nothing. The <input onChange> handler will pick up any changes shortly.
        e.preventDefault()
    }

    private onChange: React.ChangeEventHandler<HTMLInputElement> = e => {
        this.queryInputChanges.next(e.currentTarget.value)
    }

    private onDidSelectFilter = (filter: FilteredConnectionFilter) => this.activeFilterChanges.next(filter)

    private onClickShowMore = () => {
        this.showMoreClicks.next()
    }
}

function parseQueryInt(q: URLSearchParams, name: string): number | null {
    const s = q.get(name)
    if (s === null) {
        return null
    }
    const n = parseInt(s, 10)
    if (n > 0) {
        return n
    }
    return null
}

function getFilterFromURL(
    q: URLSearchParams,
    filters: FilteredConnectionFilter[] | undefined
): FilteredConnectionFilter | undefined {
    if (filters === undefined || filters.length === 0) {
        return undefined
    }
    const id = q.get('filter')
    if (id !== null) {
        const filter = filters.find(f => f.id === id)
        if (filter) {
            return filter
        }
    }
    return filters[0] // default
}
