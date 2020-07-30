import { Observable } from 'rxjs'
import { map } from 'rxjs/operators'
import {
    CloneInProgressError,
    RepoNotFoundError,
    RepoSeeOtherError,
    RevisionNotFoundError,
} from '../../../shared/src/backend/errors'
import { FetchFileCtx } from '../../../shared/src/components/CodeExcerpt'
import { gql } from '../../../shared/src/graphql/graphql'
import { createAggregateError } from '../../../shared/src/util/errors'
import { memoizeObservable } from '../../../shared/src/util/memoizeObservable'
import {
    AbsoluteRepoFile,
    makeRepoURI,
    RepoRev,
    RevisionSpec,
    RepoSpec,
    ResolvedRevisionSpec,
} from '../../../shared/src/util/url'
import { queryGraphQL } from '../backend/graphql'
import {
    RepositoryRedirectResult,
    ResolveRevResult,
    FileExternalLinksResult,
    TreeEntriesResult,
    HighlightedFileResult,
} from '../graphql-operations'

type Repository = NonNullable<RepositoryRedirectResult['repositoryRedirect']> & {
    __typename: 'Repository'
}

/**
 * Fetch the repository.
 */
export const fetchRepository = memoizeObservable(
    (args: { repoName: string }): Observable<Repository> =>
        queryGraphQL<RepositoryRedirectResult>(
            gql`
                query RepositoryRedirect($repoName: String!) {
                    repositoryRedirect(name: $repoName) {
                        __typename
                        ... on Repository {
                            id
                            name
                            url
                            externalURLs {
                                url
                                serviceType
                            }
                            description
                            viewerCanAdminister
                            defaultBranch {
                                displayName
                            }
                        }
                        ... on Redirect {
                            url
                        }
                    }
                }
            `,
            args
        ).pipe(
            map(({ data, errors }) => {
                if (!data) {
                    throw createAggregateError(errors)
                }
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
        queryGraphQL<ResolveRevResult>(
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
                if (!data.repositoryRedirect.defaultBranch || !data.repositoryRedirect.commit.tree) {
                    throw new RevisionNotFoundError('HEAD')
                }
                return {
                    commitID: data.repositoryRedirect.commit.oid,
                    defaultBranch: data.repositoryRedirect.defaultBranch.abbrevName,
                    rootTreeURL: data.repositoryRedirect.commit.tree.url,
                }
            })
        ),
    makeRepoURI
)

type HighlightedFileData = NonNullable<NonNullable<NonNullable<HighlightedFileResult['repository']>['commit']>['file']>

const fetchHighlightedFile = memoizeObservable(
    (context: FetchFileCtx): Observable<HighlightedFileData> =>
        queryGraphQL<HighlightedFileResult>(
            gql`
                query HighlightedFile(
                    $repoName: String!
                    $commitID: String!
                    $filePath: String!
                    $disableTimeout: Boolean!
                    $isLightTheme: Boolean!
                ) {
                    repository(name: $repoName) {
                        commit(rev: $commitID) {
                            file(path: $filePath) {
                                isDirectory
                                richHTML
                                highlight(disableTimeout: $disableTimeout, isLightTheme: $isLightTheme) {
                                    aborted
                                    html
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
                return data.repository.commit.file
            })
        ),
    context =>
        makeRepoURI(context) +
        `?disableTimeout=${String(context.disableTimeout)}&isLightTheme=${String(context.isLightTheme)}`
)

/**
 * Produces a list like ['<tr>...</tr>', ...]
 */
export const fetchHighlightedFileLines = memoizeObservable(
    (context: FetchFileCtx, force?: boolean): Observable<string[]> =>
        fetchHighlightedFile(context, force).pipe(
            map(result => {
                if (result.isDirectory) {
                    return []
                }
                const parsed = result.highlight.html.slice('<table>'.length, -'</table>'.length)
                const rows = parsed.split('</tr>')
                for (let index = 0; index < rows.length; ++index) {
                    rows[index] += '</tr>'
                }
                return rows
            })
        ),
    context => makeRepoURI(context) + `?isLightTheme=${String(context.isLightTheme)}`
)

export type ExternalLink = NonNullable<
    NonNullable<NonNullable<FileExternalLinksResult['repository']>['commit']>['file']
>['externalURLs'][number]

export const fetchFileExternalLinks = memoizeObservable(
    (context: RepoRev & { filePath: string }): Observable<ExternalLink[]> =>
        queryGraphQL<FileExternalLinksResult>(
            gql`
                query FileExternalLinks($repoName: String!, $revision: String!, $filePath: String!) {
                    repository(name: $repoName) {
                        commit(rev: $revision) {
                            file(path: $filePath) {
                                externalURLs {
                                    url
                                    serviceType
                                }
                            }
                        }
                    }
                }
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

export type GitTree = NonNullable<NonNullable<NonNullable<TreeEntriesResult['repository']>['commit']>['tree']>

export const fetchTreeEntries = memoizeObservable(
    (args: AbsoluteRepoFile & { first?: number }): Observable<GitTree> =>
        queryGraphQL<TreeEntriesResult>(
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
                                isRoot
                                url
                                entries(first: $first, recursiveSingleChild: true) {
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
                            }
                        }
                    }
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
