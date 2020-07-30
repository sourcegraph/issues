import { ThemeProps } from '../../../../../../shared/src/theme'
import { ChangesetCheckState, ChangesetExternalState } from '../../../../../../shared/src/graphql/schema'
import { Observer } from 'rxjs'
import { Hoverifier } from '@sourcegraph/codeintellify'
import { RepoSpec, RevisionSpec, FileSpec, ResolvedRevisionSpec } from '../../../../../../shared/src/util/url'
import { HoverMerged } from '../../../../../../shared/src/api/client/types/hover'
import { ActionItemAction } from '../../../../../../shared/src/actions/ActionItem'
import { ExtensionsControllerProps } from '../../../../../../shared/src/extensions/controller'
import {
    changesetReviewStateIcons,
    changesetCheckStateIcons,
    changesetCheckStateColors,
    changesetCheckStateTooltips,
    changesetReviewStateColors,
    changesetStateLabels,
} from './presentation'
import * as H from 'history'
import React, { useState, useCallback, useMemo } from 'react'
import { LinkOrSpan } from '../../../../../../shared/src/components/LinkOrSpan'
import ExternalLinkIcon from 'mdi-react/ExternalLinkIcon'
import classNames from 'classnames'
import { ChangesetLabel } from './ChangesetLabel'
import { Link } from '../../../../../../shared/src/components/Link'
import { ChangesetLastSynced } from './ChangesetLastSynced'
import { DiffStat } from '../../../../components/diff/DiffStat'
import { FilteredConnectionQueryArgs } from '../../../../components/FilteredConnection'
import { queryExternalChangesetWithFileDiffs } from '../backend'
import { Collapsible } from '../../../../components/Collapsible'
import { FileDiffConnection } from '../../../../components/diff/FileDiffConnection'
import { FileDiffNode } from '../../../../components/diff/FileDiffNode'
import { tap, map } from 'rxjs/operators'
import { ChangesetStateIcon } from './ChangesetStateIcon'
import { RepositoryComparisonFields, ChangesetFields } from '../../../../graphql-operations'

export interface ExternalChangesetNodeProps extends ThemeProps {
    node: ChangesetFields & { __typename: 'ExternalChangeset' }
    viewerCanAdminister: boolean
    campaignUpdates?: Pick<Observer<void>, 'next'>
    history: H.History
    location: H.Location
    extensionInfo?: {
        hoverifier: Hoverifier<RepoSpec & RevisionSpec & FileSpec & ResolvedRevisionSpec, HoverMerged, ActionItemAction>
    } & ExtensionsControllerProps
}

