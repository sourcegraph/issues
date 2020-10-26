import { last, take } from 'lodash'

import { FileSpec, RawRepoSpec, RevisionSpec } from '../../../../../shared/src/util/url'
import { commitIDFromPermalink } from '../../util/dom'
import { FileInfo } from '../shared/codeHost'
import { isExtension } from '../../context'

export enum GitLabPageKind {
    File,
    Commit,
    MergeRequest,
    Other,
}

/**
 * General information that can be found on any GitLab page that we care about. (i.e. has code)
 */
export interface GitLabInfo extends RawRepoSpec {
    pageKind: GitLabPageKind

    owner: string
    projectName: string
}

/**
 * Information about single file pages.
 */
interface GitLabFileInfo extends RawRepoSpec, FileSpec, RevisionSpec {}

export const getPageKindFromPathName = (owner: string, projectName: string, pathname: string): GitLabPageKind => {
    const pageKindMatch = pathname.match(new RegExp(`^/${owner}/${projectName}(/-)?/(commit|merge_requests|blob)/`))
    if (!pageKindMatch) {
        return GitLabPageKind.Other
    }
    switch (pageKindMatch[2]) {
        case 'commit':
            return GitLabPageKind.Commit
        case 'merge_requests':
            return GitLabPageKind.MergeRequest
        case 'blob':
            return GitLabPageKind.File
        default:
            return GitLabPageKind.Other
    }
}

/**
 * Gets information about the page.
 */
export function getPageInfo(): GitLabInfo {
    const projectLink = document.querySelector<HTMLAnchorElement>('.context-header a')
    if (!projectLink) {
        throw new Error('Unable to determine project name')
    }

    const projectFullName = new URL(projectLink.href).pathname.slice(1)

    const parts = projectFullName.split('/')

    const owner = take(parts, parts.length - 1).join('/')
    const projectName = last(parts)!

    const pageKind = getPageKindFromPathName(owner, projectName, window.location.pathname)
    const hostname = isExtension ? window.location.hostname : new URL(gon.gitlab_url).hostname

    return {
        owner,
        projectName,
        rawRepoName: [hostname, owner, projectName].join('/'),
        pageKind,
    }
}

/**
 * Gets information about a file view page.
 */
export function getFilePageInfo(): GitLabFileInfo {
    const { rawRepoName } = getPageInfo()
    const matches = window.location.pathname.match(/\/blob\/(.*?)\/(.*)/)
    if (!matches) {
        throw new Error('Unable to determine revision or file path')
    }

    const revision = decodeURIComponent(matches[1])
    const filePath = decodeURIComponent(matches[2])
    return {
        rawRepoName,
        filePath,
        revision,
    }
}

/**
 * Finds the merge request ID from the URL.
 */
export const getMergeRequestID = (): string => {
    const matches = window.location.pathname.match(/merge_requests\/(.*?)\/diffs/)
    if (!matches) {
        throw new Error('Unable to determine merge request ID')
    }
    return matches[1]
}

/**
 * Finds the diff ID, if any, from the URL.
 * The diff ID represents a specific revision in a merge request.
 */
export const getDiffID = (): string | undefined => {
    const parameters = new URLSearchParams(window.location.search)
    return parameters.get('diff_id') ?? undefined
}

const getFilePathFromElement = (element: HTMLElement): string => {
    const filePath = element.dataset.originalTitle || element.dataset.title || element.title
    if (!filePath) {
        throw new Error('Unable to get file paths from code view: no file title')
    }

    return filePath
}

/**
 * Finds the file paths from the code view. If the name has changed, it'll return the base and head file paths.
 */
export function getFilePathsFromCodeView(codeView: HTMLElement): { headFilePath: string; baseFilePath: string } {
    const filePathElements = codeView.querySelectorAll<HTMLElement>('.file-title-name')
    if (filePathElements.length === 0) {
        throw new Error('Unable to get file paths from code view: no .file-title.name')
    }

    const filePathDidChange = filePathElements.length > 1
    const filePath = getFilePathFromElement(filePathElements.item(filePathDidChange ? 1 : 0))

    return {
        headFilePath: filePath,
        baseFilePath: filePathDidChange ? getFilePathFromElement(filePathElements.item(0)) : filePath,
    }
}

interface GitLabCommitPageInfo extends RawRepoSpec, Pick<GitLabInfo, 'owner' | 'projectName'> {
    commitID: FileInfo['commitID']
}

/**
 * Get the commit from the URL.
 */
export function getCommitPageInfo(): GitLabCommitPageInfo {
    const { rawRepoName, owner, projectName } = getPageInfo()

    return {
        rawRepoName,
        owner,
        projectName,
        commitID: last(window.location.pathname.split('/'))!,
    }
}

/**
 * Get the commit ID from the permalink element on the page.
 */
export function getCommitIDFromPermalink(): string {
    return commitIDFromPermalink({
        selector: '.js-data-file-blob-permalink-url',
        hrefRegex: new RegExp('^/.*?/.*?/blob/([0-9a-f]{40})/'),
    })
}
