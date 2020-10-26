import * as H from 'history'
import * as React from 'react'
import { cleanup, fireEvent, render } from '@testing-library/react'
import _VisibilitySensor from 'react-visibility-sensor'
import { MockVisibilitySensor } from './CodeExcerpt.test'

jest.mock('react-visibility-sensor', (): typeof _VisibilitySensor => ({ children, onChange }) => (
    <>
        <MockVisibilitySensor onChange={onChange}>{children}</MockVisibilitySensor>
    </>
))

import sinon from 'sinon'

import { FileMatchChildren } from './FileMatchChildren'
import { RESULT, HIGHLIGHTED_FILE_LINES_SIMPLE_REQUEST, NOOP_SETTINGS_CASCADE } from '../util/searchTestHelpers'

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
    fetchHighlightedFileLines: HIGHLIGHTED_FILE_LINES_SIMPLE_REQUEST,
    onSelect,
    settingsCascade: NOOP_SETTINGS_CASCADE,
    isLightTheme: true,
    versionContext: undefined,
}

describe('FileMatchChildren', () => {
    afterAll(cleanup)

    it('calls onSelect callback when an item is clicked', () => {
        const { container } = render(<FileMatchChildren {...defaultProps} onSelect={onSelect} />)
        const item = container.querySelector('.file-match-children__item')
        expect(item).toBeTruthy()
        fireEvent.click(item!)
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
})
