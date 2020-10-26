import React from 'react'
import * as GQL from '../../../../../shared/src/graphql/schema'
import { formatUserCount } from '../../productSubscription/helpers'

/**
 * Displays a text label with the product name (e.g., "Sourcegraph Enterprise") and user count for the
 * subscription.
 */
export const ProductSubscriptionLabel: React.FunctionComponent<{
    productSubscription: {
        invoiceItem?:
            | ({
                  plan: Pick<GQL.IProductPlan, 'name' | 'nameWithBrand'>
              } & Pick<GQL.IProductSubscriptionInvoiceItem, 'userCount'>)
            | null
    } & Pick<GQL.IProductSubscription, 'activeLicense'>

    planField?: 'name' | 'nameWithBrand'

    className?: string
}> = ({ productSubscription, planField, className = '' }) => (
    <span className={className}>
        {productSubscription.invoiceItem ? (
            <>
                {productSubscription.invoiceItem.plan[planField || 'nameWithBrand']} (
                {formatUserCount(productSubscription.invoiceItem.userCount)})
            </>
        ) : productSubscription.activeLicense?.info ? (
            <>
                {productSubscription.activeLicense.info.productNameWithBrand} (
                {formatUserCount(productSubscription.activeLicense.info.userCount)})
            </>
        ) : (
            <span className="text-muted font-italic">No plan selected</span>
        )}
    </span>
)
