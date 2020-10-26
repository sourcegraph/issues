import { RawRepoSpec } from '../../../../../shared/src/util/url'
import { DiffResolvedRevisionSpec } from '../../repo'

/**
 * Returns the elements on the page which should be marked
 * up with tooltips & links:
 *
 * 1. blob view: a single file
 * 2. commit view: one or more file diffs
 * 3. PR conversation view: snippets with inline comments
 * 4. PR unified/split view: one or more file diffs
 */
export function getFileContainers(): NodeListOf<HTMLElement> {
    return document.querySelectorAll<HTMLElement>('.file')
}

/**
 * Returns the path of the file container. It assumes
 * the file container is for a diff (i.e. a commit or pull request view).
 */
export function getDiffFileName(container: HTMLElement): { headFilePath: string; baseFilePath?: string } {
    const fileInfoElement = container.querySelector<HTMLElement>('.file-info')
    if (fileInfoElement) {
        if (fileInfoElement.tagName === 'A') {
            // for PR conversation snippets on GHE, where the .file-info element
            // is the link containing the file paths.
            return getPathNamesFromElement(fileInfoElement)
        }
        // On commit code views, or code views on a PR's files tab,
        // find the link contained in the .file-info element.
        // It is located right of the diffstat (makes sure to not match the code owner link on PRs left of the diffstat).
        const link = fileInfoElement.querySelector<HTMLAnchorElement>('.diffstat + a')
        if (link) {
            return getPathNamesFromElement(link)
        }
    }
    // If no file info element is present, the code view is probably a PR conversation snippet
    // on github.com, where a link containing the file path can be found in the .file-header element.
    const fileHeaderLink = container.querySelector<HTMLLinkElement>('.file-header a')
    if (fileHeaderLink) {
        return getPathNamesFromElement(fileHeaderLink)
    }
    throw new Error('Could not determine diff file name')
}

function getPathNamesFromElement(element: HTMLElement): { headFilePath: string; baseFilePath: string | undefined } {
    const elements = element.title.split(' → ')
    if (elements.length > 1) {
        return { headFilePath: elements[1], baseFilePath: elements[0] }
    }
    return { headFilePath: elements[0], baseFilePath: elements[0] }
}

/**
 * Returns the base and head revision SHA, or null for non-diff views.
 */
export function getDiffResolvedRevision(codeView: HTMLElement): DiffResolvedRevisionSpec | null {
    const { pageType } = parseURL()
    if (!isDiffPageType(pageType)) {
        return null
    }

    let baseCommitID = ''
    let headCommitID = ''
    const fetchContainers = document.querySelectorAll('.js-socket-channel.js-updatable-content.js-pull-refresh-on-pjax')
    const isCommentedSnippet = codeView.classList.contains('js-comment-container')
    if (pageType === 'pull') {
        if (fetchContainers && fetchContainers.length === 1) {
            for (const element of fetchContainers) {
                // for conversation view of pull request
                const url = element.getAttribute('data-url')
                if (!url) {
                    continue
                }
                const parsed = new URL(url, window.location.href)
                baseCommitID = parsed.searchParams.get('base_commit_oid') || ''
                headCommitID = parsed.searchParams.get('end_commit_oid') || ''
            }
        } else if (isCommentedSnippet) {
            const resolvedDiffSpec = getResolvedDiffFromCommentedSnippet(codeView)
            if (resolvedDiffSpec) {
                return resolvedDiffSpec
            }
        } else {
            // Last-ditch: look for inline comment form input which has base/head on it.
            const baseInput = document.querySelector('input[name="comparison_start_oid"]')
            if (baseInput) {
                baseCommitID = (baseInput as HTMLInputElement).value
            }
            const headInput = document.querySelector('input[name="comparison_end_oid"]')
            if (headInput) {
                headCommitID = (headInput as HTMLInputElement).value
            }
        }
    } else if (pageType === 'commit') {
        // Refined GitHub adds a `.patch-diff-links` element
        const shaContainers = document.querySelectorAll('.sha-block:not(.patch-diff-links)')
        if (shaContainers && shaContainers.length === 2) {
            const baseShaElement = shaContainers[0].querySelector('a')
            if (baseShaElement) {
                // e.g "https://github.com/gorilla/mux/commit/0b13a922203ebdbfd236c818efcd5ed46097d690"
                baseCommitID = baseShaElement.href.split('/').slice(-1)[0]
            }
            const headShaElement = shaContainers[1].querySelector('span.sha') as HTMLElement
            if (headShaElement) {
                headCommitID = headShaElement.innerHTML
            }
        }
    } else if (pageType === 'compare') {
        const resolvedDiffSpec = getResolvedDiffForCompare()
        if (resolvedDiffSpec) {
            return resolvedDiffSpec
        }
    }

    if (baseCommitID === '' || headCommitID === '') {
        return getDiffResolvedRevisionFromPageSource(document.documentElement.innerHTML, pageType === 'pull')
    }
    return { baseCommitID, headCommitID }
}

// ".../files/(BASE..)?HEAD#diff-DIFF"
// https://github.com/sourcegraph/codeintellify/pull/77/files/e8ffee0c59e951d29bcc7cff7d58caff1c5c97c2..ce472adbfc6ac8ccf1bf7afbe71f18505ca994ec#diff-8a128e9e8a5a8bb9767f5f5392391217
// https://github.com/lguychard/sourcegraph-configurable-references/pull/1/files/fa32ce95d666d73cf4cb3e13b547993374eb158d#diff-45327f86d4438556066de133327f4ca2
const COMMENTED_SNIPPET_DIFF_REGEX = /\/files\/((\w+)\.\.)?(\w+)#diff-\w+$/

