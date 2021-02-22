import { useMemo, useCallback, useState } from 'react'
import { Subscription } from 'rxjs'
import { isErrorGraphQLResult } from '../../../shared/src/graphql/graphql'
import { createAggregateError } from '../../../shared/src/util/errors'
import { requestGraphQL } from '../backend/graphql'

interface UnsubmittedMutationResult {
    loading: false
}

interface InFlightMutationResult {
    loading: true
}

interface ResolvedMutationResult<TData = undefined> {
    loading: false
    data: TData
}

interface ErredMutationResult {
    loading: false
    error: Error
}

export type MutationResult<TData> =
    | UnsubmittedMutationResult
    | InFlightMutationResult
    | (ResolvedMutationResult<TData> | ErredMutationResult)

/**
 * Trigger a GraphQL mutation and render based on the returned response
 *
 * @param mutation GraphQL mutation query string
 * @param options Additional configuration
 * @returns Callback to trigger mutation and response data from GraphQL
 */
export function useMutation<TData = unknown, TVariables = unknown>(
    mutation: string,
    options?: {
        /** Throw any errors returned in the response. Use if you want to defer error handling to an ErrorBoundary */
        throwGraphQLErrors: boolean
    }
): [(variables: TVariables) => void, MutationResult<TData>] {
    const subscriptions = useMemo(() => new Subscription(), [])

    const [result, setResult] = useState<MutationResult<TData>>({ loading: false })

    const submit = useCallback(
        (variables: TVariables) => {
            setResult({ loading: true })

            subscriptions.add(
                requestGraphQL<TData, TVariables>(mutation, variables).subscribe(response => {
                    if (isErrorGraphQLResult(response)) {
                        setResult({
                            loading: false,
                            error: createAggregateError(response.errors),
                        })
                        return
                    }

                    setResult({ loading: false, data: response.data })
                })
            )
        },
        [subscriptions, mutation]
    )

    if ('error' in result && options?.throwGraphQLErrors) {
        throw result.error
    }

    return [submit, result]
}
