import * as H from 'history'
import * as React from 'react'
import { Subject, Subscription, merge, of } from 'rxjs'
import {
    debounceTime,
    distinctUntilChanged,
    filter,
    repeat,
    startWith,
    takeUntil,
    switchMap,
    map,
    catchError,
    delay,
    share,
} from 'rxjs/operators'
import { eventLogger } from '../../tracking/eventLogger'
import { scrollIntoView } from '../../util'
import { Suggestion, SuggestionItem, createSuggestion, fuzzySearchFilters } from './Suggestion'
import { PatternTypeProps, CaseSensitivityProps, InteractiveSearchProps, CopyQueryButtonProps } from '..'
import Downshift from 'downshift'
import { searchFilterSuggestions } from '../searchFilterSuggestions'
import {
    QueryState,
    filterStaticSuggestions,
    insertSuggestionInQuery,
    isFuzzyWordSearch,
    validFilterAndValueBeforeCursor,
    formatQueryForFuzzySearch,
} from '../helpers'
import { fetchSuggestions } from '../backend'
import { isDefined } from '../../../../shared/src/util/types'
import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { once } from 'lodash'
import { dedupeWhitespace } from '../../../../shared/src/util/strings'
import { FilterType } from '../../../../shared/src/search/interactive/util'
import { isSettingsValid, SettingsCascadeProps } from '../../../../shared/src/settings/settings'
import { Toggles } from './toggles/Toggles'
import { VersionContextProps } from '../../../../shared/src/search/util'
import { Shortcut } from '@slimsag/react-shortcuts'
import { KeyboardShortcut } from '../../../../shared/src/keyboardShortcuts'
import { SearchSuggestion } from '../../../../shared/src/search/suggestions'

interface Props
    extends PatternTypeProps,
        CaseSensitivityProps,
        SettingsCascadeProps,
        CopyQueryButtonProps,
        Partial<Pick<InteractiveSearchProps, 'filtersInQuery'>>,
        VersionContextProps {
    location: H.Location
    history: H.History

    /** The value of the query input */
    value: QueryState

    /** Called when the value changes */
    onChange: (newValue: QueryState) => void

    /**
     * A string that is appended to the query input's query before
     * fetching suggestions.
     */
    prependQueryForSuggestions?: string

    /** Whether the input should be autofocused (and the behavior thereof) */
    autoFocus?: boolean | 'cursor-at-end'

    /** The input placeholder, if different from the default is desired. */
    placeholder?: string

    /**
     * Whether this input should behave like the global query input: (1)
     * pressing the '/' key focuses it and (2) other components contribute a
     * query to it with their context (such as the repository area contributing
     * 'repo:foo@bar' for the current repository and revision).
     *
     * At most one query input per page should have this behavior.
     */
    hasGlobalQueryBehavior?: boolean

    /**
     * Whether to display the query input without any suggestions.
     */
    withoutSuggestions?: boolean

    /**
     * Whether the search mode toggle is attached. Used for styling.
     */
    withSearchModeToggle?: boolean

    /** Keyboard shortcut to focus the query input. */
    keyboardShortcutForFocus?: KeyboardShortcut

    /** Whether globbing is enabled for filters. */
    globbing: boolean
}

/**
 * The search suggestions and cursor position of where the last character was inserted.
 * Cursor position is used to correctly insert the suggestion when it's selected.
 */
export interface ComponentSuggestions {
    values: Suggestion[]
    cursorPosition: number
}

interface State {
    /** Only show suggestions if search input is focused */
    showSuggestions: boolean
    /** Indicates if suggestions are being loaded from the back-end */
    loadingSuggestions?: boolean
    /** The suggestions shown to the user */
    suggestions: ComponentSuggestions
}

export const noSuggestions: State['suggestions'] = { values: [], cursorPosition: 0 }

// Used for fetching suggestions and updating query history (undo/redo)
export const typingDebounceTime = 300

export class QueryInput extends React.Component<Props, State> {
    private componentUpdates = new Subject<Props>()

    /** Subscriptions to unsubscribe from on component unmount */
    private subscriptions = new Subscription()

    /** Emits new input values */
    private inputValues = new Subject<QueryState>()

    /** Emits when the suggestions are hidden */
    private suggestionsHidden = new Subject<void>()

    /** Only used for selection and focus management */
    private inputElement = React.createRef<HTMLInputElement>()

    /** Used for scrolling suggestions into view while scrolling with keyboard */
    private containerElement = React.createRef<HTMLDivElement>()

