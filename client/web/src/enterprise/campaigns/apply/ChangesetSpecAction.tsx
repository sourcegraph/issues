import React from 'react'
import { ChangesetSpecFields } from '../../../graphql-operations'
import BlankCircleIcon from 'mdi-react/CheckboxBlankCircleOutlineIcon'
import ImportIcon from 'mdi-react/ImportIcon'
import UploadIcon from 'mdi-react/UploadIcon'
import classNames from 'classnames'

export interface ChangesetSpecActionProps {
    spec: ChangesetSpecFields
    className?: string
}

export const ChangesetSpecAction: React.FunctionComponent<ChangesetSpecActionProps> = ({ spec, className }) => {
    if (spec.__typename === 'HiddenChangesetSpec') {
        return (
            <ChangesetSpecActionNoAction
                reason={NoActionReasonStrings[NoActionReason.NO_ACCESS]}
                className={className}
            />
        )
    }
    if (spec.description.__typename === 'ExistingChangesetReference') {
        return <ChangesetSpecActionImport className={className} />
    }
    if (spec.description.published === true) {
        return <ChangesetSpecActionPublish className={className} />
    }
    if (spec.description.published === 'draft') {
        return <ChangesetSpecActionPublishDraft className={className} />
    }
    return <ChangesetSpecActionNoPublish className={className} />
}

const iconClassNames = 'm-0 text-nowrap d-block d-sm-flex flex-column align-items-center justify-content-center'

export const ChangesetSpecActionPublish: React.FunctionComponent<{ className?: string }> = ({ className }) => (
    <div className={classNames(className, iconClassNames)}>
        <UploadIcon data-tooltip="This changeset will be published to its code host" />
        <span>Publish</span>
    </div>
)
export const ChangesetSpecActionPublishDraft: React.FunctionComponent<{ className?: string }> = ({ className }) => (
    <div className={classNames(className, iconClassNames)}>
        <UploadIcon className="text-muted" data-tooltip="This changeset will be published as draft to its code host" />
        <span>Publish draft</span>
    </div>
)
export const ChangesetSpecActionNoPublish: React.FunctionComponent<{ className?: string }> = ({ className }) => (
    <div className={classNames(className, iconClassNames, 'text-muted')}>
        <BlankCircleIcon data-tooltip="Set publish: true in the changeset template to publish to the code host" />
        <span>Won't publish</span>
    </div>
)
export const ChangesetSpecActionImport: React.FunctionComponent<{ className?: string }> = ({ className }) => (
    <div className={classNames(className, iconClassNames)}>
        <ImportIcon data-tooltip="This changeset will be imported and tracked in this campaign" />
        <span>Import</span>
    </div>
)
export enum NoActionReason {
    NO_ACCESS = 'no-access',
}
export const NoActionReasonStrings: Record<NoActionReason, string> = {
    [NoActionReason.NO_ACCESS]: "You don't have access to the repository this changeset spec targets.",
}
export const ChangesetSpecActionNoAction: React.FunctionComponent<{ className?: string; reason: string }> = ({
    className,
    reason,
}) => (
    <div className={classNames(className, iconClassNames, 'text-muted')}>
        <BlankCircleIcon data-tooltip={reason} />
        <span>No action</span>
    </div>
)
