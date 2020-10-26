import { RepoSettingsAreaRoute } from './RepoSettingsArea'
import { lazyComponent } from '../../util/lazyComponent'

export const repoSettingsAreaRoutes: readonly RepoSettingsAreaRoute[] = [
    {
        path: '',
        exact: true,
        render: lazyComponent(() => import('./RepoSettingsOptionsPage'), 'RepoSettingsOptionsPage'),
    },
    {
        path: '/index',
        exact: true,
        render: lazyComponent(() => import('./RepoSettingsIndexPage'), 'RepoSettingsIndexPage'),
    },
    {
        path: '/mirror',
        exact: true,
        render: lazyComponent(() => import('./RepoSettingsMirrorPage'), 'RepoSettingsMirrorPage'),
    },
]
