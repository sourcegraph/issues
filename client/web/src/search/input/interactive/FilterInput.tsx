import * as React from 'react'
import { Form } from '../../../../../branded/src/components/Form'
import CloseIcon from 'mdi-react/CloseIcon'
import PlusIcon from 'mdi-react/PlusIcon'
import MinusIcon from 'mdi-react/MinusIcon'
import { Subscription, Subject, merge, of } from 'rxjs'
import {
    distinctUntilChanged,
    switchMap,
    map,
    catchError,
    debounceTime,
    takeUntil,
    repeat,
    share,
    delay,
} from 'rxjs/operators'
import { createSuggestion, Suggestion, SuggestionItem, fuzzySearchFilters } from '../Suggestion'
import { fetchSuggestions } from '../../backend'
import { ComponentSuggestions, noSuggestions, typingDebounceTime } from '../QueryInput'
import { isDefined } from '../../../../../shared/src/util/types'
import Downshift from 'downshift'
import {
    QueryState,
    formatInteractiveQueryForFuzzySearch,
    validFilterAndValueBeforeCursor,
    filterStaticSuggestions,
} from '../../helpers'
import { dedupeWhitespace, isQuoted } from '../../../../../shared/src/util/strings'
import { FilterType, isNegatableFilter } from '../../../../../shared/src/search/interactive/util'
import { startCase, isEqual } from 'lodash'
import { searchFilterSuggestions } from '../../searchFilterSuggestions'
import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { CheckButton } from './CheckButton'
import { isTextFilter, finiteFilters, isFiniteFilter, FilterTypeToProseNames } from './filters'
import classNames from 'classnames'
import { generateFiltersQuery } from '../../../../../shared/src/util/url'
import { InteractiveSearchProps } from '../..'

interface Props extends Pick<InteractiveSearchProps, 'filtersInQuery'> {
    /**
     * The query in the main query input.
     */
    navbarQuery: QueryState

    /**
     * The key representing this filter in the top-level filtersInQuery map.
     */
    mapKey: string

    /**
     * The value for this filter.
     */
    value: string

    /**
     * The search filter type, as available in {@link SuggstionTypes}
     */
    filterType: Exclude<FilterType, FilterType.patterntype>

    /**
     * Whether or not this FilterInput is currently editable.
     *
     * This is passed as a prop rather than being a state field because
     * this component is unaware whether to render as editable or uneditable
     * on initial mount.
     */
    editable: boolean

    /**
     * Whether the current filter is negated
     */
    negated?: boolean

    /**
     * Whether globbing is enabled for filters.
     */
    globbing: boolean

    /**
     * Callback that handles a filter input being submitted. Triggers a search
     * with the new query value.
     */
    onSubmit: (event: React.FormEvent<HTMLFormElement>) => void

    /**
     * Callback to handle the filter's value being updated.
     */
    onFilterEdited: (filterKey: string, value: string) => void

    /**
     * Callback to handle the filter chip being deleted.
     */
    onFilterDeleted: (filterKey: string) => void

    /**
     * Callback to handle the editable state of this filter.
     */
    toggleFilterEditable: (filterKey: string) => void

    toggleFilterNegated: (filterKey: string) => void
}

const LOADING = 'loading' as const

interface State {
    /** Only show suggestions if search input is focused */
    showSuggestions: boolean
    /** The suggestions shown to the user */
    suggestions: typeof LOADING | ComponentSuggestions

    /** The value being typed in the filter's input*/
    inputValue: string
}

/**
 * The filter chips for each filter added to the query in interactive mode. The filter input can be either editable or non-editable.
 * If it's in an editable state, it consists of a text input field, with suggestions, and a close button. Otherwise, it's a simple
 * button with a close button.
 */
export class FilterInput extends React.Component<Props, State> {
    private subscriptions = new Subscription()
    private inputValues = new Subject<string>()
    private inputEl = React.createRef<HTMLInputElement>()
    /** Emits when the suggestions are hidden */
    private suggestionsHidden = new Subject<void>()
    /** Emits on mount to set the default value for finite-option filters */
    private setFiniteFilterDefault = new Subject<void>()

