import { Observable } from 'rxjs'
import { map } from 'rxjs/operators'
import { gql, dataOrThrowErrors } from '../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../shared/src/graphql/schema'
import { createAggregateError } from '../../../../shared/src/util/errors'
import { mutateGraphQL } from '../../backend/graphql'
import { eventLogger } from '../../tracking/eventLogger'
import { UserEvent, EventSource } from '../../../../shared/src/graphql-operations'

export function updatePassword(args: { oldPassword: string; newPassword: string }): Observable<void> {
    return mutateGraphQL(
        gql`
            mutation updatePassword($oldPassword: String!, $newPassword: String!) {
                updatePassword(oldPassword: $oldPassword, newPassword: $newPassword) {
                    alwaysNil
                }
            }
        `,
        args
    ).pipe(
        map(({ data, errors }) => {
            if (!data || !data.updatePassword) {
                eventLogger.log('UpdatePasswordFailed')
                throw createAggregateError(errors)
            }
            eventLogger.log('PasswordUpdated')
        })
    )
}

/**
 * Set the verification state for a user email address.
 *
 * @param user the user's GraphQL ID
 * @param email the email address to edit
 * @param verified the new verification state for the user email
 */
export function setUserEmailVerified(user: GQL.ID, email: string, verified: boolean): Observable<void> {
    return mutateGraphQL(
        gql`
            mutation SetUserEmailVerified($user: ID!, $email: String!, $verified: Boolean!) {
                setUserEmailVerified(user: $user, email: $email, verified: $verified) {
                    alwaysNil
                }
            }
        `,
        { user, email, verified }
    ).pipe(
        map(({ data, errors }) => {
            if (!data || (errors && errors.length > 0)) {
                throw createAggregateError(errors)
            }
        })
    )
}

/**
 * Log a user action (used to allow site admins on a Sourcegraph instance
 * to see a count of unique users on a daily, weekly, and monthly basis).
 *
 * Not used at all for public/sourcegraph.com usage.
 *
 * @deprecated Use logEvent
 */
export function logUserEvent(event: UserEvent): void {
    mutateGraphQL(
        gql`
            mutation logUserEvent($event: UserEvent!, $userCookieID: String!) {
                logUserEvent(event: $event, userCookieID: $userCookieID) {
                    alwaysNil
                }
            }
        `,
        { event, userCookieID: eventLogger.getAnonUserID() }
    )
        .pipe(
            map(({ data, errors }) => {
                if (!data || (errors && errors.length > 0)) {
                    throw createAggregateError(errors)
                }
                return
            })
        )
        // Event logs are best-effort and non-blocking
        // eslint-disable-next-line rxjs/no-ignored-subscription
        .subscribe()
}

/**
 * Log a raw user action (used to allow site admins on a Sourcegraph instance
 * to see a count of unique users on a daily, weekly, and monthly basis).
 *
 * Not used at all for public/sourcegraph.com usage.
 */
export function logEvent(event: string, eventProperties?: any): void {
    mutateGraphQL(
        gql`
            mutation logEvent(
                $event: String!
                $userCookieID: String!
                $url: String!
                $source: EventSource!
                $argument: String
            ) {
                logEvent(event: $event, userCookieID: $userCookieID, url: $url, source: $source, argument: $argument) {
                    alwaysNil
                }
            }
        `,
        {
            event,
            userCookieID: eventLogger.getAnonUserID(),
            url: window.location.href,
            source: EventSource.WEB,
            argument: eventProperties && JSON.stringify(eventProperties),
        }
    )
        .pipe(map(dataOrThrowErrors))
        // Event logs are best-effort and non-blocking
        // eslint-disable-next-line rxjs/no-ignored-subscription
        .subscribe()
}
