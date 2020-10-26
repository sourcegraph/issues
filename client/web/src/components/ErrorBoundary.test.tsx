import React from 'react'
import renderer from 'react-test-renderer'
import { ErrorBoundary } from './ErrorBoundary'

jest.mock('mdi-react/ErrorIcon', () => 'ErrorIcon')
jest.mock('mdi-react/ReloadIcon', () => 'ReloadIcon')

const ThrowError: React.FunctionComponent = () => {
    throw new Error('x')
}

/** Throws an error that resembles the Webpack error when chunk loading fails.  */
const ThrowChunkError: React.FunctionComponent = () => {
    throw new Error('Loading chunk 123 failed.')
}

describe('ErrorBoundary', () => {
    test('passes through if non-error', () =>
        expect(
            renderer
                .create(
                    <ErrorBoundary location={null}>
                        <ThrowError />
                    </ErrorBoundary>
                )
                .toJSON()
        ).toMatchSnapshot())

    test('renders error page if error', () =>
        expect(
            renderer
                .create(
                    <ErrorBoundary location={null}>
                        <span>hello</span>
                    </ErrorBoundary>
                )
                .toJSON()
        ).toMatchSnapshot())

    test('renders reload page if chunk error', () =>
        expect(
            renderer
                .create(
                    <ErrorBoundary location={null}>
                        <ThrowChunkError />
                    </ErrorBoundary>
                )
                .toJSON()
        ).toMatchSnapshot())
})
