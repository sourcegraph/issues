import { GraphQLResult, requestGraphQLCommon } from '../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../shared/src/graphql/schema'
import { Observable } from 'rxjs'
import { PlatformContext } from '../../../../shared/src/platform/context'

/**
 * A GraphQL client to be used from regression test scripts.
 */
export interface GraphQLClient {
    /**
     * mimics the `mutateGraphQL` function used by the Sourcegraph backend, but substitutes
     * in CLI-read HTTP headers rather than use the ones in the browser context (which doesn't
     * exist).
     */
    mutateGraphQL(request: string, variables?: {}): Observable<GraphQLResult<GQL.IMutation>>

    /**
     * mimics the `queryGraphQL` function used by the Sourcegraph backend, but substitutes
     * in CLI-read HTTP headers rather than use the ones in the browser context (which doesn't
     * exist).
     */
    queryGraphQL(request: string, variables?: {}): Observable<GraphQLResult<GQL.IQuery>>

    /**
     * Mimics the {@link PlatformContext#requestGraphQL} function that is injected in shared backend functions,
     * but substitutes in CLI-read HTTP headers rather than use the ones in the browser context (which doesn't
     * exist).
     */
    requestGraphQL: PlatformContext['requestGraphQL']

    /**
     * Mimics the {@link eventLogger} property expected in shared backend functions.
     */
    eventLogger: { log: (eventLabel: string, eventProperties?: any) => void }
}

/**
 * Returns a {@link GraphQLClient} for use in regression test scripts.
 *
 * @param sudoUsername should be set if and only if the token is a sudo-level token.
 */
export const createGraphQLClient = ({
    baseUrl,
    token,
    sudoUsername,
}: {
    baseUrl: string
    token: string
    sudoUsername?: string
}): GraphQLClient => {
    const headers = sudoUsername
        ? {
              Authorization: `token-sudo user="${sudoUsername}",token="${token}"`,
          }
        : {
              Authorization: `token ${token}`,
          }
    return {
        mutateGraphQL: (request, variables) =>
            requestGraphQLCommon({
                request,
                variables,
                headers,
                baseUrl,
            }),
        queryGraphQL: (request, variables) =>
            requestGraphQLCommon({
                request,
                variables,
                headers,
                baseUrl,
            }),
        requestGraphQL: ({ request, variables }) =>
            requestGraphQLCommon({
                request,
                variables,
                headers,
                baseUrl,
            }),
        eventLogger: { log: () => undefined },
    }
}
