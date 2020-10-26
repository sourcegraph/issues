import ErrorIcon from 'mdi-react/ErrorIcon'
import ExternalLinkIcon from 'mdi-react/ExternalLinkIcon'
import React, { useCallback } from 'react'
import { Observable } from 'rxjs'
import { catchError, map, mapTo, startWith, switchMap, tap } from 'rxjs/operators'
import { gql } from '../../../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../../../shared/src/graphql/schema'
import { asError, createAggregateError, isErrorLike } from '../../../../../../shared/src/util/errors'
import { mutateGraphQL } from '../../../../backend/graphql'
import { useEventObservable } from '../../../../../../shared/src/util/useObservable'

interface Props {
    /** The product subscription to show a billing link for. */
    productSubscription: Pick<GQL.IProductSubscription, 'id' | 'urlForSiteAdminBilling'>

    /** Called when the product subscription is updated. */
    onDidUpdate: () => void
}

const LOADING = 'loading' as const

/**
 * SiteAdminProductSubscriptionBillingLink shows a link to the product subscription on the billing system, if there
 * is an associated billing record. It also supports setting or unsetting the association with the billing system.
 */
export const SiteAdminProductSubscriptionBillingLink: React.FunctionComponent<Props> = ({
    productSubscription,
    onDidUpdate,
}) => {
    /** The result of updating this subscription: undefined for done or not started, loading, or an error. */
    const [nextUpdate, update] = useEventObservable(
        useCallback(
            (updates: Observable<{ id: GQL.ID; billingSubscriptionID: string | null }>) =>
                updates.pipe(
                    switchMap(({ id, billingSubscriptionID }) =>
                        setProductSubscriptionBilling({ id, billingSubscriptionID }).pipe(
                            mapTo(undefined),
                            tap(() => onDidUpdate()),
                            catchError(error => [asError(error)]),
                            startWith(LOADING)
                        )
                    )
                ),
            [onDidUpdate]
        )
    )
    const onLinkBillingClick = useCallback(() => {
        const billingSubscriptionID = window.prompt('Enter new Stripe billing subscription ID:', 'sub_ABCDEF12345678')

        // Ignore if the user pressed "Cancel" or did not enter any value.
        if (!billingSubscriptionID) {
            return
        }

        nextUpdate({ id: productSubscription.id, billingSubscriptionID })
    }, [nextUpdate, productSubscription.id])
    const onUnlinkBillingClick = useCallback(
        () => nextUpdate({ id: productSubscription.id, billingSubscriptionID: null }),
        [nextUpdate, productSubscription.id]
    )

    const productSubscriptionHasLinkedBilling = productSubscription.urlForSiteAdminBilling !== null
    return (
        <div className="site-admin-product-subscription-billing-link">
            <div className="d-flex align-items-center">
                {productSubscription.urlForSiteAdminBilling && (
                    <a href={productSubscription.urlForSiteAdminBilling} className="mr-2 d-flex align-items-center">
                        View billing subscription <ExternalLinkIcon className="icon-inline ml-1" />
                    </a>
                )}
                {isErrorLike(update) && (
                    <ErrorIcon className="icon-inline text-danger mr-2" data-tooltip={update.message} />
                )}
                <button
                    type="button"
                    className="btn btn-secondary btn-sm"
                    onClick={productSubscriptionHasLinkedBilling ? onUnlinkBillingClick : onLinkBillingClick}
                    disabled={update === LOADING}
                >
                    {productSubscriptionHasLinkedBilling ? 'Unlink' : 'Link billing subscription'}
                </button>
            </div>
        </div>
    )
}

function setProductSubscriptionBilling(
    args: GQL.ISetProductSubscriptionBillingOnDotcomMutationArguments
): Observable<void> {
    return mutateGraphQL(
        gql`
            mutation SetProductSubscriptionBilling($id: ID!, $billingSubscriptionID: String) {
                dotcom {
                    setProductSubscriptionBilling(id: $id, billingSubscriptionID: $billingSubscriptionID) {
                        alwaysNil
                    }
                }
            }
        `,
        args
    ).pipe(
        map(({ data, errors }) => {
            if (!data || !data.dotcom || !data.dotcom.setProductSubscriptionBilling || (errors && errors.length > 0)) {
                throw createAggregateError(errors)
            }
        })
    )
}
