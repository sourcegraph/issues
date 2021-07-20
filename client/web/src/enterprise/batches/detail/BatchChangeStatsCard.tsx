import classNames from 'classnames'
import CheckCircleOutlineIcon from 'mdi-react/CheckCircleOutlineIcon'
import ProgressCheckIcon from 'mdi-react/ProgressCheckIcon'
import React from 'react'

import { pluralize } from '@sourcegraph/shared/src/util/strings'

import { DiffStatStack } from '../../../components/diff/DiffStat'
import { BatchChangeFields, ChangesetsStatsFields, DiffStatFields } from '../../../graphql-operations'

import { BatchChangeStateBadge } from './BatchChangeStateBadge'
import styles from './BatchChangeStatsCard.module.scss'
import {
    ChangesetStatusUnpublished,
    ChangesetStatusOpen,
    ChangesetStatusClosed,
    ChangesetStatusMerged,
    ChangesetStatusDraft,
    ChangesetStatusArchived,
} from './changesets/ChangesetStatusCell'

interface BatchChangeStatsCardProps {
    stats: ChangesetsStatsFields
    diff: DiffStatFields
    closedAt: BatchChangeFields['closedAt']
    className?: string
}

export const BatchChangeStatsCard: React.FunctionComponent<BatchChangeStatsCardProps> = ({
    stats,
    diff,
    closedAt,
    className,
}) => {
    const percentComplete =
        stats.total === 0 ? 0 : (((stats.closed + stats.merged + stats.deleted) / stats.total) * 100).toFixed(0)
    const isCompleted = stats.closed + stats.merged + stats.deleted === stats.total
    let BatchChangeStatusIcon = ProgressCheckIcon
    if (isCompleted) {
        BatchChangeStatusIcon = CheckCircleOutlineIcon
    }
    return (
        <div className={classNames(className)}>
            <div className="d-flex flex-wrap align-items-center flex-grow-1">
                <h2 className="m-0">
                    <BatchChangeStateBadge isClosed={!!closedAt} className={styles.batchChangeStatsCardStateBadge} />
                </h2>
                <div className={classNames(styles.batchChangeStatsCardDivider, 'mx-3')} />
                <div className="d-flex align-items-center">
                    <h1 className="d-inline mb-0">
                        <BatchChangeStatusIcon
                            className={classNames(
                                'icon-inline mr-2',
                                isCompleted && 'text-success',
                                !isCompleted && 'text-muted'
                            )}
                        />
                    </h1>{' '}
                    <span className={classNames(styles.batchChangeStatsCardCompleteness, 'lead text-nowrap')}>
                        {percentComplete}% complete
                    </span>
                </div>
                <div className={classNames(styles.batchChangeStatsCardDivider, 'd-none d-md-block mx-3')} />
                <DiffStatStack className={styles.batchChangeStatsCardDiffStat} {...diff} />
                <div className="d-flex flex-wrap justify-content-end flex-grow-1">
                    <BatchChangeStatsTotalAction count={stats.total} />
                    <ChangesetStatusUnpublished
                        label={<span className="text-muted">{stats.unpublished} Unpublished</span>}
                        className={classNames(styles.batchChangeStatsCardStat, 'd-flex flex-grow-0 px-2 text-truncate')}
                    />
                    <ChangesetStatusDraft
                        label={<span className="text-muted">{stats.draft} Draft</span>}
                        className={classNames(styles.batchChangeStatsCardStat, 'd-flex flex-grow-0 px-2 text-truncate')}
                    />
                    <ChangesetStatusOpen
                        label={<span className="text-muted">{stats.open} Open</span>}
                        className={classNames(styles.batchChangeStatsCardStat, 'd-flex flex-grow-0 px-2 text-truncate')}
                    />
                    <ChangesetStatusClosed
                        label={<span className="text-muted">{stats.closed} Closed</span>}
                        className={classNames(styles.batchChangeStatsCardStat, 'd-flex flex-grow-0 px-2 text-truncate')}
                    />
                    <ChangesetStatusMerged
                        label={<span className="text-muted">{stats.merged} Merged</span>}
                        className={classNames(styles.batchChangeStatsCardStat, 'd-flex flex-grow-0 pl-2 text-truncate')}
                    />
                    <ChangesetStatusArchived
                        label={<span className="text-muted">{stats.archived} Archived</span>}
                        className={classNames(styles.batchChangeStatsCardStat, 'd-flex flex-grow-0 pl-2 text-truncate')}
                    />
                </div>
            </div>
        </div>
    )
}

export const BatchChangeStatsTotalAction: React.FunctionComponent<{ count: number }> = ({ count }) => (
    <div
        className={classNames(
            styles.batchChangeStatsCardStat,
            'm-0 flex-grow-0 pr-2 text-truncate text-nowrap d-flex flex-column align-items-center justify-content-center'
        )}
    >
        <span className={styles.batchChangeStatsCardChangesetsPill}>
            <span className="badge badge-pill badge-secondary">{count}</span>
        </span>
        <span className="text-muted">{pluralize('Changeset', count)}</span>
    </div>
)
