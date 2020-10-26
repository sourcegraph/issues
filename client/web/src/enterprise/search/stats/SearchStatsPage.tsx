import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import ChartLineIcon from 'mdi-react/ChartLineIcon'
import React, { useCallback, useState, useMemo } from 'react'
import * as H from 'history'
import { Form } from '../../../../../branded/src/components/Form'
import { useObservable } from '../../../../../shared/src/util/useObservable'
import { querySearchResultsStats } from './backend'
import { SearchStatsLanguages } from './SearchStatsLanguages'
import { catchError } from 'rxjs/operators'
import { asError, isErrorLike, ErrorLike } from '../../../../../shared/src/util/errors'
import { of } from 'rxjs'

interface Props {
    location: H.Location
    history: H.History

    /** Mockable in tests. */
    _querySearchResultsStats?: typeof querySearchResultsStats
}

/**
 * Shows statistics about the results for a search query.
 */
export const SearchStatsPage: React.FunctionComponent<Props> = ({
    location,
    history,
    _querySearchResultsStats = querySearchResultsStats,
}) => {
    const query = new URLSearchParams(location.search).get('q') || ''
    const [uncommittedQuery, setUncommittedQuery] = useState(query)
    const onUncommittedQueryChange = useCallback<React.ChangeEventHandler<HTMLInputElement>>(event => {
        setUncommittedQuery(event.currentTarget.value)
    }, [])
    const onSubmit = useCallback<React.FormEventHandler<HTMLFormElement>>(
        event => {
            event.preventDefault()
            history.push({ ...location, search: new URLSearchParams({ q: uncommittedQuery }).toString() })
        },
        [history, location, uncommittedQuery]
    )

    const DEFAULT_COUNT = 1000
    const queryWithCount = query.includes('count:') ? query : `${query} count:${DEFAULT_COUNT}`

    // TODO(sqs): reuse the user's current patternType
    const stats = useObservable(
        useMemo(
            () => _querySearchResultsStats(queryWithCount).pipe(catchError(error => of<ErrorLike>(asError(error)))),
            [queryWithCount, _querySearchResultsStats]
        )
    )

    return (
        <div className="search-stats-page container mt-4">
            <header className="d-flex align-items-center justify-content-between mb-3">
                <h2 className="d-flex align-items-center mb-0">
                    <ChartLineIcon className="icon-inline mr-2" /> Code statistics{' '}
                    <small className="badge badge-secondary text-uppercase ml-2">Experimental</small>
                </h2>
            </header>
            <Form onSubmit={onSubmit} className="form">
                <div className="form-group d-flex align-items-stretch">
                    <input
                        id="stats-page__query"
                        className="form-control flex-1 test-stats-query"
                        type="search"
                        placeholder="Enter a Sourcegraph search query"
                        value={uncommittedQuery}
                        onChange={onUncommittedQueryChange}
                        autoCapitalize="off"
                        spellCheck={false}
                        autoCorrect="off"
                        autoComplete="off"
                    />
                    {uncommittedQuery !== query && (
                        <button type="submit" className="btn btn-primary ml-2 test-stats-query-update">
                            Update
                        </button>
                    )}
                </div>
            </Form>
            <hr className="my-3" />
            {stats === undefined ? (
                <LoadingSpinner className="icon-inline" />
            ) : isErrorLike(stats) ? (
                <div className="alert alert-danger">{stats.message}</div>
            ) : stats.limitHit ? (
                <div className="alert alert-warning">
                    Limit hit. Add <code>count:{DEFAULT_COUNT * 5}</code> (or an even larger number) to your query to
                    retry with a higher limit.
                </div>
            ) : (
                <SearchStatsLanguages query={query} stats={stats} />
            )}
        </div>
    )
}
