import { trimStart } from 'lodash'
import { defer, of } from 'rxjs'
import { map } from 'rxjs/operators'
import { Omit } from 'utility-types'

import { AdjustmentDirection, PositionAdjuster } from '@sourcegraph/codeintellify'
import { NotificationType } from '@sourcegraph/shared/src/api/extension/extensionHostApi'
import { PlatformContext } from '@sourcegraph/shared/src/platform/context'
import { observeSystemIsLightTheme } from '@sourcegraph/shared/src/theme'
import {
    FileSpec,
    RepoSpec,
    ResolvedRevisionSpec,
    RevisionSpec,
    toAbsoluteBlobURL,
} from '@sourcegraph/shared/src/util/url'

import { fetchBlobContentLines } from '../../repo/backend'
import { querySelectorAllOrSelf, querySelectorOrSelf } from '../../util/dom'
import { CodeHost, MountGetter } from '../shared/codeHost'
import { CodeView, toCodeViewResolver } from '../shared/codeViews'
import { createNotificationClassNameGetter } from '../shared/getNotificationClassName'
import { NativeTooltip } from '../shared/nativeTooltips'
import { getSelectionsFromHash, observeSelectionsFromHash } from '../shared/util/selections'
import { ViewResolver } from '../shared/views'

import { markdownBodyViewResolver } from './contentViews'
import { diffDomFunctions, searchCodeSnippetDOMFunctions, singleFileDOMFunctions } from './domFunctions'
import { getCommandPaletteMount } from './extensions'
import { resolveDiffFileInfo, resolveFileInfo, resolveSnippetFileInfo } from './fileInfo'
import { setElementTooltip } from './tooltip'
import { getFileContainers, parseURL } from './util'

/**
 * Creates the mount element for the CodeViewToolbar on code views containing
 * a `.file-actions` element, for instance:
 * - A diff code view on a PR's files page, or a commit page
 * - An older GHE single file code view (newer GitHub.com code views use createFileLineContainerToolbarMount)
 */
export function createFileActionsToolbarMount(codeView: HTMLElement): HTMLElement {
    const className = 'github-file-actions-toolbar-mount'
    const existingMount = codeView.querySelector('.' + className) as HTMLElement
    if (existingMount) {
        return existingMount
    }

    const mountElement = document.createElement('div')
    mountElement.className = className

    const fileActions = codeView.querySelector('.file-actions')
    if (!fileActions) {
        throw new Error('Could not find GitHub file actions with selector .file-actions')
    }

    // Add a class to the .file-actions element, so that we can reliably match it in
    // stylesheets without bleeding CSS to other code hosts (GitLab also uses .file-actions elements).
    fileActions.classList.add('sg-github-file-actions')

    // Old GitHub Enterprise PR views have a "☑ show comments" text that we want to insert *after*
    const showCommentsElement = codeView.querySelector('.show-file-notes')
    if (showCommentsElement) {
        showCommentsElement.after(mountElement)
    } else {
        fileActions.prepend(mountElement)
    }

    return mountElement
}

const toolbarButtonProps = {
    className: 'btn btn-sm tooltipped tooltipped-s',
}

const diffCodeView: Omit<CodeView, 'element'> = {
    dom: diffDomFunctions,
    getToolbarMount: createFileActionsToolbarMount,
    resolveFileInfo: resolveDiffFileInfo,
    toolbarButtonProps,
    getScrollBoundaries: codeView => {
        const fileHeader = codeView.querySelector<HTMLElement>('.file-header')
        if (!fileHeader) {
            throw new Error('Could not find .file-header element in GitHub PR code view')
        }
        return [fileHeader]
    },
}

const diffConversationCodeView: Omit<CodeView, 'element'> = {
    ...diffCodeView,
    getToolbarMount: undefined,
}

const singleFileCodeView: Omit<CodeView, 'element'> = {
    dom: singleFileDOMFunctions,
    getToolbarMount: createFileActionsToolbarMount,
    resolveFileInfo,
    toolbarButtonProps,
    getSelections: getSelectionsFromHash,
    observeSelections: observeSelectionsFromHash,
}

/**
 * Some code snippets get leading white space trimmed. This adjusts based on
 * this. See an example here https://github.com/sourcegraph/browser-extensions/issues/188.
 */
