import classNames from 'classnames'
import CommentOutlineIcon from 'mdi-react/CommentOutlineIcon'
import ExternalLinkIcon from 'mdi-react/ExternalLinkIcon'
import LinkVariantRemoveIcon from 'mdi-react/LinkVariantRemoveIcon'
import SourceBranchIcon from 'mdi-react/SourceBranchIcon'
import SyncIcon from 'mdi-react/SyncIcon'
import UploadIcon from 'mdi-react/UploadIcon'
import React from 'react'

import { Link } from '@sourcegraph/shared/src/components/Link'
import { LinkOrSpan } from '@sourcegraph/shared/src/components/LinkOrSpan'
import { BulkOperationState, BulkOperationType } from '@sourcegraph/shared/src/graphql-operations'
import { pluralize } from '@sourcegraph/shared/src/util/strings'

import { ErrorMessage } from '../../../../components/alerts'
import { Collapsible } from '../../../../components/Collapsible'
import { Timestamp } from '../../../../components/time/Timestamp'
import { BulkOperationFields } from '../../../../graphql-operations'

import styles from './BulkOperationNode.module.scss'

const OPERATION_TITLES: Record<BulkOperationType, JSX.Element> = {
    COMMENT: (
        <>
            <CommentOutlineIcon className="icon-inline text-muted" /> Comment on changesets
        </>
    ),
    DETACH: (
        <>
            <LinkVariantRemoveIcon className="icon-inline text-muted" /> Detach changesets
        </>
    ),
    REENQUEUE: (
        <>
            <SyncIcon className="icon-inline text-muted" /> Retry changesets
        </>
    ),
    MERGE: (
        <>
            <SourceBranchIcon className="icon-inline text-muted" /> Merge changesets
        </>
    ),
    CLOSE: (
        <>
            <SourceBranchIcon className="icon-inline text-danger" /> Close changesets
        </>
    ),
    PUBLISH: (
        <>
            <UploadIcon className="icon-inline text-muted" /> Publish changesets
        </>
    ),
}

export interface BulkOperationNodeProps {
    node: BulkOperationFields
}

export const BulkOperationNode: React.FunctionComponent<BulkOperationNodeProps> = ({ node }) => (
    <>
        <div
            className={classNames(
                styles.bulkOperationNodeContainer,
                'd-flex justify-content-between align-items-center'
            )}
        >
            <div className={classNames(styles.bulkOperationNodeChangesetCounts, 'text-center')}>
                <p className="badge badge-secondary mb-2">{node.changesetCount}</p>
                <p className="mb-0">{pluralize('changeset', node.changesetCount)}</p>
            </div>
            <div className={styles.bulkOperationNodeDivider} />
            <div className="flex-grow-1 ml-3">
                <h4>{OPERATION_TITLES[node.type]}</h4>
                <p className="mb-0">
                    <Link to={node.initiator.url}>{node.initiator.username}</Link> <Timestamp date={node.createdAt} />
                </p>
            </div>
            {node.state === BulkOperationState.PROCESSING && (
                <div className={classNames(styles.bulkOperationNodeProgressBar, 'flex-grow-1 ml-3')}>
                    <meter value={node.progress} className="w-100" min={0} max={1} />
                    <p className="text-center mb-0">{Math.ceil(node.progress * 100)}%</p>
                </div>
            )}
            {node.state === BulkOperationState.FAILED && (
                <span className="badge badge-danger text-uppercase">failed</span>
            )}
            {node.state === BulkOperationState.COMPLETED && (
                <span className="badge badge-success text-uppercase">complete</span>
            )}
        </div>
        {node.errors.length > 0 && (
            <div className={classNames(styles.bulkOperationNodeErrors, 'px-4')}>
                <Collapsible
                    titleClassName="flex-grow-1 p-3"
                    title={<h4 className="mb-0">The following errors occured while running this task:</h4>}
                >
                    {node.errors.map((error, index) => (
                        <div className="mt-2 alert alert-danger" key={index}>
                            <p>
                                {error.changeset.__typename === 'HiddenExternalChangeset' ? (
                                    <span className="text-muted">On hidden repository</span>
                                ) : (
                                    <>
                                        <LinkOrSpan className="alert-link" to={error.changeset.externalURL?.url}>
                                            {error.changeset.title} <ExternalLinkIcon className="icon-inline" />
                                        </LinkOrSpan>{' '}
                                        on{' '}
                                        <Link className="alert-link" to={error.changeset.repository.url}>
                                            repository {error.changeset.repository.name}
                                        </Link>
                                        .
                                    </>
                                )}
                            </p>
                            {error.error && <ErrorMessage error={'```\n' + error.error + '\n```'} />}
                        </div>
                    ))}
                </Collapsible>
            </div>
        )}
    </>
)
