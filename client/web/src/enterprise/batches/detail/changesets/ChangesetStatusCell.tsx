import classNames from 'classnames'
import ArchiveIcon from 'mdi-react/ArchiveIcon'
import AutorenewIcon from 'mdi-react/AutorenewIcon'
import DeleteIcon from 'mdi-react/DeleteIcon'
import ErrorIcon from 'mdi-react/ErrorIcon'
import SourceBranchIcon from 'mdi-react/SourceBranchIcon'
import SourceMergeIcon from 'mdi-react/SourceMergeIcon'
import SourcePullIcon from 'mdi-react/SourcePullIcon'
import TimerSandIcon from 'mdi-react/TimerSandIcon'
import React from 'react'

import { ChangesetFields, ChangesetState, Scalars } from '../../../../graphql-operations'

import { ChangesetStatusScheduled } from './ChangesetStatusScheduled'

export interface ChangesetStatusCellProps {
    className?: string
    id?: Scalars['ID']
    state: ChangesetFields['state']
}

export const ChangesetStatusCell: React.FunctionComponent<ChangesetStatusCellProps> = ({
    id,
    state,
    className = 'd-flex',
}) => {
    switch (state) {
        case ChangesetState.FAILED:
            return <ChangesetStatusError className={className} />
        case ChangesetState.RETRYING:
            return <ChangesetStatusRetrying className={className} />
        case ChangesetState.SCHEDULED:
            return <ChangesetStatusScheduled className={className} id={id} />
        case ChangesetState.PROCESSING:
            return <ChangesetStatusProcessing className={className} />
        case ChangesetState.UNPUBLISHED:
            return <ChangesetStatusUnpublished className={className} />
        case ChangesetState.OPEN:
            return <ChangesetStatusOpen className={className} />
        case ChangesetState.DRAFT:
            return <ChangesetStatusDraft className={className} />
        case ChangesetState.CLOSED:
            return <ChangesetStatusClosed className={className} />
        case ChangesetState.MERGED:
            return <ChangesetStatusMerged className={className} />
        case ChangesetState.DELETED:
            return <ChangesetStatusDeleted className={className} />
    }
}

const iconClassNames = 'm-0 text-nowrap flex-column align-items-center justify-content-center'

export const ChangesetStatusUnpublished: React.FunctionComponent<{ label?: JSX.Element; className?: string }> = ({
    label = <span>Unpublished</span>,
    className,
}) => (
    <div className={classNames(iconClassNames, className)}>
        <SourceBranchIcon />
        {label}
    </div>
)
export const ChangesetStatusClosed: React.FunctionComponent<{ label?: JSX.Element; className?: string }> = ({
    label = <span>Closed</span>,
    className,
}) => (
    <div className={classNames(iconClassNames, className)}>
        <SourcePullIcon className="text-danger" />
        {label}
    </div>
)
export const ChangesetStatusMerged: React.FunctionComponent<{ label?: JSX.Element; className?: string }> = ({
    label = <span>Merged</span>,
    className,
}) => (
    <div className={classNames(iconClassNames, className)}>
        <SourceMergeIcon className="text-merged" />
        {label}
    </div>
)
export const ChangesetStatusOpen: React.FunctionComponent<{ label?: JSX.Element; className?: string }> = ({
    label = <span>Open</span>,
    className,
}) => (
    <div className={classNames(iconClassNames, className)}>
        <SourcePullIcon className="text-success" />
        {label}
    </div>
)
export const ChangesetStatusDraft: React.FunctionComponent<{ label?: JSX.Element; className?: string }> = ({
    label = <span>Draft</span>,
    className,
}) => (
    <div className={classNames(iconClassNames, className)}>
        <SourcePullIcon />
        {label}
    </div>
)
export const ChangesetStatusDeleted: React.FunctionComponent<{ label?: JSX.Element; className?: string }> = ({
    label = <span>Deleted</span>,
    className,
}) => (
    <div className={classNames(iconClassNames, className)}>
        <DeleteIcon />
        {label}
    </div>
)
export const ChangesetStatusError: React.FunctionComponent<{ label?: JSX.Element; className?: string }> = ({
    label = <span className="text-danger">Failed</span>,
    className,
}) => (
    <div className={classNames(iconClassNames, className)}>
        <ErrorIcon className="text-danger" />
        {label}
    </div>
)
export const ChangesetStatusRetrying: React.FunctionComponent<{ label?: JSX.Element; className?: string }> = ({
    label = <span>Retrying</span>,
    className,
}) => (
    <div className={classNames(iconClassNames, className)}>
        <AutorenewIcon />
        {label}
    </div>
)

export const ChangesetStatusProcessing: React.FunctionComponent<{ label?: JSX.Element; className?: string }> = ({
    label = <span>Processing</span>,
    className,
}) => (
    <div className={classNames(iconClassNames, className)}>
        <TimerSandIcon />
        {label}
    </div>
)

export const ChangesetStatusArchived: React.FunctionComponent<{ label?: JSX.Element; className?: string }> = ({
    label = <span>Archived</span>,
    className,
}) => (
    <div className={classNames(iconClassNames, className)}>
        <ArchiveIcon />
        {label}
    </div>
)
