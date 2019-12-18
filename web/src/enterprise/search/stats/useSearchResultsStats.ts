import { useEffect, useState } from 'react'
import { Observable } from 'rxjs'
import { map, startWith } from 'rxjs/operators'
import { dataOrThrowErrors, gql } from '../../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../../shared/src/graphql/schema'
import { asError, ErrorLike } from '../../../../../shared/src/util/errors'
import { queryGraphQL } from '../../../backend/graphql'

const LOADING = 'loading' as const

type Result = typeof LOADING | GQL.ISearchResultsStats | ErrorLike

const queryStatisticsForSearchResults = (query: string): Observable<Result> =>
    queryGraphQL(
        gql`
            query StatisticsForSearchResults($query: String!) {
                search(query: $query) {
                    results {
                        elapsedMilliseconds
                    }
                    stats {
                        languages {
                            name
                            totalLines
                        }
                    }
                }
            }
        `,
        { query }
    ).pipe(
        map(dataOrThrowErrors),
        map(data => {
            if (!data.search) {
                throw new Error('no search results')
            }
            return data.search.stats
        })
    )

/**
 * A React hook that observes the statistics for a search query.
 */
export const useSearchResultsStats = (query: string): Result => {
    const [result, setResult] = useState<Result>(LOADING)
    useEffect(() => {
        const subscription = queryStatisticsForSearchResults(query)
            .pipe(startWith(LOADING))
            .subscribe(setResult, err => setResult(asError(err)))
        return () => subscription.unsubscribe()
    }, [query])

    return result
}
