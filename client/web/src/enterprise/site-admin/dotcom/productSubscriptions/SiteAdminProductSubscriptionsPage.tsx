import AddIcon from 'mdi-react/AddIcon'
import React, { useEffect } from 'react'
import { RouteComponentProps } from 'react-router'
import { Link } from 'react-router-dom'
import { Observable } from 'rxjs'
import { map } from 'rxjs/operators'
import { dataOrThrowErrors, gql } from '../../../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../../../shared/src/graphql/schema'
import { queryGraphQL } from '../../../../backend/graphql'
import { FilteredConnection } from '../../../../components/FilteredConnection'
import { PageTitle } from '../../../../components/PageTitle'
import { eventLogger } from '../../../../tracking/eventLogger'
import {
    siteAdminProductSubscriptionFragment,
    SiteAdminProductSubscriptionNode,
    SiteAdminProductSubscriptionNodeHeader,
    SiteAdminProductSubscriptionNodeProps,
} from './SiteAdminProductSubscriptionNode'

interface Props extends RouteComponentProps<{}> {}

class FilteredSiteAdminProductSubscriptionConnection extends FilteredConnection<
    GQL.IProductSubscription,
    SiteAdminProductSubscriptionNodeProps
> {}

/**
 * Displays the product subscriptions that have been created on Sourcegraph.com.
 */
export const SiteAdminProductSubscriptionsPage: React.FunctionComponent<Props> = ({ history, location }) => {
    useEffect(() => eventLogger.logViewEvent('SiteAdminProductSubscriptions'), [])
    return (
        <div className="site-admin-product-subscriptions-page">
            <PageTitle title="Product subscriptions" />
            <div className="d-flex justify-content-between align-items-center mb-3">
                <h2 className="mb-0">Product subscriptions</h2>
                <Link to="/site-admin/dotcom/product/subscriptions/new" className="btn btn-primary">
                    <AddIcon className="icon-inline" />
                    Create product subscription
                </Link>
            </div>
            <FilteredSiteAdminProductSubscriptionConnection
                className="mt-3"
                listComponent="table"
                listClassName="table"
                noun="product subscription"
                pluralNoun="product subscriptions"
                queryConnection={queryProductSubscriptions}
                headComponent={SiteAdminProductSubscriptionNodeHeader}
                nodeComponent={SiteAdminProductSubscriptionNode}
                history={history}
                location={location}
            />
        </div>
    )
}

function queryProductSubscriptions(args: {
    first?: number
    query?: string
}): Observable<GQL.IProductSubscriptionConnection> {
    return queryGraphQL(
        gql`
            query ProductSubscriptionsDotCom($first: Int, $account: ID, $query: String) {
                dotcom {
                    productSubscriptions(first: $first, account: $account, query: $query) {
                        nodes {
                            ...SiteAdminProductSubscriptionFields
                        }
                        totalCount
                        pageInfo {
                            hasNextPage
                        }
                    }
                }
            }
            ${siteAdminProductSubscriptionFragment}
        `,
        {
            first: args.first,
            query: args.query,
        } as GQL.IProductSubscriptionsOnDotcomQueryArguments
    ).pipe(
        map(dataOrThrowErrors),
        map(data => data.dotcom.productSubscriptions)
    )
}
