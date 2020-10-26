import { DownshiftState, StateChangeOptions } from 'downshift'
import { inRange } from 'lodash'
import * as React from 'react'
import { fromEvent, merge, Subject, Subscription } from 'rxjs'
import { CompletionItem, CompletionList } from 'sourcegraph'
import getCaretCoordinates from 'textarea-caret'
import { Key } from 'ts-key-enum'
import { ErrorLike, isErrorLike } from '../../util/errors'
import { CompletionWidgetDropdown } from './CompletionWidgetDropdown'

export const LOADING = 'loading' as const
export type CompletionResult = typeof LOADING | ErrorLike | CompletionList | null

function isSuccessfulFetch(result: CompletionResult): result is CompletionList {
    return result !== LOADING && !isErrorLike(result)
}

// In order to handle keyboard events correctly, we need to explicitly manage/control some of the
// state of Downshift since we have no control over the underlying TextArea element.
//
// (See https://github.com/paypal/downshift#control-props for more information.)
export type ManagedDownShiftState = Pick<DownshiftState<CompletionItem>, 'highlightedIndex' | 'selectedItem'>

interface State extends ManagedDownShiftState {
    /**
     * Whether or not the user hid the dropdown by clicking outside of it.
     */
    hidden: boolean
}

/** CSS classes for the completion widget styling. */
export interface CompletionWidgetClassProps {
    widgetClassName?: string
    widgetContainerClassName?: string
    listClassName?: string
    listItemClassName?: string
    selectedListItemClassName?: string
    loadingClassName?: string
    noResultsClassName?: string
}

export interface CompletionWidgetProps extends CompletionWidgetClassProps {
    /**
     * A reference to the text box DOM node that this autocomplete instance is watching.
     */
    textArea: HTMLTextAreaElement

    /**
     * The current state of the completion request, or undefined if there is none.
     */
    completionListOrError: CompletionResult

    /**
     * Called when a completion item is selected.
     */
    onSelectItem: (item: CompletionItem) => void
}

export class CompletionWidget extends React.Component<CompletionWidgetProps, State> {
    public state: State = {
        hidden: false,

        // managed state for Downshift
        highlightedIndex: 0, // defaults to '0' so that the first element in the list is selected by default
        selectedItem: null,
    }

    private itemSelections = new Subject<CompletionItem>()
    private subscriptions = new Subscription()

    public componentDidMount(): void {
        this.subscriptions.add(
            fromEvent<KeyboardEvent>(this.props.textArea, 'keydown').subscribe(event =>
                this.handleKeyboardNavigationEvents(event)
            )
        )

        // Hide the widget when the user selects an item.
        this.subscriptions.add(
            this.itemSelections.subscribe(item => {
                this.props.onSelectItem(item)
                this.setState({ hidden: true })
            })
        )
        // Unhide whenever the user types something.
        this.subscriptions.add(
            merge(
                fromEvent<KeyboardEvent>(this.props.textArea, 'keypress'),
                fromEvent<KeyboardEvent>(this.props.textArea, 'input')
            ).subscribe(() => this.setState({ hidden: false }))
        )
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    private onItemSelected = (item: CompletionItem): void => {
        this.itemSelections.next(item)
    }

    private handleKeyboardNavigationEvents(event: KeyboardEvent): void {
        let handled = false

        switch (event.key) {
            case Key.ArrowDown:
                handled = this.scrollDown()
                break
            case Key.ArrowUp:
                handled = this.scrollUp()
                break
            case Key.Enter:
            case Key.Tab:
                handled = this.selectHighlightedItem()
                break
            case Key.Escape:
                this.hideDropdown()
                break
        }

        if (handled) {
            event.preventDefault()
        }
    }

    /**
     * Scrolls the list in the dropdown down by one entry, wrapping around if necessary.
     *
     * Returns 'true' iff the list was actually scrolled (after checking to see if the current state is valid).
     */
    private scrollDown(): boolean {
        return this.scrollDelta(1)
    }

    /**
     * Scrolls the list in the dropdown up by one entry, wrapping around if necessary.
     *
     * Returns 'true' iff the list was actually scrolled (after checking to see if the current state is valid).
     */
    private scrollUp(): boolean {
        return this.scrollDelta(-1)
    }

    /**
     * Scrolls the list in the dropdown by 'delta' entries, wrapping around if necessary.
     *
     * Returns 'true' iff the list was actually scrolled (after checking to see if the current state is valid).
     */
    private scrollDelta(delta: number): boolean {
        const { completionListOrError } = this.props
        const { hidden } = this.state

        if (hidden || !completionListOrError || !isSuccessfulFetch(completionListOrError)) {
            // There is nothing to scroll to if there aren't any results.
            return false
        }

        const oldIndex = this.state.highlightedIndex || 0
        const newIndex = oldIndex + delta

        // Wrap around to the other end of the list if necessary.
        const wrappedIndex =
            ((newIndex % completionListOrError.items.length) + completionListOrError.items.length) %
            completionListOrError.items.length

        this.setState({ highlightedIndex: wrappedIndex })
        return true
    }

    /**
     * Selects the currently highlighted entry in the dropdown.
     *
     * Returns 'true' iff an entry was actually selected (after checking to see if the current state is valid).
     */
    private selectHighlightedItem(): boolean {
        const { completionListOrError } = this.props
        const { hidden } = this.state

        if (hidden || !completionListOrError || !isSuccessfulFetch(completionListOrError)) {
            // There is nothing to select if there aren't symbol results for the query.
            return false
        }

        const currentIndex = this.state.highlightedIndex || 0

        if (!inRange(currentIndex, 0, completionListOrError.items.length)) {
            // There is nothing to select if the index is outside the indicies of
            // the fetched symbol results
            return false
        }

        const selectedItem = completionListOrError.items[currentIndex]
        this.setState({ selectedItem })
        this.onItemSelected(selectedItem)

        return true
    }

    private hideDropdown = (): void => {
        this.setState({ hidden: true })
    }

    /**
     * Updates our copy of Downshift's state whenever Downshift itself changes it
     */
    private onDownshiftStateChange = ({ highlightedIndex, selectedItem }: StateChangeOptions<CompletionItem>): void => {
        if (highlightedIndex !== undefined) {
            this.setState({ highlightedIndex })
        }

        if (selectedItem !== undefined) {
            this.setState({ selectedItem })
        }
    }

    public render(): JSX.Element | null {
        const { completionListOrError } = this.props
        const { hidden, highlightedIndex, selectedItem } = this.state

        if (hidden || !completionListOrError) {
            return null
        }

        const caretCoordinates = getCaretCoordinates(this.props.textArea, this.props.textArea.selectionStart)

        // Support textareas that scroll vertically. Fixes
        // https://github.com/sourcegraph/sourcegraph/issues/3424.
        caretCoordinates.top -= this.props.textArea.scrollTop

        return (
            <div className={`completion-widget ${this.props.widgetClassName || ''}`}>
                <div
                    // eslint-disable-next-line react/forbid-dom-props
                    style={{ left: caretCoordinates.left, top: caretCoordinates.top }}
                    className={`completion-widget__container ${this.props.widgetContainerClassName || ''}`}
                >
                    <CompletionWidgetDropdown
                        {...this.props}
                        completionListOrError={completionListOrError}
                        onItemSelected={this.onItemSelected}
                        onClickOutside={this.hideDropdown}
                        onDownshiftStateChange={this.onDownshiftStateChange}
                        highlightedIndex={highlightedIndex}
                        selectedItem={selectedItem}
                    />
                </div>
            </div>
        )
    }
}
