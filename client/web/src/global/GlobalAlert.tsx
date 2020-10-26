import ErrorIcon from 'mdi-react/ErrorIcon'
import InformationIcon from 'mdi-react/InformationIcon'
import WarningIcon from 'mdi-react/WarningIcon'
import React from 'react'
import { Markdown } from '../../../shared/src/components/Markdown'
import * as GQL from '../../../shared/src/graphql/schema'
import { renderMarkdown } from '../../../shared/src/util/markdown'
import { DismissibleAlert } from '../components/DismissibleAlert'
import * as H from 'history'
import { AlertType } from '../../../shared/src/graphql-operations'

/**
 * A global alert that is shown at the top of the viewport.
 */
export const GlobalAlert: React.FunctionComponent<{ alert: GQL.IAlert; className: string; history: H.History }> = ({
    alert,
    history,
    className: commonClassName,
}) => {
    const Icon = alertIconForType(alert.type)
    const content = (
        <>
            <Icon className="icon-inline mr-2 flex-shrink-0" />
            <Markdown dangerousInnerHTML={renderMarkdown(alert.message)} history={history} />
        </>
    )
    const className = `${commonClassName} alert alert-${alertClassForType(alert.type)} d-flex`
    if (alert.isDismissibleWithKey) {
        return (
            <DismissibleAlert partialStorageKey={`alert.${alert.isDismissibleWithKey}`} className={className}>
                {content}
            </DismissibleAlert>
        )
    }
    return <div className={className}>{content}</div>
}

function alertIconForType(type: AlertType): React.ComponentType<{ className?: string }> {
    switch (type) {
        case AlertType.INFO:
            return InformationIcon
        case AlertType.WARNING:
            return WarningIcon
        case AlertType.ERROR:
            return ErrorIcon
        default:
            return WarningIcon
    }
}

function alertClassForType(type: AlertType): string {
    switch (type) {
        case AlertType.INFO:
            return 'info'
        case AlertType.WARNING:
            return 'warning'
        case AlertType.ERROR:
            return 'danger'
        default:
            return 'warning'
    }
}
