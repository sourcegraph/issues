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
    /** The customer to show a billing link for. */
    customer: Pick<GQL.IUser, 'id' | 'urlForSiteAdminBilling'>

    /** Called when the customer is updated. */
    onDidUpdate: () => void
}

const LOADING = 'loading' as const

/**
 * SiteAdminCustomerBillingLink shows a link to the customer on the billing system associated with a user, if any.
 * It also supports setting or unsetting the association with the billing system.
 */
export const SiteAdminCustomerBillingLink: React.FunctionComponent<Props> = ({ customer, onDidUpdate }) => {
    /** The result of updating this customer: undefined for done or not started, loading, or an error. */
    const [nextUpdate, update] = useEventObservable(
        useCallback(
            (updates: Observable<{ user: GQL.ID; billingCustomerID: string | null }>) =>
                updates.pipe(
                    switchMap(({ user, billingCustomerID }) =>
                        setCustomerBilling({ user, billingCustomerID }).pipe(
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
        const billingCustomerID = window.prompt('Enter new Stripe billing customer ID:', 'cus_ABCDEF12345678')

        // Ignore if the user pressed "Cancel" or did not enter any value.
        if (!billingCustomerID) {
            return
        }

        nextUpdate({ user: customer.id, billingCustomerID })
    }, [customer.id, nextUpdate])
    const onUnlinkBillingClick = useCallback(() => nextUpdate({ user: customer.id, billingCustomerID: null }), [
        customer.id,
        nextUpdate,
    ])

    const customerHasLinkedBilling = customer.urlForSiteAdminBilling !== null
    return (
        <div className="site-admin-customer-billing-link">
            <div className="d-flex align-items-center">
                {customer.urlForSiteAdminBilling && (
                    <a href={customer.urlForSiteAdminBilling} className="mr-2 d-flex align-items-center">
                        View customer account <ExternalLinkIcon className="icon-inline ml-1" />
                    </a>
                )}
                {isErrorLike(update) && (
                    <ErrorIcon className="icon-inline text-danger mr-2" data-tooltip={update.message} />
                )}
                <button
                    type="button"
                    className="btn btn-secondary"
                    onClick={customerHasLinkedBilling ? onUnlinkBillingClick : onLinkBillingClick}
                    disabled={update === LOADING}
                >
                    {customerHasLinkedBilling ? 'Unlink' : 'Link billing customer'}
                </button>
            </div>
        </div>
    )
}

function setCustomerBilling(args: GQL.ISetUserBillingOnDotcomMutationArguments): Observable<void> {
    return mutateGraphQL(
        gql`
            mutation SetCustomerBilling($user: ID!, $billingCustomerID: String) {
                dotcom {
                    setUserBilling(user: $user, billingCustomerID: $billingCustomerID) {
                        alwaysNil
                    }
                }
            }
        `,
        args
    ).pipe(
        map(({ data, errors }) => {
            if (!data || !data.dotcom || !data.dotcom.setUserBilling || (errors && errors.length > 0)) {
                throw createAggregateError(errors)
            }
        })
    )
}
