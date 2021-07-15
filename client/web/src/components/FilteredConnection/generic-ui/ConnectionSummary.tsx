import * as React from 'react'

import { pluralize } from '@sourcegraph/shared/src/util/strings'

import { ConnectionNodesState, ConnectionProps } from '../ConnectionNodes'
import { Connection } from '../ConnectionType'

interface ConnectionNodesSummaryProps<C extends Connection<N>, N, NP = {}, HP = {}>
    extends Pick<
        ConnectionProps<N, NP, HP> & ConnectionNodesState,
        | 'noSummaryIfAllNodesVisible'
        | 'totalCountSummaryComponent'
        | 'noun'
        | 'pluralNoun'
        | 'connectionQuery'
        | 'emptyElement'
    > {
    /** The fetched connection data or an error (if an error occurred). */
    connection: C

    hasNextPage: boolean

    totalCount: number | null
}

export const ConnectionSummary = <C extends Connection<N>, N, NP = {}, HP = {}>({
    noSummaryIfAllNodesVisible,
    connection,
    hasNextPage,
    totalCount,
    totalCountSummaryComponent: TotalCountSummaryComponent,
    noun,
    pluralNoun,
    connectionQuery,
    emptyElement,
}: ConnectionNodesSummaryProps<C, N, NP, HP>): JSX.Element | null => {
    const shouldShowSummary = !noSummaryIfAllNodesVisible || connection.nodes.length === 0 || hasNextPage

    if (!shouldShowSummary) {
        return null
    }

    if (totalCount !== null && totalCount > 0 && TotalCountSummaryComponent) {
        return <TotalCountSummaryComponent totalCount={totalCount} />
    }

    if (totalCount !== null && totalCount > 0 && !TotalCountSummaryComponent) {
        return (
            <p className="filtered-connection__summary" data-testid="summary">
                <small>
                    <span>
                        {totalCount} {pluralize(noun, totalCount, pluralNoun)}{' '}
                        {connectionQuery ? (
                            <span>
                                {' '}
                                matching <strong>{connectionQuery}</strong>
                            </span>
                        ) : (
                            'total'
                        )}
                    </span>{' '}
                    {connection.nodes.length < totalCount && `(showing first ${connection.nodes.length})`}
                </small>
            </p>
        )
    }

    if (connection.pageInfo?.hasNextPage) {
        // No total count to show, but it will show a 'Show more' button.
        return null
    }

    return (
        emptyElement || (
            <p className="filtered-connection__summary" data-testid="summary">
                <small>
                    No {pluralNoun}{' '}
                    {connectionQuery && (
                        <span>
                            matching <strong>{connectionQuery}</strong>
                        </span>
                    )}
                </small>
            </p>
        )
    )
}
