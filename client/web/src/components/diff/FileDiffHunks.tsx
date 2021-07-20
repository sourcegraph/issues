import classNames from 'classnames'
import * as H from 'history'
import React, { useCallback, useMemo, useState, useEffect } from 'react'
import { combineLatest, from, NEVER, Observable, of, ReplaySubject, Subscription } from 'rxjs'
import { distinctUntilKeyChanged, filter, first, map, switchMap, tap } from 'rxjs/operators'
import { useDeepCompareEffectNoCheck } from 'use-deep-compare-effect'

import { findPositionsFromEvents } from '@sourcegraph/codeintellify'
import { wrapRemoteObservable } from '@sourcegraph/shared/src/api/client/api/common'
import { DecorationMapByLine, groupDecorationsByLine } from '@sourcegraph/shared/src/api/extension/api/decorations'
import { ViewerId } from '@sourcegraph/shared/src/api/viewerTypes'
import { ThemeProps } from '@sourcegraph/shared/src/theme'
import { isDefined, property } from '@sourcegraph/shared/src/util/types'
import { toURIWithPath } from '@sourcegraph/shared/src/util/url'
import { useObservable } from '@sourcegraph/shared/src/util/useObservable'

import { StatusBar } from '../../extensions/components/StatusBar'
import { FileDiffFields } from '../../graphql-operations'
import { DiffMode } from '../../repo/commit/RepositoryCommitPage'
import { diffDomFunctions } from '../../repo/compare/dom-functions'

import { DiffHunk } from './DiffHunk'
import { DiffSplitHunk } from './DiffSplitHunk'
import { ExtensionInfo } from './FileDiffConnection'

export interface FileHunksProps extends ThemeProps {
    /** The anchor (URL hash link) of the file diff. The component creates sub-anchors with this prefix. */
    fileDiffAnchor: string

    /**
     * Information needed to apply extensions (hovers, decorations, ...) on the diff.
     * If undefined, extensions will not be applied on this diff.
     */
    extensionInfo?: ExtensionInfo<
        {
            observeViewerId?: (uri: string) => Observable<ViewerId | undefined>
        },
        {
            /**
             * `null` if the file does not exist in this diff part.
             */
            filePath: string | null
        }
    >

    /** The file's hunks. */
    hunks: FileDiffFields['hunks']

    /** Whether to show line numbers. */
    lineNumbers: boolean

    className: string
    location: H.Location
    history: H.History
    /** Reflect selected line in url */
    persistLines?: boolean
    diffMode: DiffMode
}

