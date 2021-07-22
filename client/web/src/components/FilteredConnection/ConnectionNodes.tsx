import classNames from 'classnames'
import * as H from 'history'
import * as React from 'react'

import { useRedesignToggle } from '@sourcegraph/shared/src/util/useRedesignToggle'

import { ConnectionNodesSummary } from './ConnectionNodesSummary'
import { Connection } from './ConnectionType'
import { hasID, hasNextPage } from './utils'

/**
 * Props for the FilteredConnection component's result nodes and associated summary/pagination controls.
 *
 * @template N The node type of the GraphQL connection, such as GQL.IRepository (if the connection is GQL.IRepositoryConnection)
 * @template NP Props passed to `nodeComponent` in addition to `{ node: N }`
 * @template HP Props passed to `headComponent` in addition to `{ nodes: N[]; totalCount?: number | null }`.
 */
export interface ConnectionProps<N, NP = {}, HP = {}> extends ConnectionNodesDisplayProps {
    /** Header row to appear above all nodes. */
    headComponent?: React.ComponentType<{ nodes: N[]; totalCount?: number | null } & HP>

    /** Props to pass to each headComponent in addition to `{ nodes: N[]; totalCount?: number | null }`. */
    headComponentProps?: HP

    /** Footer row to appear below all nodes. */
    footComponent?: React.ComponentType<{ nodes: N[] }>

    /** The component type to use to display each node. */
    nodeComponent: React.ComponentType<{ node: N } & NP>

    /** Props to pass to each nodeComponent in addition to `{ node: N }`. */
    nodeComponentProps?: NP

    /** An element rendered as a sibling of the filters. */
    additionalFilterElement?: React.ReactElement
}

/** State related to the ConnectionNodes component. */
export interface ConnectionNodesState {
    query: string
    first: number

    connectionQuery?: string

    /**
     * Whether the connection is loading. It is not equivalent to connection === undefined because we preserve the
     * old data for ~250msec while loading to reduce jitter.
     */
    loading: boolean
}

/**
 * Fields that belong in ConnectionNodesProps and that don't depend on the type parameters. These are the fields
 * that are most likely to be needed by callers, and it's simpler for them if they are in a parameter-less type.
 */
export interface ConnectionNodesDisplayProps {
    /** list HTML element type. Default is <ul>. */
    listComponent?: 'ul' | 'table' | 'div'

    /** CSS class name for the list element (<ul>, <table>, or <div>). */
    listClassName?: string

    /** CSS class name for the summary container element. */
    summaryClassName?: string

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

    /** The component displayed when all nodes have been fetched. */
    totalCountSummaryComponent?: React.ComponentType<{ totalCount: number }>
}

interface ConnectionNodesProps<C extends Connection<N>, N, NP = {}, HP = {}>
    extends ConnectionProps<N, NP, HP>,
        ConnectionNodesState {
    /** The fetched connection data or an error (if an error occurred). */
    connection: C

    location: H.Location

    onShowMore: () => void
}

export const getTotalCount = <N,>({ totalCount, nodes, pageInfo }: Connection<N>, first: number): number | null => {
    if (typeof totalCount === 'number') {
        return totalCount
    }

    if (
        // TODO(sqs): this line below is wrong because `first` might've just been changed and
        // `nodes` is still the data fetched from before `first` was changed.
        // this causes the UI to incorrectly show "N items total" even when the count is indeterminate right
        // after the user clicks "Show more" but before the new data is loaded.
        nodes.length < first ||
        (nodes.length === first && pageInfo && typeof pageInfo.hasNextPage === 'boolean' && !pageInfo.hasNextPage)
    ) {
        return nodes.length
    }

    return null
}

export const ConnectionNodes = <C extends Connection<N>, N, NP = {}, HP = {}>({
    nodeComponent: NodeComponent,
    nodeComponentProps,
    listComponent: ListComponent = 'ul',
    listClassName,
    summaryClassName,
    headComponent: HeadComponent,
    headComponentProps,
    footComponent: FootComponent,
    emptyElement,
    totalCountSummaryComponent,
    connection,
    first,
    noSummaryIfAllNodesVisible,
    noun,
    pluralNoun,
    connectionQuery,
    loading,
    noShowMore,
    onShowMore,
    showMoreClassName,
}: ConnectionNodesProps<C, N, NP, HP>): JSX.Element => {
    const nextPage = hasNextPage(connection)
    const [isRedesignEnabled] = useRedesignToggle()

    const totalCount = getTotalCount(connection, first)
    const summary = (
        <ConnectionNodesSummary
            noSummaryIfAllNodesVisible={noSummaryIfAllNodesVisible}
            totalCount={totalCount}
            totalCountSummaryComponent={totalCountSummaryComponent}
            noun={noun}
            pluralNoun={pluralNoun}
            connectionQuery={connectionQuery}
            emptyElement={emptyElement}
            connection={connection}
            hasNextPage={nextPage}
        />
    )

    const nodes = connection.nodes.map((node, index) => (
        <NodeComponent key={hasID(node) ? node.id : index} node={node} {...nodeComponentProps!} />
    ))

    const summaryContainerClassName = classNames(summaryClassName, 'filtered-connection__summary-container')

    return (
        <>
            <div className={summaryContainerClassName}>{connectionQuery && summary}</div>
            {connection.nodes.length > 0 && (
                <ListComponent className={classNames('filtered-connection__nodes', listClassName)} data-testid="nodes">
                    {HeadComponent && (
                        <HeadComponent
                            nodes={connection.nodes}
                            totalCount={connection.totalCount}
                            {...headComponentProps!}
                        />
                    )}
                    {ListComponent === 'table' ? <tbody>{nodes}</tbody> : nodes}
                    {FootComponent && <FootComponent nodes={connection.nodes} />}
                </ListComponent>
            )}
            {!loading && (
                <div className={summaryContainerClassName}>
                    {!connectionQuery && summary}
                    {!noShowMore && nextPage && (
                        <button
                            type="button"
                            className={classNames(
                                'btn btn-sm filtered-connection__show-more',
                                isRedesignEnabled ? 'btn-link' : 'btn-secondary',
                                showMoreClassName
                            )}
                            onClick={onShowMore}
                        >
                            Show more
                        </button>
                    )}
                </div>
            )}
        </>
    )
}
