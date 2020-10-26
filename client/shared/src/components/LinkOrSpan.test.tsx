import React from 'react'
import renderer from 'react-test-renderer'
import { LinkOrSpan } from './LinkOrSpan'

describe('LinkOrSpan', () => {
    test('render a link when "to" is set', () => {
        const component = renderer.create(<LinkOrSpan to="http://example.com">foo</LinkOrSpan>)
        expect(component.toJSON()).toMatchSnapshot()
    })

    test('render a span when "to" is undefined', () => {
        const component = renderer.create(<LinkOrSpan to={undefined}>foo</LinkOrSpan>)
        expect(component.toJSON()).toMatchSnapshot()
    })
})