    public state: State = {
        showSuggestions: false,
        loadingSuggestions: false,
        suggestions: {
            cursorPosition: 0,
            values: [],
        },
    }

    constructor(props: Props) {
        super(props)

        // Update parent component
        // (will be used in next PR to push to queryHistory (undo/redo))
        this.subscriptions.add(this.inputValues.subscribe(queryState => this.props.onChange(queryState)))

        const hideSuggestionsSetting =
            isSettingsValid(props.settingsCascade) &&
            props.settingsCascade.final &&
            props.settingsCascade.final['search.hideSuggestions']

        if (!hideSuggestionsSetting && !this.props.withoutSuggestions) {
            // Show suggestions on input change.
            this.subscriptions.add(
                this.inputValues.subscribe(() => {
                    if (!this.state.showSuggestions) {
                        this.setState({ showSuggestions: true })
                    }
                })
            )
            // Trigger suggestions.
            // This is set on componentDidUpdate so the data flow can be easier to manage, making it
            // only depend on props.value updates, and not both from props.value and this.inputValues
            this.subscriptions.add(
                this.componentUpdates
                    .pipe(
                        debounceTime(typingDebounceTime),
                        // Only show suggestions for when the user has typed (explicitly changed the query).
                        // Also: Prevents suggestions from showing on page load because of componentUpdates.
                        filter(props => !!props.value.fromUserInput),
                        distinctUntilChanged(
                            (previous, current) =>
                                dedupeWhitespace(previous.value.query) === dedupeWhitespace(current.value.query)
                        ),
                        switchMap(({ value: queryState }) => {
                            if (queryState.query.length === 0) {
                                return [{ suggestions: noSuggestions }]
                            }

                            // A filter value (in "archive:yes", "archive" is the filter and "yes" is the value)
                            // can either be from `searchFilterSuggestions` or from the fuzzy-search.

                            // First get static suggestions
                            const staticSuggestions = {
                                cursorPosition: queryState.cursorPosition,
                                values: filterStaticSuggestions(queryState, searchFilterSuggestions),
                            }

                            // Used to know if a filter value, and not just a separate word, is being typed
                            const filterAndValueBeforeCursor = validFilterAndValueBeforeCursor(queryState)

                            // If a filter value is being typed but selected filter does not use
                            // fuzzy-search suggestions, then return only static suggestions
                            if (
                                filterAndValueBeforeCursor &&
                                !fuzzySearchFilters.includes(filterAndValueBeforeCursor.resolvedFilterType)
                            ) {
                                return [{ suggestions: staticSuggestions }]
                            }

                            // Because of API limitations, we need to modify the query before the request,
                            // see definition of `formatQueryForFuzzySearch`
                            const queryForFuzzySearch = formatQueryForFuzzySearch(queryState)
                            const fullQuery = this.props.prependQueryForSuggestions
                                ? this.props.prependQueryForSuggestions + ' ' + queryForFuzzySearch
                                : queryForFuzzySearch

                            const fuzzySearchSuggestions = fetchSuggestions(fullQuery).pipe(
                                map((suggestions): Suggestion[] =>
                                    suggestions
                                        .map((item: SearchSuggestion) => createSuggestion(item, this.props.globbing))
                                        .filter(isDefined)
                                        .map((suggestion): Suggestion => ({ ...suggestion, fromFuzzySearch: true }))
                                        .filter(suggestion => {
                                            // Only show fuzzy-suggestions that are relevant to the typed filter
                                            if (filterAndValueBeforeCursor?.resolvedFilterType) {
                                                switch (filterAndValueBeforeCursor.resolvedFilterType) {
                                                    case FilterType.repohasfile:
                                                        return suggestion.type === FilterType.file
                                                    default:
                                                        return (
                                                            suggestion.type ===
                                                            filterAndValueBeforeCursor.resolvedFilterType
                                                        )
                                                }
                                            }
                                            return suggestion.type !== FilterType.repogroup
                                        })
                                ),
                                map(suggestions => ({
                                    suggestions: {
                                        cursorPosition: queryState.cursorPosition,
                                        values: staticSuggestions.values.concat(suggestions),
                                    },
                                })),
                                catchError(error => {
                                    console.error(error)
                                    // If fuzzy-search is not capable of returning suggestions for the query
                                    // or there is an internal error, then at least return the static suggestions
                                    return [{ suggestions: staticSuggestions }]
                                }),
                                map(state => ({
                                    ...state,
                                    loadingSuggestions: false,
                                })),
                                share()
                            )

                            // Prevent jitter when no static suggestions are found but fuzzy-suggestions are.
                            // Jitter being the suggestions list going blank unnecessarily during update.
                            // (This is a fix for 3.10 release, and will be improved on next PR)
                            const currentSuggestions = {
                                ...staticSuggestions,
                                values: staticSuggestions.values.concat(
                                    this.state.suggestions.values.filter(({ fromFuzzySearch }) => fromFuzzySearch)
                                ),
                            }

                            return merge(
                                // Render static suggestions first
                                [{ suggestions: currentSuggestions }],
                                // Prevent loading indicator jitter, only showing it after 1s delay
                                of({ suggestions: staticSuggestions, loadingSuggestions: true }).pipe(
                                    delay(1000),
                                    takeUntil(fuzzySearchSuggestions)
                                ),
                                // Fetch and format fuzzy-search suggestions
                                fuzzySearchSuggestions
                            )
                        }),
                        // Abort suggestion display on route change or suggestion hiding
                        takeUntil(this.suggestionsHidden),
                        // But resubscribe afterwards
                        repeat()
                    )
                    .subscribe(
                        state => {
                            this.setState(state)
                        },
                        error => {
                            console.error(error)
                        }
                    )
            )
        }

        if (this.props.hasGlobalQueryBehavior) {
            /** Whenever the URL query has a "focus" property, remove it and focus the query input. */
            this.subscriptions.add(
                this.componentUpdates
                    .pipe(
                        startWith(props),
                        filter(({ location }) => new URLSearchParams(location.search).get('focus') !== null)
                    )
                    .subscribe(props => {
                        this.focusInputAndPositionCursorAtEnd()
                        const searchParameters = new URLSearchParams(props.location.search)
                        searchParameters.delete('focus')
                        this.props.history.replace({ search: searchParameters.toString() })
                    })
            )
        }
    }

