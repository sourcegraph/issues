import * as path from 'path'
import { createAggregateError } from '../../../../../shared/src/util/errors'
import { DiffResolvedRevisionSpec } from '../../repo'
import { FileInfo, DiffInfo } from '../shared/codeHost'

export interface BitbucketRepoInfo {
    repoSlug: string
    project: string
}

const LINK_SELECTORS = ['a.raw-view-link', 'a.source-view-link', 'a.mode-source']

const bitbucketToSourcegraphRepoName = ({ repoSlug, project }: BitbucketRepoInfo): string =>
    [window.location.hostname, project, repoSlug].join('/')

/**
 * Attempts to parse the file info from a link element contained in the given
 * single-file code view (both source and "diff to previous" views).
 * Depending on the configuration of the page, this can be a link to the raw file,
 * or to the original source view, so a few different selectors are tried.
 *
 * The href of these links contains:
 * - project name
 * - repo name
 * - file path
 * - revision (through the query parameter `at`)
 */
const getFileInfoFromLinkInSingleFileView = (
    codeView: HTMLElement
): Pick<FileInfo, 'rawRepoName' | 'filePath' | 'revision'> & BitbucketRepoInfo => {
    const errors: Error[] = []
    for (const selector of LINK_SELECTORS) {
        try {
            const linkElement = codeView.querySelector<HTMLLinkElement>(selector)
            if (!linkElement) {
                throw new Error(`Could not find selector ${selector} in code view`)
            }
            const url = new URL(linkElement.href)
            const path = url.pathname

            // Looks like /projects/<project>/repos/<repo>/(browse|raw)/<file path>?at=<revision>
            const pathMatch = path.match(/\/projects\/(.*?)\/repos\/(.*?)\/(?:browse|raw)\/(.*)$/)
            if (!pathMatch) {
                throw new Error(`Path of link matching selector ${selector} did not match path regex: ${path}`)
            }

            const [, project, repoSlug, filePath] = pathMatch

            // Looks like 'refs/heads/<revision>'
            const atParameter = url.searchParams.get('at')
            if (!atParameter) {
                throw new Error(
                    `href of link matching selector ${selector} did not have 'at' search param: ${url.href}`
                )
            }

            const atMatch = atParameter.match(/refs\/heads\/(.*?)$/)

            const revision = atMatch ? atMatch[1] : atParameter

            return {
                rawRepoName: bitbucketToSourcegraphRepoName({ repoSlug, project }),
                filePath: decodeURIComponent(filePath),
                revision,
                project,
                repoSlug,
            }
        } catch (error) {
            errors.push(error)
            continue
        }
    }
    throw createAggregateError(errors)
}

/**
 * Attempts to retrieve the commitid from a link to the commit,
 * found on single file views (both source and "diff to previous" views) and commit pages.
 */
export const getCommitIDFromLink = (selector = 'a.commitid'): string => {
    const commitLink = document.querySelector<HTMLElement>(selector)
    if (!commitLink) {
        throw new Error('No element found matching a.commitid')
    }
    const commitID = commitLink.dataset.commitid
    if (!commitID) {
        throw new Error('Element matching a.commitid has no data-commitid')
    }
    return commitID
}

const getCommitIDFromRevisionSelector = (): string => {
    const revisionSelectorSpan = document.querySelector<HTMLElement>('span[data-revision-ref]')
    if (!revisionSelectorSpan) {
        throw new Error('Could not find span[data-revision-ref] element')
    }
    try {
        const { latestCommit }: { latestCommit: string } = JSON.parse(revisionSelectorSpan.dataset.revisionRef!)
        return latestCommit
    } catch {
        throw new Error('Could not parse JSON from revision selector')
    }
}

/**
 * Gets the file info on a single-file source code view
 */
export const getFileInfoFromSingleFileSourceCodeView = (codeViewElement: HTMLElement): BitbucketRepoInfo & FileInfo => {
    const { rawRepoName, filePath, revision, project, repoSlug } = getFileInfoFromLinkInSingleFileView(codeViewElement)
    const commitID = getCommitIDFromRevisionSelector()
    return {
        rawRepoName,
        filePath,
        revision,
        commitID,
        project,
        repoSlug,
    }
}

