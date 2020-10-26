import SearchIcon from 'mdi-react/SearchIcon'
import React from 'react'
import { SearchHelpDropdownButton } from './SearchHelpDropdownButton'

interface Props {
    /** Hide the "help" icon and dropdown. */
    noHelp?: boolean
}

/**
 * A search button with a dropdown with related links. It must be wrapped in a form whose onSubmit
 * handler performs the search.
 */
export const SearchButton: React.FunctionComponent<Props> = ({ noHelp }) => (
    <div className="search-button d-flex">
        <button className="btn btn-primary search-button__btn test-search-button" type="submit" aria-label="Search">
            <SearchIcon className="icon-inline" aria-hidden="true" />
        </button>
        {!noHelp && <SearchHelpDropdownButton />}
    </div>
)
