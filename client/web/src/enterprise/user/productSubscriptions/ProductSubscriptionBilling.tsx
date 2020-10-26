import React from 'react'
import { Link } from 'react-router-dom'
import * as GQL from '../../../../../shared/src/graphql/schema'
import { ProductSubscriptionLabel } from '../../dotcom/productSubscriptions/ProductSubscriptionLabel'

export const ProductSubscriptionBilling: React.FunctionComponent<{
    productSubscription: GQL.IProductSubscription
}> = ({ productSubscription }) => (
    <table className="table mb-0">
        <tbody>
            <tr>
                <th className="text-nowrap align-middle">Plan</th>
                <td className="w-100 d-flex align-items-center justify-content-between">
                    <ProductSubscriptionLabel productSubscription={productSubscription} planField="name" />
                    <Link to={`${productSubscription.url}/edit`} className="btn btn-secondary btn-sm">
                        Change plan or add/remove users
                    </Link>
                </td>
            </tr>
        </tbody>
    </table>
)
