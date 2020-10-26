import React from 'react'
import renderer from 'react-test-renderer'
import { SiteAdminGenerateProductLicenseForSubscriptionForm } from './SiteAdminGenerateProductLicenseForSubscriptionForm'
import { createMemoryHistory } from 'history'

describe('SiteAdminGenerateProductLicenseForSubscriptionForm', () => {
    test('renders', () => {
        expect(
            renderer
                .create(
                    <SiteAdminGenerateProductLicenseForSubscriptionForm
                        subscriptionID="s"
                        onGenerate={() => undefined}
                        history={createMemoryHistory()}
                    />
                )
                .toJSON()
        ).toMatchSnapshot()
    })
})
