import * as React from 'react'
import { Redirect, RouteComponentProps } from 'react-router'

import { RepoSettingsAreaRoute } from '../../../repo/settings/RepoSettingsArea'
import { repoSettingsAreaRoutes } from '../../../repo/settings/routes'
import { lazyComponent } from '../../../util/lazyComponent'

import { RepoSettingsPermissionsPageProps } from './RepoSettingsPermissionsPage'

const RepoSettingsPermissionsPage = lazyComponent<RepoSettingsPermissionsPageProps, 'RepoSettingsPermissionsPage'>(
    () => import('./RepoSettingsPermissionsPage'),
    'RepoSettingsPermissionsPage'
)

export const enterpriseRepoSettingsAreaRoutes: readonly RepoSettingsAreaRoute[] = [
    ...repoSettingsAreaRoutes,
    {
        path: '/permissions',
        exact: true,
        render: props => <RepoSettingsPermissionsPage {...props} />,
    },

    // Legacy routes
    {
        path: '/code-intelligence/lsif-uploads/:id',
        exact: true,
        render: ({
            match: {
                params: { id },
            },
        }: RouteComponentProps<{ id: string }>) => <Redirect to={`../uploads/${id}`} />,
    },
    {
        path: '/code-intelligence',
        exact: false,
        render: props => <Redirect to={props.location.pathname.replace('/settings/', '/')} />,
    },
]
