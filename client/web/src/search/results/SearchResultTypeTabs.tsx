import * as React from 'react'
import * as H from 'history'
import { SearchResultTabHeader } from './SearchResultTab'
import { PatternTypeProps, CaseSensitivityProps, InteractiveSearchProps } from '..'
import { VersionContextProps } from '../../../../shared/src/search/util'

interface Props
    extends Omit<PatternTypeProps, 'setPatternType'>,
        Omit<CaseSensitivityProps, 'setCaseSensitivity'>,
        Pick<InteractiveSearchProps, 'filtersInQuery'>,
        VersionContextProps {
    location: H.Location
    history: H.History
    query: string
}

export const SearchResultTypeTabs: React.FunctionComponent<Props> = props => (
    <div className="mt-2 border-bottom test-search-result-type-tabs">
        <ul className="nav nav-tabs border-bottom-0">
            <SearchResultTabHeader {...props} type={null} />
            <SearchResultTabHeader {...props} type="diff" />
            <SearchResultTabHeader {...props} type="commit" />
            <SearchResultTabHeader {...props} type="symbol" />
            <SearchResultTabHeader {...props} type="repo" />
            <SearchResultTabHeader {...props} type="path" />
        </ul>
    </div>
)
