import { AdjustmentDirection, PositionAdjuster } from '@sourcegraph/codeintellify'
import { of } from 'rxjs'
import { Omit } from 'utility-types'
import { FileSpec, RepoSpec, ResolvedRevisionSpec, RevisionSpec } from '../../../../../shared/src/util/url'
import { querySelectorOrSelf } from '../../util/dom'
import { CodeHost, MountGetter } from '../shared/codeHost'
import { CodeView, DOMFunctions } from '../shared/codeViews'
import { ViewResolver } from '../shared/views'
import { getContext } from './context'
import { diffDOMFunctions, singleFileDOMFunctions } from './domFunctions'
import {
    resolveCommitViewFileInfo,
    resolveCompareFileInfo,
    resolveFileInfoForSingleFileSourceView,
    resolvePullRequestFileInfo,
    resolveSingleFileDiffFileInfo,
} from './fileInfo'
import { isCommitsView, isCompareView, isPullRequestView, isSingleFileView } from './scrape'
import { NotificationType } from '../../../../../shared/src/api/client/services/notifications'

/**
 * Gets or creates the toolbar mount for allcode views.
 */
export const getToolbarMount = (codeView: HTMLElement): HTMLElement => {
    const existingMount = codeView.querySelector<HTMLElement>('.sg-toolbar-mount')
    if (existingMount) {
        return existingMount
    }

    const fileActions = codeView.querySelector<HTMLElement>('.file-toolbar .secondary')
    if (!fileActions) {
        throw new Error('Unable to find mount location')
    }

    const mount = document.createElement('div')
    mount.classList.add('btn-group')
    mount.classList.add('sg-toolbar-mount')
    mount.classList.add('sg-toolbar-mount-bitbucket-server')

    fileActions.prepend(mount)

    return mount
}

/**
 * Sometimes tabs are converted to spaces so we need to adjust. Luckily, there
 * is an attribute `cm-text` that contains the real text.
 */
const createPositionAdjuster = (
    dom: DOMFunctions
): PositionAdjuster<RepoSpec & RevisionSpec & FileSpec & ResolvedRevisionSpec> => ({
    direction,
    codeView,
    position,
}) => {
    const codeElement = dom.getCodeElementFromLineNumber(codeView, position.line, position.part)
    if (!codeElement) {
        throw new Error('(adjustPosition) could not find code element for line provided')
    }

    let delta = 0
    for (const modifiedTextElement of codeElement.querySelectorAll('[cm-text]')) {
        const actualText = modifiedTextElement.getAttribute('cm-text') || ''
        const adjustedText = modifiedTextElement.textContent || ''

        delta += actualText.length - adjustedText.length
    }

    const modifier = direction === AdjustmentDirection.ActualToCodeView ? -1 : 1

    const newPosition = {
        line: position.line,
        character: position.character + modifier * delta,
    }

    return of(newPosition)
}

const toolbarButtonProps = {
    className: 'aui-button',
}

/**
 * A code view spec for single file code view in the "source" view (not diff).
 */
const singleFileSourceCodeView: Omit<CodeView, 'element'> = {
    getToolbarMount,
    dom: singleFileDOMFunctions,
    resolveFileInfo: resolveFileInfoForSingleFileSourceView,
    getPositionAdjuster: () => createPositionAdjuster(singleFileDOMFunctions),
    toolbarButtonProps,
}

const baseDiffCodeView: Omit<CodeView, 'element' | 'resolveFileInfo'> = {
    getToolbarMount,
    dom: diffDOMFunctions,
    getPositionAdjuster: () => createPositionAdjuster(diffDOMFunctions),
    toolbarButtonProps,
}
/**
 * A code view spec for a single file "diff to previous" view
 */
const singleFileDiffCodeView: Omit<CodeView, 'element'> = {
    ...baseDiffCodeView,
    resolveFileInfo: resolveSingleFileDiffFileInfo,
}

/**
 * A code view spec for pull requests
 */
const pullRequestDiffCodeView: Omit<CodeView, 'element'> = {
    ...baseDiffCodeView,
    resolveFileInfo: resolvePullRequestFileInfo,
}

/**
 * A code view spec for compare pages
 */
const compareDiffCodeView: Omit<CodeView, 'element'> = {
    ...baseDiffCodeView,
    resolveFileInfo: resolveCompareFileInfo,
}

/**
 * A code view spec for commit pages
 */
const commitDiffCodeView: Omit<CodeView, 'element'> = {
    ...baseDiffCodeView,
    resolveFileInfo: resolveCommitViewFileInfo,
}

