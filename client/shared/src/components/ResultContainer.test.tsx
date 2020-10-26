import * as H from 'history'
import FileIcon from 'mdi-react/FileIcon'
import * as React from 'react'
import { cleanup, fireEvent, getByTestId, getByText, render } from '@testing-library/react'
import sinon from 'sinon'
import { FileMatchChildren } from './FileMatchChildren'
import { RepoFileLink } from './RepoFileLink'
import { ResultContainer } from './ResultContainer'
import {
    MULTIPLE_MATCH_RESULT,
    HIGHLIGHTED_FILE_LINES_SIMPLE_REQUEST,
    NOOP_SETTINGS_CASCADE,
} from '../util/searchTestHelpers'

describe('ResultContainer', () => {
    afterAll(cleanup)

    const history = H.createBrowserHistory()
    history.replace({ pathname: '/search' })
    const onSelect = sinon.spy()
    const fileMatchChildrenProps = {
        location: history.location,
        items: [
            {
                preview: '\t"net/http/httptest"',
                line: 11,
                highlightRanges: [{ start: 15, highlightLength: 4 }],
            },
            {
                preview: '\tts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {',
                line: 39,
                highlightRanges: [{ start: 11, highlightLength: 4 }],
            },
            {
                preview: 'func TestTokenRequest(t *testing.T) {',
                line: 73,
                highlightRanges: [{ start: 5, highlightLength: 4 }],
            },
            {
                preview: '\tts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {',
                line: 117,
                highlightRanges: [{ start: 11, highlightLength: 4 }],
            },
            {
                preview: '\t\tio.WriteString(w, `{"access_token": "foo", "refresh_token": "bar"}`)',
                line: 134,
                highlightRanges: [{ start: 8, highlightLength: 4 }],
            },
        ],
        result: MULTIPLE_MATCH_RESULT,
        allMatches: true,
        subsetMatches: 1,
        fetchHighlightedFileLines: HIGHLIGHTED_FILE_LINES_SIMPLE_REQUEST,
        onSelect,
        settingsCascade: NOOP_SETTINGS_CASCADE,
        isLightTheme: true,
    }

    // Props that represent a FileMatch with multiple results, totaling more than subsetMatch.
    // The FileMatch is not expanded by default, and is collapsible. These are the props passed
    // down to ResultContainers when they are used for search results.
    const defaultProps = {
        location: history.location,
        collapsible: true,
        defaultExpanded: false,
        icon: FileIcon,
        title: (
            <RepoFileLink
                repoName="example.com/my/repo"
                repoURL="https://example.com"
                filePath="file.go"
                fileURL="https://example.com/file"
            />
        ),
        expandedChildren: <FileMatchChildren {...fileMatchChildrenProps} />,
        collapsedChildren: <FileMatchChildren {...fileMatchChildrenProps} allMatches={false} />,
        collapseLabel: 'Hide matches',
        expandLabel: 'Show matches',
        allExpanded: false,
    }

    const findReferencesProps = {
        location: history.location,
        collapsible: true,
        defaultExpanded: true,
        icon: FileIcon,
        title: (
            <RepoFileLink
                repoName="example.com/my/repo"
                repoURL="https://example.com"
                filePath="file.go"
                fileURL="https://example.com/file"
            />
        ),
        expandedChildren: <FileMatchChildren {...fileMatchChildrenProps} />,
        allExpanded: true,
    }
    it('displays only one result when collapsed, which is the equivalent of subsetMatches', () => {
        const { container } = render(<ResultContainer {...defaultProps} />)

        const expandedItems = container.querySelectorAll('.file-match-children__item')
        // 1 is the value of subsetMatches
        expect(expandedItems.length).toBe(1)
    })

    it('expands to display all results when the header is clicked', () => {
        const { container } = render(<ResultContainer {...defaultProps} />)

        let expandedItems = container.querySelectorAll('.file-match-children__item')
        // 1 is the value of subsetMatches
        expect(expandedItems.length).toBe(1)

        const header = container.querySelector('.result-container__header--collapsible')
        expect(header).toBeTruthy()

        fireEvent.click(header!)

        expandedItems = container.querySelectorAll('.file-match-children__item')
        expect(expandedItems.length).toBe(5)
    })

    it('displays the expand label when collapsed', () => {
        const { container } = render(<ResultContainer {...defaultProps} />)
        const header = getByTestId(container, 'result-container-header')
        expect(header).toBeTruthy()
        expect(getByText(container, 'Show matches')).toBeTruthy()
    })

    it('displays the collapse label when expanded', () => {
        const { container } = render(<ResultContainer {...defaultProps} />)

        const clickableHeader = container.querySelector('.result-container__header--collapsible')
        expect(clickableHeader).toBeTruthy()

        fireEvent.click(clickableHeader!)

        expect(getByText(container, 'Hide matches')).toBeTruthy()
    })

    it('displays all results by default, when allExpanded is true', () => {
        const { container } = render(<ResultContainer {...findReferencesProps} />)

        const expandedItems = container.querySelectorAll('.file-match-children__item')
        expect(expandedItems.length).toBe(5)
    })

    it('collapses to show no results when the header is clicked, when allExpanded is true', () => {
        const { container } = render(<ResultContainer {...findReferencesProps} />)

        let expandedItems = container.querySelectorAll('.file-match-children__item')
        expect(expandedItems.length).toBe(5)

        const header = container.querySelector('.result-container__header--collapsible')
        expect(header).toBeTruthy()
        fireEvent.click(header!)

        expandedItems = container.querySelectorAll('.file-match-children__item')
        expect(expandedItems.length).toBe(0)
    })
})
