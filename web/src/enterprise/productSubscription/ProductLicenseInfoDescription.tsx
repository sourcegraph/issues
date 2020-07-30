import React from 'react'
import { formatUserCount } from './helpers'
import { ProductLicenseFields } from '../../graphql-operations'

export const ProductLicenseInfoDescription: React.FunctionComponent<{
    licenseInfo: NonNullable<ProductLicenseFields['info']>
    className?: string
}> = ({ licenseInfo, className = '' }) => (
    <span
        className={className}
        title={licenseInfo.tags.length > 0 ? `Tags: ${licenseInfo.tags.join(', ')}` : 'No tags'}
    >
        {licenseInfo.productNameWithBrand} ({formatUserCount(licenseInfo.userCount)})
    </span>
)