const codeViewResolver: ViewResolver<CodeView> = {
    selector: '.file-content',
    resolveView: element => {
        const contentView = element.querySelector('.content-view')
        if (!contentView) {
            return null
        }
        if (isCompareView()) {
            return { element, ...compareDiffCodeView }
        }
        if (isCommitsView(window.location)) {
            return { element, ...commitDiffCodeView }
        }
        if (isSingleFileView(element)) {
            const isDiff = contentView.classList.contains('diff-view')
            return isDiff ? { element, ...singleFileDiffCodeView } : { element, ...singleFileSourceCodeView }
        }
        if (isPullRequestView(window.location)) {
            return { element, ...pullRequestDiffCodeView }
        }
        console.error('Unknown code view', element)
        return null
    },
}

const getCommandPaletteMount: MountGetter = (container: HTMLElement): HTMLElement | null => {
    const headerElement = querySelectorOrSelf(container, '.aui-header-primary .aui-nav')
    if (!headerElement) {
        return null
    }
    const classNames = ['command-palette-button', 'command-palette-button--bitbucket-server']
    const create = (): HTMLElement => {
        const mount = document.createElement('li')
        mount.className = classNames.join(' ')
        headerElement.append(mount)
        return mount
    }
    const preexisting = headerElement.querySelector<HTMLElement>(classNames.map(className => `.${className}`).join(''))
    return preexisting || create()
}

function getViewContextOnSourcegraphMount(container: HTMLElement): HTMLElement | null {
    const branchSelectorButtons = querySelectorOrSelf(container, '.branch-selector-toolbar .aui-buttons')
    if (!branchSelectorButtons) {
        return null
    }
    const preexisting = branchSelectorButtons.querySelector<HTMLElement>('#open-on-sourcegraph')
    if (preexisting) {
        return preexisting
    }
    const mount = document.createElement('span')
    mount.id = 'open-on-sourcegraph'
    mount.className = 'open-on-sourcegraph--bitbucket-server'
    branchSelectorButtons.append(mount)
    return mount
}

export const checkIsBitbucket = (): boolean =>
    !!document.querySelector('.bitbucket-header-logo') ||
    !!document.querySelector('.aui-header-logo.aui-header-logo-bitbucket')

const iconClassName = 'aui-icon'

const notificationClassNames = {
    [NotificationType.Log]: 'aui-message aui-message-info',
    [NotificationType.Success]: 'aui-message aui-message-success',
    [NotificationType.Info]: 'aui-message aui-message-info',
    [NotificationType.Warning]: 'aui-message aui-message-warning',
    [NotificationType.Error]: 'aui-message aui-message-error',
}

export const bitbucketServerCodeHost: CodeHost = {
    type: 'bitbucket-server',
    name: 'Bitbucket Server',
    check: checkIsBitbucket,
    codeViewResolvers: [codeViewResolver],
    getCommandPaletteMount,
    notificationClassNames,
    commandPaletteClassProps: {
        buttonClassName:
            'command-list-popover-button--bitbucket-server aui-alignment-target aui-alignment-abutted aui-alignment-abutted-left aui-alignment-element-attached-top aui-alignment-element-attached-left aui-alignment-target-attached-bottom aui-alignment-target-attached-left',
        buttonElement: 'a',
        buttonOpenClassName: 'aui-dropdown2-active active aui-alignment-enabled',
        showCaret: false,
        popoverClassName:
            'command-palette-popover--bitbucket-server aui-dropdown2 aui-style-default aui-layer aui-dropdown2-in-header aui-alignment-element aui-alignment-side-bottom aui-alignment-snap-left aui-alignment-enabled aui-alignment-abutted aui-alignment-abutted-left aui-alignment-element-attached-top aui-alignment-element-attached-left aui-alignment-target-attached-bottom aui-alignment-target-attached-left',
        popoverInnerClassName: 'aui-dropdown2-section',
        formClassName: 'aui',
        inputClassName: 'text',
        resultsContainerClassName: 'results',
        listClassName: 'results-list',
        listItemClassName: 'result',
        selectedListItemClassName: 'focused',
        noResultsClassName: 'no-results',
        iconClassName,
    },
    codeViewToolbarClassProps: {
        className: 'code-view-toolbar--bitbucket aui-buttons',
        actionItemClass: 'aui-button action-item--bitbucket-server',
        // actionItemPressedClass is not needed because Bitbucket applies styling to aria-pressed="true"
        actionItemIconClass: 'aui-icon',
        listItemClass: 'action-nav-item--bitbucket',
    },
    hoverOverlayClassProps: {
        className: 'aui-dialog',
        actionItemClassName: 'aui-button hover-action-item--bitbucket-server',
        iconButtonClassName: 'aui-button btn-icon--bitbucket-server',
        infoAlertClassName: notificationClassNames[NotificationType.Info],
        errorAlertClassName: notificationClassNames[NotificationType.Error],
        iconClassName,
    },
    getViewContextOnSourcegraphMount,
    getContext,
    viewOnSourcegraphButtonClassProps: {
        className: 'aui-button',
        iconClassName,
    },
    codeViewsRequireTokenization: false,
}