const getSnippetPositionAdjuster = (
    requestGraphQL: PlatformContext['requestGraphQL']
): PositionAdjuster<RepoSpec & RevisionSpec & FileSpec & ResolvedRevisionSpec> => ({ direction, codeView, position }) =>
    fetchBlobContentLines({ ...position, requestGraphQL }).pipe(
        map(lines => {
            const codeElement = singleFileDOMFunctions.getCodeElementFromLineNumber(
                codeView,
                position.line,
                position.part
            )
            if (!codeElement) {
                throw new Error('(adjustPosition) could not find code element for line provided')
            }

            const actualLine = lines[position.line - 1]
            const documentLine = codeElement.textContent || ''

            const actualLeadingWhiteSpace = actualLine.length - trimStart(actualLine).length
            const documentLeadingWhiteSpace = documentLine.length - trimStart(documentLine).length

            const modifier = direction === AdjustmentDirection.ActualToCodeView ? -1 : 1
            const delta = Math.abs(actualLeadingWhiteSpace - documentLeadingWhiteSpace) * modifier

            return {
                line: position.line,
                character: position.character + delta,
            }
        })
    )

const searchResultCodeViewResolver = toCodeViewResolver('.code-list-item', {
    dom: searchCodeSnippetDOMFunctions,
    getPositionAdjuster: getSnippetPositionAdjuster,
    resolveFileInfo: resolveSnippetFileInfo,
    toolbarButtonProps,
})

const snippetCodeView: Omit<CodeView, 'element'> = {
    dom: singleFileDOMFunctions,
    resolveFileInfo: resolveSnippetFileInfo,
    getPositionAdjuster: getSnippetPositionAdjuster,
}

export const createFileLineContainerToolbarMount: NonNullable<CodeView['getToolbarMount']> = (
    codeViewElement: HTMLElement
): HTMLElement => {
    const className = 'sourcegraph-github-file-code-view-toolbar-mount'
    const existingMount = codeViewElement.querySelector(`.${className}`) as HTMLElement
    if (existingMount) {
        return existingMount
    }
    const mountElement = document.createElement('div')
    mountElement.style.display = 'inline-flex'
    mountElement.style.verticalAlign = 'middle'
    mountElement.style.alignItems = 'center'
    mountElement.className = className
    const rawURLLink = codeViewElement.querySelector('#raw-url')
    const buttonGroup = rawURLLink?.closest('.BtnGroup')
    if (!buttonGroup?.parentNode) {
        throw new Error('File actions not found')
    }
    buttonGroup.parentNode.insertBefore(mountElement, buttonGroup)
    return mountElement
}

/**
 * Matches the modern single-file code view, or snippets embedded in comments.
 *
 */
export const fileLineContainerResolver: ViewResolver<CodeView> = {
    selector: '.js-file-line-container',
    resolveView: (fileLineContainer: HTMLElement): CodeView | null => {
        const embeddedBlobWrapper = fileLineContainer.closest('.blob-wrapper-embedded')
        if (embeddedBlobWrapper) {
            // This is a snippet embedded in a comment.
            // Resolve to `.blob-wrapper-embedded`'s parent element,
            // the smallest element that contains both the code and
            // the HTML anchor allowing to resolve the file info.
            const element = embeddedBlobWrapper.parentElement!
            return {
                element,
                ...snippetCodeView,
            }
        }
        const { pageType } = parseURL()
        if (pageType !== 'blob') {
            // this is not a single-file code view
            return null
        }
        const repositoryContent = fileLineContainer.closest('.repository-content')
        if (!repositoryContent) {
            throw new Error('Could not find repository content element')
        }
        return {
            element: repositoryContent as HTMLElement,
            ...singleFileCodeView,
            getToolbarMount: createFileLineContainerToolbarMount,
        }
    },
}

const genericCodeViewResolver: ViewResolver<CodeView> = {
    selector: target => {
        const codeViews = new Set<HTMLElement>()

        // Logic to support large diffs that are loaded asynchronously:
        // https://github.com/sourcegraph/sourcegraph/issues/18337
        // - Don't return `.file` elements that have yet to be loaded (loading is triggered by user)
        // - When the user triggers diff loading, the mutation observer will tell us about
        // .js-blob-wrapper, since the actual '.file' has been in the DOM the whole time. Return
        // the closest ancestor '.file'

        for (const file of querySelectorAllOrSelf<HTMLElement>(target, '.file')) {
            if (file.querySelectorAll('.js-diff-load-container').length === 0) {
                codeViews.add(file)
            }
        }

        for (const blobWrapper of querySelectorAllOrSelf(target, '.js-blob-wrapper')) {
            const file = blobWrapper.closest('.file')
            if (file instanceof HTMLElement) {
                codeViews.add(file)
            }
        }

        return [...codeViews]
    },
    resolveView: (element: HTMLElement): CodeView | null => {
        if (element.querySelector('article.markdown-body')) {
            // This code view is rendered markdown, we shouldn't add code intelligence
            return null
        }

        // This is a suggested change on a GitHub PR
        if (element.closest('.js-suggested-changes-blob')) {
            return null
        }

        const { pageType } = parseURL()
        const isSingleCodeFile =
            pageType === 'blob' &&
            document.querySelectorAll('.file').length === 1 &&
            document.querySelectorAll('.diff-view').length === 0

        if (isSingleCodeFile) {
            return { element, ...singleFileCodeView }
        }

        if (element.closest('.discussion-item-body') || element.classList.contains('js-comment-container')) {
            // This code view is embedded on a PR conversation page.
            return { element, ...diffConversationCodeView }
        }

        return { element, ...diffCodeView }
    },
}

