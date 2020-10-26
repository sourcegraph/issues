import React, { useEffect } from 'react'
import { PageTitle } from '../../../components/PageTitle'
import { eventLogger } from '../../../tracking/eventLogger'
import { ProductSubscriptionStatus } from './ProductSubscriptionStatus'
import { RouteComponentProps } from 'react-router'

/**
 * Displays the product subscription information from the license key in site configuration.
 */
export const SiteAdminProductSubscriptionPage: React.FunctionComponent<RouteComponentProps> = props => {
    useEffect(() => eventLogger.logViewEvent('SiteAdminProductSubscription'), [])

    return (
        <div className="site-admin-product-subscription-page">
            <PageTitle title="Sourcegraph product subscription" />
            <ProductSubscriptionStatus {...props} showTrueUpStatus={true} />
        </div>
    )
}
