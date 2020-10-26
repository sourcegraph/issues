import { ExtensionAreaRoute } from '../../../extensions/extension/ExtensionArea'
import { extensionAreaRoutes } from '../../../extensions/extension/routes'
import { lazyComponent } from '../../../util/lazyComponent'

export const enterpriseExtensionAreaRoutes: readonly ExtensionAreaRoute[] = [
    ...extensionAreaRoutes,
    {
        path: '/-/manage',
        exact: true,
        render: lazyComponent(() => import('./RegistryExtensionManagePage'), 'RegistryExtensionManagePage'),
    },
    {
        path: '/-/releases/new',
        exact: true,
        render: lazyComponent(() => import('./RegistryExtensionNewReleasePage'), 'RegistryExtensionNewReleasePage'),
    },
]