/** The type of the change of a file in a diff */
type ChangeType = 'MOVE' | 'RENAME' | 'MODIFY' | 'DELETE' | 'COPY' | 'ADD'

/**
 * Returns true if the active page is a compare view.
 */
export const isCompareView = (): boolean => !!document.querySelector('#branch-compare')

/**
 * Returns true if the active page is a commit view.
 */
export const isCommitsView = ({ pathname }: Pick<Location, 'pathname'>): boolean =>
    /\/projects\/[^/]+\/repos\/[^/]+\/commits\/\w+$/.test(pathname)

/**
 * Returns true if the active page is a pull request view.
 */
export const isPullRequestView = ({ pathname }: Pick<Location, 'pathname'>): boolean =>
    /\/projects\/[^/]+\/repos\/[^/]+\/pull-requests\/\d+/.test(pathname)

/**
 * Returns true if the given code view is a single file source or "diff to previous" view.
 * These views have a toggle to toggle between "source" and "diff to previous".
 */
export const isSingleFileView = (codeViewElement: HTMLElement): boolean =>
    !!codeViewElement.querySelector('.mode-toggle')

/**
 * Gets the change type indicator badge from the given diff code view.
 * Returns `null` if there is no badge on the page (this is expected on single-file diff pages if the file was _modified_).
 */
const getChangeTypeElement = ({ codeViewElement }: { codeViewElement: HTMLElement }): HTMLElement | null =>
    codeViewElement.querySelector<HTMLElement>('.change-type-lozenge')

/**
 * Reads the change type from the change type indicator badge.
 */
const getChangeType = ({ changeTypeElement }: { changeTypeElement: HTMLElement | null }): ChangeType => {
    if (!changeTypeElement) {
        return 'MODIFY'
    }
    const className = [...changeTypeElement.classList].find(className => /^change-type-[A-Z]+/.test(className))
    if (!className) {
        throw new Error('Could not detect change type from change type element')
    }
    return className.replace(/^change-type-/, '') as ChangeType
}

/**
 * Gets the base file path for a diff code view ("diff to previous" or PR) by inspecting the change type badge.
 * Returns `undefined` if there is no base file path (if the file was _added_).
 * Returns `filePath` if the file was _modified_.
 *
 * @param filePath The head file path
 */
const getBaseFilePathForDiffCodeView = ({
    filePath,
    changeType,
    changeTypeElement,
}: {
    changeTypeElement: HTMLElement | null
    changeType: ChangeType
    filePath: string
}): string | undefined => {
    if (changeType === 'ADD') {
        // This file didn't exist in the base
        return undefined
    }
    if (changeType === 'MODIFY' || changeType === 'DELETE') {
        // File path is the same
        return filePath
    }
    if (changeType === 'MOVE' || changeType === 'RENAME' || changeType === 'COPY') {
        if (!changeTypeElement) {
            throw new Error(`Change type is ${changeType} but no change type indicator found`)
        }
        // Need to read previous file path from change type indicator
        // Contains HTML content, example:
        // <span class="deleted">.github</span>/stale.yml &rarr;<br><span class="added">test-dir</span>/stale.yml
        const tooltip = changeTypeElement.getAttribute('original-title')
        if (!tooltip) {
            throw new Error('Moved change type badge did not have original-title attribute')
        }
        const span = document.createElement('span')
        span.innerHTML = tooltip
        const tooltipText = span.textContent!
        if (changeType === 'MOVE' || changeType === 'COPY') {
            const from = tooltipText.split('→')[0].trim()
            if (!from) {
                throw new Error(`Unexpected move change type badge content "${tooltipText}"`)
            }
            return from
        }
        if (changeType === 'RENAME') {
            const renameRegexp = /Renamed from '(.+)'/
            const match = tooltipText.match(renameRegexp)
            if (!match) {
                throw new Error(
                    `Rename change type badge content did not match ${renameRegexp.toString()}: "${tooltipText}"`
                )
            }
            return path.join(path.dirname(filePath), match[1])
        }
    }
    throw new Error(`Unexpected change type "${changeType as string}"`)
}

/**
 * Returns most file info from a single file "diff to previous" code view (excluding `baseCommitID`).
 * The base commit ID needs to be resolved through the API.
 */
