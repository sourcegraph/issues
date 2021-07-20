import classNames from 'classnames'
import React, { useCallback, useEffect, useMemo, useState } from 'react'
import { Observable } from 'rxjs'

import { Link } from '@sourcegraph/shared/src/components/Link'
import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { buildSearchURLQuery } from '@sourcegraph/shared/src/util/url'
import { useObservable } from '@sourcegraph/shared/src/util/useObservable'

import { AuthenticatedUser } from '../../auth'
import { SyntaxHighlightedSearchQuery } from '../../components/SyntaxHighlightedSearchQuery'
import { Timestamp } from '../../components/time/Timestamp'
import { SearchPatternType } from '../../graphql-operations'
import { EventLogResult } from '../backend'

import { LoadingPanelView } from './LoadingPanelView'
import { PanelContainer } from './PanelContainer'
import { ShowMoreButton } from './ShowMoreButton'

interface RecentSearch {
    count: number
    searchText: string
    timestamp: string
    url: string
}

interface Props extends TelemetryProps {
    className?: string
    authenticatedUser: AuthenticatedUser | null
    fetchRecentSearches: (userId: string, first: number) => Observable<EventLogResult | null>

    /** Function that returns current time (for stability in visual tests). */
    now?: () => Date
}

export const RecentSearchesPanel: React.FunctionComponent<Props> = ({
    className,
    authenticatedUser,
    fetchRecentSearches,
    now,
    telemetryService,
}) => {
    const pageSize = 20

    const [itemsToLoad, setItemsToLoad] = useState(pageSize)
    const recentSearches = useObservable(
        useMemo(() => fetchRecentSearches(authenticatedUser?.id || '', itemsToLoad), [
            authenticatedUser?.id,
            fetchRecentSearches,
            itemsToLoad,
        ])
    )
    const [processedResults, setProcessedResults] = useState<RecentSearch[] | null>(null)

    // Only update processed results when results are valid to prevent
    // flashing loading screen when "Show more" button is clicked
    useEffect(() => {
        if (recentSearches) {
            setProcessedResults(processRecentSearches(recentSearches))
        }
    }, [recentSearches])

    useEffect(() => {
        // Only log the first load (when items to load is equal to the page size)
        if (processedResults && itemsToLoad === pageSize) {
            telemetryService.log('RecentSearchesPanelLoaded', { empty: processedResults.length === 0 })
        }
    }, [processedResults, telemetryService, itemsToLoad])

    const logSearchClicked = useCallback(() => telemetryService.log('RecentSearchesPanelSearchClicked'), [
        telemetryService,
    ])

    const loadingDisplay = <LoadingPanelView text="Loading recent searches" />
    const emptyDisplay = (
        <div className="panel-container__empty-container text-muted">
            <small className="mb-2">
                Your recent searches will be displayed here. Here are a few searches to get you started:
            </small>

            <ul className="recent-searches-panel__examples-list">
                <li className="recent-searches-panel__examples-list-item">
                    <small>
                        <Link
                            to={
                                '/search?' +
                                buildSearchURLQuery(
                                    'lang:c if(:[eval_match]) { :[statement_match] }',
                                    SearchPatternType.structural,
                                    false
                                )
                            }
                            className="text-monospace"
                        >
                            <SyntaxHighlightedSearchQuery query="lang:c if(:[eval_match]) { :[statement_match] }" />
                        </Link>
                    </small>
                </li>
                <li className="recent-searches-panel__examples-list-item">
                    <small>
                        <Link
                            to={
                                '/search?' +
                                buildSearchURLQuery(
                                    'lang:java type:diff after:"1 week ago"',
                                    SearchPatternType.literal,
                                    false
                                )
                            }
                            className="text-monospace"
                        >
                            <SyntaxHighlightedSearchQuery query='lang:java type:diff after:"1 week ago"' />
                        </Link>
                    </small>
                </li>
                <li className="recent-searches-panel__examples-list-item">
                    <small>
                        <Link
                            to={'/search?' + buildSearchURLQuery('lang:java', SearchPatternType.literal, false)}
                            className="text-monospace"
                        >
                            <SyntaxHighlightedSearchQuery query="lang:java" />
                        </Link>
                    </small>
                </li>
            </ul>
        </div>
    )

    function loadMoreItems(): void {
        setItemsToLoad(current => current + pageSize)
        telemetryService.log('RecentSearchesPanelShowMoreClicked')
    }

    const contentDisplay = (
        <>
            <table className="recent-searches-panel__results-table mt-2">
                <thead>
                    <tr className="recent-searches-panel__results-table-row">
                        <th>
                            <small>Search</small>
                        </th>
                        <th>
                            <small>Date</small>
                        </th>
                    </tr>
                </thead>
                <tbody className="recent-searches-panel__results-table-body">
                    {processedResults?.map((recentSearch, index) => (
                        <tr key={index} className="recent-searches-panel__results-table-row">
                            <td>
                                <small>
                                    <Link to={recentSearch.url} onClick={logSearchClicked}>
                                        <SyntaxHighlightedSearchQuery query={recentSearch.searchText} />
                                    </Link>
                                </small>
                            </td>
                            <td className="recent-searches-panel__results-table-date-col">
                                <Timestamp noAbout={true} date={recentSearch.timestamp} now={now} strict={true} />
                            </td>
                        </tr>
                    ))}
                </tbody>
            </table>
            {recentSearches?.pageInfo.hasNextPage && (
                <ShowMoreButton onClick={loadMoreItems} className="test-recent-searches-panel-show-more" />
            )}
        </>
    )

    return (
        <PanelContainer
            className={classNames(className, 'recent-searches-panel')}
            title="Recent searches"
            state={processedResults ? (processedResults.length > 0 ? 'populated' : 'empty') : 'loading'}
            loadingContent={loadingDisplay}
            populatedContent={contentDisplay}
            emptyContent={emptyDisplay}
        />
    )
}

function processRecentSearches(eventLogResult?: EventLogResult): RecentSearch[] | null {
    if (!eventLogResult) {
        return null
    }

    const recentSearches: RecentSearch[] = []

    for (const node of eventLogResult.nodes) {
        if (node.argument) {
            const parsedArguments = JSON.parse(node.argument)
            const searchText: string | undefined = parsedArguments?.code_search?.query_data?.combined

            if (searchText) {
                if (recentSearches.length > 0 && recentSearches[recentSearches.length - 1].searchText === searchText) {
                    recentSearches[recentSearches.length - 1].count += 1
                } else {
                    const parsedUrl = new URL(node.url)
                    recentSearches.push({
                        count: 1,
                        url: parsedUrl.pathname + parsedUrl.search, // Strip domain from URL so clicking on it doesn't reload page
                        searchText,
                        timestamp: node.timestamp,
                    })
                }
            }
        }
    }

    return recentSearches
}
