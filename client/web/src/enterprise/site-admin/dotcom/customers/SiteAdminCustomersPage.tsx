import React, { useEffect, useMemo, useCallback } from 'react'
import { RouteComponentProps } from 'react-router'
import { Observable, Subject } from 'rxjs'
import { map } from 'rxjs/operators'
import { gql } from '../../../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../../../shared/src/graphql/schema'
import { createAggregateError } from '../../../../../../shared/src/util/errors'
import { queryGraphQL } from '../../../../backend/graphql'
import { FilteredConnection } from '../../../../components/FilteredConnection'
import { PageTitle } from '../../../../components/PageTitle'
import { eventLogger } from '../../../../tracking/eventLogger'
import { userURL } from '../../../../user'
import { AccountName } from '../../../dotcom/productSubscriptions/AccountName'
import { SiteAdminCustomerBillingLink } from './SiteAdminCustomerBillingLink'

const siteAdminCustomerFragment = gql`
    fragment CustomerFields on User {
        id
        username
        displayName
        urlForSiteAdminBilling
    }
`

interface SiteAdminCustomerNodeProps {
    node: Pick<GQL.IUser, 'id' | 'username' | 'displayName' | 'urlForSiteAdminBilling'>
    onDidUpdate: () => void
}

/**
 * Displays a customer in a connection in the site admin area.
 */
const SiteAdminCustomerNode: React.FunctionComponent<SiteAdminCustomerNodeProps> = ({ node, onDidUpdate }) => (
    <li className="list-group-item py-2">
        <div className="d-flex align-items-center justify-content-between">
            <span className="mr-3">
                <AccountName account={node} link={`${userURL(node.username)}/subscriptions`} />
            </span>
            <SiteAdminCustomerBillingLink customer={node} onDidUpdate={onDidUpdate} />
        </div>
    </li>
)

interface Props extends RouteComponentProps<{}> {}

class FilteredSiteAdminCustomerConnection extends FilteredConnection<
    Pick<GQL.IUser, 'id' | 'username' | 'displayName' | 'urlForSiteAdminBilling'>,
    Pick<SiteAdminCustomerNodeProps, Exclude<keyof SiteAdminCustomerNodeProps, 'node'>>
> {}

/**
 * Displays a list of customers associated with user accounts on Sourcegraph.com.
 */
export const SiteAdminProductCustomersPage: React.FunctionComponent<Props> = props => {
    useEffect(() => eventLogger.logViewEvent('SiteAdminProductCustomers'), [])

    const updates = useMemo(() => new Subject<void>(), [])
    const onUserUpdate = useCallback(() => updates.next(), [updates])
    const nodeProps: Pick<SiteAdminCustomerNodeProps, Exclude<keyof SiteAdminCustomerNodeProps, 'node'>> = {
        onDidUpdate: onUserUpdate,
    }

    return (
        <div className="site-admin-customers-page">
            <PageTitle title="Customers" />
            <div className="d-flex justify-content-between align-items-center mb-1">
                <h2 className="mb-0">Customers</h2>
            </div>
            <p>User accounts may be linked to a customer on the billing system.</p>
            <FilteredSiteAdminCustomerConnection
                className="list-group list-group-flush mt-3"
                noun="customer"
                pluralNoun="customers"
                queryConnection={queryCustomers}
                nodeComponent={SiteAdminCustomerNode}
                nodeComponentProps={nodeProps}
                noSummaryIfAllNodesVisible={true}
                updates={updates}
                history={props.history}
                location={props.location}
            />
        </div>
    )
}

function queryCustomers(args: { first?: number; query?: string }): Observable<GQL.IUserConnection> {
    return queryGraphQL(
        gql`
            query Customers($first: Int, $query: String) {
                users(first: $first, query: $query) {
                    nodes {
                        ...CustomerFields
                    }
                    totalCount
                    pageInfo {
                        hasNextPage
                    }
                }
            }
            ${siteAdminCustomerFragment}
        `,
        {
            first: args.first,
            query: args.query,
        } as GQL.IUsersOnQueryArguments
    ).pipe(
        map(({ data, errors }) => {
            if (!data || !data.users || (errors && errors.length > 0)) {
                throw createAggregateError(errors)
            }
            return data.users
        })
    )
}
