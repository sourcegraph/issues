import React from 'react'
import renderer from 'react-test-renderer'
import { ProductPlanPrice } from './ProductPlanPrice'

describe('ProductPlanPrice', () => {
    test('free, no max', () => {
        expect(
            renderer
                .create(
                    <ProductPlanPrice
                        plan={{
                            minQuantity: null,
                            maxQuantity: null,
                            tiersMode: '',
                            pricePerUserPerYear: 0,
                            planTiers: [],
                        }}
                    />
                )
                .toJSON()
        ).toMatchInlineSnapshot(`
            Array [
              "$0/month",
              " ",
            ]
        `)
    })

    test('max', () => {
        expect(
            renderer
                .create(
                    <ProductPlanPrice
                        plan={{
                            minQuantity: 50,
                            maxQuantity: 100,
                            tiersMode: '',
                            pricePerUserPerYear: 0,
                            planTiers: [],
                        }}
                    />
                )
                .toJSON()
        ).toMatchInlineSnapshot(`
            Array [
              "$0/month",
              " ",
              "(up to ",
              "100",
              " ",
              "users",
              ")",
            ]
        `)
    })

    test('priced', () => {
        expect(
            renderer
                .create(
                    <ProductPlanPrice
                        plan={{
                            minQuantity: null,
                            maxQuantity: null,
                            tiersMode: '',
                            pricePerUserPerYear: 100,
                            planTiers: [],
                        }}
                    />
                )
                .toJSON()
        ).toMatchInlineSnapshot(`
            Array [
              "$0.08",
              "/user/month (paid yearly)",
            ]
        `)
    })
})
