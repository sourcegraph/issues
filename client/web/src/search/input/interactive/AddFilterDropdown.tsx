import React, { useCallback } from 'react'
import { FilterType, filterTypeKeys } from '../../../../../shared/src/search/interactive/util'
import { defaultFilterTypes } from './AddFilterRow'
import { FilterTypeToProseNames } from './filters'

interface Props {
    onAddNewFilter: (filterType: FilterType) => void
}

export const AddFilterDropdown: React.FunctionComponent<Props> = ({ onAddNewFilter }) => {
    const addNewFilter = useCallback(
        (event: React.ChangeEvent<HTMLSelectElement>): void => {
            onAddNewFilter(event.target.value as FilterType)
        },
        [onAddNewFilter]
    )

    return (
        <select
            className="form-control add-filter-dropdown test-filter-dropdown"
            onChange={addNewFilter}
            value="default"
        >
            <option value="default" disabled={true}>
                Add filter…
            </option>
            {filterTypeKeys
                .filter(
                    filter =>
                        !defaultFilterTypes.includes(filter) &&
                        filter !== FilterType.case &&
                        filter !== FilterType.patterntype
                )
                .map(filter => (
                    <option key={filter} value={filter} className={`test-filter-dropdown-option-${filter}`}>
                        {FilterTypeToProseNames[filter]}
                    </option>
                ))}
        </select>
    )
}
