import InformationIcon from 'mdi-react/InformationIcon'
import KeyIcon from 'mdi-react/KeyIcon'
import React, { useState, useCallback } from 'react'
import { CopyableText } from '../../../components/CopyableText'
import { ExpirationDate } from '../../productSubscription/ExpirationDate'
import { formatUserCount, mailtoSales } from '../../productSubscription/helpers'
import { LicenseGenerationKeyWarning } from '../../productSubscription/LicenseGenerationKeyWarning'
import { ProductCertificate } from '../../productSubscription/ProductCertificate'

interface Props {
    subscriptionName: string
    productNameWithBrand: string
    userCount: number
    expiresAt: Date | number
    licenseKey: string | null
}

/**
 * Displays a certificate with information about and status for a user's product subscription. It
 * supports both billing-linked and non-billing-linked subscriptions.
 */
export const UserProductSubscriptionStatus: React.FunctionComponent<Props> = ({
    subscriptionName,
    productNameWithBrand,
    userCount,
    expiresAt,
    licenseKey,
}) => {
    const [showLicenseKey, setShowLicenseKey] = useState(false)

    const toggleShowLicenseKey = useCallback((): void => setShowLicenseKey(!showLicenseKey), [showLicenseKey])

    return (
        <ProductCertificate
            title={productNameWithBrand}
            subtitle={
                <>
                    {formatUserCount(userCount, true)} license,{' '}
                    <ExpirationDate date={expiresAt} showRelative={true} showPrefix={true} lowercase={true} />
                </>
            }
            footer={
                <>
                    <div className="card-footer d-flex align-items-center justify-content-between flex-wrap">
                        <button type="button" className="btn btn-primary mr-4 my-1" onClick={toggleShowLicenseKey}>
                            <KeyIcon className="icon-inline" /> {showLicenseKey ? 'Hide' : 'Reveal'} license key
                        </button>
                        <div className="flex-fill" />
                        <div className="my-1" />
                    </div>
                    {showLicenseKey && (
                        <div className="card-footer">
                            <h3>License key</h3>
                            {licenseKey ? (
                                <>
                                    <CopyableText text={licenseKey} className="d-block" />
                                    <small className="mt-2 d-flex align-items-center">
                                        <InformationIcon className="icon-inline mr-1" />{' '}
                                        <span>
                                            Use this license key as the{' '}
                                            <code>
                                                <strong>licenseKey</strong>
                                            </code>{' '}
                                            property value in Sourcegraph site configuration.
                                        </span>
                                    </small>
                                    <LicenseGenerationKeyWarning className="mb-0 mt-1" />
                                </>
                            ) : (
                                <div className="text-muted">
                                    No license key found.{' '}
                                    <a
                                        href={mailtoSales({
                                            subject: `No license key for subscription ${subscriptionName}`,
                                        })}
                                    >
                                        Contact sales
                                    </a>{' '}
                                    for help.
                                </div>
                            )}
                        </div>
                    )}
                </>
            }
        />
    )
}
