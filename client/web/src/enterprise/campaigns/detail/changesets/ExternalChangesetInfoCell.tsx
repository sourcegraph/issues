import React from 'react'
import {
    ExternalChangesetFields,
    ChangesetExternalState,
    ChangesetPublicationState,
} from '../../../../graphql-operations'
import { LinkOrSpan } from '../../../../../../shared/src/components/LinkOrSpan'
import ExternalLinkIcon from 'mdi-react/ExternalLinkIcon'
import { ChangesetLabel } from './ChangesetLabel'
import { Link } from '../../../../../../shared/src/components/Link'
import { ChangesetLastSynced } from './ChangesetLastSynced'
import { ChangesetReconcilerState } from '../../../../../../shared/src/graphql/schema'
import classNames from 'classnames'

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
                    /* Deleted changesets most likely don't exist on the codehost anymore and would return 404 pages */
                    to={
                        node.externalURL && node.externalState !== ChangesetExternalState.DELETED
                            ? node.externalURL.url
                            : undefined
                    }
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
                    {node.externalURL && node.externalState !== ChangesetExternalState.DELETED && (
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
            </span>
            {node.publicationState === ChangesetPublicationState.PUBLISHED && (
                <ChangesetLastSynced changeset={node} viewerCanAdminister={viewerCanAdminister} />
            )}
        </div>
    </div>
)

function isImporting(node: ExternalChangesetFields): boolean {
    return (
        [ChangesetReconcilerState.QUEUED, ChangesetReconcilerState.PROCESSING].includes(node.reconcilerState) &&
        !node.title
    )
}

function importingFailed(node: ExternalChangesetFields): boolean {
    return node.reconcilerState === ChangesetReconcilerState.ERRORED && !node.title
}
