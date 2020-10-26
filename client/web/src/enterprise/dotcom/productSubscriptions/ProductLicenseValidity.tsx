import { parseISO } from 'date-fns'
import format from 'date-fns/format'
import React from 'react'
import * as GQL from '../../../../../shared/src/graphql/schema'
import { formatRelativeExpirationDate, isProductLicenseExpired } from '../../productSubscription/helpers'

/**
 * Displays an alert indicating the validity of a product license.
 */
export const ProductLicenseValidity: React.FunctionComponent<{
    licenseInfo: GQL.IProductLicenseInfo
    primary: boolean
    className?: string
}> = ({ licenseInfo: { expiresAt }, primary, className = '' }) => {
    const isExpired = isProductLicenseExpired(expiresAt)
    const validityClass = isExpired ? 'danger' : 'success'
    return (
        <div
            className={`${className} ${primary ? `alert alert-${validityClass} py-1 px-2` : ''}`}
            data-tooltip={format(parseISO(expiresAt), 'PPpp')}
        >
            <strong className={primary ? '' : `text-${validityClass}`}>{isExpired ? 'Expired' : 'Valid'}</strong> (
            {formatRelativeExpirationDate(expiresAt)})
        </div>
    )
}
