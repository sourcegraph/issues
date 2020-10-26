import React from 'react'
import renderer from 'react-test-renderer'
import { ProductSubscriptionUserCountFormControl } from './ProductSubscriptionUserCountFormControl'

describe('ProductSubscriptionUserCountFormControl', () => {
    test('renders', () => {
        expect(
            renderer.create(<ProductSubscriptionUserCountFormControl value={123} onChange={() => undefined} />).toJSON()
        ).toMatchSnapshot()
    })
})