export const ExternalChangesetNode: React.FunctionComponent<ExternalChangesetNodeProps> = ({
    node,
    viewerCanAdminister,
    campaignUpdates,
    isLightTheme,
    history,
    location,
    extensionInfo,
}) => {
    const ReviewStateIcon = node.reviewState && changesetReviewStateIcons[node.reviewState]
    const ChangesetCheckStateIcon = node.checkState
        ? changesetCheckStateIcons[node.checkState]
        : changesetCheckStateIcons[ChangesetCheckState.PENDING]
    const changesetState = node.externalState

    const changesetNodeRow = (
        <div className="d-flex align-items-start m-1 ml-2">
            <div className="changeset-node__content flex-fill">
                <div className="d-flex flex-column">
                    <div className="m-0 mb-2">
                        <h3 className="m-0 d-inline">
                            <ChangesetStateIcon externalState={changesetState || ChangesetExternalState.OPEN} />
                            <LinkOrSpan
                                /* Deleted changesets most likely don't exist on the codehost anymore and would return 404 pages */
                                to={
                                    node.externalURL && node.externalState !== ChangesetExternalState.DELETED
                                        ? node.externalURL.url
                                        : undefined
                                }
                                target="_blank"
                                rel="noopener noreferrer"
                            >
                                {node.title} (#{node.externalID}){' '}
                                {node.externalURL && node.externalState !== ChangesetExternalState.DELETED && (
                                    <ExternalLinkIcon size="1rem" />
                                )}
                            </LinkOrSpan>
                        </h3>
                        {node.checkState && (
                            <small>
                                <ChangesetCheckStateIcon
                                    className={classNames(
                                        'ml-1 changeset-node__check-state',
                                        changesetCheckStateColors[node.checkState]
                                    )}
                                    data-tooltip={changesetCheckStateTooltips[node.checkState]}
                                />
                            </small>
                        )}
                        {node.labels.length > 0 && (
                            <span className="ml-2">
                                {node.labels.map(label => (
                                    <ChangesetLabel label={label} key={label.text} />
                                ))}
                            </span>
                        )}
                    </div>
                    <div>
                        <strong className="mr-2">
                            <Link to={node.repository.url} target="_blank" rel="noopener noreferrer">
                                {node.repository.name}
                            </Link>
                        </strong>
                        <ChangesetLastSynced
                            changeset={node}
                            viewerCanAdminister={viewerCanAdminister}
                            campaignUpdates={campaignUpdates}
                        />
                    </div>
                </div>
            </div>
            <div className="flex-shrink-0 flex-grow-0 ml-1 align-items-end">
                {node.diffStat && <DiffStat {...node.diffStat} expandedCounts={true} />}
            </div>
            {ReviewStateIcon && (
                <div className="flex-shrink-0 flex-grow-0 ml-1 align-items-end">
                    <ReviewStateIcon
                        className={
                            node.externalState === ChangesetExternalState.DELETED
                                ? 'text-muted'
                                : `text-${changesetReviewStateColors[node.reviewState!]}`
                        }
                        data-tooltip={changesetStateLabels[node.reviewState!]}
                    />
                </div>
            )}
        </div>
    )

    const [range, setRange] = useState<RepositoryComparisonFields['range']>()

    /** Fetches the file diffs for the changeset */
    const queryFileDiffs = useCallback(
        (args: FilteredConnectionQueryArgs) =>
            queryExternalChangesetWithFileDiffs(node.id, { ...args, isLightTheme }).pipe(
                map(changeset => {
                    if (!changeset.diff) {
                        throw new Error('The given changeset has no diff')
                    }
                    return changeset.diff
                }),
                tap(diff => {
                    if (diff.__typename === 'RepositoryComparison') {
                        setRange(diff.range)
                    }
                }),
                map(diff => diff.fileDiffs)
            ),
        [node.id, isLightTheme]
    )

    const hydratedExtensionInfo = useMemo(() => {
        if (!extensionInfo || !range) {
            return
        }
        const baseRevision = commitOIDForGitRevision(range.base)
        const headRevision = commitOIDForGitRevision(range.head)
        return {
            ...extensionInfo,
            head: {
                commitID: headRevision,
                repoID: node.repository.id,
                repoName: node.repository.name,
                revision: headRevision,
            },
            base: {
                commitID: baseRevision,
                repoID: node.repository.id,
                repoName: node.repository.name,
                revision: baseRevision,
            },
        }
    }, [extensionInfo, range, node.repository.id, node.repository.name])

    return (
        <li className="list-group-item test-changeset-node">
            {node.diff?.fileDiffs ? (
                <Collapsible
                    titleClassName="changeset-node__content flex-fill"
                    expandedButtonClassName="mb-3"
                    title={changesetNodeRow}
                    wholeTitleClickable={false}
                >
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
                            extensionInfo: hydratedExtensionInfo,
                            lineNumbers: true,
                        }}
                        updateOnChange={node.repository.id}
                        defaultFirst={15}
                        hideSearch={true}
                        noSummaryIfAllNodesVisible={true}
                        history={history}
                        location={location}
                        useURLQuery={false}
                        cursorPaging={true}
                    />
                </Collapsible>
            ) : (
                <div className="changeset-node__content changeset-node__content--no-collapse flex-fill">
                    {changesetNodeRow}
                </div>
            )}
        </li>
    )
}

function commitOIDForGitRevision(revision: RepositoryComparisonFields['range']['head']): string {
    switch (revision.__typename) {
        case 'GitObject':
            return revision.oid
        case 'GitRef':
            return revision.target.oid
        case 'GitRevSpecExpr':
            if (!revision.object) {
                throw new Error('Could not resolve commit for revision')
            }
            return revision.object.oid
    }
}
