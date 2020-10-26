import { createBrowserHistory } from 'history'
import FileIcon from 'mdi-react/FileIcon'
import * as React from 'react'
import { cleanup, getAllByTestId, getByTestId, render } from '@testing-library/react'
import _VisibilitySensor from 'react-visibility-sensor'
import sinon from 'sinon'
import { MockVisibilitySensor } from './CodeExcerpt.test'
import { FileMatch, IFileMatch } from './FileMatch'
import { HIGHLIGHTED_FILE_LINES_REQUEST, NOOP_SETTINGS_CASCADE, RESULT } from '../util/searchTestHelpers'

jest.mock('react-visibility-sensor', (): typeof _VisibilitySensor => ({ children, onChange }) => (
    <>
        <MockVisibilitySensor onChange={onChange}>{children}</MockVisibilitySensor>
    </>
))

describe('FileMatch', () => {
    afterAll(cleanup)
    const history = createBrowserHistory()
    const defaultProps = {
        location: history.location,
        result: RESULT as IFileMatch,
        icon: FileIcon,
        onSelect: sinon.spy(),
        expanded: true,
        showAllMatches: true,
        isLightTheme: true,
        fetchHighlightedFileLines: HIGHLIGHTED_FILE_LINES_REQUEST,
        settingsCascade: NOOP_SETTINGS_CASCADE,
    }

    it('renders one result container', () => {
        const { container } = render(<FileMatch {...defaultProps} />)
        expect(getByTestId(container, 'result-container')).toBeTruthy()
        expect(getAllByTestId(container, 'result-container').length).toBe(1)
    })
})
