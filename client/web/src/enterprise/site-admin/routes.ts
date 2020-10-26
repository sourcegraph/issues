import { siteAdminAreaRoutes } from '../../site-admin/routes'
import { SiteAdminAreaRoute } from '../../site-admin/SiteAdminArea'
import { lazyComponent } from '../../util/lazyComponent'

export const enterpriseSiteAdminAreaRoutes: readonly SiteAdminAreaRoute[] = [
    ...siteAdminAreaRoutes,
    {
        path: '/license',
        render: lazyComponent(
            () => import('./productSubscription/SiteAdminProductSubscriptionPage'),
            'SiteAdminProductSubscriptionPage'
        ),
        exact: true,
    },
    {
        path: '/dotcom/customers',
        render: lazyComponent(
            () => import('./dotcom/customers/SiteAdminCustomersPage'),
            'SiteAdminProductCustomersPage'
        ),
        exact: true,
    },
    {
        path: '/dotcom/product/subscriptions/new',
        render: lazyComponent(
            () => import('./dotcom/productSubscriptions/SiteAdminCreateProductSubscriptionPage'),
            'SiteAdminCreateProductSubscriptionPage'
        ),
        exact: true,
    },
    {
        path: '/dotcom/product/subscriptions/:subscriptionUUID',
        render: lazyComponent(
            () => import('./dotcom/productSubscriptions/SiteAdminProductSubscriptionPage'),
            'SiteAdminProductSubscriptionPage'
        ),
        exact: true,
    },
    {
        path: '/dotcom/product/subscriptions',
        render: lazyComponent(
            () => import('./dotcom/productSubscriptions/SiteAdminProductSubscriptionsPage'),
            'SiteAdminProductSubscriptionsPage'
        ),
        exact: true,
    },
    {
        path: '/dotcom/product/licenses',
        render: lazyComponent(
            () => import('./dotcom/productSubscriptions/SiteAdminProductLicensesPage'),
            'SiteAdminProductLicensesPage'
        ),
        exact: true,
    },
    {
        path: '/auth/providers',
        render: lazyComponent(
            () => import('./SiteAdminAuthenticationProvidersPage'),
            'SiteAdminAuthenticationProvidersPage'
        ),
        exact: true,
    },
    {
        path: '/auth/external-accounts',
        render: lazyComponent(() => import('./SiteAdminExternalAccountsPage'), 'SiteAdminExternalAccountsPage'),
        exact: true,
    },
    {
        path: '/registry/extensions',
        render: lazyComponent(() => import('./SiteAdminRegistryExtensionsPage'), 'SiteAdminRegistryExtensionsPage'),
        exact: true,
    },
    {
        path: '/code-intelligence/uploads',
        render: lazyComponent(() => import('../codeintel/CodeIntelUploadsPage'), 'CodeIntelUploadsPage'),
        exact: true,
    },
    {
        path: '/code-intelligence/uploads/:id',
        render: lazyComponent(() => import('../codeintel/CodeIntelUploadPage'), 'CodeIntelUploadPage'),
        exact: true,
    },
    {
        path: '/code-intelligence/indexes',
        render: lazyComponent(() => import('../codeintel/CodeIntelIndexesPage'), 'CodeIntelIndexesPage'),
        exact: true,
    },
    {
        path: '/code-intelligence/indexes/:id',
        render: lazyComponent(() => import('../codeintel/CodeIntelIndexPage'), 'CodeIntelIndexPage'),
        exact: true,
    },
    {
        path: '/lsif-uploads/:id',
        render: lazyComponent(() => import('./SiteAdminLsifUploadPage'), 'SiteAdminLsifUploadPage'),
        exact: true,
    },
]
