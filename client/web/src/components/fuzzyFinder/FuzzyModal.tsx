import Dialog from '@reach/dialog'
import classNames from 'classnames'
import CloseIcon from 'mdi-react/CloseIcon'
import React, { useState } from 'react'

import { pluralize } from '@sourcegraph/shared/src/util/strings'
import { useLocalStorage } from '@sourcegraph/shared/src/util/useLocalStorage'

import { CaseInsensitiveFuzzySearch } from '../../fuzzyFinder/CaseInsensitiveFuzzySearch'
import { FuzzySearch, FuzzySearchResult, SearchIndexing, SearchValue } from '../../fuzzyFinder/FuzzySearch'
import { WordSensitiveFuzzySearch } from '../../fuzzyFinder/WordSensitiveFuzzySearch'
import { parseBrowserRepoURL } from '../../util/url'

import { FuzzyFinderProps, Indexing, FuzzyFSM } from './FuzzyFinder'
import styles from './FuzzyModal.module.scss'
import { HighlightedLink } from './HighlightedLink'

// The default value of 80k filenames is picked from the following observations:
// - case-insensitive search is slow but works in the torvalds/linux repo (72k files)
// - case-insensitive search is almost unusable in the chromium/chromium repo (360k files)
const DEFAULT_CASE_INSENSITIVE_FILE_COUNT_THRESHOLD = 80000

const FUZZY_MODAL_TITLE = 'fuzzy-modal-title'
const FUZZY_MODAL_RESULTS = 'fuzzy-modal-results'

// Cache for the last fuzzy query. This value is only used to avoid redoing the
// full fuzzy search on every re-render when the user presses the down/up arrow
// keys to move the "focus index".
const lastFuzzySearchResult = new Map<string, FuzzySearchResult>()

// The number of results to jump by on PageUp/PageDown keyboard shortcuts.
const PAGE_DOWN_INCREMENT = 10

export interface FuzzyModalProps extends FuzzyFinderProps {
    initialMaxResults: number
    initialQuery: string
    downloadFilenames: () => Promise<string[]>

    isVisible: boolean
    onClose: () => void

    fsm: FuzzyFSM
    setFsm: (fsm: FuzzyFSM) => void
}

interface FuzzyModalState {
    query: string
    setQuery: (query: string) => void

    focusIndex: number
    setFocusIndex: (focusIndex: number) => void

    maxResults: number
    increaseMaxResults: () => void
}

/**
 * Component that interactively displays filenames in the open repository when given fuzzy queries.
 *
 * Similar to "Go to file" in VS Code or the "t" keyboard shortcut on github.com
 */
