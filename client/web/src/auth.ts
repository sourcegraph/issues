import { Observable, ReplaySubject } from 'rxjs'
import { catchError, map, mergeMap, tap } from 'rxjs/operators'
import { dataOrThrowErrors, gql } from '../../shared/src/graphql/graphql'
import { requestGraphQL } from './backend/graphql'
import { CurrentAuthStateResult } from './graphql-operations'

/**
 * Always represents the latest state of the currently authenticated user.
 *
 * Note that authenticatedUser is not designed to survive across changes in the currently authenticated user. Sign
 * in, sign out, and account changes all require a full-page reload in the browser to take effect.
 */
export const authenticatedUser = new ReplaySubject<AuthenticatedUser | null>(1)

export type AuthenticatedUser = NonNullable<CurrentAuthStateResult['currentUser']>

/**
 * Fetches the current user, orgs, and config state from the remote. Emits no items, completes when done.
 */
export function refreshAuthenticatedUser(): Observable<never> {
    return requestGraphQL<CurrentAuthStateResult>(gql`
        query CurrentAuthState {
            currentUser {
                __typename
                id
                databaseID
                username
                avatarURL
                email
                displayName
                siteAdmin
                tags
                url
                settingsURL
                organizations {
                    nodes {
                        id
                        name
                        displayName
                        url
                        settingsURL
                    }
                }
                session {
                    canSignOut
                }
                viewerCanAdminister
                tags
            }
        }
    `).pipe(
        map(dataOrThrowErrors),
        tap(data => authenticatedUser.next(data.currentUser)),
        catchError(() => {
            authenticatedUser.next(null)
            return []
        }),
        mergeMap(() => [])
    )
}

/**
 * Whether auth is required to perform any action.
 *
 * If an HTTP request might be triggered by an unauthenticated user on a server that is not Sourcegraph.com
 * the caller must first check authRequired. If authRequired is true, then the component must
 * not initiate the HTTP request. This prevents the browser's devtools console from showing HTTP 401
 * errors, which mislead the user into thinking there is a problem (and make debugging any actual
 * issue much harder).
 */
export const authRequired = authenticatedUser.pipe(map(user => user === null && !window.context?.sourcegraphDotComMode))

// Populate authenticatedUser.
if (window.context?.isAuthenticatedUser) {
    refreshAuthenticatedUser()
        .toPromise()
        .then(
            () => undefined,
            error => console.error(error)
        )
} else {
    authenticatedUser.next(null)
}
