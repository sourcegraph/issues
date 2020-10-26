import React, { useEffect, useCallback } from 'react'
import { RouteComponentProps } from 'react-router'
import { Link } from 'react-router-dom'
import { Observable } from 'rxjs'
import { map } from 'rxjs/operators'
import { gql } from '../../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../../shared/src/graphql/schema'
import { createAggregateError } from '../../../../../shared/src/util/errors'
import { queryGraphQL } from '../../../backend/graphql'
import { FilteredConnection } from '../../../components/FilteredConnection'
import { PageTitle } from '../../../components/PageTitle'
import { eventLogger } from '../../../tracking/eventLogger'
import {
    productSubscriptionFragment,
    ProductSubscriptionNode,
    ProductSubscriptionNodeHeader,
    ProductSubscriptionNodeProps,
} from '../../dotcom/productSubscriptions/ProductSubscriptionNode'
import { UserAreaUserFields } from '../../../graphql-operations'

interface Props extends RouteComponentProps<{}> {
    user: UserAreaUserFields
}

class FilteredProductSubscriptionConnection extends FilteredConnection<
    GQL.IProductSubscription,
    ProductSubscriptionNodeProps
> {}

/**
 * Displays the product subscriptions associated with this account.
 */
export const UserSubscriptionsProductSubscriptionsPage: React.FunctionComponent<Props> = props => {
    useEffect(() => {
        eventLogger.logViewEvent('UserSubscriptionsProductSubscriptions')
    }, [])

    const queryLicenses = useCallback(
        (args: { first?: number }): Observable<GQL.IProductSubscriptionConnection> => {
            const vars: GQL.IProductSubscriptionsOnDotcomQueryArguments = {
                first: args.first,
                account: props.user.id,
            }
            return queryGraphQL(
                gql`
                    query ProductSubscriptions($first: Int, $account: ID) {
                        dotcom {
                            productSubscriptions(first: $first, account: $account) {
                                nodes {
                                    ...ProductSubscriptionFields
                                }
                                totalCount
                                pageInfo {
                                    hasNextPage
                                }
                            }
                        }
                    }
                    ${productSubscriptionFragment}
                `,
                vars
            ).pipe(
                map(({ data, errors }) => {
                    if (!data || !data.dotcom || !data.dotcom.productSubscriptions || (errors && errors.length > 0)) {
                        throw createAggregateError(errors)
                    }
                    return data.dotcom.productSubscriptions
                })
            )
        },
        [props.user.id]
    )

    return (
        <div className="user-subscriptions-product-subscriptions-page">
            <PageTitle title="Subscriptions" />
            <div className="d-flex justify-content-between align-items-center mb-3">
                <h2 className="mb-0">Subscriptions</h2>
                <Link to={`${props.match.path}/new`} className="btn btn-primary">
                    New subscription
                </Link>
            </div>
            <p>
                A subscription gives you a license key to run a self-hosted Sourcegraph instance. See{' '}
                <a href="https://about.sourcegraph.com/pricing">Sourcegraph pricing</a> for more information.
            </p>
            <FilteredProductSubscriptionConnection
                className="mt-3"
                listComponent="table"
                listClassName="table"
                noun="subscription"
                pluralNoun="subscriptions"
                queryConnection={queryLicenses}
                headComponent={ProductSubscriptionNodeHeader}
                nodeComponent={ProductSubscriptionNode}
                hideSearch={true}
                noSummaryIfAllNodesVisible={true}
                history={props.history}
                location={props.location}
            />
        </div>
    )
}
