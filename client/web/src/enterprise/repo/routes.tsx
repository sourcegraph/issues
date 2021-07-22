import React from 'react'

import { RepoContainerRoute } from '../../repo/RepoContainer'
import { RepoRevisionContainerRoute } from '../../repo/RepoRevisionContainer'
import { repoContainerRoutes, repoRevisionContainerRoutes } from '../../repo/routes'
import { lazyComponent } from '../../util/lazyComponent'

const RepositoryGitDataContainer = lazyComponent(
    () => import('../../repo/RepositoryGitDataContainer'),
    'RepositoryGitDataContainer'
)

const RepositoryBatchChangesArea = lazyComponent(
    () => import('../batches/repo/RepositoryBatchChangesArea'),
    'RepositoryBatchChangesArea'
)

export const enterpriseRepoContainerRoutes: readonly RepoContainerRoute[] = [
    ...repoContainerRoutes,

    {
        path: '/-/batch-changes',
        condition: ({ batchChangesEnabled }) => batchChangesEnabled,
        render: context => (
            <RepositoryGitDataContainer {...context} repoName={context.repo.name}>
                <RepositoryBatchChangesArea {...context} />
            </RepositoryGitDataContainer>
        ),
    },
]

export const enterpriseRepoRevisionContainerRoutes: readonly RepoRevisionContainerRoute[] = repoRevisionContainerRoutes
