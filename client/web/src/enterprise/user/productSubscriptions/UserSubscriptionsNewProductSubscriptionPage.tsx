import * as H from 'history'
import AlertCircleIcon from 'mdi-react/AlertCircleIcon'
import React, { useEffect, useCallback } from 'react'
import { RouteComponentProps } from 'react-router'
import { Link } from 'react-router-dom'
import { Observable } from 'rxjs'
import { catchError, map, mapTo, startWith, switchMap, tap } from 'rxjs/operators'
import { gql } from '../../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../../shared/src/graphql/schema'
import { asError, createAggregateError } from '../../../../../shared/src/util/errors'
import { mutateGraphQL } from '../../../backend/graphql'
import { HeroPage } from '../../../components/HeroPage'
import { PageTitle } from '../../../components/PageTitle'
import { eventLogger } from '../../../tracking/eventLogger'
import { BackToAllSubscriptionsLink } from './BackToAllSubscriptionsLink'
import { ProductSubscriptionForm } from './ProductSubscriptionForm'
import { ThemeProps } from '../../../../../shared/src/theme'
import { useEventObservable } from '../../../../../shared/src/util/useObservable'
import { UserAreaUserFields } from '../../../graphql-operations'

interface Props extends RouteComponentProps<{}>, ThemeProps {
    /**
     * The user who will own the new subscription when created, or null when there is no
     * authenticated user and this page is accessed at /subscriptions/new.
     */
    user: UserAreaUserFields | null
    history: H.History
}

const LOADING = 'loading' as const

/**
 * Displays a form and payment flow to purchase a product subscription.
 *
 * This page is visible to both authenticated and unauthenticated users. Unauthenticated users may
 * view it at /subscriptions/new and are allowed to price out a subscription, but they must sign in
 * to buy the subscription.
 */
export const UserSubscriptionsNewProductSubscriptionPage: React.FunctionComponent<Props> = ({
    user,
    location,
    history,
    isLightTheme,
}) => {
    useEffect(() => eventLogger.logViewEvent('UserSubscriptionsNewProductSubscription'), [])

    /**
     * The result of creating the paid product subscription: undefined when complete or not started yet,
     * loading, or an error.
     */
    const [nextCreation, creation] = useEventObservable(
        useCallback(
            (creations: Observable<GQL.ICreatePaidProductSubscriptionOnDotcomMutationArguments>) =>
                creations.pipe(
                    switchMap(args =>
                        createPaidProductSubscription(args).pipe(
                            tap(({ productSubscription }) => {
                                // Redirect to new subscription upon success.
                                history.push(productSubscription.url)
                            }),
                            mapTo(undefined),
                            catchError(error => [asError(error)]),
                            startWith(LOADING)
                        )
                    )
                ),
            [history]
        )
    )

    if (user && !user.viewerCanAdminister) {
        return <HeroPage icon={AlertCircleIcon} title="Not authorized" />
    }

    return (
        <div className="user-subscriptions-new-product-subscription-page">
            <PageTitle title="New product subscription" />
            {user && <BackToAllSubscriptionsLink user={user} />}
            <h2>New subscription</h2>
            <ProductSubscriptionForm
                accountID={user ? user.id : null}
                subscriptionID={null}
                initialValue={parseProductSubscriptionInputFromLocation(location) || undefined}
                isLightTheme={isLightTheme}
                onSubmit={nextCreation}
                submissionState={creation}
                primaryButtonText="Buy subscription"
                primaryButtonTextNoPaymentRequired="Create subscription"
                afterPrimaryButton={
                    <small className="form-text text-muted">
                        Your license key will be available immediately.
                        <br />
                        <br />
                        <Link to="/terms" target="_blank">
                            Terms of Service
                        </Link>{' '}
                        |{' '}
                        <Link to="/privacy" target="_blank">
                            Privacy Policy
                        </Link>
                    </small>
                }
                history={history}
            />
        </div>
    )
}

/**
 * Parses product subscription input from the URL hash.
 *
 * Inverse of {@link productSubscriptionInputForLocationHash}.
 */
function parseProductSubscriptionInputFromLocation(location: H.Location): GQL.IProductSubscriptionInput | null {
    if (location.hash) {
        const parameters = new URLSearchParams(location.hash.slice('#'.length))
        const billingPlanID = parameters.get('plan')
        const userCount = parseInt(parameters.get('userCount') || '0', 10)
        if (billingPlanID && userCount) {
            return { billingPlanID, userCount }
        }
    }
    return null
}

/**
 * Generates the URL hash value to represent the product subscription input.
 *
 * Inverse of {@link parseProductSubscriptionInputFromLocation}.
 */
export function productSubscriptionInputForLocationHash(value: GQL.IProductSubscriptionInput | null): string {
    if (value === null) {
        return ''
    }
    const parameters = new URLSearchParams()
    parameters.set('plan', value.billingPlanID)
    parameters.set('userCount', value.userCount.toString())
    return '#' + parameters.toString()
}

function createPaidProductSubscription(
    args: GQL.ICreatePaidProductSubscriptionOnDotcomMutationArguments
): Observable<GQL.ICreatePaidProductSubscriptionResult> {
    return mutateGraphQL(
        gql`
            mutation CreatePaidProductSubscription(
                $accountID: ID!
                $productSubscription: ProductSubscriptionInput!
                $paymentToken: String
            ) {
                dotcom {
                    createPaidProductSubscription(
                        accountID: $accountID
                        productSubscription: $productSubscription
                        paymentToken: $paymentToken
                    ) {
                        productSubscription {
                            id
                            name
                            url
                        }
                    }
                }
            }
        `,
        args
    ).pipe(
        map(({ data, errors }) => {
            if (!data || !data.dotcom || !data.dotcom.createPaidProductSubscription || (errors && errors.length > 0)) {
                throw createAggregateError(errors)
            }
            return data.dotcom.createPaidProductSubscription
        })
    )
}
