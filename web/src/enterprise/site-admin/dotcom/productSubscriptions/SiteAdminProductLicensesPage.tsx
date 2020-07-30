import React, { useEffect } from 'react'
import { RouteComponentProps } from 'react-router'
import { Observable, of } from 'rxjs'
import { map } from 'rxjs/operators'
import { gql } from '../../../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../../../shared/src/graphql/schema'
import { createAggregateError } from '../../../../../../shared/src/util/errors'
import { queryGraphQL } from '../../../../backend/graphql'
import { FilteredConnection } from '../../../../components/FilteredConnection'
import { PageTitle } from '../../../../components/PageTitle'
import { eventLogger } from '../../../../tracking/eventLogger'
import {
    siteAdminProductLicenseFragment,
    SiteAdminProductLicenseNode,
    SiteAdminProductLicenseNodeProps,
} from './SiteAdminProductLicenseNode'
import { DotComProductLicensesResult, DotComProductLicensesVariables } from '../../../../graphql-operations'

interface Props extends RouteComponentProps<{}> {}

class FilteredProductLicenseConnection extends FilteredConnection<
    GQL.ProductLicense,
    Pick<SiteAdminProductLicenseNodeProps, 'showSubscription'>
> {}

/**
 * Displays the product licenses that have been created on Sourcegraph.com.
 */
export const SiteAdminProductLicensesPage: React.FunctionComponent<Props> = ({ history, location }) => {
    useEffect(() => eventLogger.logViewEvent('SiteAdminProductLicenses'), [])

    const nodeProps: Pick<SiteAdminProductLicenseNodeProps, 'showSubscription'> = {
        showSubscription: true,
    }

    return (
        <div className="site-admin-product-subscriptions-page">
            <PageTitle title="Product subscriptions" />
            <h2>License key lookup</h2>
            <p>Find matching licenses and their associated product subscriptions.</p>
            <FilteredProductLicenseConnection
                className="list-group list-group-flush mt-3"
                noun="product license"
                pluralNoun="product licenses"
                queryConnection={queryLicenses}
                nodeComponent={SiteAdminProductLicenseNode}
                nodeComponentProps={nodeProps}
                emptyElement={<span className="text-muted mt-2">Enter a partial license key to find matches.</span>}
                noSummaryIfAllNodesVisible={true}
                autoFocus={true}
                history={history}
                location={location}
            />
        </div>
    )
}

function queryLicenses(args: {
    first: number | null
    query: string | null
}): Observable<DotComProductLicensesResult['dotcom']['productLicenses']> {
    const vars: DotComProductLicensesVariables = {
        first: args.first,
        licenseKeySubstring: args.query,
    }
    return args.query
        ? queryGraphQL<DotComProductLicensesResult>(
              gql`
                  query DotComProductLicenses($first: Int, $licenseKeySubstring: String) {
                      dotcom {
                          productLicenses(first: $first, licenseKeySubstring: $licenseKeySubstring) {
                              nodes {
                                  ...ProductLicenseFields
                              }
                              totalCount
                              pageInfo {
                                  hasNextPage
                              }
                          }
                      }
                  }
                  ${siteAdminProductLicenseFragment}
              `,
              vars
          ).pipe(
              map(({ data, errors }) => {
                  if (!data || !data.dotcom || !data.dotcom.productLicenses || (errors && errors.length > 0)) {
                      throw createAggregateError(errors)
                  }
                  return data.dotcom.productLicenses
              })
          )
        : of({
              nodes: [],
              totalCount: 0,
              pageInfo: {
                  hasNextPage: false,
              },
          })
}
