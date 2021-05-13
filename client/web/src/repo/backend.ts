import { Observable } from 'rxjs'
import { map, tap } from 'rxjs/operators'

import {
    CloneInProgressError,
    RepoNotFoundError,
    RepoSeeOtherError,
    RevisionNotFoundError,
} from '@sourcegraph/shared/src/backend/errors'
import { FetchFileParameters } from '@sourcegraph/shared/src/components/CodeExcerpt'
import { dataOrThrowErrors, gql } from '@sourcegraph/shared/src/graphql/graphql'
import { IPageInfo, IRepositoryMetadataTag } from '@sourcegraph/shared/src/graphql/schema'
import { createAggregateError } from '@sourcegraph/shared/src/util/errors'
import { memoizeObservable } from '@sourcegraph/shared/src/util/memoizeObservable'
import {
    AbsoluteRepoFile,
    makeRepoURI,
    RepoRevision,
    RevisionSpec,
    RepoSpec,
    ResolvedRevisionSpec,
} from '@sourcegraph/shared/src/util/url'

import { queryGraphQL, requestGraphQL } from '../backend/graphql'
import {
    TreeFields,
    ExternalLinkFields,
    RepositoryRedirectResult,
    RepositoryRedirectVariables,
    RepositoryFields,
    RepoTagsResult,
    RepoTagsVariables,
    AddRepoTagResult,
    AddRepoTagVariables,
    DeleteRepoTagResult,
    DeleteRepoTagVariables,
} from '../graphql-operations'

export const externalLinkFieldsFragment = gql`
    fragment ExternalLinkFields on ExternalLink {
        url
        serviceKind
    }
`

export const repositoryFragment = gql`
    fragment RepositoryFields on Repository {
        id
        name
        url
        externalURLs {
            url
            serviceKind
        }
        description
        viewerCanAdminister
        defaultBranch {
            displayName
        }
    }
`

/**
 * Fetch the repository.
 */
export const fetchRepository = memoizeObservable(
    (args: { repoName: string }): Observable<RepositoryFields> =>
        requestGraphQL<RepositoryRedirectResult, RepositoryRedirectVariables>(
            gql`
                query RepositoryRedirect($repoName: String!) {
                    repositoryRedirect(name: $repoName) {
                        __typename
                        ... on Repository {
                            ...RepositoryFields
                        }
                        ... on Redirect {
                            url
                        }
                    }
                }
                ${repositoryFragment}
            `,
            args
        ).pipe(
            map(dataOrThrowErrors),
            map(data => {
                if (!data.repositoryRedirect) {
                    throw new RepoNotFoundError(args.repoName)
                }
                if (data.repositoryRedirect.__typename === 'Redirect') {
                    throw new RepoSeeOtherError(data.repositoryRedirect.url)
                }
                return data.repositoryRedirect
            })
        ),
    makeRepoURI
)

export interface ResolvedRevision extends ResolvedRevisionSpec {
    defaultBranch: string

    /** The URL to the repository root tree at the revision. */
    rootTreeURL: string
}

/**
 * When `revision` is undefined, the default branch is resolved.
 *
 * @returns Observable that emits the commit ID. Errors with a `CloneInProgressError` if the repo is still being cloned.
 */
export const resolveRevision = memoizeObservable(
    ({ repoName, revision }: RepoSpec & Partial<RevisionSpec>): Observable<ResolvedRevision> =>
        queryGraphQL(
            gql`
                query ResolveRev($repoName: String!, $revision: String!) {
                    repositoryRedirect(name: $repoName) {
                        __typename
                        ... on Repository {
                            mirrorInfo {
                                cloneInProgress
                                cloneProgress
                                cloned
                            }
                            commit(rev: $revision) {
                                oid
                                tree(path: "") {
                                    url
                                }
                            }
                            defaultBranch {
                                abbrevName
                            }
                        }
                        ... on Redirect {
                            url
                        }
                    }
                }
            `,
            { repoName, revision: revision || '' }
        ).pipe(
            map(({ data, errors }) => {
                if (!data) {
                    throw createAggregateError(errors)
                }
                if (!data.repositoryRedirect) {
                    throw new RepoNotFoundError(repoName)
                }
                if (data.repositoryRedirect.__typename === 'Redirect') {
                    throw new RepoSeeOtherError(data.repositoryRedirect.url)
                }
                if (data.repositoryRedirect.mirrorInfo.cloneInProgress) {
                    throw new CloneInProgressError(
                        repoName,
                        data.repositoryRedirect.mirrorInfo.cloneProgress || undefined
                    )
                }
                if (!data.repositoryRedirect.mirrorInfo.cloned) {
                    throw new CloneInProgressError(repoName, 'queued for cloning')
                }
                if (!data.repositoryRedirect.commit) {
                    throw new RevisionNotFoundError(revision)
                }

                const defaultBranch = data.repositoryRedirect.defaultBranch?.abbrevName || 'HEAD'

                if (!data.repositoryRedirect.commit.tree) {
                    throw new RevisionNotFoundError(defaultBranch)
                }
                return {
                    commitID: data.repositoryRedirect.commit.oid,
                    defaultBranch,
                    rootTreeURL: data.repositoryRedirect.commit.tree.url,
                }
            })
        ),
    makeRepoURI
)

/**
 * Fetches the specified highlighted file line ranges (`FetchFileParameters.ranges`) and returns
 * them as a list of ranges, each describing a list of lines in the form of HTML table '<tr>...</tr>'.
 */
