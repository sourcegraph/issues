import { findPositionsFromEvents, Hoverifier } from '@sourcegraph/codeintellify'
import { TextDocumentDecoration } from '@sourcegraph/extension-api-types'
import * as H from 'history'
import { isEqual } from 'lodash'
import React, { useCallback, useMemo, useState, useEffect } from 'react'
import { combineLatest, NEVER, Observable, of, Subject } from 'rxjs'
import { distinctUntilChanged, filter, switchMap } from 'rxjs/operators'
import { ActionItemAction } from '../../../../shared/src/actions/ActionItem'
import { DecorationMapByLine, groupDecorationsByLine } from '../../../../shared/src/api/client/services/decoration'
import { HoverMerged } from '../../../../shared/src/api/client/types/hover'
import { ExtensionsControllerProps } from '../../../../shared/src/extensions/controller'
import { isDefined } from '../../../../shared/src/util/types'
import { FileSpec, RepoSpec, ResolvedRevisionSpec, RevisionSpec, toURIWithPath } from '../../../../shared/src/util/url'
import { ThemeProps } from '../../../../shared/src/theme'
import { DiffHunk } from './DiffHunk'
import { diffDomFunctions } from '../../repo/compare/dom-functions'
import { FileDiffFields, Scalars } from '../../graphql-operations'

interface PartFileInfo {
    repoName: string
    repoID: Scalars['ID']
    revision: string
    commitID: string

    /**
     * `null` if the file does not exist in this diff part.
     */
    filePath: string | null
}

interface FileHunksProps extends ThemeProps {
    /** The anchor (URL hash link) of the file diff. The component creates sub-anchors with this prefix. */
    fileDiffAnchor: string

    /**
     * Information needed to apply extensions (hovers, decorations, ...) on the diff.
     * If undefined, extensions will not be applied on this diff.
     */
    extensionInfo?: {
        /** The base repository, revision, and file. */
        base: PartFileInfo

        /** The head repository, revision, and file. */
        head: PartFileInfo
        hoverifier: Hoverifier<RepoSpec & RevisionSpec & FileSpec & ResolvedRevisionSpec, HoverMerged, ActionItemAction>
    } & ExtensionsControllerProps

    /** The file's hunks. */
    hunks: FileDiffFields['hunks']

    /** Whether to show line numbers. */
    lineNumbers: boolean

    className: string
    location: H.Location
    history: H.History
    /** Reflect selected line in url */
    persistLines?: boolean
}

/** Displays hunks in a unified file diff. */
export const FileDiffHunks: React.FunctionComponent<FileHunksProps> = ({
    className,
    fileDiffAnchor,
    history,
    hunks,
    isLightTheme,
    lineNumbers,
    location,
    extensionInfo,
    persistLines,
}) => {
    /**
     * Decorations for the file at the two revisions of the diff
     */
    const [decorations, setDecorations] = useState<Record<'head' | 'base', DecorationMapByLine>>({
        head: new Map(),
        base: new Map(),
    })

    /** Emits whenever the ref callback for the code element is called */
    const codeElements = useMemo(() => new Subject<HTMLElement | null>(), [])
    const nextCodeElement = useCallback((element: HTMLElement | null): void => codeElements.next(element), [
        codeElements,
    ])

    /** Emits whenever the ref callback for the blob element is called */
    const blobElements = useMemo(() => new Subject<HTMLElement | null>(), [])
    const nextBlobElement = useCallback((element: HTMLElement | null): void => blobElements.next(element), [
        blobElements,
    ])

    useEffect(() => {
        if (!extensionInfo) {
            return () => undefined
        }
        const subscription = extensionInfo.hoverifier.hoverify({
            dom: diffDomFunctions,
            positionEvents: codeElements.pipe(
                filter(isDefined),
                findPositionsFromEvents({ domFunctions: diffDomFunctions })
            ),
            positionJumps: NEVER, // TODO support diff URLs
            resolveContext: hoveredToken => {
                // if part is undefined, it doesn't matter whether we chose head or base, the line stayed the same
                const { repoName, revision, filePath, commitID } = extensionInfo[hoveredToken.part || 'head']
                // If a hover or go-to-definition was invoked on this part, we know the file path must exist
                return { repoName, filePath: filePath!, revision, commitID }
            },
        })
        return () => subscription.unsubscribe()
    }, [codeElements, extensionInfo])

    // Listen to decorations from extensions and group them by line
    useEffect(() => {
        const subscription = of(extensionInfo)
            .pipe(
                filter(isDefined),
                distinctUntilChanged(
                    (a, b) =>
                        isEqual(a.head, b.head) &&
                        isEqual(a.base, b.base) &&
                        a.extensionsController !== b.extensionsController
                ),
                switchMap(({ head, base, extensionsController }) => {
                    const getDecorationsForPart = ({
                        repoName,
                        commitID,
                        filePath,
                    }: PartFileInfo): Observable<TextDocumentDecoration[] | null> =>
                        filePath !== null
                            ? extensionsController.services.textDocumentDecoration.getDecorations({
                                  uri: toURIWithPath({ repoName, commitID, filePath }),
                              })
                            : of(null)
                    return combineLatest([getDecorationsForPart(head), getDecorationsForPart(base)])
                })
            )
            .subscribe(([headDecorations, baseDecorations]) => {
                setDecorations({
                    head: groupDecorationsByLine(headDecorations),
                    base: groupDecorationsByLine(baseDecorations),
                })
            })
        return () => subscription.unsubscribe()
    }, [extensionInfo])

    return (
        <div className={`file-diff-hunks ${className}`} ref={nextBlobElement}>
            {hunks.length === 0 ? (
                <div className="text-muted m-2">No changes</div>
            ) : (
                <div className="file-diff-hunks__container" ref={nextCodeElement}>
                    <table className="file-diff-hunks__table">
                        {lineNumbers && (
                            <colgroup>
                                <col width="40" />
                                <col width="40" />
                                <col />
                            </colgroup>
                        )}
                        <tbody>
                            {hunks.map((hunk, index) => (
                                <DiffHunk
                                    fileDiffAnchor={fileDiffAnchor}
                                    history={history}
                                    isLightTheme={isLightTheme}
                                    lineNumbers={lineNumbers}
                                    location={location}
                                    persistLines={persistLines}
                                    key={index}
                                    hunk={hunk}
                                    decorations={decorations}
                                />
                            ))}
                        </tbody>
                    </table>
                </div>
            )}
        </div>
    )
}
