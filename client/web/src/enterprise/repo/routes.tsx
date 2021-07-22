import React from 'react'

import { RepoContainerRoute } from '../../repo/RepoContainer'
import { RepoRevisionContainerRoute } from '../../repo/RepoRevisionContainer'
import { repoContainerRoutes, repoRevisionContainerRoutes } from '../../repo/routes'
import { lazyComponent } from '../../util/lazyComponent'

const RepositoryGitDataContainer = lazyComponent(
    () => import('../../repo/RepositoryGitDataContainer'),
    'RepositoryGitDataContainer'
)

const RepositoryCodeIntelArea = lazyComponent(
    () => import('../codeintel/repo/RepositoryCodeIntelArea'),
    'RepositoryCodeIntelArea'
)

const RepositoryBatchChangesArea = lazyComponent(
    () => import('../batches/repo/RepositoryBatchChangesArea'),
    'RepositoryBatchChangesArea'
)

export const enterpriseRepoContainerRoutes: readonly RepoContainerRoute[] = [
    ...repoContainerRoutes,

    {
        path: '/-/code-intelligence',
        exact: false,
        render: context => (
            <RepositoryGitDataContainer {...context} repoName={context.repo.name}>
                <RepositoryCodeIntelArea {...context} />
            </RepositoryGitDataContainer>
        ),
    },

    {
        path: '/-/batch-changes',
        condition: ({ showBatchChanges }) => showBatchChanges,
        render: context => (
            <RepositoryGitDataContainer {...context} repoName={context.repo.name}>
                <RepositoryBatchChangesArea {...context} />
            </RepositoryGitDataContainer>
        ),
    },
]

export const enterpriseRepoRevisionContainerRoutes: readonly RepoRevisionContainerRoute[] = repoRevisionContainerRoutes
