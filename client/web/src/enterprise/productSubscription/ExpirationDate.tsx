import format from 'date-fns/format'
import React from 'react'
import { formatRelativeExpirationDate, isProductLicenseExpired } from './helpers'

/** Displays an expiration date (for product subscriptions or licenses). */
export const ExpirationDate: React.FunctionComponent<{
    /** The expiration date of a product subscription or license. */
    date: Date | number

    /** Show the time of day of expiration. */
    showTime?: boolean

    /** Show "(T remaining)" or "(T ago)" relative times. */
    showRelative?: boolean

    /** Show the "Expired on" or "Valid until" prefix. */
    showPrefix?: boolean

    lowercase?: boolean
}> = ({ date, showTime, showRelative, showPrefix, lowercase }) => {
    let text: string | undefined
    if (showPrefix) {
        text = isProductLicenseExpired(date) ? 'Expired on ' : 'Valid until '
    }
    return (
        <span>
            {text && lowercase ? text.toLowerCase() : text}
            {showTime ? format(date, 'PPpp') : <span title={format(date, 'PPpp')}>{format(date, 'yyyy-MM-dd')}</span>}
            {showRelative && ` (${formatRelativeExpirationDate(date)})`}
        </span>
    )
}
