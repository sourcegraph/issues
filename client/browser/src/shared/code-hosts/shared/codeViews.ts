import { DiffPart, DOMFunctions as CodeIntellifyDOMFuncions, PositionAdjuster } from '@sourcegraph/codeintellify'
import { Selection } from '@sourcegraph/extension-api-types'
import { Observable, of, zip, OperatorFunction } from 'rxjs'
import { catchError, map, switchMap } from 'rxjs/operators'
import { Omit } from 'utility-types'
import { isPrivateRepoPublicSourcegraphComErrorLike } from '../../../../../shared/src/backend/errors'
import { PlatformContext } from '../../../../../shared/src/platform/context'
import { FileSpec, RepoSpec, ResolvedRevisionSpec, RevisionSpec } from '../../../../../shared/src/util/url'
import { ButtonProps } from '../../components/CodeViewToolbar'
import { fetchBlobContentLines } from '../../repo/backend'
import { CodeHost, FileInfoWithRepoName, DiffOrBlobInfo, FileInfoWithContent } from './codeHost'
import { trackViews, ViewResolver, ViewWithSubscriptions } from './views'
import { MutationRecordLike } from '../../util/dom'
import { ensureRevisionIsClonedForFileInfo } from './util/fileInfo'

export interface DOMFunctions extends CodeIntellifyDOMFuncions {
    /**
     * Gets the element for the entire line. This element is used for whole-line
     * background decorations. It should span the entire width of the line
     * independent on how long the code on that line is. This may be a parent
     * element of the code element, but keep in mind that even in split diff
     * views it must only contain the line the given diff part.
     */
    getLineElementFromLineNumber: (codeView: HTMLElement, line: number, part?: DiffPart) => HTMLElement | null
}

/**
 * Defines a code view that is present on a page.
 * Exposes operations for manipulating it, and CSS classes to be applied to injected UI elements.
 */
export interface CodeView {
    /**
     * The code view element on the page.
     */
    element: HTMLElement
    /** The DOMFunctions for the code view. */
    dom: DOMFunctions
    /**
     * Finds or creates a DOM element where we should inject the
     * `CodeViewToolbar`. This function is responsible for ensuring duplicate
     * mounts aren't created.
     */
    getToolbarMount?: (codeView: HTMLElement) => HTMLElement
    /**
     * Resolves the file info for a given code view. It returns an observable
     * because some code hosts need to resolve this asynchronously. The
     * observable should only emit once.
     */
    resolveFileInfo: (
        codeView: HTMLElement,
        requestGraphQL: PlatformContext['requestGraphQL']
    ) => Observable<DiffOrBlobInfo> | DiffOrBlobInfo
    /**
     * In some situations, we need to be able to adjust the position going into
     * and coming out of codeintellify. For example, Phabricator converts tabs
     * to spaces in it's DOM.
     */
    getPositionAdjuster?: (
        requestGraphQL: PlatformContext['requestGraphQL']
    ) => PositionAdjuster<RepoSpec & RevisionSpec & FileSpec & ResolvedRevisionSpec>
    /** Props for styling the buttons in the `CodeViewToolbar`. */
    toolbarButtonProps?: ButtonProps
    /**
     * Gets the current selections for a code view.
     */
    getSelections?: (codeViewElement: HTMLElement) => Selection[]
    /**
     * Returns a stream of selections changes for a code view.
     */
    observeSelections?: (codeViewElement: HTMLElement) => Observable<Selection[]>

    /**
     * Returns the scrollBoundaries of the code view, used by codeintellify.
     * This is called once per code view, when calling Hoverifier.hoverify().
     */
    getScrollBoundaries?: (codeViewElement: HTMLElement) => HTMLElement[]
}

/**
 * Builds a CodeViewResolver from a static CodeView and a selector.
 */
export const toCodeViewResolver = (selector: string, spec: Omit<CodeView, 'element'>): ViewResolver<CodeView> => ({
    selector,
    resolveView: element => ({ ...spec, element }),
})

/**
 * Find all the code views on a page using both the code view specs and the code view spec
 * resolvers, calling down to {@link trackViews}.
 */
export const trackCodeViews = ({
    codeViewResolvers,
}: Pick<CodeHost, 'codeViewResolvers'>): OperatorFunction<MutationRecordLike[], ViewWithSubscriptions<CodeView>> =>
    trackViews<CodeView>(codeViewResolvers)

const fetchFileContentForFileInfo = (
    fileInfo: FileInfoWithRepoName,
    requestGraphQL: PlatformContext['requestGraphQL']
): Observable<FileInfoWithContent> =>
    ensureRevisionIsClonedForFileInfo(fileInfo, requestGraphQL).pipe(
        switchMap(() =>
            fetchBlobContentLines({
                repoName: fileInfo.repoName,
                filePath: fileInfo.filePath,
                commitID: fileInfo.commitID,
                requestGraphQL,
            })
        ),
        map(content => {
            if (content) {
                return { ...fileInfo, content: content.join('\n') }
            }
            return { ...fileInfo }
        }),
        catchError(error => {
            if (isPrivateRepoPublicSourcegraphComErrorLike(error)) {
                // In this case, fileInfo will have undefined content.
                return of(fileInfo)
            }
            throw error
        })
    )

export const fetchFileContentForDiffOrFileInfo = (
    diffOrBlobInfo: DiffOrBlobInfo<FileInfoWithRepoName>,
    requestGraphQL: PlatformContext['requestGraphQL']
): Observable<DiffOrBlobInfo<FileInfoWithContent>> => {
    if ('blob' in diffOrBlobInfo) {
        return fetchFileContentForFileInfo(diffOrBlobInfo.blob, requestGraphQL).pipe(
            map(fileInfo => ({ blob: fileInfo }))
        )
    }
    if (diffOrBlobInfo.head && diffOrBlobInfo.base) {
        const fetchingBaseFile = fetchFileContentForFileInfo(diffOrBlobInfo.base, requestGraphQL)
        const fetchingHeadFile = fetchFileContentForFileInfo(diffOrBlobInfo.head, requestGraphQL)

        return zip(fetchingBaseFile, fetchingHeadFile).pipe(
            map(([base, head]): DiffOrBlobInfo<FileInfoWithContent> => ({ head, base }))
        )
    }
    if (diffOrBlobInfo.head) {
        return fetchFileContentForFileInfo(diffOrBlobInfo.head, requestGraphQL).pipe(
            map((head): DiffOrBlobInfo<FileInfoWithContent> => ({ head }))
        )
    }
    return fetchFileContentForFileInfo(diffOrBlobInfo.base, requestGraphQL).pipe(map(base => ({ base })))
}
