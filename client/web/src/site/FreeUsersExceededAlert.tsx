import WarningIcon from 'mdi-react/WarningIcon'
import * as React from 'react'

/**
 * A global alert that appears telling all users that they have exceeded the limit of free users allowed.
 */
export const FreeUsersExceededAlert: React.FunctionComponent<{
    noLicenseWarningUserCount: number | null
    className?: string
}> = ({ noLicenseWarningUserCount, className = '' }) => (
    <div className={`alert alert-danger ${className}`}>
        <WarningIcon className="icon-inline mr-2" />
        This Sourcegraph instance has reached{' '}
        {noLicenseWarningUserCount === null ? 'the limit for' : noLicenseWarningUserCount} free users, and an admin must{' '}
        <a className="site-alert__link" href="https://sourcegraph.com/user/subscriptions/new">
            <span className="underline">purchase a license</span>
        </a>{' '}
        or{' '}
        <a className="site-alert__link" href="https://about.sourcegraph.com/contact/sales">
            <span className="underline">contact Sourcegraph for a free trial</span>
        </a>{' '}
        to add more
    </div>
)