export const getFileInfoFromSingleFileDiffCodeView = (
    codeViewElement: HTMLElement
): BitbucketRepoInfo & FileInfo & { baseFilePath: string; changeType: ChangeType } => {
    const { rawRepoName, project, repoSlug, filePath } = getFileInfoFromLinkInSingleFileView(codeViewElement)
    const commitID = getCommitIDFromLink()
    const changeTypeElement = getChangeTypeElement({ codeViewElement })
    const changeType = getChangeType({ changeTypeElement })

    const baseFilePath = getBaseFilePathForDiffCodeView({ changeTypeElement, changeType, filePath }) || filePath
    return {
        changeType,
        rawRepoName,
        filePath,
        baseFilePath,
        commitID,
        project,
        repoSlug,
    }
}

/**
 * Gets most of the file info from the DOM of a multi-file diff code view (PR, compare or commit page).
 *
 * The returned file info does not have the commit ID and base commit ID.
 * Those need to be fetched from the Bitbucket API for PRs,
 * or taken from links on the page for compare and commit pages {@link getCommitInfoFromComparePage}.
 */
export const getFileInfoWithoutCommitIDsFromMultiFileDiffCodeView = (
    codeViewElement: HTMLElement
): BitbucketRepoInfo & {
    changeType: ChangeType
    rawRepoName: string
    filePath: string
    baseFilePath: string
} => {
    // Get the file path from the breadcrumbs
    const breadcrumbsElement = codeViewElement.querySelector('.breadcrumbs')
    if (!breadcrumbsElement) {
        throw new Error('Could not find diff code view breadcrumbs element through selector .breadcrumbs')
    }
    const filePath = breadcrumbsElement.textContent
    if (!filePath) {
        throw new Error('Unexpected empty file path in breadcrumbs')
    }

    // Get project and repo from the URL
    const pathMatch = location.pathname.match(/\/projects\/(.*?)\/repos\/(.*?)\//)
    if (!pathMatch) {
        throw new Error('Location did not match regexp')
    }
    const [, project, repoSlug] = pathMatch
    const rawRepoName = bitbucketToSourcegraphRepoName({ project, repoSlug })

    // Get base file path from the change type indicator
    const changeTypeElement = getChangeTypeElement({ codeViewElement })
    const changeType = getChangeType({ changeTypeElement })
    const baseFilePath = getBaseFilePathForDiffCodeView({ changeTypeElement, changeType, filePath }) || filePath

    return {
        changeType,
        rawRepoName,
        filePath,
        baseFilePath,
        project,
        repoSlug,
    }
}

export const getFileInfoFromCommitDiffCodeView = (codeViewElement: HTMLElement): DiffInfo => {
    const commitID = getCommitIDFromLink('.commit-badge-oneline .commitid')
    const baseCommitID = getCommitIDFromLink('.commit-parents .commitid')
    const { rawRepoName, filePath, baseFilePath } = getFileInfoWithoutCommitIDsFromMultiFileDiffCodeView(
        codeViewElement
    )
    return {
        head: { rawRepoName, filePath, commitID },
        base: {
            rawRepoName,
            filePath: baseFilePath,
            commitID: baseCommitID,
        },
    }
}

export function getPRIDFromPathName(): number {
    const prIDMatch = window.location.pathname.match(/pull-requests\/(\d*?)\/(diff|overview|commits)/)
    if (!prIDMatch) {
        throw new Error(`Could not parse PR ID from pathname: ${window.location.pathname}`)
    }
    return parseInt(prIDMatch[1], 10)
}

/**
 * Gets the head and base commit ID from the comparison pickers on the compare page.
 */
export function getCommitInfoFromComparePage(): DiffResolvedRevisionSpec {
    const headCommitElement = document.querySelector('#branch-compare .source-selector a.commitid[data-commitid]')
    const baseCommitElement = document.querySelector('#branch-compare .target-selector a.commitid[data-commitid]')
    if (!headCommitElement || !baseCommitElement) {
        throw new Error('Could not resolve Bitbucket compare diff spec')
    }
    return {
        headCommitID: headCommitElement.getAttribute('data-commitid')!,
        baseCommitID: baseCommitElement.getAttribute('data-commitid')!,
    }
}
