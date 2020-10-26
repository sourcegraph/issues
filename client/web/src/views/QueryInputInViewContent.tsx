import React, { useCallback, useState } from 'react'
import { Form } from '../../../branded/src/components/Form'
import { QueryInput } from '../search/input/QueryInput'
import { CaseSensitivityProps, PatternTypeProps, CopyQueryButtonProps } from '../search'
import { SearchButton } from '../search/input/SearchButton'
import { SettingsCascadeProps } from '../../../shared/src/settings/settings'
import { QueryState, submitSearch } from '../search/helpers'
import * as H from 'history'
import { VersionContextProps } from '../../../shared/src/search/util'

interface Props
    extends SettingsCascadeProps,
        PatternTypeProps,
        CaseSensitivityProps,
        CopyQueryButtonProps,
        VersionContextProps {
    implicitQueryPrefix: string

    location: H.Location
    history: H.History

    /** Whether globbing is enabled for filters. */
    globbing: boolean
}

/**
 * A query input rendered in a view from an extension.
 */
export const QueryInputInViewContent: React.FunctionComponent<Props> = ({
    implicitQueryPrefix,
    settingsCascade,
    ...props
}) => {
    const [queryState, setQueryState] = useState<QueryState>({ query: '', cursorPosition: 0 })

    const onSubmit = useCallback(
        (event: React.FormEvent<HTMLFormElement>): void => {
            event.preventDefault()
            submitSearch({
                ...props,
                query: `${implicitQueryPrefix} ${queryState.query}`,
                source: 'scopePage',
            })
        },
        [implicitQueryPrefix, props, queryState.query]
    )
    return (
        <Form className="d-flex" onSubmit={onSubmit}>
            <QueryInput
                {...props}
                value={queryState}
                onChange={setQueryState}
                prependQueryForSuggestions={implicitQueryPrefix}
                autoFocus={true}
                location={props.location}
                history={props.history}
                settingsCascade={settingsCascade}
                placeholder="Search..."
            />
            <SearchButton />
        </Form>
    )
}