/**
 * Returns true if the current page is GitHub Enterprise.
 */
export function checkIsGitHubEnterprise(): boolean {
    const ogSiteName = document.head.querySelector<HTMLMetaElement>('meta[property="og:site_name"]')
    return (
        !!ogSiteName &&
        // GitHub Enterprise v2.14.11 has "GitHub" as og:site_name
        (ogSiteName.content === 'GitHub Enterprise' || ogSiteName.content === 'GitHub') &&
        document.body.classList.contains('enterprise')
    )
}

/**
 * Returns true if the current page is github.com.
 */
export const checkIsGitHubDotCom = (url = window.location.href): boolean => /^https?:\/\/(www\.)?github\.com/.test(url)

/**
 * Returns true if the current page is either github.com or GitHub Enterprise.
 */
export const checkIsGitHub = (): boolean => checkIsGitHubDotCom() || checkIsGitHubEnterprise()

const OPEN_ON_SOURCEGRAPH_ID = 'open-on-sourcegraph'

const adjustOverlayPosition: CodeHost['adjustOverlayPosition'] = ({ top, left }) => {
    const octotree = document.querySelector('.octotree-sidebar')
    // When browser install octotree plugin, it shows a sidebar on left.
    if (octotree) {
        left += octotree.getBoundingClientRect().right
    }
    return {
        top,
        left,
    }
}

export const createOpenOnSourcegraphIfNotExists: MountGetter = (container: HTMLElement): HTMLElement | null => {
    const pageheadActions = querySelectorOrSelf(container, '.pagehead-actions')
    // If ran on page that isn't under a repository namespace.
    if (!pageheadActions || pageheadActions.children.length === 0) {
        return null
    }
    // Check for existing
    let mount = pageheadActions.querySelector<HTMLElement>('#' + OPEN_ON_SOURCEGRAPH_ID)
    if (mount) {
        return mount
    }
    // Create new
    mount = document.createElement('li')
    mount.id = OPEN_ON_SOURCEGRAPH_ID
    pageheadActions.prepend(mount)
    return mount
}

const nativeTooltipResolver: ViewResolver<NativeTooltip> = {
    selector: '.js-tagsearch-popover',
    resolveView: element => ({ element }),
}

const iconClassName = 'icon--github v-align-text-bottom'

const notificationClassNames = {
    [NotificationType.Log]: 'flash',
    [NotificationType.Success]: 'flash flash-success',
    [NotificationType.Info]: 'flash',
    [NotificationType.Warning]: 'flash flash-warn',
    [NotificationType.Error]: 'flash flash-error',
}

