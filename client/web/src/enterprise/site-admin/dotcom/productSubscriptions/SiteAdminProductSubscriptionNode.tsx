import * as React from 'react'
import { LinkOrSpan } from '../../../../../../shared/src/components/LinkOrSpan'
import { gql } from '../../../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../../../shared/src/graphql/schema'
import { Timestamp } from '../../../../components/time/Timestamp'
import { AccountName } from '../../../dotcom/productSubscriptions/AccountName'
import { ProductSubscriptionLabel } from '../../../dotcom/productSubscriptions/ProductSubscriptionLabel'
import { ProductLicenseTags } from '../../../productSubscription/ProductLicenseTags'

export const siteAdminProductSubscriptionFragment = gql`
    fragment SiteAdminProductSubscriptionFields on ProductSubscription {
        id
        name
        account {
            id
            username
            displayName
            emails {
                email
                isPrimary
            }
        }
        invoiceItem {
            plan {
                nameWithBrand
            }
            userCount
            expiresAt
        }
        activeLicense {
            id
            info {
                productNameWithBrand
                tags
                userCount
                expiresAt
            }
            licenseKey
            createdAt
        }
        createdAt
        isArchived
        urlForSiteAdmin
    }
`

export const SiteAdminProductSubscriptionNodeHeader: React.FunctionComponent = () => (
    <thead>
        <tr>
            <th>ID</th>
            <th>Customer</th>
            <th>Plan</th>
            <th>Expiration</th>
            <th>Tags</th>
        </tr>
    </thead>
)

export interface SiteAdminProductSubscriptionNodeProps {
    node: GQL.IProductSubscription
}

/**
 * Displays a product subscription in a connection in the site admin area.
 */
export const SiteAdminProductSubscriptionNode: React.FunctionComponent<SiteAdminProductSubscriptionNodeProps> = ({
    node,
}) => (
    <tr>
        <td className="text-nowrap">
            <LinkOrSpan to={node.urlForSiteAdmin} className="mr-3">
                {node.name}
            </LinkOrSpan>
        </td>
        <td className="w-100">
            <AccountName account={node.account} />
            {node.account && (
                <div>
                    <small>
                        {node.account.emails
                            .filter(email => email.isPrimary)
                            .map(({ email }) => email)
                            .join(', ')}
                    </small>
                </div>
            )}
        </td>
        <td className="text-nowrap">
            <ProductSubscriptionLabel productSubscription={node} className="mr-3" />
        </td>
        <td className="text-nowrap">
            {node.activeLicense?.info ? (
                <Timestamp date={node.activeLicense.info.expiresAt} />
            ) : (
                <span className="text-muted font-italic">None</span>
            )}
        </td>
        <td className="w-100">
            {node.activeLicense?.info && node.activeLicense.info.tags.length > 0 ? (
                <ProductLicenseTags tags={node.activeLicense.info.tags} />
            ) : (
                <span className="text-muted font-italic">None</span>
            )}
        </td>
    </tr>
)