export const FuzzyModal: React.FunctionComponent<FuzzyModalProps> = props => {
    // NOTE: the query is cached in local storage to mimic the file pickers in
    // IntelliJ (by default) and VS Code (when "Workbench > Quick Open >
    // Preserve Input" is enabled).
    const [query, setQuery] = useLocalStorage(`fuzzy-modal.query.${props.repoName}`, props.initialQuery)

    // The "focus index" is the index of the file result that the user has
    // select with up/down arrow keys. The focused item is highlighted and the
    // window.location is moved to that URL when the user presses the enter key.
    const [focusIndex, setFocusIndex] = useState(0)

    // The maximum number of results to display in the fuzzy finder. For large
    // repositories, a generic query like "src" may return thousands of results
    // making DOM rendering slow.  The user can increase this number by clicking
    // on a button at the bottom of the result list.
    const [maxResults, setMaxResults] = useState(props.initialMaxResults)

    const state: FuzzyModalState = {
        query,
        setQuery,
        focusIndex,
        setFocusIndex,
        maxResults,
        increaseMaxResults: () => {
            setMaxResults(maxResults + props.initialMaxResults)
        },
    }

    const fuzzyResult = renderFuzzyResult(props, state)

    // Sets the new "focus index" so that it's rounded by the number of
    // displayed filenames.  Cycles so that the user can press-hold the down
    // arrow and it goes all the way down and back up to the top result.
    function setRoundedFocusIndex(increment: number): void {
        const newNumber = state.focusIndex + increment
        const index = newNumber % fuzzyResult.resultsCount
        const nextIndex = index < 0 ? fuzzyResult.resultsCount + index : index
        state.setFocusIndex(nextIndex)
        document.querySelector(`#fuzzy-modal-result-${nextIndex}`)?.scrollIntoView(false)
    }

    function onInputKeyDown(event: React.KeyboardEvent): void {
        switch (event.key) {
            case 'Escape':
                props.onClose()
                break
            case 'ArrowDown':
                event.preventDefault() // Don't move the cursor to the end of the input.
                setRoundedFocusIndex(1)
                break
            case 'PageDown':
                setRoundedFocusIndex(PAGE_DOWN_INCREMENT)
                break
            case 'ArrowUp':
                event.preventDefault() // Don't move the cursor to the start of input.
                setRoundedFocusIndex(-1)
                break
            case 'PageUp':
                setRoundedFocusIndex(-PAGE_DOWN_INCREMENT)
                break
            case 'Enter':
                if (state.focusIndex < fuzzyResult.resultsCount) {
                    const fileAnchor = document.querySelector<HTMLAnchorElement>(
                        `#fuzzy-modal-result-${state.focusIndex} a`
                    )
                    fileAnchor?.click()
                    props.onClose()
                }
                break
            default:
        }
    }

    return (
        <Dialog
            className={classNames(styles.modal, 'modal-body p-4 rounded border')}
            onDismiss={() => props.onClose()}
            aria-labelledby={FUZZY_MODAL_TITLE}
        >
            <div className={styles.content}>
                <div className={styles.header}>
                    <h3 className="mb-0" id={FUZZY_MODAL_TITLE}>
                        Find file
                    </h3>
                    <button type="button" className="btn btn-icon" onClick={() => props.onClose()} aria-label="Close">
                        <CloseIcon className={`icon-inline ${styles.closeIcon}`} />
                    </button>
                </div>
                <input
                    autoComplete="off"
                    spellCheck="false"
                    role="combobox"
                    aria-autocomplete="list"
                    aria-controls={FUZZY_MODAL_RESULTS}
                    aria-owns={FUZZY_MODAL_RESULTS}
                    aria-expanded={props.fsm.key !== 'downloading'}
                    aria-activedescendant={fuzzyResultId(state.focusIndex)}
                    id="fuzzy-modal-input"
                    className={classNames('form-control py-1', styles.input)}
                    placeholder="Enter a partial file path or name"
                    value={state.query}
                    onChange={event => {
                        state.setQuery(event.target.value)
                        state.setFocusIndex(0)
                    }}
                    type="text"
                    onKeyDown={onInputKeyDown}
                />
                <div className={styles.summary}>
                    <FuzzyResultsSummary fsm={props.fsm} files={fuzzyResult} />
                </div>
                {fuzzyResult.element}
                {!fuzzyResult.isComplete && (
                    <button
                        className={classNames('btn btn-secondary', styles.showMore)}
                        type="button"
                        onClick={() => state.increaseMaxResults()}
                    >
                        Show more
                    </button>
                )}
            </div>
        </Dialog>
    )
}

function plural(what: string, count: number, isComplete: boolean): string {
    return `${count.toLocaleString()}${isComplete ? '' : '+'} ${pluralize(what, count)}`
}
interface FuzzyResultsSummaryProps {
    fsm: FuzzyFSM
    files: RenderedFuzzyResult
}

const FuzzyResultsSummary: React.FunctionComponent<FuzzyResultsSummaryProps> = ({ fsm, files }) => (
    <>
        <span className={styles.resultCount}>
            {plural('result', files.resultsCount, files.isComplete)} -{' '}
            {fsm.key === 'indexing' && indexingProgressBar(fsm)} {plural('total file', files.totalFileCount, true)}
        </span>
        <i className="text-muted">
            <kbd>↑</kbd> and <kbd>↓</kbd> arrow keys browse. <kbd>⏎</kbd> selects.
        </i>
    </>
)

function indexingProgressBar(indexing: Indexing): JSX.Element {
    const indexedFiles = indexing.indexing.indexedFileCount
    const totalFiles = indexing.indexing.totalFileCount
    const percentage = Math.round((indexedFiles / totalFiles) * 100)
    return (
        <progress value={indexedFiles} max={totalFiles}>
            {percentage}%
        </progress>
    )
}

interface RenderedFuzzyResult {
    element: JSX.Element
    resultsCount: number
    isComplete: boolean
    totalFileCount: number
    elapsedMilliseconds?: number
    falsePositiveRatio?: number
}

function renderFuzzyResult(props: FuzzyModalProps, state: FuzzyModalState): RenderedFuzzyResult {
    function empty(element: JSX.Element): RenderedFuzzyResult {
        return {
            element,
            resultsCount: 0,
            isComplete: true,
            totalFileCount: 0,
        }
    }

    function onError(what: string): (error: Error) => void {
        return error => {
            props.setFsm({ key: 'failed', errorMessage: JSON.stringify(error) })
            throw new Error(what)
        }
    }

    switch (props.fsm.key) {
        case 'empty':
            handleEmpty(props).then(() => {}, onError('onEmpty'))
            return empty(<></>)
        case 'downloading':
            return empty(<p>Downloading...</p>)
        case 'failed':
            return empty(<p>Error: {props.fsm.errorMessage}</p>)
        case 'indexing': {
            const loader = props.fsm.indexing
            later()
                .then(() => continueIndexing(loader))
                .then(next => props.setFsm(next), onError('onIndexing'))
            return renderFiles(props, state, props.fsm.indexing.partialFuzzy, props.fsm.indexing)
        }
        case 'ready':
            return renderFiles(props, state, props.fsm.fuzzy)
        default:
            return empty(<p>ERROR</p>)
    }
}

