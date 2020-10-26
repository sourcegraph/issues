import React from 'react'
import renderer, { act } from 'react-test-renderer'
import { UserProductSubscriptionStatus } from './UserProductSubscriptionStatus'

jest.mock('mdi-react/KeyIcon', () => 'KeyIcon')
jest.mock('mdi-react/InformationIcon', () => 'InformationIcon')
jest.mock('../../../components/CopyableText', () => ({ CopyableText: 'CopyableText' }))

describe('UserProductSubscriptionStatus', () => {
    test('toggle', () => {
        const component = renderer.create(
            <UserProductSubscriptionStatus
                subscriptionName="L-123"
                productNameWithBrand="P"
                userCount={123}
                expiresAt={23456}
                licenseKey="lk"
            />
        )
        expect(component.toJSON()).toMatchSnapshot('license key hidden')

        // Click "Reveal license key" button.
        act(() => component.root.findByType('button').props.onClick())
        expect(component.toJSON()).toMatchSnapshot('license key revealed')
    })
})