    public componentDidMount(): void {
        switch (this.props.autoFocus) {
            case 'cursor-at-end':
                this.focusInputAndPositionCursorAtEnd()
                break
        }
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public componentDidUpdate(previousProps: Props): void {
        if (this.props.value.cursorPosition && previousProps.value.cursorPosition !== this.props.value.cursorPosition) {
            this.focusInputAndPositionCursor(this.props.value.cursorPosition)
        }
        this.componentUpdates.next(this.props)
    }

    public render(): JSX.Element | null {
        const hideSuggestionsSetting =
            isSettingsValid(this.props.settingsCascade) &&
            this.props.settingsCascade.final &&
            this.props.settingsCascade.final['search.hideSuggestions']

        const showSuggestions =
            !this.props.withoutSuggestions &&
            !hideSuggestionsSetting &&
            this.state.showSuggestions &&
            (this.state.suggestions.values.length > 0 || this.state.loadingSuggestions)

        // If last typed word is not a filter type,
        // suggestions should show url label and redirect on select.
        const showUrlLabel = isFuzzyWordSearch({
            query: this.props.value.query,
            cursorPosition: this.state.suggestions.cursorPosition,
        })
        return (
            <>
                <Downshift
                    scrollIntoView={this.downshiftScrollIntoView}
                    onSelect={this.onSuggestionSelect}
                    itemToString={this.downshiftItemToString}
                >
                    {({ getInputProps, getItemProps, getMenuProps, highlightedIndex }) => {
                        const { onChange: downshiftChange, onKeyDown } = getInputProps()
                        return (
                            <div className="query-input2">
                                <div ref={this.containerElement}>
                                    <input
                                        onFocus={this.onInputFocus}
                                        onBlur={this.onInputBlur}
                                        className={`form-control query-input2__input test-query-input ${
                                            this.props.withSearchModeToggle
                                                ? 'query-input2__input-with-mode--toggle'
                                                : 'rounded-left'
                                        }`}
                                        value={this.props.value.query}
                                        autoFocus={this.props.autoFocus === true}
                                        onChange={(event: React.ChangeEvent<HTMLInputElement>) => {
                                            downshiftChange(event)
                                            this.onInputChange(event)
                                        }}
                                        onKeyDown={event => {
                                            this.onInputKeyDown(event)
                                            onKeyDown(event)
                                        }}
                                        spellCheck={false}
                                        autoCapitalize="off"
                                        placeholder={
                                            this.props.placeholder === undefined
                                                ? 'Search code...'
                                                : this.props.placeholder
                                        }
                                        ref={this.inputElement}
                                        name="query"
                                        autoComplete="off"
                                    />
                                    {showSuggestions && (
                                        <ul
                                            className="query-input2__suggestions test-query-suggestions"
                                            {...getMenuProps()}
                                        >
                                            {this.state.suggestions.values.map((suggestion, index) => {
                                                const isSelected = highlightedIndex === index
                                                const key = `${index}-${suggestion.value}`
                                                return (
                                                    <SuggestionItem
                                                        key={key}
                                                        {...getItemProps({
                                                            key,
                                                            index,
                                                            item: suggestion,
                                                        })}
                                                        suggestion={suggestion}
                                                        isSelected={isSelected}
                                                        showUrlLabel={showUrlLabel}
                                                        defaultLabel="add to query"
                                                    />
                                                )
                                            })}
                                            {this.state.loadingSuggestions && (
                                                <li className="suggestion suggestion--selected">
                                                    <LoadingSpinner className="icon-inline" />
                                                    <div className="suggestion__description">Loading</div>
                                                </li>
                                            )}
                                        </ul>
                                    )}
                                    <Toggles
                                        {...this.props}
                                        navbarSearchQuery={this.props.value.query}
                                        className="query-input2__toggle-container"
                                    />
                                </div>
                            </div>
                        )
                    }}
                </Downshift>
                {this.props.keyboardShortcutForFocus?.keybindings.map((keybinding, index) => (
                    <Shortcut key={index} {...keybinding} onMatch={this.focusInputAndPositionCursorAtEnd} />
                ))}
            </>
        )
    }

    private downshiftItemToString = (suggestion?: Suggestion): string => (suggestion ? suggestion.value : '')

    private downshiftScrollIntoView = (node: HTMLElement, menuNode: HTMLElement): void => {
        scrollIntoView(menuNode, node)
    }

    private setShowSuggestions = (showSuggestions: boolean): void => {
        this.setState({ showSuggestions }, () => !showSuggestions && this.suggestionsHidden.next())
    }

    private onInputKeyDown = (event: React.KeyboardEvent<HTMLInputElement>): void => {
        // Ctrl+Space to show all available filter type suggestions
        if (event.ctrlKey && event.key === ' ') {
            this.setState({
                suggestions: {
                    cursorPosition: event.currentTarget.selectionStart ?? 0,
                    values: searchFilterSuggestions.filters.values,
                },
            })
        }
        if (event.key === 'Enter') {
            this.setShowSuggestions(false)
        }
    }

    private onInputBlur = (): void => {
        this.setShowSuggestions(false)
    }

    private onInputFocus = (): void => {
        this.setShowSuggestions(true)
    }

    /**
     * if query only has one word and selected suggestion is not a filter: redirect to suggestion URL
     * else: add selected suggestion to query
     */
    private onSuggestionSelect = (suggestion: Suggestion | undefined): void => {
        this.setState(({ suggestions }, { value, history }) => {
            // If downshift selects an item with value undefined
            if (!suggestion) {
                return { suggestions: noSuggestions }
            }

            // 🚨 PRIVACY: never provide any private data in { code_search: { suggestion: { type } } }.
            eventLogger.log('SearchSuggestionSelected', { type: suggestion.type })

            // if separate word is being typed and suggestion with url is selected
            if (
                isFuzzyWordSearch({
                    query: value.query,
                    cursorPosition: suggestions.cursorPosition,
                }) &&
                suggestion.url
            ) {
                history.push(suggestion.url)
                return { suggestions: noSuggestions }
            }

            this.inputValues.next({
                ...insertSuggestionInQuery(value.query, suggestion, suggestions.cursorPosition),
                fromUserInput: true,
            })

            return { suggestions: noSuggestions }
        })
    }

    private focusInputAndPositionCursor(cursorPosition: number): void {
        if (this.inputElement.current) {
            this.inputElement.current.focus()
            this.inputElement.current.setSelectionRange(cursorPosition, cursorPosition)
        }
    }

    private focusInputAndPositionCursorAtEnd = (): void => {
        if (
            this.inputElement.current &&
            !!document.activeElement &&
            !['INPUT', 'TEXTAREA'].includes(document.activeElement.nodeName)
        ) {
            this.focusInputAndPositionCursor(this.inputElement.current.value.length)
        }
    }

    /** Only log when user has typed the first character into the input. */
    private logFirstInput = once((): void => {
        eventLogger.log('SearchInitiated')
    })

    private onInputChange: React.ChangeEventHandler<HTMLInputElement> = event => {
        this.logFirstInput()
        this.inputValues.next({
            fromUserInput: true,
            query: event.currentTarget.value,
            cursorPosition: event.currentTarget.selectionStart || 0,
        })
    }
}
