import { Observable } from 'rxjs'
import { GraphQLResult, requestGraphQL as requestGraphQLCommon } from '../../../shared/src/graphql/graphql'

const getHeaders = (): { [header: string]: string } => ({
    ...window.context.xhrHeaders,
    Accept: 'application/json',
    'Content-Type': 'application/json',
    'X-Sourcegraph-Should-Trace': new URLSearchParams(window.location.search).get('trace') || 'false',
})

/**
 * Does a GraphQL request to the Sourcegraph GraphQL API running under `/.api/graphql`
 *
 * @param request The GraphQL request (query or mutation)
 * @param variables A key/value object with variable values
 * @returns Observable That emits the result or errors if the HTTP request failed
 */
export const requestGraphQL = <T>(
    request: string,
    variables?: {}
): Observable<GraphQLResult<T>> =>
    requestGraphQLCommon({
        request,
        variables,
        headers: getHeaders(),
    })

/**
 * Does a GraphQL query to the Sourcegraph GraphQL API running under `/.api/graphql`
 *
 * @param request The GraphQL query
 * @param variables A key/value object with variable values
 * @returns Observable That emits the result or errors if the HTTP request failed
 */
export const queryGraphQL = <T>(request: string, variables?: {}): Observable<GraphQLResult<T>> =>
    requestGraphQLCommon({
        request,
        variables,
        headers: getHeaders(),
    })

/**
 * Does a GraphQL mutation to the Sourcegraph GraphQL API running under `/.api/graphql`
 *
 * @param request The GraphQL mutation
 * @param variables A key/value object with variable values
 * @returns Observable That emits the result or errors if the HTTP request failed
 */
export const mutateGraphQL = <T>(request: string, variables?: {}): Observable<GraphQLResult<T>> =>
    requestGraphQLCommon({
        request,
        variables,
        headers: getHeaders(),
    })
