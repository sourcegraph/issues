import renderer from 'react-test-renderer'
import { LoaderInput } from './LoaderInput'
import React from 'react'

jest.mock('@sourcegraph/react-loading-spinner', () => ({ LoadingSpinner: 'LoadingSpinner' }))

describe('LoaderInput', () => {
    it('should render a loading spinner when loading prop is true', () => {
        expect(
            renderer
                .create(
                    <LoaderInput loading={true}>
                        <input type="text" />
                    </LoaderInput>
                )
                .toJSON()
        ).toMatchSnapshot()
    })

    it('should not render a loading spinner when loading prop is false', () => {
        expect(
            renderer
                .create(
                    <LoaderInput loading={false}>
                        <input type="text" />
                    </LoaderInput>
                )
                .toJSON()
        ).toMatchSnapshot()
    })
})
