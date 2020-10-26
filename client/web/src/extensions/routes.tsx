import React from 'react'
import { lazyComponent } from '../util/lazyComponent'
import { ExtensionsAreaRoute } from './ExtensionsArea'

const ExtensionArea = lazyComponent(() => import('./extension/ExtensionArea'), 'ExtensionArea')

export const extensionsAreaRoutes: readonly ExtensionsAreaRoute[] = [
    {
        path: '',
        exact: true,
        render: lazyComponent(() => import('./ExtensionRegistry'), 'ExtensionRegistry'),
    },
    {
        path: '/:extensionID(.*)/-/',
        render: props => <ExtensionArea {...props} routes={props.extensionAreaRoutes} />,
    },
    {
        path: '/:extensionID(.*)',
        render: props => <ExtensionArea {...props} routes={props.extensionAreaRoutes} />,
    },
]
