import { parseISO } from 'date-fns'
import React, { useMemo } from 'react'
import { Link } from 'react-router-dom'
import { Observable } from 'rxjs'
import { catchError, map } from 'rxjs/operators'
import { gql, dataOrThrowErrors } from '../../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../../shared/src/graphql/schema'
import { asError, ErrorLike, isErrorLike } from '../../../../../shared/src/util/errors'
import { numberWithCommas } from '../../../../../shared/src/util/strings'
import { queryGraphQL } from '../../../backend/graphql'
import { ExpirationDate } from '../../productSubscription/ExpirationDate'
import { formatUserCount } from '../../productSubscription/helpers'
import { ProductCertificate } from '../../productSubscription/ProductCertificate'
import { TrueUpStatusSummary } from '../../productSubscription/TrueUpStatusSummary'
import { ErrorAlert } from '../../../components/alerts'
import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { useObservable } from '../../../../../shared/src/util/useObservable'
import * as H from 'history'

const queryProductLicenseInfo = (): Observable<GQL.IProductSubscriptionStatus> =>
    queryGraphQL(gql`
        query ProductLicenseInfo {
            site {
                productSubscription {
                    productNameWithBrand
                    actualUserCount
                    actualUserCountDate
                    noLicenseWarningUserCount
                    license {
                        tags
                        userCount
                        expiresAt
                    }
                }
            }
        }
    `).pipe(
        map(dataOrThrowErrors),
        map(data => data.site.productSubscription)
    )

interface Props {
    className?: string

    /**
     * If true, always show the license true-up status.
     * If undefined or false, never show the full license true-up status, and instead only show an alert
     * if the user count is over the license limit.
     *
     */
    showTrueUpStatus?: boolean
    history: H.History
}

/**
 * A component displaying information about and the status of the product subscription.
 */
export const ProductSubscriptionStatus: React.FunctionComponent<Props> = ({ className, showTrueUpStatus, history }) => {
    /** The product subscription status, or an error, or undefined while loading. */
    const statusOrError = useObservable(
        useMemo(() => queryProductLicenseInfo().pipe(catchError((error): [ErrorLike] => [asError(error)])), [])
    )
    if (statusOrError === undefined) {
        return (
            <div className="text-center">
                <LoadingSpinner className="icon-inline" />
            </div>
        )
    }
    if (isErrorLike(statusOrError)) {
        return <ErrorAlert error={statusOrError} prefix="Error checking product license" history={history} />
    }

    const {
        productNameWithBrand,
        actualUserCount,
        actualUserCountDate,
        noLicenseWarningUserCount,
        license,
    } = statusOrError

    // No license means Sourcegraph Free. For that, show the user that they can use this for free
    // forever, and show them how to upgrade.

    return (
        <div>
            <ProductCertificate
                title={productNameWithBrand}
                detail={
                    license ? (
                        <>
                            {formatUserCount(license.userCount, true)} license,{' '}
                            <ExpirationDate
                                date={parseISO(license.expiresAt)}
                                showRelative={true}
                                lowercase={true}
                                showPrefix={true}
                            />
                        </>
                    ) : null
                }
                footer={
                    <div className="card-footer d-flex align-items-center justify-content-between">
                        {license ? (
                            <>
                                <div>
                                    <strong>User licenses:</strong> {numberWithCommas(actualUserCount)} used /{' '}
                                    {numberWithCommas(license.userCount - actualUserCount)} remaining
                                </div>
                                <a
                                    href="https://about.sourcegraph.com/pricing"
                                    className="btn btn-primary btn-sm"
                                    // eslint-disable-next-line react/jsx-no-target-blank
                                    target="_blank"
                                    rel="noopener"
                                >
                                    Upgrade
                                </a>
                            </>
                        ) : (
                            <>
                                <div className="mr-2">
                                    Add a license key to activate Sourcegraph Enterprise features{' '}
                                    {typeof noLicenseWarningUserCount === 'number'
                                        ? `or to exceed ${noLicenseWarningUserCount} users`
                                        : ''}
                                </div>
                                <div className="text-nowrap flex-wrap-reverse">
                                    <a
                                        href="http://about.sourcegraph.com/contact/sales"
                                        className="btn btn-primary btn-sm"
                                        // eslint-disable-next-line react/jsx-no-target-blank
                                        target="_blank"
                                        rel="noopener"
                                        data-tooltip="Buy a Sourcegraph Enterprise subscription to get a license key"
                                    >
                                        Get license
                                    </a>
                                </div>
                            </>
                        )}
                    </div>
                }
                className={className}
            />
            {license &&
                (showTrueUpStatus ? (
                    <TrueUpStatusSummary
                        actualUserCount={actualUserCount}
                        actualUserCountDate={actualUserCountDate}
                        license={license}
                    />
                ) : (
                    license.userCount - actualUserCount < 0 && (
                        <div className="alert alert-warning">
                            You have exceeded your licensed users.{' '}
                            <Link to="/site-admin/license">View your license details</Link> or{' '}
                            {/* eslint-disable-next-line react/jsx-no-target-blank */}
                            <a href="https://about.sourcegraph.com/pricing" target="_blank" rel="noopener">
                                upgrade your license
                            </a>{' '}
                            to true up and prevent a retroactive charge.
                        </div>
                    )
                ))}
        </div>
    )
}
