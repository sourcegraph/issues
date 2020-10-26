import React from 'react'
import renderer from 'react-test-renderer'
import { SiteAdminCustomerBillingLink } from './SiteAdminCustomerBillingLink'

jest.mock('mdi-react/ExternalLinkIcon', () => 'ExternalLinkIcon')

describe('SiteAdminCustomerBillingLink', () => {
    test('linked billing account', () => {
        expect(
            renderer
                .create(
                    <SiteAdminCustomerBillingLink
                        customer={{ id: 'u', urlForSiteAdminBilling: 'https://example.com' }}
                        onDidUpdate={() => undefined}
                    />
                )
                .toJSON()
        ).toMatchSnapshot()
    })

    test('no linked billing account', () => {
        expect(
            renderer
                .create(
                    <SiteAdminCustomerBillingLink
                        customer={{ id: 'u', urlForSiteAdminBilling: null }}
                        onDidUpdate={() => undefined}
                    />
                )
                .toJSON()
        ).toMatchSnapshot()
    })
})