function getResolvedDiffFromCommentedSnippet(codeView: HTMLElement): DiffResolvedRevisionSpec | null {
    // For commented snippets, try to get the HEAD commit ID from the file header,
    // as it will always be the most accurate (for example in the case of outdated snippets).
    const linkToFile: HTMLLinkElement | null = codeView.querySelector('.file-header a')
    if (!linkToFile) {
        return null
    }
    const match = linkToFile.href.match(COMMENTED_SNIPPET_DIFF_REGEX)
    if (!match) {
        return null
    }
    const headCommitID = match[3]
    // The file header may not contain the base commit ID, so we get it from the page source.
    const resolvedRevisionFromPageSource = getDiffResolvedRevisionFromPageSource(
        document.documentElement.innerHTML,
        true
    )
    return headCommitID && resolvedRevisionFromPageSource
        ? {
              ...resolvedRevisionFromPageSource,
              headCommitID,
          }
        : null
}

function getResolvedDiffForCompare(): DiffResolvedRevisionSpec | undefined {
    const [base, head] = document.querySelectorAll<HTMLElement>('.commitish-suggester .select-menu-button span')
    if (base && head && base.textContent && head.textContent) {
        return {
            baseCommitID: base.textContent,
            headCommitID: head.textContent,
        }
    }
    return undefined
}

function getDiffResolvedRevisionFromPageSource(
    pageSource: string,
    isPullRequest: boolean
): DiffResolvedRevisionSpec | null {
    if (!isPullRequest) {
        return null
    }
    const baseShaComment = '<!-- base sha1: &quot;'
    const baseIndex = pageSource.indexOf(baseShaComment)

    if (baseIndex === -1) {
        return null
    }

    const headShaComment = '<!-- head sha1: &quot;'
    const headIndex = pageSource.indexOf(headShaComment, baseIndex)
    if (headIndex === -1) {
        return null
    }

    const baseCommitID = pageSource.slice(baseIndex + baseShaComment.length, 40)
    const headCommitID = pageSource.slice(headIndex + headShaComment.length, 40)
    return {
        baseCommitID,
        headCommitID,
    }
}

/**
 * Returns the file path for the current page. Must be on a blob or tree page.
 *
 * Implementation details:
 *
 * This scrapes the file path from the permalink on GitHub blob pages:
 * ```html
 * <a class="d-none js-permalink-shortcut" data-hotkey="y" href="/gorilla/mux/blob/ed099d42384823742bba0bf9a72b53b55c9e2e38/mux.go">Permalink</a>
 * ```
 *
 * We can't get the file path from the URL because the branch name can contain
 * slashes which make the boundary between the branch name and file path
 * ambiguous. For example: https://github.com/sourcegraph/sourcegraph/blob/bext/release/cmd/frontend/internal/session/session.go
 *
 * TODO ideally, this should only scrape the code view itself.
 */
export function getFilePath(): string {
    const permalink = document.querySelector<HTMLAnchorElement>('a.js-permalink-shortcut')
    if (!permalink) {
        throw new Error('Unable to determine the file path because no a.js-permalink-shortcut element was found.')
    }
    const url = new URL(permalink.href)
    // <empty>/<user>/<repo>/(blob|tree)/<commitID>/<path/to/file>
    // eslint-disable-next-line unicorn/no-unreadable-array-destructuring
    const [, , , pageType, , ...path] = url.pathname.split('/')
    // Check for page type because a tree page can be the repo root, so it shouldn't throw an error despite an empty path
    if (pageType !== 'tree' && path.length === 0) {
        throw new Error(
            `Unable to determine the file path because the a.js-permalink-shortcut element's href's path was ${url.pathname} (it is expected to be of the form /<user>/<repo>/blob/<commitID>/<path/to/file>).`
        )
    }
    return decodeURIComponent(path.join('/'))
}

type GitHubURL = RawRepoSpec &
    (
        | { pageType: 'commit' | 'pull' | 'compare' | 'other' }
        | {
              pageType: 'blob' | 'tree'
              /** revision and file path separated by a slash, URL-decoded. */
              revisionAndFilePath: string
          }
    )

export function isDiffPageType(pageType: GitHubURL['pageType']): boolean {
    switch (pageType) {
        case 'commit':
        case 'pull':
        case 'compare':
            return true
        default:
            return false
    }
}

export function parseURL(location: Pick<Location, 'host' | 'pathname' | 'href'> = window.location): GitHubURL {
    const { host, pathname } = location
    const [user, ghRepoName, pageType, ...rest] = pathname.slice(1).split('/')
    if (!user || !ghRepoName) {
        throw new Error(`Could not parse repoName from GitHub url: ${location.href}`)
    }
    const rawRepoName = `${host}/${user}/${ghRepoName}`
    switch (pageType) {
        case 'blob':
        case 'tree':
            return {
                pageType,
                rawRepoName,
                revisionAndFilePath: decodeURIComponent(rest.join('/')),
            }
        case 'pull':
        case 'commit':
        case 'compare':
            return { pageType, rawRepoName }
        default:
            return { pageType: 'other', rawRepoName }
    }
}