/** Displays hunks in a unified file diff. */
export const FileDiffHunks: React.FunctionComponent<FileHunksProps> = ({
    className,
    fileDiffAnchor,
    hunks,
    isLightTheme,
    lineNumbers,
    location,
    extensionInfo,
    persistLines,
    diffMode,
}) => {
    /**
     * Decorations for the file at the two revisions of the diff
     */
    const [decorations, setDecorations] = useState<Record<'head' | 'base', DecorationMapByLine>>({
        head: new Map(),
        base: new Map(),
    })

    /** Emits whenever the ref callback for the code element is called */
    const codeElements = useMemo(() => new ReplaySubject<HTMLElement | null>(1), [])
    const nextCodeElement = useCallback((element: HTMLElement | null): void => codeElements.next(element), [
        codeElements,
    ])

    /** Emits whenever the ref callback for the blob element is called */
    const blobElements = useMemo(() => new ReplaySubject<HTMLElement | null>(1), [])
    const nextBlobElement = useCallback((element: HTMLElement | null): void => blobElements.next(element), [
        blobElements,
    ])

    const extensionInfoChanges = useMemo(() => new ReplaySubject<FileHunksProps['extensionInfo'] | undefined>(1), [])
    useDeepCompareEffectNoCheck(() => {
        extensionInfoChanges.next(extensionInfo)
        // Use `useDeepCompareEffectNoCheck` since extensionInfo can be undefined
    }, [extensionInfo])

    // Observe base and head viewerIds along with reference to extension host API
    const baseAndHeadViewerIds = useMemo(
        () =>
            extensionInfoChanges.pipe(
                filter(isDefined),
                filter(property('observeViewerId', isDefined)),
                switchMap(extensionInfo => {
                    const baseUri = extensionInfo.base.filePath
                        ? toURIWithPath({
                              repoName: extensionInfo.base.repoName,
                              commitID: extensionInfo.base.commitID,
                              filePath: extensionInfo.base.filePath,
                          })
                        : null
                    const baseViewerIds = baseUri ? extensionInfo.observeViewerId(baseUri) : of(null)

                    const headUri = extensionInfo.head.filePath
                        ? toURIWithPath({
                              repoName: extensionInfo.head.repoName,
                              commitID: extensionInfo.head.commitID,
                              filePath: extensionInfo.head.filePath,
                          })
                        : null
                    const headViewerIds = headUri ? extensionInfo.observeViewerId(headUri) : of(null)

                    return combineLatest([
                        baseViewerIds,
                        headViewerIds,
                        from(extensionInfo.extensionsController.extHostAPI),
                    ])
                }),
                map(([baseViewerId, headViewerId, extensionHostAPI]) => ({
                    baseViewerId,
                    headViewerId,
                    extensionHostAPI,
                }))
            ),
        [extensionInfoChanges]
    )

    const getHeadStatusBarItems = useCallback(
        () =>
            baseAndHeadViewerIds.pipe(
                distinctUntilKeyChanged('headViewerId'),
                switchMap(({ headViewerId, extensionHostAPI }) =>
                    headViewerId ? wrapRemoteObservable(extensionHostAPI.getStatusBarItems(headViewerId)) : of(null)
                ),
                map(statusBarItems => statusBarItems || [])
            ),
        [baseAndHeadViewerIds]
    )

    const getBaseStatusBarItems = useCallback(
        () =>
            baseAndHeadViewerIds.pipe(
                distinctUntilKeyChanged('baseViewerId'),
                switchMap(({ baseViewerId, extensionHostAPI }) =>
                    baseViewerId ? wrapRemoteObservable(extensionHostAPI.getStatusBarItems(baseViewerId)) : of(null)
                ),
                map(statusBarItems => statusBarItems || [])
            ),
        [baseAndHeadViewerIds]
    )

    // Listen for line decorations from extensions
    useObservable(
        useMemo(
            () =>
                baseAndHeadViewerIds.pipe(
                    switchMap(({ baseViewerId, headViewerId, extensionHostAPI }) =>
                        combineLatest([
                            baseViewerId
                                ? wrapRemoteObservable(extensionHostAPI.getTextDecorations(baseViewerId))
                                : of(null),
                            headViewerId
                                ? wrapRemoteObservable(extensionHostAPI.getTextDecorations(headViewerId))
                                : of(null),
                        ])
                    ),

                    tap(([baseDecorations, headDecorations]) => {
                        setDecorations({
                            base: groupDecorationsByLine(baseDecorations),
                            head: groupDecorationsByLine(headDecorations),
                        })
                    })
                ),

            [baseAndHeadViewerIds]
        )
    )

    // Hoverify
    useEffect(() => {
        const subscription = new Subscription()

        let hoverSubscription: Subscription
        subscription.add(
            extensionInfoChanges.pipe(filter(isDefined), first()).subscribe(extensionInfo => {
                hoverSubscription?.unsubscribe()

                hoverSubscription = extensionInfo.hoverifier.hoverify({
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
                        // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
                        return { repoName, filePath: filePath!, revision, commitID }
                    },
                })
                subscription.add(hoverSubscription)
            })
        )

        return () => subscription.unsubscribe()
    }, [codeElements, extensionInfoChanges])

    const isSplitMode = diffMode === 'split'

    return (
        <div className="file-diff-node__body">
            {extensionInfo && (
                <div className={classNames('w-100', isSplitMode && 'd-flex ')}>
                    {/* Always render base status bar even though it isn't displayed in unified mode
                    in order to prevent overloading the extension host with messages (`api.getStatusBarItems`) on
                    mode switch, which noticeably decreases status bar performance. */}
                    <StatusBar
                        getStatusBarItems={getBaseStatusBarItems}
                        className={classNames(
                            isSplitMode && 'flex-1 w-50',
                            'file-diff-node__status-bar border-bottom border-top-0'
                        )}
                        statusBarItemClassName="mx-0"
                        extensionsController={extensionInfo.extensionsController}
                        location={location}
                        badgeText="BASE"
                    />
                    <StatusBar
                        getStatusBarItems={getHeadStatusBarItems}
                        className={classNames(
                            isSplitMode && 'w-50',
                            'flex-1 file-diff-node__status-bar border-bottom border-top-0'
                        )}
                        statusBarItemClassName="mx-0"
                        extensionsController={extensionInfo.extensionsController}
                        location={location}
                        badgeText="HEAD"
                    />
                </div>
            )}
            <div className={`file-diff-hunks ${className}`} ref={nextBlobElement}>
                {hunks.length === 0 ? (
                    <div className="text-muted m-2">No changes</div>
                ) : (
                    <div className="file-diff-hunks__container" ref={nextCodeElement}>
                        <table
                            className={classNames('file-diff-hunks__table file-diff-hunks__table', {
                                'diff-hunk--split': isSplitMode,
                            })}
                        >
                            {lineNumbers && (
                                <colgroup>
                                    {isSplitMode ? (
                                        <>
                                            <col width="40" />
                                            <col />
                                            <col width="40" />
                                            <col />
                                        </>
                                    ) : (
                                        <>
                                            <col width="40" />
                                            <col width="40" />
                                            <col />
                                        </>
                                    )}
                                </colgroup>
                            )}
                            <tbody>
                                {hunks.map(hunk =>
                                    isSplitMode ? (
                                        <DiffSplitHunk
                                            fileDiffAnchor={fileDiffAnchor}
                                            isLightTheme={isLightTheme}
                                            lineNumbers={lineNumbers}
                                            persistLines={persistLines}
                                            key={hunk.oldRange.startLine}
                                            hunk={hunk}
                                            decorations={decorations}
                                        />
                                    ) : (
                                        <DiffHunk
                                            fileDiffAnchor={fileDiffAnchor}
                                            isLightTheme={isLightTheme}
                                            lineNumbers={lineNumbers}
                                            persistLines={persistLines}
                                            key={hunk.oldRange.startLine}
                                            hunk={hunk}
                                            decorations={decorations}
                                        />
                                    )
                                )}
                            </tbody>
                        </table>
                    </div>
                )}
            </div>
        </div>
    )
}
