import { cleanup, fireEvent, render } from '@testing-library/react'
import * as H from 'history'
import * as React from 'react'
import { MemoryRouter } from 'react-router'
import _VisibilitySensor from 'react-visibility-sensor'
import { of } from 'rxjs'
import sinon from 'sinon'

import {
    RESULT,
    HIGHLIGHTED_FILE_LINES_SIMPLE_REQUEST,
    NOOP_SETTINGS_CASCADE,
    HIGHLIGHTED_FILE_LINES,
} from '../util/searchTestHelpers'

import { MockVisibilitySensor } from './CodeExcerpt.test'
import { FileMatchChildren } from './FileMatchChildren'

jest.mock('react-visibility-sensor', (): typeof _VisibilitySensor => ({ children, onChange }) => (
    <>
        <MockVisibilitySensor onChange={onChange}>{children}</MockVisibilitySensor>
    </>
))

const history = H.createBrowserHistory()
history.replace({ pathname: '/search' })

const onSelect = sinon.spy()

const defaultProps = {
    location: history.location,
    items: [
        {
            preview: 'third line of code',
            line: 3,
            highlightRanges: [{ start: 7, highlightLength: 4 }],
        },
    ],
    result: RESULT,
    allMatches: true,
    subsetMatches: 10,
    fetchHighlightedFileLineRanges: HIGHLIGHTED_FILE_LINES_SIMPLE_REQUEST,
    onSelect,
    settingsCascade: NOOP_SETTINGS_CASCADE,
    isLightTheme: true,
    versionContext: undefined,
}

describe('FileMatchChildren', () => {
    const realWindowGetSelection = window.getSelection
    beforeAll(() => {
        window.getSelection = () => null
    })

    afterAll(() => {
        window.getSelection = realWindowGetSelection
    })

    afterAll(cleanup)

    it('calls onSelect callback on item mouse up', () => {
        const { container } = render(
            <MemoryRouter>
                <FileMatchChildren {...defaultProps} onSelect={onSelect} />
            </MemoryRouter>
        )
        const item = container.querySelector('.file-match-children__item')
        expect(item).toBeVisible()
        fireEvent.mouseUp(item!)
        expect(onSelect.calledOnce).toBe(true)
    })

    it('correctly shows number of context lines when search.contextLines setting is set', () => {
        const settingsCascade = {
            final: { 'search.contextLines': 3 },
            subjects: [
                {
                    lastID: 1,
                    settings: { 'search.contextLines': '3' },
                    extensions: null,
                    subject: {
                        __typename: 'User' as const,
                        username: 'f',
                        id: 'abc',
                        settingsURL: '/users/f/settings',
                        viewerCanAdminister: true,
                        displayName: 'f',
                    },
                },
            ],
        }
        const { container } = render(<FileMatchChildren {...defaultProps} settingsCascade={settingsCascade} />)
        const tableRows = container.querySelectorAll('.code-excerpt tr')
        expect(tableRows.length).toBe(7)
    })

    it('does not disable the highlighting timeout', () => {
        /*
            Because disabling the timeout should only ever be done in response
            to the user asking us to do so, something that we do not do for
            file matches because falling back to plaintext rendering is most
            ideal.
        */
        const fetchHighlightedFileLineRanges = sinon.spy(context => of(HIGHLIGHTED_FILE_LINES))
        render(<FileMatchChildren {...defaultProps} fetchHighlightedFileLineRanges={fetchHighlightedFileLineRanges} />)
        sinon.assert.calledOnce(fetchHighlightedFileLineRanges)
        sinon.assert.calledWithMatch(fetchHighlightedFileLineRanges, { disableTimeout: false })
    })
})
