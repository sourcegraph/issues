import React from 'react'
import { ChangesetFields } from '../../../../graphql-operations'
import SourceBranchIcon from 'mdi-react/SourceBranchIcon'
import SourcePullIcon from 'mdi-react/SourcePullIcon'
import SourceMergeIcon from 'mdi-react/SourceMergeIcon'
import DeleteIcon from 'mdi-react/DeleteIcon'
import ErrorIcon from 'mdi-react/ErrorIcon'
import TimerSandIcon from 'mdi-react/TimerSandIcon'
import classNames from 'classnames'
import { computeChangesetUIState, ChangesetUIState } from '../../utils'

export interface ChangesetStatusCellProps {
    className?: string
    changeset: Pick<ChangesetFields, 'publicationState' | 'externalState' | 'reconcilerState'>
}

export const ChangesetStatusCell: React.FunctionComponent<ChangesetStatusCellProps> = ({
    changeset,
    className = 'd-flex',
}) => {
    switch (computeChangesetUIState(changeset)) {
        case ChangesetUIState.ERRORED:
            return <ChangesetStatusError className={className} />
        case ChangesetUIState.PROCESSING:
            return <ChangesetStatusProcessing className={className} />
        case ChangesetUIState.UNPUBLISHED:
            return <ChangesetStatusUnpublished className={className} />
        case ChangesetUIState.OPEN:
            return <ChangesetStatusOpen className={className} />
        case ChangesetUIState.DRAFT:
            return <ChangesetStatusDraft className={className} />
        case ChangesetUIState.CLOSED:
            return <ChangesetStatusClosed className={className} />
        case ChangesetUIState.MERGED:
            return <ChangesetStatusMerged className={className} />
        case ChangesetUIState.DELETED:
            return <ChangesetStatusDeleted className={className} />
    }
}

const iconClassNames = 'm-0 text-nowrap flex-column align-items-center justify-content-center'

export const ChangesetStatusUnpublished: React.FunctionComponent<{ label?: JSX.Element; className?: string }> = ({
    label = <span className="text-muted">Unpublished</span>,
    className,
}) => (
    <div
        className={classNames(iconClassNames, 'text-muted', className)}
        data-tooltip="Set published: true to publish to code host"
    >
        <SourceBranchIcon />
        {label}
    </div>
)
export const ChangesetStatusClosed: React.FunctionComponent<{ label?: JSX.Element; className?: string }> = ({
    label = <span className="text-muted">Closed</span>,
    className,
}) => (
    <div className={classNames(iconClassNames, 'text-danger', className)}>
        <SourcePullIcon />
        {label}
    </div>
)
export const ChangesetStatusMerged: React.FunctionComponent<{ label?: JSX.Element; className?: string }> = ({
    label = <span className="text-muted">Merged</span>,
    className,
}) => (
    <div className={classNames(iconClassNames, 'text-merged', className)}>
        <SourceMergeIcon />
        {label}
    </div>
)
export const ChangesetStatusOpen: React.FunctionComponent<{ label?: JSX.Element; className?: string }> = ({
    label = <span className="text-muted">Open</span>,
    className,
}) => (
    <div className={classNames(iconClassNames, 'text-success', className)}>
        <SourcePullIcon />
        {label}
    </div>
)
export const ChangesetStatusDraft: React.FunctionComponent<{ label?: JSX.Element; className?: string }> = ({
    label = <span className="text-muted">Draft</span>,
    className,
}) => (
    <div className={classNames(iconClassNames, 'text-muted', className)}>
        <SourcePullIcon />
        {label}
    </div>
)
export const ChangesetStatusDeleted: React.FunctionComponent<{ label?: JSX.Element; className?: string }> = ({
    label = <span className="text-muted">Deleted</span>,
    className,
}) => (
    <div className={classNames(iconClassNames, 'text-muted', className)}>
        <DeleteIcon />
        {label}
    </div>
)
export const ChangesetStatusError: React.FunctionComponent<{ label?: JSX.Element; className?: string }> = ({
    label = <span className="text-danger">Error</span>,
    className,
}) => (
    <div className={classNames(iconClassNames, 'text-danger', className)}>
        <ErrorIcon />
        {label}
    </div>
)
export const ChangesetStatusProcessing: React.FunctionComponent<{ label?: JSX.Element; className?: string }> = ({
    label = <span className="text-muted">Processing</span>,
    className,
}) => (
    <div className={classNames(iconClassNames, className)}>
        <TimerSandIcon />
        {label}
    </div>
)
