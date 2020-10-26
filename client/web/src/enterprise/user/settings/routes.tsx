import { userSettingsAreaRoutes } from '../../../user/settings/routes'
import { UserSettingsAreaRoute } from '../../../user/settings/UserSettingsArea'
import { lazyComponent } from '../../../util/lazyComponent'
import { SHOW_BUSINESS_FEATURES } from '../../dotcom/productSubscriptions/features'
import { authExp } from '../../site-admin/SiteAdminAuthenticationProvidersPage'
import React from 'react'

const UserEventLogsPage = lazyComponent(() => import('../../../user/UserEventLogsPage'), 'UserEventLogsPage')

export const enterpriseUserSettingsAreaRoutes: readonly UserSettingsAreaRoute[] = [
    ...userSettingsAreaRoutes,
    {
        path: '/permissions',
        exact: true,
        render: lazyComponent(() => import('./auth/UserSettingsPermissionsPage'), 'UserSettingsPermissionsPage'),
        condition: ({ authenticatedUser }) => authenticatedUser.siteAdmin,
    },
    {
        path: '/event-log',
        exact: true,
        render: props => <UserEventLogsPage {...props} />,
    },
    {
        path: '/external-accounts',
        exact: true,
        render: lazyComponent(() => import('./UserSettingsExternalAccountsPage'), 'UserSettingsExternalAccountsPage'),
        condition: () => authExp,
    },
    {
        path: '/subscriptions/new',
        exact: true,
        render: lazyComponent(
            () => import('../productSubscriptions/UserSubscriptionsNewProductSubscriptionPage'),
            'UserSubscriptionsNewProductSubscriptionPage'
        ),
        condition: () => SHOW_BUSINESS_FEATURES,
    },
    {
        path: '/subscriptions/:subscriptionUUID',
        exact: true,
        render: lazyComponent(
            () => import('../productSubscriptions/UserSubscriptionsProductSubscriptionPage'),
            'UserSubscriptionsProductSubscriptionPage'
        ),
        condition: () => SHOW_BUSINESS_FEATURES,
    },
    {
        path: '/subscriptions/:subscriptionUUID/edit',
        exact: true,
        render: lazyComponent(
            () => import('../productSubscriptions/UserSubscriptionsEditProductSubscriptionPage'),
            'UserSubscriptionsEditProductSubscriptionPage'
        ),
        condition: () => SHOW_BUSINESS_FEATURES,
    },
    {
        path: '/subscriptions',
        exact: true,
        render: lazyComponent(
            () => import('../productSubscriptions/UserSubscriptionsProductSubscriptionsPage'),
            'UserSubscriptionsProductSubscriptionsPage'
        ),
        condition: () => SHOW_BUSINESS_FEATURES,
    },
]
