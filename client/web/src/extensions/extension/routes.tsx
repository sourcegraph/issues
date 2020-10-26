import React from 'react'
import { eventLogger } from '../../tracking/eventLogger'
import { lazyComponent } from '../../util/lazyComponent'
import { ExtensionAreaRoute } from './ExtensionArea'

const RegistryExtensionOverviewPage = lazyComponent(
    () => import('./RegistryExtensionOverviewPage'),
    'RegistryExtensionOverviewPage'
)

export const extensionAreaRoutes: readonly ExtensionAreaRoute[] = [
    {
        path: '',
        exact: true,
        render: props => <RegistryExtensionOverviewPage eventLogger={eventLogger} {...props} />,
    },
    {
        path: '/-/manifest',
        exact: true,
        render: lazyComponent(() => import('./RegistryExtensionManifestPage'), 'RegistryExtensionManifestPage'),
    },
    {
        path: '/-/contributions',
        exact: true,
        render: lazyComponent(
            () => import('./RegistryExtensionContributionsPage'),
            'RegistryExtensionContributionsPage'
        ),
    },
]
