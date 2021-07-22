import { GraphQLError } from 'graphql'
import type { Location } from 'history'

import { Scalars } from '@sourcegraph/shared/src/graphql-operations'
import { GraphQLResult } from '@sourcegraph/shared/src/graphql/graphql'
import { hasProperty } from '@sourcegraph/shared/src/util/types'

import { Connection } from './ConnectionType'
import { QUERY_KEY } from './constants'
import type { FilteredConnectionFilter, FilteredConnectionFilterValue } from './FilterControl'

/** Checks if the passed value satisfies the GraphQL Node interface */
export const hasID = (value: unknown): value is { id: Scalars['ID'] } =>
    typeof value === 'object' && value !== null && hasProperty('id')(value) && typeof value.id === 'string'

export const getFilterFromURL = (
    searchParameters: URLSearchParams,
    filters: FilteredConnectionFilter[] | undefined
): Map<string, FilteredConnectionFilterValue> => {
    const values: Map<string, FilteredConnectionFilterValue> = new Map<string, FilteredConnectionFilterValue>()

    if (filters === undefined || filters.length === 0) {
        return values
    }
    for (const filter of filters) {
        const urlValue = searchParameters.get(filter.id)
        if (urlValue !== null) {
            const value = filter.values.find(value => value.value === urlValue)
            if (value !== undefined) {
                values.set(filter.id, value)
                continue
            }
        }
        // couldn't find a value, add default
        values.set(filter.id, filter.values[0])
    }
    return values
}

export const parseQueryInt = (searchParameters: URLSearchParams, name: string): number | null => {
    const valueString = searchParameters.get(name)
    if (valueString === null) {
        return null
    }
    const valueNumber = parseInt(valueString, 10)
    if (valueNumber > 0) {
        return valueNumber
    }
    return null
}

/**
 * Determine if a connection has a next page.
 * Provides fallback logic to support queries where `hasNextPage` is undefined.
 */
export const hasNextPage = (connection: Connection<unknown>): boolean =>
    connection.pageInfo
        ? connection.pageInfo.hasNextPage
        : typeof connection.totalCount === 'number' && connection.nodes.length < connection.totalCount

export interface GetUrlQueryParameters {
    first: {
        actual: number
        default: number
    }
    query?: string
    values?: Map<string, FilteredConnectionFilterValue>
    filters?: FilteredConnectionFilter[]
    visibleResultCount?: number
    search: Location['search']
}

/**
 * Determines the URL search parameters for a connection.
 */
export const getUrlQuery = ({
    first,
    query,
    values,
    visibleResultCount,
    filters,
    search,
}: GetUrlQueryParameters): string => {
    const searchParameters = new URLSearchParams(search)

    if (query) {
        searchParameters.set(QUERY_KEY, query)
    }

    if (first.actual !== first.default) {
        searchParameters.set('first', String(first.actual))
    }

    if (values && filters) {
        for (const filter of filters) {
            const value = values.get(filter.id)
            if (value === undefined) {
                continue
            }
            if (value !== filter.values[0]) {
                searchParameters.set(filter.id, value.value)
            } else {
                searchParameters.delete(filter.id)
            }
        }
    }

    if (visibleResultCount && visibleResultCount !== 0 && visibleResultCount !== first.actual) {
        searchParameters.set('visible', String(visibleResultCount))
    }

    return searchParameters.toString()
}

interface AsGraphQLResultParameters<TResult> {
    data?: TResult
    errors: readonly GraphQLError[]
}

/**
 * Map non-conforming GraphQL responses to a GraphQLResult.
 */
export const asGraphQLResult = <T>({ data, errors }: AsGraphQLResultParameters<T>): GraphQLResult<T> => {
    if (!data) {
        return { data: undefined, errors }
    }
    return {
        data,
        errors: undefined,
    }
}
