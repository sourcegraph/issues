import React from 'react'
import * as GQL from '../../../../../shared/src/graphql/schema'
import renderer, { act } from 'react-test-renderer'
import { UserSubscriptionsProductSubscriptionPage } from './UserSubscriptionsProductSubscriptionPage'
import { of } from 'rxjs'
import { MemoryRouter } from 'react-router'
import { createMemoryHistory } from 'history'

jest.mock('./BackToAllSubscriptionsLink', () => ({
    BackToAllSubscriptionsLink: 'BackToAllSubscriptionsLink',
}))

jest.mock('./UserProductSubscriptionStatus', () => ({
    UserProductSubscriptionStatus: 'UserProductSubscriptionStatus',
}))

jest.mock('./ProductSubscriptionBilling', () => ({
    ProductSubscriptionBilling: 'ProductSubscriptionBilling',
}))

jest.mock('./ProductSubscriptionHistory', () => ({
    ProductSubscriptionHistory: 'ProductSubscriptionHistory',
}))

jest.mock('../../../tracking/eventLogger', () => ({
    eventLogger: { logViewEvent: () => undefined },
}))

describe('UserSubscriptionsProductSubscriptionPage', () => {
    test('renders', () => {
        const component = renderer.create(
            <MemoryRouter>
                <UserSubscriptionsProductSubscriptionPage
                    user={{ settingsURL: '/u' }}
                    match={{ isExact: true, params: { subscriptionUUID: 's' }, path: '/p', url: '/p' }}
                    _queryProductSubscription={() =>
                        of<GQL.IProductSubscription>({
                            __typename: 'ProductSubscription',
                        } as GQL.IProductSubscription)
                    }
                    history={createMemoryHistory()}
                />
            </MemoryRouter>
        )
        act(() => undefined)
        expect(component.toJSON()).toMatchSnapshot()
    })
})
