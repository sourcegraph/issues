import { Observable, ReplaySubject } from 'rxjs'
import { filter, mergeMap, take, tap } from 'rxjs/operators'
import { SiteFlags } from '.'
import { gql } from '../../../shared/src/graphql/graphql'
import { createAggregateError } from '../../../shared/src/util/errors'
import { authRequired } from '../auth'
import { queryGraphQL } from '../backend/graphql'

/**
 * The latest state of the site flags.
 */
export const siteFlags = new ReplaySubject<SiteFlags>(1)

/**
 * refreshSiteFlags refreshes the site flags. The result is available from
 * the siteFlags const.
 */
export function refreshSiteFlags(): Observable<never> {
    return authRequired.pipe(
        take(1),
        filter(authRequired => !authRequired),
        mergeMap(() =>
            queryGraphQL(gql`
                query SiteFlags {
                    site {
                        needsRepositoryConfiguration
                        freeUsersExceeded
                        alerts {
                            type
                            message
                            isDismissibleWithKey
                        }
                        authProviders {
                            nodes {
                                serviceType
                                serviceID
                                clientID
                                displayName
                                isBuiltin
                                authenticationURL
                            }
                        }
                        disableBuiltInSearches
                        sendsEmailVerificationEmails
                        updateCheck {
                            pending
                            checkedAt
                            errorMessage
                            updateVersionAvailable
                        }
                        productSubscription {
                            license {
                                expiresAt
                            }
                            noLicenseWarningUserCount
                        }
                        productVersion
                    }
                }
            `)
        ),
        tap(({ data, errors }) => {
            if (!data || !data.site) {
                throw createAggregateError(errors)
            }
            siteFlags.next(data.site)
        }),
        mergeMap(() => [])
    )
}

refreshSiteFlags()
    .toPromise()
    .then(
        () => undefined,
        error => console.error(error)
    )
