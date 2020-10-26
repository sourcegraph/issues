import React from 'react'
import * as GQL from '../../../../../shared/src/graphql/schema'
import renderer, { act } from 'react-test-renderer'
import { NewProductSubscriptionPaymentSection } from './NewProductSubscriptionPaymentSection'
import { of } from 'rxjs'

jest.mock('./ProductSubscriptionBeforeAfterInvoiceItem', () => ({
    ProductSubscriptionBeforeAfterInvoiceItem: 'ProductSubscriptionBeforeAfterInvoiceItem',
}))

describe('NewProductSubscriptionPaymentSection', () => {
    test('new subscription', () => {
        const component = renderer.create(
            <NewProductSubscriptionPaymentSection
                accountID={null}
                subscriptionID={null}
                productSubscription={{ billingPlanID: 'p', userCount: 123 }}
                onValidityChange={() => undefined}
                _queryPreviewProductSubscriptionInvoice={() =>
                    of<GQL.IProductSubscriptionPreviewInvoice>({
                        __typename: 'ProductSubscriptionPreviewInvoice',
                        beforeInvoiceItem: null,
                        afterInvoiceItem: {
                            __typename: 'ProductSubscriptionInvoiceItem',
                            expiresAt: '2020-01-01',
                            userCount: 123,
                            plan: {} as GQL.IProductPlan,
                        },
                        prorationDate: null,
                        isDowngradeRequiringManualIntervention: false,
                        price: 12345,
                    })
                }
            />
        )
        act(() => undefined)
        expect(component.toJSON()).toMatchSnapshot()
    })

    test('valid change to existing subscription', () => {
        const component = renderer.create(
            <NewProductSubscriptionPaymentSection
                accountID="a"
                subscriptionID="s"
                productSubscription={{ billingPlanID: 'p', userCount: 123 }}
                onValidityChange={() => undefined}
                _queryPreviewProductSubscriptionInvoice={() =>
                    of<GQL.IProductSubscriptionPreviewInvoice>({
                        __typename: 'ProductSubscriptionPreviewInvoice',
                        beforeInvoiceItem: {
                            __typename: 'ProductSubscriptionInvoiceItem',
                            expiresAt: '2020-01-01',
                            userCount: 123,
                            plan: {} as GQL.IProductPlan,
                        },
                        afterInvoiceItem: {
                            __typename: 'ProductSubscriptionInvoiceItem',
                            expiresAt: '2021-01-01',
                            userCount: 456,
                            plan: {} as GQL.IProductPlan,
                        },
                        prorationDate: null,
                        isDowngradeRequiringManualIntervention: false,
                        price: 23456,
                    })
                }
            />
        )
        act(() => undefined)
        expect(component.toJSON()).toMatchSnapshot()
    })

    test('no change to existing subscription', () => {
        const component = renderer.create(
            <NewProductSubscriptionPaymentSection
                accountID="a"
                subscriptionID="s"
                productSubscription={{ billingPlanID: 'p', userCount: 123 }}
                onValidityChange={() => undefined}
                _queryPreviewProductSubscriptionInvoice={() =>
                    of<GQL.IProductSubscriptionPreviewInvoice>({
                        __typename: 'ProductSubscriptionPreviewInvoice',
                        beforeInvoiceItem: {
                            __typename: 'ProductSubscriptionInvoiceItem',
                            expiresAt: '2020-01-01',
                            userCount: 123,
                            plan: {} as GQL.IProductPlan,
                        },
                        afterInvoiceItem: {
                            __typename: 'ProductSubscriptionInvoiceItem',
                            expiresAt: '2020-01-01',
                            userCount: 123,
                            plan: {} as GQL.IProductPlan,
                        },
                        prorationDate: null,
                        isDowngradeRequiringManualIntervention: false,
                        price: 0,
                    })
                }
            />
        )
        act(() => undefined)
        expect(component.toJSON()).toMatchSnapshot()
    })

    test('downgrade to existing subscription', () => {
        const component = renderer.create(
            <NewProductSubscriptionPaymentSection
                accountID="a"
                subscriptionID="s"
                productSubscription={{ billingPlanID: 'p', userCount: 123 }}
                onValidityChange={() => undefined}
                _queryPreviewProductSubscriptionInvoice={() =>
                    of<GQL.IProductSubscriptionPreviewInvoice>({
                        __typename: 'ProductSubscriptionPreviewInvoice',
                        beforeInvoiceItem: {
                            __typename: 'ProductSubscriptionInvoiceItem',
                            expiresAt: '2020-01-01',
                            userCount: 123,
                            plan: {} as GQL.IProductPlan,
                        },
                        afterInvoiceItem: {
                            __typename: 'ProductSubscriptionInvoiceItem',
                            expiresAt: '2021-01-01',
                            userCount: 456,
                            plan: {} as GQL.IProductPlan,
                        },
                        prorationDate: null,
                        isDowngradeRequiringManualIntervention: true,
                        price: 23456,
                    })
                }
            />
        )
        act(() => undefined)
        expect(component.toJSON()).toMatchSnapshot()
    })
})