function renderFiles(
    props: FuzzyModalProps,
    state: FuzzyModalState,
    search: FuzzySearch,
    indexing?: SearchIndexing
): RenderedFuzzyResult {
    const { revision } = parseBrowserRepoURL(location.pathname + location.search + location.hash)
    const revisionUrlPart = revision ? `@${revision}` : ''
    const indexedFileCount = indexing ? indexing.indexedFileCount : ''
    const cacheKey = `${state.query}-${state.maxResults}${indexedFileCount}-${revisionUrlPart}`
    let fuzzyResult = lastFuzzySearchResult.get(cacheKey)
    if (!fuzzyResult) {
        const start = window.performance.now()
        fuzzyResult = search.search({
            query: state.query,
            maxResults: state.maxResults,
            createUrl: filename => `/${props.repoName}${revisionUrlPart}/-/blob/${filename}`,
            onClick: () => props.onClose(),
        })
        fuzzyResult.elapsedMilliseconds = window.performance.now() - start
        lastFuzzySearchResult.clear() // Only cache the last query.
        lastFuzzySearchResult.set(cacheKey, fuzzyResult)
    }
    const links = fuzzyResult.links
    if (links.length === 0) {
        return {
            element: <p>No files matching '{state.query}'</p>,
            resultsCount: 0,
            totalFileCount: search.totalFileCount,
            isComplete: fuzzyResult.isComplete,
        }
    }
    const linksToRender = links.slice(0, state.maxResults)
    return {
        element: (
            <ul id={FUZZY_MODAL_RESULTS} className={styles.results} role="listbox" aria-label="Fuzzy finder results">
                {linksToRender.map((file, fileIndex) => (
                    <li
                        id={fuzzyResultId(fileIndex)}
                        key={file.text}
                        role="option"
                        aria-selected={fileIndex === state.focusIndex}
                        className={classNames('p-1', fileIndex === state.focusIndex && styles.focused)}
                    >
                        <HighlightedLink {...file} />
                    </li>
                ))}
            </ul>
        ),
        resultsCount: linksToRender.length,
        totalFileCount: search.totalFileCount,
        isComplete: fuzzyResult.isComplete,
        elapsedMilliseconds: fuzzyResult.elapsedMilliseconds,
        falsePositiveRatio: fuzzyResult.falsePositiveRatio,
    }
}

async function later(): Promise<void> {
    return new Promise(resolve => setTimeout(() => resolve(), 0))
}

async function continueIndexing(indexing: SearchIndexing): Promise<FuzzyFSM> {
    const next = await indexing.continue()
    if (next.key === 'indexing') {
        return { key: 'indexing', indexing: next }
    }
    return {
        key: 'ready',
        fuzzy: next.value,
    }
}

async function handleEmpty(props: FuzzyModalProps): Promise<void> {
    props.setFsm({ key: 'downloading' })
    try {
        const filenames = await props.downloadFilenames()
        props.setFsm(handleFilenames(filenames))
    } catch (error) {
        props.setFsm({
            key: 'failed',
            errorMessage: JSON.stringify(error),
        })
    }
    cleanLegacyCacheStorage()
}

function handleFilenames(filenames: string[]): FuzzyFSM {
    const values: SearchValue[] = filenames.map(file => ({ text: file }))
    if (filenames.length < DEFAULT_CASE_INSENSITIVE_FILE_COUNT_THRESHOLD) {
        return {
            key: 'ready',
            fuzzy: new CaseInsensitiveFuzzySearch(values),
        }
    }
    const indexing = WordSensitiveFuzzySearch.fromSearchValuesAsync(values)
    if (indexing.key === 'ready') {
        return {
            key: 'ready',
            fuzzy: indexing.value,
        }
    }
    return {
        key: 'indexing',
        indexing,
    }
}

/**
 * Removes unused cache storage from the initial implementation of the fuzzy finder.
 *
 * This method can be removed in the future. The cache storage was no longer
 * needed after we landed an optimization in the backend that made it faster to
 * download filenames.
 */
function cleanLegacyCacheStorage(): void {
    const cacheAvailable = 'caches' in self
    if (!cacheAvailable) {
        return
    }

    caches.delete('fuzzy-modal').then(
        () => {},
        () => {}
    )
}

function fuzzyResultId(id: number): string {
    return `fuzzy-modal-result-${id}`
}