    constructor(props: Props) {
        super(props)

        this.state = {
            suggestions: noSuggestions,
            showSuggestions: false,
            inputValue: props.value || '',
        }

        this.subscriptions.add(this.inputValues.subscribe(query => this.setState({ inputValue: query })))

        this.subscriptions.add(
            this.inputValues
                .pipe(
                    debounceTime(typingDebounceTime),
                    distinctUntilChanged(
                        (previous, current) => dedupeWhitespace(previous) === dedupeWhitespace(current)
                    ),
                    switchMap(inputValue => {
                        if (inputValue.length === 0) {
                            return [{ suggestions: noSuggestions }]
                        }
                        const filterType = props.filterType
                        const newFiltersQuery = { ...props.filtersInQuery }
                        if (newFiltersQuery[props.mapKey]) {
                            newFiltersQuery[props.mapKey].value = inputValue
                        } else {
                            newFiltersQuery[props.mapKey] = {
                                type: props.filterType,
                                value: inputValue,
                                editable: true,
                            }
                        }
                        const filterAndValue = `${filterType}:${inputValue}`
                        const filterAndValueBeforeCursor = validFilterAndValueBeforeCursor({
                            query: `${filterType}:${inputValue}`,
                            cursorPosition: filterAndValue.length,
                        })

                        // First get static suggestions
                        const staticSuggestions = {
                            cursorPosition: filterAndValue.length,
                            values: filterStaticSuggestions(
                                {
                                    query: `${filterType}:${inputValue}`,
                                    cursorPosition: filterAndValue.length,
                                },
                                searchFilterSuggestions
                            ),
                        }

                        // If a filter value is being typed but selected filter does not use
                        // fuzzy-search suggestions, then return only static suggestions
                        if (
                            filterAndValueBeforeCursor &&
                            !fuzzySearchFilters.includes(filterAndValueBeforeCursor.resolvedFilterType)
                        ) {
                            return [{ suggestions: staticSuggestions }]
                        }

                        let fullQuery = `${props.navbarQuery.query} ${generateFiltersQuery(newFiltersQuery)}`

                        fullQuery = formatInteractiveQueryForFuzzySearch(fullQuery, filterType, inputValue)
                        const suggestions = fetchSuggestions(fullQuery).pipe(
                            map((suggestions): Suggestion[] =>
                                suggestions
                                    .map(item => createSuggestion(item, props.globbing))
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
                                        return true
                                    })
                            ),
                            map(suggestions => ({
                                suggestions: {
                                    values: suggestions,
                                    cursorPosition: fullQuery.length,
                                },
                            })),
                            catchError(error => {
                                console.error(error)
                                return [{ suggestions: noSuggestions }]
                            }),
                            share()
                        )

                        return merge(
                            of({ suggestions: LOADING }).pipe(delay(1000), takeUntil(suggestions)),
                            suggestions
                        )
                    }),
                    takeUntil(this.suggestionsHidden),
                    repeat()
                )
                .subscribe(state => this.setState({ ...state, showSuggestions: true }))
        )

        if (isFiniteFilter(this.props.filterType)) {
            this.subscriptions.add(
                this.setFiniteFilterDefault.subscribe(() => {
                    this.setState(({ inputValue }) => ({
                        inputValue:
                            isFiniteFilter(this.props.filterType) && inputValue === ''
                                ? finiteFilters[this.props.filterType].default
                                : inputValue,
                    }))
                })
            )
        }
    }

    public componentDidMount(): void {
        if (this.inputEl.current) {
            this.inputEl.current.focus()
        }

        this.setFiniteFilterDefault.next()
    }

    public componentDidUpdate(previousProps: Props): void {
        if (isFiniteFilter(this.props.filterType) && this.props.value !== previousProps.value) {
            this.inputValues.next(this.props.value)
        }
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    private onInputUpdate: React.ChangeEventHandler<HTMLInputElement> = event => {
        this.inputValues.next(event.target.value)
    }

    private onSubmitInput = (event: React.FormEvent<HTMLFormElement>): void => {
        event.preventDefault()
        event.stopPropagation()

        if (this.state.inputValue !== '' || this.props.filterType === FilterType.type) {
            // Don't allow empty filters, unless it's the type filter.
            let inputValue = this.state.inputValue

            // Filters should always be quoted and escaped before being sent to the backend.
            inputValue = JSON.stringify(inputValue)

            this.inputValues.next(inputValue)
            // Update the top-level filtersInQueryMap with the new value for this filter.
            this.props.onFilterEdited(this.props.mapKey, inputValue)
        }
    }

    private onSuggestionSelect = (suggestion: Suggestion | undefined): void => {
        // Insert value into filter input. For any suggestion selected, the whole value should be updated,
        // not just appended.
        if (suggestion) {
            this.inputValues.next(suggestion.value)
        }

        this.setState({ suggestions: noSuggestions, showSuggestions: false }, () => this.suggestionsHidden.next())
    }

    /**
     * Handles clicking a filter chip in an uneditable state. Makes the filter editable
     * and focuses the input field.
     */
    private onClickFilterChip = (): void => {
        if (this.inputEl.current) {
            this.inputEl.current.focus()
        }
        // Filters are always quoted and escaped, but we don't display the quoted and escaped value
        // to the user when editing. This makes queries easier to edit and makes sure URLs are always
        // properly escaped.
        const { inputValue } = this.state
        // Check for isQuoted before parsing to support old URLs that don't have quotes around filters.
        this.inputValues.next(isQuoted(inputValue) ? JSON.parse(inputValue) : inputValue)
        this.props.toggleFilterEditable(this.props.mapKey)
    }

    /** Handles clicking the delete button on an uneditable filter chip. */
    private onClickDelete = (): void => {
        this.props.onFilterDeleted(this.props.mapKey)
    }

    /**
     * Checks that the newly focused element is not a child of the previously focused element.
     * Prevents onBlur from firing if we are clicking inside the filter input chip.
     */
    private focusInCurrentTarget = (event: React.FocusEvent<HTMLDivElement>): boolean => {
        const { relatedTarget, currentTarget } = event

        if (relatedTarget === null) {
            return false
        }
        const node = relatedTarget as HTMLElement
        return currentTarget.contains(node) || isEqual(currentTarget, node)
    }

    /**
     * Handles the input field in the filter input becoming unfocused.
     */
    private onInputBlur: React.FocusEventHandler<HTMLDivElement> = event => {
        const focusIsNotChildElement = this.focusInCurrentTarget(event)
        if (focusIsNotChildElement) {
            return
        }

        this.handleDiscard()
    }

    /**
     * Handles discarding while editing a filter.
     *
     * If the filter had no value, and no new value was submitted, the filter is deleted.
     * If the filter had an old value, and no new value was submitted, the inputValue is reverted
     * to the initial value, and the filter becomes uneditable.
     * Any suggestions get hidden.
     */
    private handleDiscard = (): void => {
        if (this.props.value === '') {
            // Don't allow empty filters
            this.onClickDelete()
            return
        }

        this.props.toggleFilterEditable(this.props.mapKey)

        // Revert to the last locked-in value for this filter, since the user didn't submit their new value.
        this.setState({ suggestions: noSuggestions, inputValue: this.props.value })
    }

    private onInputKeyDown = (event: React.KeyboardEvent<HTMLInputElement>): void => {
        if (event.ctrlKey && event.key === ' ') {
            this.setState({
                suggestions: {
                    cursorPosition: event.currentTarget.selectionStart ?? 0,
                    values: searchFilterSuggestions[this.props.filterType].values,
                },
                showSuggestions: true,
            })
        }

        // Escape to cancel editing a filter
        if (event.key === 'Escape' && this.props.editable) {
            this.handleDiscard()
        }
    }

    private toggleNegation = (): void => {
        this.props.toggleFilterNegated(this.props.mapKey)
    }

    private downshiftItemToString = (suggestion?: Suggestion): string => (suggestion ? suggestion.value : '')

    public render(): JSX.Element | null {
        const isText = isTextFilter(this.props.filterType)
        const isEditableAndText = this.props.editable && isText
        return (
            <div
                className={`${classNames('filter-input', `test-filter-input-${this.props.mapKey}`, {
                    'filter-input--active': isEditableAndText,
                })}`}
                onBlur={this.onInputBlur}
                tabIndex={0}
            >
                {this.props.editable
                    ? isText
                        ? this.renderTextFilterForm()
                        : this.renderFiniteFilterForm()
                    : this.renderUneditableFilter()}
            </div>
        )
    }

    private renderTextFilterForm(): JSX.Element {
        const suggestionsAreLoading = this.state.suggestions === LOADING
        const showSuggestions =
            this.state.showSuggestions &&
            ((this.state.suggestions !== LOADING && this.state.suggestions.values.length > 0) || suggestionsAreLoading)

        return (
            <Form onSubmit={this.onSubmitInput}>
                <Downshift onSelect={this.onSuggestionSelect} itemToString={this.downshiftItemToString}>
                    {({ getInputProps, getItemProps, getMenuProps, highlightedIndex }) => {
                        const { onKeyDown } = getInputProps()
                        return (
                            <div>
                                <div className="filter-input__form">
                                    {isNegatableFilter(this.props.filterType) && this.props.editable && (
                                        <button
                                            type="button"
                                            className={classNames('btn', 'btn-icon', 'filter-input__negation-button', {
                                                'filter-input__negation-button--active': this.props.negated,
                                            })}
                                            onClick={this.toggleNegation}
                                            data-tooltip={this.props.negated ? 'Include results' : 'Exclude results'}
                                        >
                                            {this.props.negated ? <MinusIcon size={16} /> : <PlusIcon size={16} />}
                                        </button>
                                    )}
                                    <span
                                        tabIndex={0}
                                        className="filter-input__label"
                                    >{`${this.props.filterType}:`}</span>
                                    <div className="filter-input__input-wrapper">
                                        <input
                                            ref={this.inputEl}
                                            className={`form-control form-control-sm filter-input__input-field test-filter-input__input-field-${this.props.mapKey}`}
                                            value={this.state.inputValue}
                                            onChange={this.onInputUpdate}
                                            onKeyDown={event => {
                                                this.onInputKeyDown(event)
                                                onKeyDown(event)
                                            }}
                                            autoFocus={true}
                                        />
                                        {showSuggestions && (
                                            <ul
                                                className="filter-input__suggestions test-filter-input__suggestions"
                                                {...getMenuProps()}
                                            >
                                                {this.state.suggestions === LOADING ? (
                                                    <li className="suggestion suggestion--selected">
                                                        <LoadingSpinner className="icon-inline" />
                                                        <div className="suggestion__description">Loading</div>
                                                    </li>
                                                ) : (
                                                    this.state.suggestions.values.map((suggestion, index) => {
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
                                                                showUrlLabel={false}
                                                            />
                                                        )
                                                    })
                                                )}
                                            </ul>
                                        )}
                                    </div>
                                    <CheckButton className="check-button__btn" />
                                    <button
                                        type="button"
                                        onClick={this.handleDiscard}
                                        className={`btn btn-icon icon-inline test-filter-input__cancel-button-${this.props.mapKey}`}
                                        aria-label="Cancel"
                                        data-tooltip="Cancel"
                                    >
                                        <CloseIcon />
                                    </button>
                                </div>
                            </div>
                        )
                    }}
                </Downshift>
            </Form>
        )
    }

    private renderFiniteFilterForm(): JSX.Element {
        return (
            <Form onSubmit={this.onSubmitInput}>
                <div className="filter-input__form test-filter-input-finite-form">
                    <div className="filter-input__radio-button-container">
                        <span>{`${FilterTypeToProseNames[this.props.filterType]}:`}</span>
                        {isFiniteFilter(this.props.filterType) &&
                            finiteFilters[this.props.filterType].values.map(value => (
                                <div key={value.value} className="filter-input__radio">
                                    <input
                                        type="radio"
                                        className={`test-filter-input-radio-button-${value.value}`}
                                        id={value.value}
                                        name={value.value}
                                        onChange={() => this.setState({ inputValue: value.value })}
                                        checked={this.state.inputValue === value.value}
                                        autoFocus={true}
                                    />
                                    <label htmlFor={value.value} tabIndex={0} className="filter-input__radio-label">
                                        {value.displayValue ? startCase(value.displayValue) : startCase(value.value)}
                                    </label>
                                </div>
                            ))}
                    </div>
                    <CheckButton />
                    <button
                        type="button"
                        onClick={this.handleDiscard}
                        className={`btn btn-icon icon-inline test-filter-input__cancel-button-${this.props.mapKey}`}
                        aria-label="Cancel"
                        data-tooltip="Cancel"
                    >
                        <CloseIcon />
                    </button>
                </div>
            </Form>
        )
    }

    private renderUneditableFilter(): JSX.Element {
        return (
            <div className="filter-input--uneditable d-flex">
                <button
                    type="button"
                    className={`filter-input__button-text btn text-nowrap test-filter-input__button-text-${this.props.mapKey}`}
                    onClick={this.onClickFilterChip}
                    data-tooltip="Edit filter"
                    aria-label="Edit filter"
                    tabIndex={0}
                >
                    {this.props.negated ? '-' : ''}
                    {this.props.filterType}:{this.state.inputValue}
                </button>
                <button
                    type="button"
                    onClick={this.onClickDelete}
                    className="btn btn-icon icon-inline test-filter-input__delete-button"
                    aria-label="Delete filter"
                    data-tooltip="Delete filter"
                >
                    <CloseIcon />
                </button>
            </div>
        )
    }
}
