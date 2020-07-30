import * as React from 'react'
import { Link } from 'react-router-dom'
import { gql } from '../../../../../shared/src/graphql/graphql'
import { ProductSubscriptionLabel } from './ProductSubscriptionLabel'
import { ProductSubscriptionFields } from '../../../graphql-operations'

export const productSubscriptionFragment = gql`
    fragment ProductSubscriptionFields on ProductSubscription {
        id
        name
        account {
            id
            username
            displayName
            emails {
                email
                verified
            }
        }
        invoiceItem {
            plan {
                name
                nameWithBrand
            }
            userCount
            expiresAt
        }
        activeLicense {
            licenseKey
            info {
                productNameWithBrand
                tags
                userCount
                expiresAt
            }
        }
        createdAt
        isArchived
        url
    }
`

export const ProductSubscriptionNodeHeader: React.FunctionComponent = () => (
    <thead>
        <tr>
            <th>ID</th>
            <th>Plan</th>
        </tr>
    </thead>
)

export interface ProductSubscriptionNodeProps {
    node: ProductSubscriptionFields
}

export const ProductSubscriptionNode: React.FunctionComponent<ProductSubscriptionNodeProps> = ({ node }) => (
    <tr>
        <td className="text-nowrap">
            <Link to={node.url} className="mr-3 font-weight-bold">
                {node.name}
            </Link>
        </td>
        <td className="w-100">
            <ProductSubscriptionLabel productSubscription={node} className="mr-3" />
        </td>
    </tr>
)
