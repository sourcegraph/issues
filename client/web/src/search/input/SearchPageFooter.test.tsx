import React from 'react'
import { cleanup, render } from '@testing-library/react'
import { SearchPageFooter } from './SearchPageFooter'

describe('SearchPageFooter', () => {
    afterAll(cleanup)

    let container: HTMLElement

    it('should render correctly', () => {
        container = render(<SearchPageFooter className="test-class" />).container
        expect(container).toMatchSnapshot()
    })
})
