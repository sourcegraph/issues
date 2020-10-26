import { from, Observable } from 'rxjs'
import { delay, filter, map, retryWhen } from 'rxjs/operators'
import {
    CloneInProgressError,
    RepoNotFoundError,
    RevisionNotFoundError,
    isCloneInProgressErrorLike,
} from '../../../../shared/src/backend/errors'
import { dataOrThrowErrors, gql } from '../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../shared/src/graphql/schema'
import { PlatformContext } from '../../../../shared/src/platform/context'
import { createAggregateError } from '../../../../shared/src/util/errors'
import { memoizeObservable } from '../../../../shared/src/util/memoizeObservable'
import {
    FileSpec,
    makeRepoURI,
    RawRepoSpec,
    RepoSpec,
    ResolvedRevisionSpec,
    RevisionSpec,
} from '../../../../shared/src/util/url'

/**
 * @returns Observable that emits if the repo exists on the instance.
 * Emits the repo name on the Sourcegraph instance as affected by `repositoryPathPattern`.
 * Errors with a `RepoNotFoundError` if the repo is not found.
 */
export const resolveRepo = memoizeObservable(
    ({ rawRepoName, requestGraphQL }: RawRepoSpec & Pick<PlatformContext, 'requestGraphQL'>): Observable<string> =>
        requestGraphQL<GQL.IQuery>({
            request: gql`
                query ResolveRepo($rawRepoName: String!) {
                    repository(name: $rawRepoName) {
                        name
                    }
                }
            `,
            variables: { rawRepoName },
            // This request may leak private repository names
            mightContainPrivateInfo: true,
        }).pipe(
            map(dataOrThrowErrors),
            map(({ repository }) => {
                if (!repository?.name) {
                    throw new RepoNotFoundError(rawRepoName)
                }
                return repository.name
            })
        ),
    ({ rawRepoName }) => rawRepoName
)

/**
 * @returns Observable that emits the commit ID. Errors with a `CloneInProgressError` if the repo is still being cloned.
 */
export const resolveRevision = memoizeObservable(
    ({
        requestGraphQL,
        ...context
    }: RepoSpec & Partial<RevisionSpec> & Pick<PlatformContext, 'requestGraphQL'>): Observable<string> =>
        from(
            requestGraphQL<GQL.IQuery>({
                request: gql`
                    query ResolveRev($repoName: String!, $revision: String!) {
                        repository(name: $repoName) {
                            mirrorInfo {
                                cloned
                            }
                            commit(rev: $revision) {
                                oid
                            }
                        }
                    }
                `,
                variables: { ...context, revision: context.revision || '' },
                mightContainPrivateInfo: true,
            })
        ).pipe(
            map(dataOrThrowErrors),
            map(({ repository }) => {
                if (!repository) {
                    throw new RepoNotFoundError(context.repoName)
                }
                if (!repository.mirrorInfo.cloned) {
                    throw new CloneInProgressError(context.repoName)
                }
                if (!repository.commit) {
                    throw new RevisionNotFoundError(context.revision)
                }
                return repository.commit.oid
            })
        ),
    makeRepoURI
)

export function retryWhenCloneInProgressError<T>(): (v: Observable<T>) => Observable<T> {
    return (maybeErrors: Observable<T>) =>
        maybeErrors.pipe(
            retryWhen(errors =>
                errors.pipe(
                    filter(error => {
                        if (isCloneInProgressErrorLike(error)) {
                            return true
                        }

                        // Don't swallow other errors.
                        throw error
                    }),
                    delay(1000)
                )
            )
        )
}

/**
 * Fetches the lines of a given file at a given commit from the Sourcegraph API.
 * Will return an empty array if the repo, commit or file does not exist or an error happened (TODO change this!).
 *
 * Only emits once.
 */
export const fetchBlobContentLines = memoizeObservable(
    ({
        requestGraphQL,
        ...context
    }: RepoSpec & ResolvedRevisionSpec & FileSpec & Pick<PlatformContext, 'requestGraphQL'>): Observable<string[]> =>
        from(
            requestGraphQL<GQL.IQuery>({
                request: gql`
                    query BlobContent($repoName: String!, $commitID: String!, $filePath: String!) {
                        repository(name: $repoName) {
                            commit(rev: $commitID) {
                                file(path: $filePath) {
                                    content
                                }
                            }
                        }
                    }
                `,
                variables: context,
                mightContainPrivateInfo: true,
            })
        ).pipe(
            map(({ data, errors }) => {
                if (!data) {
                    throw new Error('Invalid response')
                }
                if (errors) {
                    if (errors.length === 1) {
                        const error = errors[0]
                        const errorPath = error.path.join('.')

                        // Originally this checked only for 'repository.commit.file.content'.
                        // But if a file doesn't exist, the error path is 'repository.commit.file'
                        const isFileContent =
                            errorPath === 'repository.commit.file.content' || errorPath === 'repository.commit.file'
                        const isDNE = error.message.includes('does not exist')

                        // The error is the file DNE. Just ignore it and pass an empty array
                        // to represent this.
                        if (isFileContent && isDNE) {
                            return []
                        }
                    }
                    throw createAggregateError(errors)
                }
                const { repository } = data
                if (!repository || !repository.commit || !repository.commit.file || !repository.commit.file.content) {
                    return []
                }
                return repository.commit.file.content.split('\n')
            })
        ),
    makeRepoURI
)
