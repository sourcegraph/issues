import React from 'react'
import renderer from 'react-test-renderer'
import { ProductPlanTiered } from './ProductPlanTiered'

describe('ProductPlanTiered', () => {
    test('volume', () => {
        expect(
            renderer
                .create(
                    <ProductPlanTiered
                        plan={{
                            minQuantity: null,
                            tiersMode: 'volume',
                            planTiers: [
                                { __typename: 'PlanTier', flatAmount: 100, unitAmount: 200, upTo: 300 },
                                { __typename: 'PlanTier', flatAmount: 400, unitAmount: 500, upTo: 600 },
                            ],
                        }}
                    />
                )
                .toJSON()
        ).toMatchInlineSnapshot(`
            Array [
              <div>
                $0.17/user/month
                 
                for 1–300 users
              </div>,
              <div>
                $0.42/user/month
                 
                for 301–600 users
              </div>,
            ]
        `)
    })

    test('volume', () => {
        expect(
            renderer
                .create(
                    <ProductPlanTiered
                        plan={{
                            minQuantity: 50,
                            tiersMode: 'graduated',
                            planTiers: [
                                { __typename: 'PlanTier', flatAmount: 100, unitAmount: 200, upTo: 300 },
                                { __typename: 'PlanTier', flatAmount: 400, unitAmount: 500, upTo: 600 },
                            ],
                        }}
                    />
                )
                .toJSON()
        ).toMatchInlineSnapshot(`
            Array [
              <div>
                $0.17/user/month
                 
                for the first 300 users
              </div>,
              <div>
                $0.42/user/month
                 
                for the next 300 users
              </div>,
            ]
        `)
    })
})
