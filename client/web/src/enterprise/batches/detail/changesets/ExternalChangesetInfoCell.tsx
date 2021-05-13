import classNames from 'classnames'
import ExternalLinkIcon from 'mdi-react/ExternalLinkIcon'
import React from 'react'

import { Link } from '@sourcegraph/shared/src/components/Link'
import { LinkOrSpan } from '@sourcegraph/shared/src/components/LinkOrSpan'

import {
    ExternalChangesetFields,
    GitBranchChangesetDescriptionFields,
    ChangesetState,
} from '../../../../graphql-operations'

import { ChangesetLabel } from './ChangesetLabel'
import { ChangesetLastSynced } from './ChangesetLastSynced'

export interface ExternalChangesetInfoCellProps {
    node: ExternalChangesetFields
    viewerCanAdminister: boolean
    className?: string
}

export const ExternalChangesetInfoCell: React.FunctionComponent<ExternalChangesetInfoCellProps> = ({
    node,
    viewerCanAdminister,
    className,
}) => (
    <div className={classNames('d-flex flex-column', className)}>
        <div className="m-0">
            <h3 className="m-0 d-block d-md-inline">
                <LinkOrSpan
                    to={node.externalURL?.url ?? undefined}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="mr-2"
                >
                    {(isImporting(node) || importingFailed(node)) && (
                        <>
                            Importing changeset
                            {node.externalID && <> #{node.externalID} </>}
                        </>
                    )}
                    {!isImporting(node) && !importingFailed(node) && (
                        <>
                            {node.title}
                            {node.externalID && <> (#{node.externalID}) </>}
                        </>
                    )}
                    {node.externalURL?.url && (
                        <>
                            {' '}
                            <ExternalLinkIcon size="1rem" />
                        </>
                    )}
                </LinkOrSpan>
            </h3>
            {node.labels.length > 0 && (
                <span className="d-block d-md-inline-block">
                    {node.labels.map(label => (
                        <ChangesetLabel label={label} key={label.text} />
                    ))}
                </span>
            )}
        </div>
        <div>
            <span className="mr-2 d-block d-mdinline-block">
                <Link to={node.repository.url} target="_blank" rel="noopener noreferrer">
                    {node.repository.name}
                </Link>
                {node.repository.metadataTags.nodes.map(({ tag }) => (
                    <span key={tag} className="badge badge-secondary ml-1">
                        {tag}
                    </span>
                ))}
                {node.repository.metadataTags.pageInfo.hasNextPage && (
                    <span key="overflow" className="badge badge-secondary ml-1">
                        &hellip;
                    </span>
                )}
                {hasHeadReference(node) && (
                    <div className="d-block d-sm-inline-block ml-1">
                        <span className="badge badge-info text-monospace">{headReference(node)}</span>
                    </div>
                )}
            </span>
            {![
                ChangesetState.FAILED,
                ChangesetState.PROCESSING,
                ChangesetState.RETRYING,
                ChangesetState.UNPUBLISHED,
                ChangesetState.SCHEDULED,
            ].includes(node.state) && (
                <ChangesetLastSynced changeset={node} viewerCanAdminister={viewerCanAdminister} />
            )}
        </div>
    </div>
)

function isImporting(node: ExternalChangesetFields): boolean {
    return node.state === ChangesetState.PROCESSING && !hasHeadReference(node)
}

function importingFailed(node: ExternalChangesetFields): boolean {
    return node.state === ChangesetState.FAILED && !hasHeadReference(node)
}

function headReference(node: ExternalChangesetFields): string | undefined {
    if (hasHeadReference(node)) {
        return node.currentSpec?.description.headRef
    }
    return undefined
}

function hasHeadReference(
    node: ExternalChangesetFields
): node is ExternalChangesetFields & {
    currentSpec: ExternalChangesetFields & { description: GitBranchChangesetDescriptionFields }
} {
    return node.currentSpec?.description.__typename === 'GitBranchChangesetDescription'
}
