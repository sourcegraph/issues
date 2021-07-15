import classNames from 'classnames'
import React, { useCallback } from 'react'

import { Form } from '@sourcegraph/branded/src/components/Form'

import { FilterControl, FilteredConnectionFilter, FilteredConnectionFilterValue } from '../FilterControl'

export interface ConnectionFormProps {
    /** Hides the filter input field. */
    hideSearch?: boolean

    /** CSS class name for the <input> element */
    inputClassName?: string

    /** Placeholder text for the <input> element */
    inputPlaceholder?: string

    query: string

    onChange: React.ChangeEventHandler<HTMLInputElement>

    /** Autofocuses the filter input field. */
    autoFocus?: boolean

    /**
     * Filters to display next to the filter input field.
     *
     * Filters are mutually exclusive.
     */
    filters?: FilteredConnectionFilter[]

    onDidSelectValue?: (filter: FilteredConnectionFilter, value: FilteredConnectionFilterValue) => void

    /** An element rendered as a sibling of the filters. */
    additionalFilterElement?: React.ReactElement

    values?: Map<string, FilteredConnectionFilterValue>
}

export const ConnectionForm = React.forwardRef<HTMLInputElement, ConnectionFormProps>(
    (
        {
            hideSearch,
            inputClassName,
            inputPlaceholder,
            query,
            onChange,
            autoFocus,
            filters,
            onDidSelectValue,
            additionalFilterElement,
            values,
        },
        reference
    ) => {
        const handleSubmit = useCallback<React.FormEventHandler<HTMLFormElement>>(event => {
            // Do nothing. The <input onChange> handler will pick up any changes shortly.
            event.preventDefault()
        }, [])

        return (
            <Form
                className="w-100 d-inline-flex justify-content-between flex-row filtered-connection__form"
                onSubmit={handleSubmit}
            >
                {filters && onDidSelectValue && values && (
                    <FilterControl filters={filters} onDidSelectValue={onDidSelectValue} values={values}>
                        {additionalFilterElement}
                    </FilterControl>
                )}
                {!hideSearch && (
                    <input
                        className={classNames('form-control', inputClassName)}
                        type="search"
                        placeholder={inputPlaceholder}
                        name="query"
                        value={query}
                        onChange={onChange}
                        autoFocus={autoFocus}
                        autoComplete="off"
                        autoCorrect="off"
                        autoCapitalize="off"
                        ref={reference}
                        spellCheck={false}
                    />
                )}
            </Form>
        )
    }
)
