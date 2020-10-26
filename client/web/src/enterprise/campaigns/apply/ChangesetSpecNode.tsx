import * as H from 'history'
import React from 'react'
import { ChangesetSpecFields } from '../../../graphql-operations'
import { HiddenChangesetSpecNode } from './HiddenChangesetSpecNode'
import { VisibleChangesetSpecNode } from './VisibleChangesetSpecNode'
import { ThemeProps } from '../../../../../shared/src/theme'
import { queryChangesetSpecFileDiffs } from './backend'

export interface ChangesetSpecNodeProps extends ThemeProps {
    node: ChangesetSpecFields
    history: H.History
    location: H.Location

    /** Used for testing. */
    queryChangesetSpecFileDiffs?: typeof queryChangesetSpecFileDiffs
    /** Expand changeset descriptions, for testing only. */
    expandChangesetDescriptions?: boolean
}

export const ChangesetSpecNode: React.FunctionComponent<ChangesetSpecNodeProps> = ({
    node,
    history,
    location,
    isLightTheme,
    queryChangesetSpecFileDiffs,
    expandChangesetDescriptions,
}) => {
    if (node.__typename === 'HiddenChangesetSpec') {
        return (
            <>
                <span className="changeset-spec-node__separator" />
                <HiddenChangesetSpecNode node={node} />
            </>
        )
    }
    return (
        <>
            <span className="changeset-spec-node__separator" />
            <VisibleChangesetSpecNode
                node={node}
                history={history}
                location={location}
                isLightTheme={isLightTheme}
                queryChangesetSpecFileDiffs={queryChangesetSpecFileDiffs}
                expandChangesetDescriptions={expandChangesetDescriptions}
            />
        </>
    )
}
