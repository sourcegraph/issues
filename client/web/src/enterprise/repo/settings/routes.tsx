import * as React from 'react'
import { Redirect, RouteComponentProps } from 'react-router'
import { repoSettingsAreaRoutes } from '../../../repo/settings/routes'
import { RepoSettingsAreaRoute } from '../../../repo/settings/RepoSettingsArea'
import { lazyComponent } from '../../../util/lazyComponent'
import { CodeIntelUploadsPageProps } from '../../codeintel/CodeIntelUploadsPage'
import { CodeIntelIndexesPageProps } from '../../codeintel/CodeIntelIndexesPage'
import { CodeIntelIndexPageProps } from '../../codeintel/CodeIntelIndexPage'
import { CodeIntelUploadPageProps } from '../../codeintel/CodeIntelUploadPage'
import { RepoSettingsPermissionsPageProps } from './RepoSettingsPermissionsPage'

const RepoSettingsPermissionsPage = lazyComponent<RepoSettingsPermissionsPageProps, 'RepoSettingsPermissionsPage'>(
    () => import('./RepoSettingsPermissionsPage'),
    'RepoSettingsPermissionsPage'
)
const CodeIntelUploadsPage = lazyComponent<CodeIntelUploadsPageProps, 'CodeIntelUploadsPage'>(
    () => import('../../codeintel/CodeIntelUploadsPage'),
    'CodeIntelUploadsPage'
)
const CodeIntelUploadPage = lazyComponent<CodeIntelUploadPageProps, 'CodeIntelUploadPage'>(
    () => import('../../codeintel/CodeIntelUploadPage'),
    'CodeIntelUploadPage'
)
const CodeIntelIndexesPage = lazyComponent<CodeIntelIndexesPageProps, 'CodeIntelIndexesPage'>(
    () => import('../../codeintel/CodeIntelIndexesPage'),
    'CodeIntelIndexesPage'
)
const CodeIntelIndexPage = lazyComponent<CodeIntelIndexPageProps, 'CodeIntelIndexPage'>(
    () => import('../../codeintel/CodeIntelIndexPage'),
    'CodeIntelIndexPage'
)

export const enterpriseRepoSettingsAreaRoutes: readonly RepoSettingsAreaRoute[] = [
    ...repoSettingsAreaRoutes,
    {
        path: '/permissions',
        exact: true,
        render: props => <RepoSettingsPermissionsPage {...props} />,
    },
    {
        path: '/code-intelligence/uploads',
        exact: true,
        render: props => <CodeIntelUploadsPage {...props} />,
    },
    {
        path: '/code-intelligence/uploads/:id',
        exact: true,
        render: props => <CodeIntelUploadPage {...props} />,
    },
    {
        path: '/code-intelligence/indexes',
        exact: true,
        render: props => <CodeIntelIndexesPage {...props} />,
    },
    {
        path: '/code-intelligence/indexes/:id',
        exact: true,
        render: props => <CodeIntelIndexPage {...props} />,
    },
    {
        path: '/code-intelligence/lsif-uploads/:id',
        exact: true,
        render: ({
            match: {
                params: { id },
            },
        }: RouteComponentProps<{ id: string }>) => <Redirect to={`../uploads/${id}`} />,
    },
]
