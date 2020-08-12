import * as H from 'history'
import React, { useCallback, useState } from 'react'
import { ChangesetSpecFields } from '../../../graphql-operations'
import { ThemeProps } from '../../../../../shared/src/theme'
import { FileDiffNode } from '../../../components/diff/FileDiffNode'
import { FileDiffConnection } from '../../../components/diff/FileDiffConnection'
import { map } from 'rxjs/operators'
import { queryChangesetSpecFileDiffs } from './backend'
import { FilteredConnectionQueryArgs } from '../../../components/FilteredConnection'
import { Link } from '../../../../../shared/src/components/Link'
import { DiffStat } from '../../../components/diff/DiffStat'
import { ChangesetSpecAction } from './ChangesetSpecAction'
import ChevronDownIcon from 'mdi-react/ChevronDownIcon'
import ChevronRightIcon from 'mdi-react/ChevronRightIcon'

export interface VisibleChangesetSpecNodeProps extends ThemeProps {
    node: ChangesetSpecFields & { __typename: 'VisibleChangesetSpec' }
    history: H.History
    location: H.Location
}

export const VisibleChangesetSpecNode: React.FunctionComponent<VisibleChangesetSpecNodeProps> = ({
    node,
    isLightTheme,
    history,
    location,
}) => {
    const [isExpanded, setIsExpanded] = useState(false)
    const toggleIsExpanded = useCallback<React.MouseEventHandler<HTMLButtonElement>>(
        event => {
            event.preventDefault()
            setIsExpanded(!isExpanded)
        },
        [isExpanded]
    )

    /** Fetches the file diffs for the changeset */
    const queryFileDiffs = useCallback(
        (args: FilteredConnectionQueryArgs) =>
            queryChangesetSpecFileDiffs({
                after: args.after ?? null,
                first: args.first ?? null,
                changesetSpec: node.id,
                isLightTheme,
            }).pipe(
                map(diff => {
                    if (!diff) {
                        throw new Error('The given changeset spec has no diff')
                    }
                    return diff.fileDiffs
                })
            ),
        [node.id, isLightTheme]
    )

    return (
        <>
            <button
                type="button"
                className="btn btn-icon"
                aria-label={isExpanded ? 'Collapse section' : 'Expand section'}
                onClick={toggleIsExpanded}
            >
                {isExpanded ? (
                    <ChevronDownIcon className="icon-inline" aria-label="Close section" />
                ) : (
                    <ChevronRightIcon className="icon-inline" aria-label="Expand section" />
                )}
            </button>
            <ChangesetSpecAction spec={node} />
            <div>
                <div className="d-flex flex-column">
                    <div className="m-0 mb-2">
                        <h3 className="m-0 d-inline">
                            {node.description.__typename === 'ExistingChangesetReference' && (
                                <span>Import changeset #{node.description.externalID}</span>
                            )}
                            {node.description.__typename === 'GitBranchChangesetDescription' && (
                                <span>{node.description.title}</span>
                            )}
                        </h3>
                    </div>
                    <div>
                        <strong className="mr-2">
                            <Link to={node.description.baseRepository.url} target="_blank" rel="noopener noreferrer">
                                {node.description.baseRepository.name}
                            </Link>{' '}
                            {node.description.__typename === 'GitBranchChangesetDescription' && (
                                <>
                                    <span className="badge badge-primary">{node.description.baseRef}</span> &larr;{' '}
                                    <span className="badge badge-primary">{node.description.headRef}</span>
                                </>
                            )}
                        </strong>
                    </div>
                </div>
            </div>
            <div className="justify-self-end align-self-start">
                {node.description.__typename === 'GitBranchChangesetDescription' && (
                    <DiffStat {...node.description.diff.fileDiffs.diffStat} expandedCounts={true} />
                )}
            </div>
            {isExpanded && (
                <div className="grid-row">
                    {node.description.__typename === 'GitBranchChangesetDescription' && (
                        <FileDiffConnection
                            listClassName="list-group list-group-flush"
                            noun="changed file"
                            pluralNoun="changed files"
                            queryConnection={queryFileDiffs}
                            nodeComponent={FileDiffNode}
                            nodeComponentProps={{
                                history,
                                location,
                                isLightTheme,
                                persistLines: true,
                                lineNumbers: true,
                            }}
                            defaultFirst={15}
                            hideSearch={true}
                            noSummaryIfAllNodesVisible={true}
                            history={history}
                            location={location}
                            useURLQuery={false}
                            cursorPaging={true}
                        />
                    )}
                    {node.description.__typename === 'ExistingChangesetReference' && (
                        <div className="alert alert-info mb-0">
                            When run, the changeset with ID {node.description.externalID} will be imported from{' '}
                            {node.description.baseRepository.name}.
                        </div>
                    )}
                </div>
            )}
        </>
    )
}
