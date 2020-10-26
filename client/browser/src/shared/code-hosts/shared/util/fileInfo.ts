import { Observable, of, zip } from 'rxjs'
import { catchError, map } from 'rxjs/operators'

import { isPrivateRepoPublicSourcegraphComErrorLike } from '../../../../../../shared/src/backend/errors'
import { PlatformContext } from '../../../../../../shared/src/platform/context'
import { resolveRepo, resolveRevision, retryWhenCloneInProgressError } from '../../../repo/backend'
import { FileInfo, FileInfoWithRepoName, DiffOrBlobInfo } from '../codeHost'

/**
 * Use `rawRepoName` for the value of `repoName`, as a fallback if `repoName`
 * was not available.
 */
const useRawRepoNameAsFallback = (fileInfo: FileInfo): FileInfoWithRepoName => ({
    ...fileInfo,
    repoName: fileInfo.rawRepoName,
})

/**
 * Resolve a `FileInfo`'s raw repo names to their Sourcegraph
 * repo names as affected by `repositoryPathPattern`.
 */
const resolveRepoNameForFileInfo = (
    fileInfo: FileInfo,
    requestGraphQL: PlatformContext['requestGraphQL']
): Observable<FileInfoWithRepoName> =>
    resolveRepo({ rawRepoName: fileInfo.rawRepoName, requestGraphQL }).pipe(
        map(repoName => ({ ...fileInfo, repoName })),
        catchError(error => {
            // ERPRIVATEREPOPUBLICSOURCEGRAPHCOM likely means that the user is viewing private code
            // without having pointed his browser extension to a self-hosted Sourcegraph instance that
            // has access to that code. In that case, it's impossible to resolve the repo names,
            // so we keep the repo names inferred from the code host's DOM.
            if (isPrivateRepoPublicSourcegraphComErrorLike(error)) {
                return of(useRawRepoNameAsFallback(fileInfo))
            }
            throw error
        })
    )

/**
 * Uses the `commitID` as the default value for `revision`, if no `revision`
 * value is present.
 */
export const defaultRevisionToCommitID = <T extends FileInfo>(fileInfo: T): T & { revision: string } => ({
    ...fileInfo,
    revision: fileInfo.revision || fileInfo.commitID,
})

/**
 * Ensures that the revision is cloned on Sourcegraph, by issuing a
 * `resolveRevision` request and retrying until the clone is complete.
 */
export const ensureRevisionIsClonedForFileInfo = (
    fileInfo: FileInfoWithRepoName,
    requestGraphQL: PlatformContext['requestGraphQL']
): Observable<string> => {
    // Although we get the commit SHA's from elsewhere, we still need to
    // use `resolveRev` otherwise we can't guarantee Sourcegraph has the
    // revision cloned.
    const { repoName, commitID } = fileInfo
    return resolveRevision({ repoName, revision: commitID, requestGraphQL }).pipe(retryWhenCloneInProgressError())
}

export const resolveRepoNamesForDiffOrFileInfo = (
    diffOrFileInfo: DiffOrBlobInfo,
    requestGraphQL: PlatformContext['requestGraphQL']
): Observable<DiffOrBlobInfo<FileInfoWithRepoName>> => {
    if ('blob' in diffOrFileInfo) {
        return resolveRepoNameForFileInfo(diffOrFileInfo.blob, requestGraphQL).pipe(
            map(fileInfo => ({ blob: { ...diffOrFileInfo.blob, ...fileInfo } }))
        )
    }
    if (diffOrFileInfo.head && diffOrFileInfo.base) {
        const resolvingHeadWithRepoName = resolveRepoNameForFileInfo(diffOrFileInfo.head, requestGraphQL)
        const resolvingBaseWithRepoName = resolveRepoNameForFileInfo(diffOrFileInfo.base, requestGraphQL)

        return zip(resolvingHeadWithRepoName, resolvingBaseWithRepoName).pipe(map(([head, base]) => ({ head, base })))
    }
    if (diffOrFileInfo.head) {
        return resolveRepoNameForFileInfo(diffOrFileInfo.head, requestGraphQL).pipe(map(head => ({ head })))
    }
    return resolveRepoNameForFileInfo(diffOrFileInfo.base, requestGraphQL).pipe(map(base => ({ base })))
}
