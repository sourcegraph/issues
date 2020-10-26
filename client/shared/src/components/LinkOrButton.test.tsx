import React from 'react'
import renderer from 'react-test-renderer'
import { ButtonLink } from './LinkOrButton'

describe('LinkOrButton', () => {
    test('render a link when "to" is set', () => {
        const component = renderer.create(<ButtonLink to="http://example.com">foo</ButtonLink>)
        expect(component.toJSON()).toMatchSnapshot()
    })

    test('render a button when "to" is undefined', () => {
        const component = renderer.create(<ButtonLink to={undefined}>foo</ButtonLink>)
        expect(component.toJSON()).toMatchSnapshot()
    })
})
