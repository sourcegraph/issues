import { Observable, of } from 'rxjs'
import { map } from 'rxjs/operators'
import { DiffInfo, BlobInfo } from '../shared/codeHost'
import { getBaseCommit, getCommitsForPR } from './api'
import {
    getCommitInfoFromComparePage,
    getFileInfoFromCommitDiffCodeView,
    getFileInfoFromSingleFileDiffCodeView,
    getFileInfoFromSingleFileSourceCodeView,
    getFileInfoWithoutCommitIDsFromMultiFileDiffCodeView,
    getPRIDFromPathName,
} from './scrape'

/**
 * Resolves file information for a page with a single file in source (not diff) view.
 */
export const resolveFileInfoForSingleFileSourceView = (codeView: HTMLElement): BlobInfo => {
    const fileInfo = getFileInfoFromSingleFileSourceCodeView(codeView)
    return { blob: fileInfo }
}

/**
 * Gets the file info for a PR diff code view.
 */
export const resolvePullRequestFileInfo = (codeView: HTMLElement): Observable<DiffInfo> => {
    const partialFileInfo = getFileInfoWithoutCommitIDsFromMultiFileDiffCodeView(codeView)
    const { rawRepoName, filePath, baseFilePath } = partialFileInfo
    const prID = getPRIDFromPathName()
    return getCommitsForPR({ ...partialFileInfo, prID }).pipe(
        map(
            ({ headCommitID, baseCommitID }): DiffInfo => ({
                head: {
                    rawRepoName,
                    filePath,
                    commitID: headCommitID,
                },
                base: {
                    rawRepoName,
                    filePath: baseFilePath,
                    commitID: baseCommitID,
                },
            })
        )
    )
}

/**
 * Gets the file info for a single-file "diff to previous" code view.
 */
export const resolveSingleFileDiffFileInfo = (codeView: HTMLElement): Observable<DiffInfo> => {
    const {
        changeType,
        rawRepoName,
        filePath,
        commitID,
        baseFilePath,
        revision,
        ...bitbucketInfo
    } = getFileInfoFromSingleFileDiffCodeView(codeView)
    if (changeType === 'ADD') {
        return of({ head: { rawRepoName, filePath, commitID } })
    }

    return getBaseCommit({ commitID, ...bitbucketInfo }).pipe(
        map(
            (baseCommitID): DiffInfo => {
                if (changeType === 'DELETE') {
                    return { base: { rawRepoName, filePath: baseFilePath, commitID: baseCommitID } }
                }
                return {
                    base: { rawRepoName, filePath: baseFilePath, commitID: baseCommitID },
                    head: { rawRepoName, filePath, commitID },
                }
            }
        )
    )
}

export const resolveCommitViewFileInfo = (codeView: HTMLElement): DiffInfo =>
    getFileInfoFromCommitDiffCodeView(codeView)

/**
 * Resolves the file info on a compare page.
 */
export const resolveCompareFileInfo = (codeView: HTMLElement): DiffInfo => {
    const { rawRepoName, filePath, baseFilePath } = getFileInfoWithoutCommitIDsFromMultiFileDiffCodeView(codeView)
    const { baseCommitID, headCommitID } = getCommitInfoFromComparePage()
    return {
        base: {
            rawRepoName,
            filePath: baseFilePath,
            commitID: headCommitID,
        },
        head: { rawRepoName, filePath, commitID: baseCommitID },
    }
}
