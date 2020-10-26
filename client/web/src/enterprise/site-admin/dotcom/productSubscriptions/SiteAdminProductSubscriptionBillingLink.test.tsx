import React from 'react'
import renderer from 'react-test-renderer'
import { SiteAdminProductSubscriptionBillingLink } from './SiteAdminProductSubscriptionBillingLink'

jest.mock('mdi-react/ExternalLinkIcon', () => 'ExternalLinkIcon')

describe('SiteAdminProductSubscriptionBillingLink', () => {
    test('linked billing', () => {
        expect(
            renderer
                .create(
                    <SiteAdminProductSubscriptionBillingLink
                        productSubscription={{ id: 'u', urlForSiteAdminBilling: 'https://example.com' }}
                        onDidUpdate={() => undefined}
                    />
                )
                .toJSON()
        ).toMatchSnapshot()
    })

    test('no linked billing', () => {
        expect(
            renderer
                .create(
                    <SiteAdminProductSubscriptionBillingLink
                        productSubscription={{ id: 'u', urlForSiteAdminBilling: null }}
                        onDidUpdate={() => undefined}
                    />
                )
                .toJSON()
        ).toMatchSnapshot()
    })
})
