import * as React from 'react'
import { FilterInput } from './FilterInput'
import { QueryState } from '../../helpers'
import { FilterType } from '../../../../../shared/src/search/interactive/util'
import { InteractiveSearchProps } from '../..'
import classNames from 'classnames'

interface Props extends Pick<InteractiveSearchProps, 'filtersInQuery'> {
    /**
     * The query in the main query input.
     */
    navbarQuery: QueryState

    /**
     * Callback to trigger a search when a filter is submitted.
     */
    onSubmit: (event: React.FormEvent<HTMLFormElement>) => void

    /**
     * Callback to handle a filter's value being updated.
     */
    onFilterEdited: (filterKey: string, value: string) => void

    /**
     * Callback to handle a filter being deleted from the selected filter row.
     */
    onFilterDeleted: (filterKey: string) => void

    /**
     * Callback to handle the editable state of a filter.
     */
    toggleFilterEditable: (filterKey: string) => void

    /**
     * Callback to handle the negation state of a filter.
     */
    toggleFilterNegated: (filterKey: string) => void

    /**
     * Whether globbing is enabled for filters.
     */
    globbing: boolean

    emptyClassName?: string
}

/**
 * The row displaying the filters that have been added to the query in interactive mode.
 */
export const SelectedFiltersRow: React.FunctionComponent<Props> = ({
    filtersInQuery,
    navbarQuery,
    onSubmit,
    onFilterEdited,
    onFilterDeleted,
    toggleFilterEditable,
    toggleFilterNegated,
    globbing,
    emptyClassName = '',
}) => {
    const filterKeys = Object.keys(filtersInQuery)
    return (
        <div className={classNames('selected-filters-row', { [emptyClassName]: filterKeys.length === 0 })}>
            {filtersInQuery &&
                filterKeys.map(field => (
                    /** Replace this with new input component, which can be an input when editable, and button when non-editable */
                    <FilterInput
                        globbing={globbing}
                        key={field}
                        mapKey={field}
                        filterType={filtersInQuery[field].type as Exclude<FilterType, FilterType.patterntype>}
                        value={filtersInQuery[field].value}
                        editable={filtersInQuery[field].editable}
                        negated={filtersInQuery[field].negated}
                        filtersInQuery={filtersInQuery}
                        navbarQuery={navbarQuery}
                        onSubmit={onSubmit}
                        onFilterDeleted={onFilterDeleted}
                        onFilterEdited={onFilterEdited}
                        toggleFilterEditable={toggleFilterEditable}
                        toggleFilterNegated={toggleFilterNegated}
                    />
                ))}
        </div>
    )
}
