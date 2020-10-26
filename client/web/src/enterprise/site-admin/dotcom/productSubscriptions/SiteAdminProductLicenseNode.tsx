import * as React from 'react'
import { LinkOrSpan } from '../../../../../../shared/src/components/LinkOrSpan'
import { gql } from '../../../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../../../shared/src/graphql/schema'
import { CopyableText } from '../../../../components/CopyableText'
import { Timestamp } from '../../../../components/time/Timestamp'
import { AccountName } from '../../../dotcom/productSubscriptions/AccountName'
import { ProductLicenseValidity } from '../../../dotcom/productSubscriptions/ProductLicenseValidity'
import { ProductLicenseInfoDescription } from '../../../productSubscription/ProductLicenseInfoDescription'
import { ProductLicenseTags } from '../../../productSubscription/ProductLicenseTags'

export const siteAdminProductLicenseFragment = gql`
    fragment ProductLicenseFields on ProductLicense {
        id
        subscription {
            id
            name
            account {
                id
                username
                displayName
            }
            activeLicense {
                id
            }
            urlForSiteAdmin
        }
        licenseKey
        info {
            productNameWithBrand
            tags
            userCount
            expiresAt
        }
        createdAt
    }
`

export interface SiteAdminProductLicenseNodeProps {
    node: GQL.IProductLicense
    showSubscription: boolean
}

/**
 * Displays a product license in a connection in the site admin area.
 */
export const SiteAdminProductLicenseNode: React.FunctionComponent<SiteAdminProductLicenseNodeProps> = ({
    node,
    showSubscription,
}) => (
    <li className="list-group-item py-2">
        <div className="d-flex align-items-center justify-content-between">
            {showSubscription && (
                <div className="mr-3 text-truncate">
                    <strong>
                        License in{' '}
                        <LinkOrSpan to={node.subscription.urlForSiteAdmin} className="mr-3">
                            {node.subscription.name}
                        </LinkOrSpan>
                    </strong>
                    <span className="mr-3">
                        <AccountName account={node.subscription.account} />
                    </span>
                </div>
            )}
            <div>
                {node.info && <ProductLicenseInfoDescription licenseInfo={node.info} className="mr-3" />}
                {node.info && node.subscription.activeLicense && node.subscription.activeLicense.id === node.id ? (
                    <ProductLicenseValidity licenseInfo={node.info} primary={false} className="d-inline-block mr-3" />
                ) : (
                    <span
                        className="text-warning font-weight-bold mr-3"
                        data-tooltip="A newer license was generated for this subscription. This license should no longer be used."
                    >
                        Inactive
                    </span>
                )}
                <span className="text-muted">
                    Created <Timestamp date={node.createdAt} />
                </span>
            </div>
        </div>
        {node.info && node.info.tags.length > 0 && (
            <div>
                Tags: <ProductLicenseTags tags={node.info.tags} />
            </div>
        )}
        <CopyableText text={node.licenseKey} className="mt-2 d-block" />
    </li>
)
