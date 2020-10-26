import { Observable, from } from 'rxjs'
import { map, switchMap } from 'rxjs/operators'

import { DiffInfo, BlobInfo } from '../shared/codeHost'

import { getBaseCommitIDForCommit, getMergeRequestDetailsFromAPI } from './api'
import {
    getCommitIDFromPermalink,
    getCommitPageInfo,
    getFilePageInfo,
    getFilePathsFromCodeView,
    getPageInfo,
    getMergeRequestID,
    getDiffID,
} from './scrape'
import { asObservable } from '../../../../../shared/src/util/rxjs/asObservable'

/**
 * Resolves file information for a page with a single file, not including diffs with only one file.
 */
export const resolveFileInfo = (): BlobInfo => {
    const { rawRepoName, filePath, revision } = getFilePageInfo()
    if (!filePath) {
        throw new Error(
            `Unable to determine the file path of the current file because the current URL (window.location ${window.location.href}) does not have a file path.`
        )
    }
    const commitID = getCommitIDFromPermalink()
    return { blob: { rawRepoName, filePath, commitID, revision } }
}

/**
 * Gets `FileInfo` for a diff file.
 */
export const resolveDiffFileInfo = (codeView: HTMLElement): Observable<DiffInfo> =>
    from(
        getMergeRequestDetailsFromAPI({
            ...getPageInfo(),
            mergeRequestID: getMergeRequestID(),
            diffID: getDiffID(),
        })
    ).pipe(
        map(
            (info): DiffInfo => {
                const { rawRepoName, baseRawRepoName, commitID, baseCommitID } = info
                const { headFilePath, baseFilePath } = getFilePathsFromCodeView(codeView)

                return {
                    head: { rawRepoName, filePath: headFilePath, commitID },
                    base: { rawRepoName: baseRawRepoName, filePath: baseFilePath, commitID: baseCommitID },
                }
            }
        )
    )

/**
 * Resolves file information for commit pages.
 */
export const resolveCommitFileInfo = (codeView: HTMLElement): Observable<DiffInfo> =>
    asObservable(getCommitPageInfo).pipe(
        // Resolve base commit ID.
        switchMap(({ owner, projectName, commitID, rawRepoName }) =>
            getBaseCommitIDForCommit({ owner, projectName, commitID }).pipe(
                map(baseCommitID => ({ commitID, baseCommitID, rawRepoName }))
            )
        ),
        map(
            ({ commitID, baseCommitID, rawRepoName }): DiffInfo => {
                const { headFilePath, baseFilePath } = getFilePathsFromCodeView(codeView)
                return {
                    head: { rawRepoName, filePath: headFilePath, commitID },
                    base: { rawRepoName, filePath: baseFilePath, commitID: baseCommitID },
                }
            }
        )
    )
