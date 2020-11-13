import { Observable } from 'rxjs'
import { fromFetch } from 'rxjs/fetch'
import { Omit } from 'utility-types'
import { createAggregateError } from '../util/errors'
import { checkOk } from '../backend/fetch'
import * as GQL from './schema'
import { hasProperty } from '../util/types'
import { isEqual, isObject } from 'lodash'

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
    errors: GraphQLError[]
}

/**
 * A spec-compliant member of the GraphQL `errors` array.
 */
export interface GraphQLError {
    /**
     * Every error must contain an entry with the key message with a string description of the error intended for
     * the developer as a guide to understand and correct the error.
     */
    message: string

    /**
     * If an error can be associated to a particular point in the requested GraphQL document, it should contain an
     * entry with the key locations with a list of locations, where each location is a map with the keys line and
     * column, both positive numbers starting from 1 which describe the beginning of an associated syntax element.
     */
    locations?: GQL.IGraphQLResponseErrorLocation[]

    /**
     * If an error can be associated to a particular field in the GraphQL result, it must contain an entry with the
     * key path that details the path of the response field which experienced the error. This allows clients to
     * identify whether a null result is intentional or caused by a runtime error.
     *
     * This field should be a list of path segments starting at the root of the response and ending with the field
     * associated with the error. Path segments that represent fields should be strings, and path segments that
     * represent list indices should be 0‐indexed integers. If the error happens in an aliased field, the path to
     * the error should use the aliased name, since it represents a path in the response, not in the query.
     */
    path?: (string | number)[]

    /**
     * GraphQL services may provide an additional entry to errors with key extensions. This entry, if set, must
     * have a map as its value. This entry is reserved for implementors to add additional information to errors
     * however they see fit, and there are no additional restrictions on its contents.
     */
    extensions?: Record<string, unknown>
}

export const hasExactGraphQLPath = (error: unknown, path: (string | number)[]): boolean =>
    isObject(error) && hasProperty('path')(error) && isEqual(error.path, path)

/**
 * Returns true if the given error has the given GraphQL path or a path below it.
 */
export const isUnderGraphQLPath = (error: unknown, path: (string | number)[]): boolean =>
    isObject(error) &&
    hasProperty('path')(error) &&
    Array.isArray(error.path) &&
    isEqual(error.path?.slice(0, path.length), path)
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

export interface InvalidGraphQLResponseError extends Error {
    queryName: string
}
export const createInvalidGraphQLQueryResponseError = (queryName: string): InvalidGraphQLResponseError =>
    Object.assign(new Error(`Invalid GraphQL response: query ${queryName}`), {
        queryName,
    })
export const createInvalidGraphQLMutationResponseError = (queryName: string): InvalidGraphQLResponseError =>
    Object.assign(new Error(`Invalid GraphQL response: mutation ${queryName}`), {
        queryName,
    })

export interface GraphQLRequestOptions extends Omit<RequestInit, 'method' | 'body'> {
    baseUrl?: string
}

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
    const apiURL = `/.api/graphql${nameMatch ? '?' + nameMatch[1] : ''}`
    return fromFetch(baseUrl ? new URL(apiURL, baseUrl).href : apiURL, {
        ...options,
        method: 'POST',
        body: JSON.stringify({ query: request, variables }),
        selector: response => checkOk(response).json(),
    })
}
