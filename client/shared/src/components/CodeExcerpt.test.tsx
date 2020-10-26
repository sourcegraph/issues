import * as React from 'react'
import _VisibilitySensor from 'react-visibility-sensor'
import sinon from 'sinon'
export class MockVisibilitySensor extends React.Component<{ onChange?: (isVisible: boolean) => void }> {
    constructor(props: { onChange?: (isVisible: boolean) => void }) {
        super(props)
        if (props.onChange) {
            props.onChange(true)
        }
    }

    public render(): JSX.Element {
        return <>{this.props.children}</>
    }
}

jest.mock('react-visibility-sensor', (): typeof _VisibilitySensor => ({ children, onChange }) => (
    <>
        <MockVisibilitySensor onChange={onChange}>{children}</MockVisibilitySensor>
    </>
))

import { cleanup, getAllByText, getByText, render } from '@testing-library/react'
import { of } from 'rxjs'

import { CodeExcerpt } from './CodeExcerpt'
import {
    HIGHLIGHTED_FILE_LINES,
    HIGHLIGHTED_FILE_LINES_REQUEST,
    HIGHLIGHTED_FILE_LINES_SIMPLE_REQUEST,
    HIGHLIGHTED_FILE_LINES_LONG,
} from '../util/searchTestHelpers'

describe('CodeExcerpt', () => {
    afterAll(cleanup)

    const defaultProps = {
        repoName: 'github.com/golang/oauth2',
        commitID: 'e64efc72b421e893cbf63f17ba2221e7d6d0b0f3',
        filePath: '.travis.yml',
        highlightRanges: [
            { line: 0, character: 6, highlightLength: 4 },
            { line: 1, character: 7, highlightLength: 4 },
            { line: 2, character: 6, highlightLength: 4 },
        ],
        lastSubsetMatchLineNumber: 2,
        isLightTheme: false,
        className: 'file-match__item-code-excerpt',
        fetchHighlightedFileLines: HIGHLIGHTED_FILE_LINES_SIMPLE_REQUEST,
    }

    it('renders correct number of rows', () => {
        const { container } = render(<CodeExcerpt {...defaultProps} />)
        expect(container.querySelectorAll('.code-excerpt tr').length).toBe(4)
    })

    it('renders correct number of rows with context value as 5', () => {
        const { container } = render(
            <CodeExcerpt
                {...defaultProps}
                context={5}
                highlightRanges={[{ line: 4, character: 1, highlightLength: 2 }]}
                lastSubsetMatchLineNumber={4}
            />
        )
        expect(container.querySelectorAll('.code-excerpt tr').length).toBe(10)
    })

    it('renders the line number container on each row', () => {
        // We can't evaluate the content of these containers since the numbers are drawn
        // by CSS. This is a proxy to make sure the <td> tags with data-line attributes
        // at least exist.
        const { container } = render(<CodeExcerpt {...defaultProps} />)
        const dataLines = container.querySelectorAll('[data-line]')
        expect(dataLines.length).toBe(4)
    })

    it('renders the code portion of each row', () => {
        const { container } = render(<CodeExcerpt {...defaultProps} />)
        expect(getByText(container, 'first of code')).toBeTruthy()
        expect(getByText(container, 'second of code')).toBeTruthy()
        expect(getByText(container, 'third of code')).toBeTruthy()
        expect(getAllByText(container, 'line').length).toBe(3)
        expect(getByText(container, 'fourth')).toBeTruthy()
    })

    it('highlights matches correctly', () => {
        const { container } = render(<CodeExcerpt {...defaultProps} />)
        const highlightedSpans = container.querySelectorAll('.selection-highlight')
        expect(highlightedSpans.length).toBe(3)
        for (const span of highlightedSpans) {
            expect(span.textContent === 'line')
        }
    })

    it('highlights matches correctly in syntax highlighted code blocks', () => {
        const highlightRange = [{ line: 12, character: 7, highlightLength: 4 }]
        const { container } = render(
            <CodeExcerpt
                {...defaultProps}
                highlightRanges={highlightRange}
                fetchHighlightedFileLines={HIGHLIGHTED_FILE_LINES_REQUEST}
            />
        )
        const highlightedSpans = container.querySelectorAll('.selection-highlight')
        expect(highlightedSpans.length).toBe(1)
        for (const span of highlightedSpans) {
            expect(span.textContent === 'test')
        }
    })

    it('does not disable the highlighting timeout', () => {
        /*
            Because disabling the timeout should only ever be done in response
            to the user asking us to do so, something that we do not do for
            code excerpts because falling back to plaintext rendering is most
            ideal.
        */
        const fetchHighlightedFileLines = sinon.spy(context => of(HIGHLIGHTED_FILE_LINES))
        const highlightRange = [{ line: 12, character: 7, highlightLength: 4 }]
        render(
            <CodeExcerpt
                {...defaultProps}
                highlightRanges={highlightRange}
                fetchHighlightedFileLines={fetchHighlightedFileLines}
            />
        )
        sinon.assert.calledOnce(fetchHighlightedFileLines)
        sinon.assert.calledWithMatch(fetchHighlightedFileLines, { disableTimeout: false })
    })

    it('displays the correct number of lines, with no highlight matches on the context lines', () => {
        const fetchHighlightedFileLines = sinon.spy(context => of(HIGHLIGHTED_FILE_LINES_LONG))
        const highlightRanges = [
            { line: 0, character: 0, highlightLength: 1 },
            { line: 1, character: 0, highlightLength: 1 },
            { line: 2, character: 0, highlightLength: 1 },
            { line: 4, character: 0, highlightLength: 1 },
            { line: 7, character: 0, highlightLength: 1 },
            { line: 10, character: 0, highlightLength: 1 },
            { line: 11, character: 0, highlightLength: 1 },
            { line: 16, character: 0, highlightLength: 1 },
            { line: 17, character: 0, highlightLength: 1 },
            { line: 19, character: 0, highlightLength: 1 },
        ]
        const { container } = render(
            <CodeExcerpt
                {...defaultProps}
                context={2}
                lastSubsetMatchLineNumber={20}
                highlightRanges={highlightRanges}
                fetchHighlightedFileLines={fetchHighlightedFileLines}
            />
        )
        const rows = container.querySelectorAll('tr')
        expect(rows.length).toBe(22)
    })

    it('displays the correct number of lines, with matches on the context line', () => {
        const fetchHighlightedFileLines = sinon.spy(context => of(HIGHLIGHTED_FILE_LINES_LONG))
        const highlightRanges = [
            { line: 0, character: 0, highlightLength: 1 },
            { line: 1, character: 0, highlightLength: 1 },
            { line: 2, character: 0, highlightLength: 1 },
            { line: 4, character: 0, highlightLength: 1 },
            { line: 7, character: 0, highlightLength: 1 },
            { line: 10, character: 0, highlightLength: 1 },
            { line: 11, character: 0, highlightLength: 1 },
            { line: 16, character: 0, highlightLength: 1 },
            { line: 17, character: 0, highlightLength: 1 },
            { line: 19, character: 0, highlightLength: 1 },
            { line: 20, character: 0, highlightLength: 1 },
        ]
        const { container } = render(
            <CodeExcerpt
                {...defaultProps}
                context={2}
                lastSubsetMatchLineNumber={19}
                highlightRanges={highlightRanges}
                fetchHighlightedFileLines={fetchHighlightedFileLines}
            />
        )
        const rows = container.querySelectorAll('tr')
        expect(rows.length).toBe(22)
    })
})
