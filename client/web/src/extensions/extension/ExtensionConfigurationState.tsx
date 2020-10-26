import CheckIcon from 'mdi-react/CheckIcon'
import * as React from 'react'

/** Displays the extension's configuration state (not added, added and enabled, added and disabled). */
export const ExtensionConfigurationState: React.FunctionComponent<{
    isAdded: boolean
    isEnabled: boolean

    enabledIconOnly?: boolean

    className?: string
}> = ({ isAdded, isEnabled, enabledIconOnly, className = '' }) =>
    isAdded && isEnabled ? (
        <span className={`text-success ${className}`}>
            <CheckIcon className="icon-inline" /> {!enabledIconOnly && 'Enabled'}
        </span>
    ) : (
        <span className={`text-muted ${className}`}>Disabled</span>
    )