export const fetchHighlightedFileLineRanges = memoizeObservable(
    (context: FetchFileParameters, force?: boolean): Observable<string[][]> =>
        queryGraphQL(
            gql`
                query HighlightedFile(
                    $repoName: String!
                    $commitID: String!
                    $filePath: String!
                    $disableTimeout: Boolean!
                    $isLightTheme: Boolean!
                    $ranges: [HighlightLineRange!]!
                ) {
                    repository(name: $repoName) {
                        commit(rev: $commitID) {
                            file(path: $filePath) {
                                isDirectory
                                richHTML
                                highlight(disableTimeout: $disableTimeout, isLightTheme: $isLightTheme) {
                                    aborted
                                    lineRanges(ranges: $ranges)
                                }
                            }
                        }
                    }
                }
            `,
            context
        ).pipe(
            map(({ data, errors }) => {
                if (!data?.repository?.commit?.file?.highlight) {
                    throw createAggregateError(errors)
                }
                const file = data.repository.commit.file
                if (file.isDirectory) {
                    return []
                }
                return file.highlight.lineRanges
            })
        ),
    context =>
        makeRepoURI(context) +
        `?disableTimeout=${String(context.disableTimeout)}&isLightTheme=${String(
            context.isLightTheme
        )}&ranges=${context.ranges.map(range => `${range.startLine}:${range.endLine}`).join(',')}`
)

export const fetchFileExternalLinks = memoizeObservable(
    (context: RepoRevision & { filePath: string }): Observable<ExternalLinkFields[]> =>
        queryGraphQL(
            gql`
                query FileExternalLinks($repoName: String!, $revision: String!, $filePath: String!) {
                    repository(name: $repoName) {
                        commit(rev: $revision) {
                            file(path: $filePath) {
                                externalURLs {
                                    ...ExternalLinkFields
                                }
                            }
                        }
                    }
                }

                ${externalLinkFieldsFragment}
            `,
            context
        ).pipe(
            map(({ data, errors }) => {
                if (!data?.repository?.commit?.file?.externalURLs) {
                    throw createAggregateError(errors)
                }
                return data.repository.commit.file.externalURLs
            })
        ),
    makeRepoURI
)

export const fetchTreeEntries = memoizeObservable(
    (args: AbsoluteRepoFile & { first?: number }): Observable<TreeFields> =>
        queryGraphQL(
            gql`
                query TreeEntries(
                    $repoName: String!
                    $revision: String!
                    $commitID: String!
                    $filePath: String!
                    $first: Int
                ) {
                    repository(name: $repoName) {
                        commit(rev: $commitID, inputRevspec: $revision) {
                            tree(path: $filePath) {
                                ...TreeFields
                            }
                        }
                    }
                }
                fragment TreeFields on GitTree {
                    isRoot
                    url
                    entries(first: $first, recursiveSingleChild: true) {
                        ...TreeEntryFields
                    }
                }
                fragment TreeEntryFields on TreeEntry {
                    name
                    path
                    isDirectory
                    url
                    submodule {
                        url
                        commit
                    }
                    isSingleChild
                }
            `,
            args
        ).pipe(
            map(({ data, errors }) => {
                if (errors || !data?.repository?.commit?.tree) {
                    throw createAggregateError(errors)
                }
                return data.repository.commit.tree
            })
        ),
    ({ first, ...args }) => `${makeRepoURI(args)}:first-${String(first)}`
)

interface FetchRepoTagsArguments {
    id: string
    first?: number
    after?: string
}

export interface FetchRepoTagsResult {
    nodes: Pick<IRepositoryMetadataTag, 'id' | 'tag'>[]
    pageInfo: Pick<IPageInfo, 'endCursor' | 'hasNextPage'>
}

export const fetchRepoTags = memoizeObservable(
    ({ id, first, after }: FetchRepoTagsArguments, force?: boolean): Observable<FetchRepoTagsResult> =>
        requestGraphQL<RepoTagsResult, RepoTagsVariables>(
            gql`
                query RepoTags($id: ID!, $first: Int, $after: String) {
                    node(id: $id) {
                        ... on Repository {
                            metadataTags(first: $first, after: $after) {
                                nodes {
                                    id
                                    tag
                                }
                                pageInfo {
                                    endCursor
                                    hasNextPage
                                }
                            }
                        }
                    }
                }
            `,
            { id, first: first || 15, after: after || null }
        ).pipe(
            tap(({ data }) => console.log('tap', data)),
            map(({ data, errors }) => {
                const metadataTags = data?.node?.metadataTags
                if (!metadataTags) {
                    throw createAggregateError(errors)
                }
                return metadataTags
            })
        ),
    ({ id, first, after }) => `${id}-${first || ''}-${after || ''}`
)

export const addRepoTag = async (repo: string, tag: string): Promise<void> => {
    const result = await requestGraphQL<AddRepoTagResult, AddRepoTagVariables>(
        gql`
            mutation AddRepoTag($repo: ID!, $tag: String!) {
                setTag(node: $repo, tag: $tag, present: true) {
                    alwaysNil
                }
            }
        `,
        { repo, tag }
    ).toPromise()
    dataOrThrowErrors(result)
}

export const deleteRepoTag = async (repo: string, tag: string): Promise<void> => {
    const result = await requestGraphQL<DeleteRepoTagResult, DeleteRepoTagVariables>(
        gql`
            mutation DeleteRepoTag($repo: ID!, $tag: String!) {
                setTag(node: $repo, tag: $tag, present: false) {
                    alwaysNil
                }
            }
        `,
        { repo, tag }
    ).toPromise()
    dataOrThrowErrors(result)
}