export const githubCodeHost: CodeHost = {
    type: 'github',
    name: checkIsGitHubEnterprise() ? 'GitHub Enterprise' : 'GitHub',
    codeViewResolvers: [genericCodeViewResolver, fileLineContainerResolver, searchResultCodeViewResolver],
    contentViewResolvers: [markdownBodyViewResolver],
    nativeTooltipResolvers: [nativeTooltipResolver],
    adjustOverlayPosition,
    getContext: () => {
        const repoHeaderHasPrivateMarker =
            !!document.querySelector('.repohead .private') ||
            !!document.querySelector('h1 .octicon-lock ~ [itemprop="author"] ~ [itemprop="name"]') ||
            !!(
                document
                    .querySelector('h1 [itemprop="author"] ~ [itemprop="name"] ~ .Label')
                    ?.textContent?.trim()
                    .toLowerCase() === 'private'
            )
        const parsedURL = parseURL()
        return {
            ...parsedURL,
            revision:
                parsedURL.pageType === 'blob' || parsedURL.pageType === 'tree'
                    ? resolveFileInfo().blob.revision
                    : undefined,
            privateRepository: window.location.hostname !== 'github.com' || repoHeaderHasPrivateMarker,
        }
    },
    isLightTheme: defer(() => {
        const mode = document.documentElement.dataset.colorMode as 'auto' | 'light' | 'dark' | undefined
        if (mode === 'auto') {
            return observeSystemIsLightTheme()
        }
        return of(mode !== 'dark')
    }),
    getViewContextOnSourcegraphMount: createOpenOnSourcegraphIfNotExists,
    viewOnSourcegraphButtonClassProps: {
        className: 'btn btn-sm tooltipped tooltipped-s',
        iconClassName,
    },
    check: checkIsGitHub,
    getCommandPaletteMount,
    notificationClassNames,
    commandPaletteClassProps: {
        buttonClassName: 'Header-link',
        popoverClassName: 'Box',
        formClassName: 'p-1',
        inputClassName: 'form-control input-sm header-search-input jump-to-field',
        listClassName: 'p-0 m-0 js-navigation-container jump-to-suggestions-results-container',
        selectedListItemClassName: 'navigation-focus',
        listItemClassName:
            'd-flex flex-justify-start flex-items-center p-0 f5 navigation-item js-navigation-item js-jump-to-scoped-search',
        actionItemClassName:
            'command-palette-action-item--github no-underline d-flex flex-auto flex-items-center jump-to-suggestions-path p-2',
        noResultsClassName: 'd-flex flex-auto flex-items-center jump-to-suggestions-path p-2',
        iconClassName,
    },
    codeViewToolbarClassProps: {
        className: 'code-view-toolbar--github',
        listItemClass: 'code-view-toolbar__item--github BtnGroup',
        actionItemClass: 'btn btn-sm tooltipped tooltipped-s BtnGroup-item action-item--github',
        actionItemPressedClass: 'selected',
        actionItemIconClass: 'icon--github v-align-text-bottom',
    },
    hoverOverlayClassProps: {
        className: 'Box',
        actionItemClassName: 'btn btn-secondary',
        actionItemPressedClassName: 'active',
        closeButtonClassName: 'btn-octicon p-0',
        badgeClassName: 'label',
        getAlertClassName: createNotificationClassNameGetter(notificationClassNames, 'flash-full'),
        iconClassName,
    },
    setElementTooltip,
    linkPreviewContentClass: 'text-small text-gray p-1 mx-1 border rounded-1 bg-gray text-gray-dark',
    urlToFile: (sourcegraphURL, target, context) => {
        if (target.viewState) {
            // A view state means that a panel must be shown, and panels are currently only supported on
            // Sourcegraph (not code hosts).
            return toAbsoluteBlobURL(sourcegraphURL, target)
        }

        // Make sure the location is also on this github instance, return an absolute URL otherwise.
        const sameCodeHost = target.rawRepoName.startsWith(window.location.hostname)
        if (!sameCodeHost) {
            return toAbsoluteBlobURL(sourcegraphURL, target)
        }

        const revision = target.revision || 'HEAD'
        // If we're provided options, we can make the j2d URL more specific.
        const { rawRepoName } = parseURL()

        // Stay on same page in PR if possible.
        // TODO to be entirely correct, this would need to compare the revision of the code view with the target revision.
        const isSameRepo = rawRepoName === target.rawRepoName
        if (isSameRepo && context.part !== undefined) {
            const containers = getFileContainers()
            for (const container of containers) {
                const header = container.querySelector<HTMLElement & { dataset: { path: string; anchor: string } }>(
                    '.file-header[data-path][data-anchor]'
                )
                if (!header) {
                    // E.g. suggestion snippet
                    continue
                }
                const anchorPath = header.dataset.path
                if (anchorPath === target.filePath) {
                    const anchorUrl = header.dataset.anchor
                    const url = new URL(window.location.href)
                    url.hash = anchorUrl
                    if (target.position) {
                        // GitHub uses L for the left side, R for both right side and the unchanged/white parts
                        url.hash += `${context.part === 'base' ? 'L' : 'R'}${target.position.line}`
                    }
                    // Only use URL if it is visible
                    // TODO: Expand hidden lines to reveal
                    if (!document.querySelector(url.hash)) {
                        break
                    }
                    return url.href
                }
            }
        }

        // Go to blob URL
        const fragment = target.position
            ? `#L${target.position.line}${target.position.character ? `:${target.position.character}` : ''}`
            : ''
        return `https://${target.rawRepoName}/blob/${revision}/${target.filePath}${fragment}`
    },
    codeViewsRequireTokenization: true,
}
