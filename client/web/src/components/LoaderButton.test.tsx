import renderer from 'react-test-renderer'
import { LoaderButton } from './LoaderButton'
import React from 'react'

jest.mock('@sourcegraph/react-loading-spinner', () => ({ LoadingSpinner: 'LoadingSpinner' }))

describe('LoaderButton', () => {
    it('should render a loading spinner when loading prop is true', () => {
        expect(renderer.create(<LoaderButton label="Test" loading={true} />).toJSON()).toMatchSnapshot()
    })

    it('should not render a loading spinner when loading prop is false', () => {
        expect(renderer.create(<LoaderButton label="Test" loading={false} />).toJSON()).toMatchSnapshot()
    })
})
