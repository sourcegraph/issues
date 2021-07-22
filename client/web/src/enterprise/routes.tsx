import React from 'react'
import { Redirect } from 'react-router'

import { LayoutRouteProps, routes } from '../routes'
import { lazyComponent } from '../util/lazyComponent'

export const enterpriseRoutes: readonly LayoutRouteProps<{}>[] = [
    {
        // Allow unauthenticated viewers to view the "new subscription" page to price out a subscription (instead
        // of just dumping them on a sign-in page).
        path: '/subscriptions/new',
        exact: true,
        render: lazyComponent(
            () => import('./user/productSubscriptions/NewProductSubscriptionPageOrRedirectUser'),
            'NewProductSubscriptionPageOrRedirectUser'
        ),
    },
    {
        // Redirect from old /user/subscriptions/new -> /subscriptions/new.
        path: '/user/subscriptions/new',
        exact: true,
        render: () => <Redirect to="/subscriptions/new" />,
    },
    {
        path: '/campaigns',
        render: ({ match }) => <Redirect to={match.path.replace('/campaigns', '/batch-changes')} />,
    },
    {
        path: '/batch-changes',
        render: lazyComponent(() => import('./batches/global/GlobalBatchChangesArea'), 'GlobalBatchChangesArea'),
        condition: ({ batchChangesEnabled }) => batchChangesEnabled,
    },
    {
        path: '/stats',
        render: lazyComponent(() => import('./search/stats/SearchStatsPage'), 'SearchStatsPage'),
    },
    {
        path: '/code-monitoring',
        render: lazyComponent(
            () => import('./code-monitoring/global/GlobalCodeMonitoringArea'),
            'GlobalCodeMonitoringArea'
        ),
    },
    ...routes,
]
