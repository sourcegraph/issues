import { Observable, of, throwError } from 'rxjs'
import { SuccessGraphQLResult } from '../../../../../shared/src/graphql/graphql'
import { IQuery } from '../../../../../shared/src/graphql/schema'
import { PlatformContext } from '../../../../../shared/src/platform/context'

export interface GraphQLResponseMap {
    [requestName: string]: (
        variables: { [k: string]: any },
        mightContainPrivateInfo?: boolean
    ) => Observable<SuccessGraphQLResult<any>>
}

export const DEFAULT_GRAPHQL_RESPONSES: GraphQLResponseMap = {
    SiteProductVersion: () =>
        // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
        of({
            data: {
                site: {
                    productVersion: 'dev',
                    buildVersion: 'dev',
                    hasCodeIntelligence: true,
                },
            },
            errors: undefined,
        } as SuccessGraphQLResult<IQuery>),
    CurrentUSer: () =>
        // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
        of({
            data: {
                currentUser: {
                    id: 'u1',
                    displayName: 'Alice',
                    username: 'alice',
                    avatarURL: null,
                    url: 'https://example.com/alice',
                    settingsURL: 'https://example.com/alice/settings',
                    emails: [{ email: 'alice@example.com' }],
                    siteAdmin: false,
                },
            },
        } as SuccessGraphQLResult<IQuery>),

    ResolveRev: () =>
        // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
        of({
            data: {
                repository: {
                    mirrorInfo: {
                        cloned: true,
                    },
                    commit: {
                        oid: 'foo',
                    },
                },
            },
            errors: undefined,
        } as SuccessGraphQLResult<IQuery>),
    BlobContent: () =>
        // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
        of({
            data: {
                repository: {
                    commit: {
                        file: {
                            content: 'Hello World',
                        },
                    },
                },
            },
            errors: undefined,
        } as SuccessGraphQLResult<IQuery>),
    ResolveRepo: variables =>
        // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
        of({
            data: {
                repository: {
                    name: variables.rawRepoName,
                },
            },
            errors: undefined,
        } as SuccessGraphQLResult<IQuery>),
}

/**
 * @param responseMap a {@link GraphQLResponseMap} of request names (eg. `ResolveRev`) to response builders.
 *
 * @returns a mock implementation of {@link PlatformContext#requestGraphQL}
 */
export const mockRequestGraphQL = (
    responseMap: GraphQLResponseMap = DEFAULT_GRAPHQL_RESPONSES
): PlatformContext['requestGraphQL'] => <R, V = object>({
    request,
    variables,
    mightContainPrivateInfo,
}: {
    request: string
    variables: V
    mightContainPrivateInfo?: boolean
}) => {
    const nameMatch = request.match(/^\s*(?:query|mutation)\s+(\w+)/)
    const requestName = nameMatch?.[1]
    if (!requestName || !responseMap[requestName]) {
        return throwError(new Error(`No mock for GraphQL request ${String(requestName)}`))
    }
    return responseMap[requestName](variables, mightContainPrivateInfo) as Observable<SuccessGraphQLResult<R>>
}
