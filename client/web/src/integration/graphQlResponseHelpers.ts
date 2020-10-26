import {
    TreeEntriesResult,
    BlobResult,
    FileExternalLinksResult,
    RepositoryRedirectResult,
    ResolveRevResult,
} from '../graphql-operations'

export const createTreeEntriesResult = (url: string, toplevelFiles: string[]): TreeEntriesResult => ({
    repository: {
        commit: {
            tree: {
                isRoot: true,
                url,
                entries: toplevelFiles.map(name => ({
                    name,
                    path: name,
                    isDirectory: false,
                    url: `${url}/-/blob/${name}`,
                    submodule: null,
                    isSingleChild: false,
                })),
            },
        },
    },
})

export const createBlobContentResult = (
    content: string,
    html: string = `<div style="color:red">${content}<div>`
): BlobResult => ({
    repository: {
        commit: {
            file: {
                content,
                richHTML: '',
                highlight: {
                    aborted: false,
                    html,
                },
            },
        },
    },
})

export const createFileExternalLinksResult = (
    url: string,
    serviceType: string = 'github'
): FileExternalLinksResult => ({
    repository: {
        commit: {
            file: {
                externalURLs: [{ url, serviceType }],
            },
        },
    },
})

export const createRepositoryRedirectResult = (
    repoName: string,
    serviceType: string = 'github'
): RepositoryRedirectResult => ({
    repositoryRedirect: {
        __typename: 'Repository',
        id: `RepositoryID:${repoName}`,
        name: repoName,
        url: `/${repoName}`,
        externalURLs: [{ url: new URL(`https://${repoName}`).href, serviceType }],
        description: 'bla',
        viewerCanAdminister: false,
        defaultBranch: { displayName: 'master' },
    },
})

export const createResolveRevisionResult = (treeUrl: string, oid = '1'.repeat(40)): ResolveRevResult => ({
    repositoryRedirect: {
        __typename: 'Repository',
        mirrorInfo: { cloneInProgress: false, cloneProgress: '', cloned: true },
        commit: {
            oid,
            tree: { url: treeUrl },
        },
        defaultBranch: { abbrevName: 'master' },
    },
})
