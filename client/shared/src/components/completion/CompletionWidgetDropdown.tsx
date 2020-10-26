import Downshift, { StateChangeOptions } from 'downshift'
import React from 'react'
import { CompletionItem } from 'sourcegraph'
import { isErrorLike } from '../../util/errors'
import { CompletionResult, CompletionWidgetClassProps, LOADING, ManagedDownShiftState } from './CompletionWidget'

interface CompletionWidgetDropdownProps
    extends ManagedDownShiftState,
        Pick<
            CompletionWidgetClassProps,
            Exclude<keyof CompletionWidgetClassProps, 'widgetClassName' | 'widgetContainerClassName'>
        > {
    /**
     * The completion results to render inside the dropdown.
     */
    completionListOrError: CompletionResult

    /**
     * Callback to run whenever the user selects an item in the dropdown.
     */
    onItemSelected: (selectedItem: CompletionItem) => void

    /**
     * Callback to run whenever the user clicks outside the dropdown component.
     */
    onClickOutside: () => void

    /**
     * Callback to run whenever the state of the Downshift component changes. This is needed
     * to manage keyboard events.
     *
     * (See https://github.com/paypal/downshift#control-props for more information.)
     */
    onDownshiftStateChange: (options: StateChangeOptions<CompletionItem>) => void
}

function completionItemToString(item: CompletionItem | null): string {
    return item ? item.label : ''
}

export const CompletionWidgetDropdown: React.FunctionComponent<CompletionWidgetDropdownProps> = ({
    completionListOrError,

    onItemSelected,
    onClickOutside,
    onDownshiftStateChange,

    highlightedIndex,
    selectedItem,

    listClassName = '',
    listItemClassName = '',
    selectedListItemClassName = '',
    loadingClassName = '',
    noResultsClassName = '',
}: CompletionWidgetDropdownProps) => (
    <Downshift
        initialHighlightedIndex={0}
        isOpen={true}
        itemToString={completionItemToString}
        onChange={onItemSelected}
        onOuterClick={onClickOutside}
        onStateChange={onDownshiftStateChange}
        highlightedIndex={highlightedIndex}
        selectedItem={selectedItem}
    >
        {({ getItemProps, highlightedIndex }) =>
            isErrorLike(completionListOrError) ? null : (
                <ul className={`completion-widget-dropdown ${listClassName}`}>
                    {completionListOrError === LOADING ? (
                        <li className={loadingClassName}>Loading ...</li>
                    ) : !completionListOrError || completionListOrError.items.length === 0 ? (
                        <li className={noResultsClassName}>No results.</li>
                    ) : (
                        completionListOrError.items.map((item, index) => (
                            // eslint-disable-next-line react/jsx-key
                            <li
                                {...getItemProps({
                                    key: index,
                                    index,
                                    item,
                                    className: `${
                                        highlightedIndex === index ? selectedListItemClassName : ''
                                    } ${listItemClassName}`,
                                })}
                            >
                                <div className="completion-widget-dropdown__item-text">
                                    {item.description ? (
                                        <>
                                            <strong>{item.label}</strong>&nbsp; {item.description}
                                        </>
                                    ) : (
                                        item.label
                                    )}
                                </div>
                            </li>
                        ))
                    )}
                </ul>
            )
        }
    </Downshift>
)
