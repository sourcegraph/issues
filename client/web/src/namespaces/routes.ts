import { lazyComponent } from '../util/lazyComponent'
import { NamespaceAreaRoute } from './NamespaceArea'

export const namespaceAreaRoutes: readonly NamespaceAreaRoute[] = [
    {
        path: '/searches',
        exact: true,
        render: lazyComponent(() => import('../savedSearches/SavedSearchListPage'), 'SavedSearchListPage'),
    },
    {
        path: '/searches/add',
        render: lazyComponent(() => import('../savedSearches/SavedSearchCreateForm'), 'SavedSearchCreateForm'),
    },
    {
        path: '/searches/:id',
        render: lazyComponent(() => import('../savedSearches/SavedSearchUpdateForm'), 'SavedSearchUpdateForm'),
    },
]
