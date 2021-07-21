import {
    gql as apolloGql,
    useQuery as useApolloQuery,
    useMutation as useApolloMutation,
    useLazyQuery as useApolloLazyQuery,
    DocumentNode,
    ApolloClient,
    createHttpLink,
    NormalizedCacheObject,
    OperationVariables,
    QueryHookOptions,
    QueryResult,
    MutationHookOptions,
    MutationTuple,
    QueryTuple,
} from '@apollo/client'
import { GraphQLError } from 'graphql'
import { useMemo } from 'react'
import { Observable } from 'rxjs'
import { fromFetch } from 'rxjs/fetch'
import { Omit } from 'utility-types'

import { checkOk } from '../backend/fetch'
import { createAggregateError } from '../util/errors'

import { cache } from './cache'

/**
 * Use this template string tag for all GraphQL queries.
 */
export const gql = (template: TemplateStringsArray, ...substitutions: any[]): string =>
    String.raw(template, ...substitutions)

export interface SuccessGraphQLResult<T> {
    data: T
    errors: undefined
}
export interface ErrorGraphQLResult {
    data: undefined
    errors: readonly GraphQLError[]
}

export type GraphQLResult<T> = SuccessGraphQLResult<T> | ErrorGraphQLResult

/**
 * Guarantees that the GraphQL query resulted in an error.
 */
export function isErrorGraphQLResult<T>(result: GraphQLResult<T>): result is ErrorGraphQLResult {
    return !!(result as ErrorGraphQLResult).errors && (result as ErrorGraphQLResult).errors.length > 0
}

export function dataOrThrowErrors<T>(result: GraphQLResult<T>): T {
    if (isErrorGraphQLResult(result)) {
        throw createAggregateError(result.errors)
    }
    return result.data
}

export const createInvalidGraphQLQueryResponseError = (queryName: string): GraphQLError =>
    new GraphQLError(`Invalid GraphQL response: query ${queryName}`)

export const createInvalidGraphQLMutationResponseError = (queryName: string): GraphQLError =>
    new GraphQLError(`Invalid GraphQL response: mutation ${queryName}`)

export interface GraphQLRequestOptions extends Omit<RequestInit, 'method' | 'body'> {
    baseUrl?: string
}

const GRAPHQL_URI = '/.api/graphql'

/**
 * This function should not be called directly as it does not
 * add the necessary headers to authorize the GraphQL API call.
 * Use `requestGraphQL()` in `client/web/src/backend/graphql.ts` instead.
 */
export function requestGraphQLCommon<T, V = object>({
    request,
    baseUrl,
    variables,
    ...options
}: GraphQLRequestOptions & {
    request: string
    variables?: V
}): Observable<GraphQLResult<T>> {
    const nameMatch = request.match(/^\s*(?:query|mutation)\s+(\w+)/)
    const apiURL = `${GRAPHQL_URI}${nameMatch ? '?' + nameMatch[1] : ''}`
    return fromFetch(baseUrl ? new URL(apiURL, baseUrl).href : apiURL, {
        ...options,
        method: 'POST',
        body: JSON.stringify({ query: request, variables }),
        selector: response => checkOk(response).json(),
    })
}

export const graphQLClient = ({ headers }: { headers: RequestInit['headers'] }): ApolloClient<NormalizedCacheObject> =>
    new ApolloClient({
        uri: GRAPHQL_URI,
        cache,
        link: createHttpLink({
            uri: ({ operationName }) => `${GRAPHQL_URI}?${operationName}`,
            headers,
        }),
    })

type RequestDocument = string | DocumentNode

/**
 * Returns a `DocumentNode` value to support integrations with GraphQL clients that require this.
 *
 * @param document The GraphQL operation payload
 * @returns The created `DocumentNode`
 */
export const getDocumentNode = (document: RequestDocument): DocumentNode => {
    if (typeof document === 'string') {
        return apolloGql(document)
    }
    return document
}

const useDocumentNode = (document: RequestDocument): DocumentNode =>
    useMemo(() => getDocumentNode(document), [document])

/**
 * Send a query to GraphQL and respond to updates.
 * Wrapper around Apollo `useQuery` that supports `DocumentNode` and `string` types.
 *
 * @param query GraphQL operation payload.
 * @param options Operation variables and request configuration
 * @returns GraphQL response
 */
export function useQuery<TData = any, TVariables = OperationVariables>(
    query: RequestDocument,
    options: QueryHookOptions<TData, TVariables>
): QueryResult<TData, TVariables> {
    const documentNode = useDocumentNode(query)
    return useApolloQuery(documentNode, options)
}

export function useLazyQuery<TData = any, TVariables = OperationVariables>(
    query: RequestDocument,
    options: QueryHookOptions<TData, TVariables>
): QueryTuple<TData, TVariables> {
    const documentNode = useDocumentNode(query)
    return useApolloLazyQuery(documentNode, options)
}

/**
 * Send a mutation to GraphQL and respond to updates.
 * Wrapper around Apollo `useMutation` that supports `DocumentNode` and `string` types.
 *
 * @param mutation GraphQL operation payload.
 * @param options Operation variables and request configuration
 * @returns GraphQL response
 */
export function useMutation<TData = any, TVariables = OperationVariables>(
    mutation: RequestDocument,
    options?: MutationHookOptions<TData, TVariables>
): MutationTuple<TData, TVariables> {
    const documentNode = useDocumentNode(mutation)
    return useApolloMutation(documentNode, options)
}
