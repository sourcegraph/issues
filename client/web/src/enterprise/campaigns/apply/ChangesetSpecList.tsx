import React, { useCallback } from 'react'
import * as H from 'history'
import { ThemeProps } from '../../../../../shared/src/theme'
import { FilteredConnection, FilteredConnectionQueryArguments } from '../../../components/FilteredConnection'
import { Scalars, ChangesetSpecFields } from '../../../graphql-operations'
import { queryChangesetSpecs as _queryChangesetSpecs, queryChangesetSpecFileDiffs } from './backend'
import { ChangesetSpecNode, ChangesetSpecNodeProps } from './ChangesetSpecNode'
import { ChangesetSpecListHeader } from './ChangesetSpecListHeader'
import { EmptyChangesetSpecListElement } from './EmptyChangesetSpecListElement'

interface Props extends ThemeProps {
    campaignSpecID: Scalars['ID']
    history: H.History
    location: H.Location

    /** For testing only. */
    queryChangesetSpecs?: typeof _queryChangesetSpecs
    /** For testing only. */
    queryChangesetSpecFileDiffs?: typeof queryChangesetSpecFileDiffs
    /** Expand changeset descriptions, for testing only. */
    expandChangesetDescriptions?: boolean
}

/**
 * A list of a campaign spec's changeset specs.
 */
export const ChangesetSpecList: React.FunctionComponent<Props> = ({
    campaignSpecID,
    history,
    location,
    isLightTheme,
    queryChangesetSpecs = _queryChangesetSpecs,
    queryChangesetSpecFileDiffs,
    expandChangesetDescriptions,
}) => {
    const queryChangesetSpecsConnection = useCallback(
        (args: FilteredConnectionQueryArguments) =>
            queryChangesetSpecs({
                first: args.first ?? null,
                after: args.after ?? null,
                campaignSpec: campaignSpecID,
            }),
        [campaignSpecID, queryChangesetSpecs]
    )

    return (
        <>
            <h3>Changesets</h3>
            <hr className="mb-3" />
            <FilteredConnection<ChangesetSpecFields, Omit<ChangesetSpecNodeProps, 'node'>>
                className="mt-2"
                nodeComponent={ChangesetSpecNode}
                nodeComponentProps={{
                    isLightTheme,
                    history,
                    location,
                    queryChangesetSpecFileDiffs,
                    expandChangesetDescriptions,
                }}
                queryConnection={queryChangesetSpecsConnection}
                hideSearch={true}
                defaultFirst={15}
                noun="changeset"
                pluralNoun="changesets"
                history={history}
                location={location}
                useURLQuery={true}
                listComponent="div"
                listClassName="changeset-spec-list__grid mb-3"
                headComponent={ChangesetSpecListHeader}
                cursorPaging={true}
                noSummaryIfAllNodesVisible={true}
                emptyElement={<EmptyChangesetSpecListElement />}
            />
        </>
    )
}
