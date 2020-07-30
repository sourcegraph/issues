import { findPositionsFromEvents, Hoverifier } from '@sourcegraph/codeintellify'
import { TextDocumentDecoration } from '@sourcegraph/extension-api-types'
import * as H from 'history'
import { isEqual } from 'lodash'
import * as React from 'react'
import { combineLatest, NEVER, Observable, of, Subject, Subscription } from 'rxjs'
import { distinctUntilChanged, filter, map, switchMap } from 'rxjs/operators'
import { ActionItemAction } from '../../../../shared/src/actions/ActionItem'
import { DecorationMapByLine, groupDecorationsByLine } from '../../../../shared/src/api/client/services/decoration'
import { HoverMerged } from '../../../../shared/src/api/client/types/hover'
import { ExtensionsControllerProps } from '../../../../shared/src/extensions/controller'
import * as GQL from '../../../../shared/src/graphql/schema'
import { isDefined } from '../../../../shared/src/util/types'
import { FileSpec, RepoSpec, ResolvedRevisionSpec, RevisionSpec, toURIWithPath } from '../../../../shared/src/util/url'
import { ThemeProps } from '../../../../shared/src/theme'
import { DiffHunk } from './DiffHunk'
import { diffDomFunctions } from '../../repo/compare/dom-functions'
import { FileDiffHunkFields } from '../../graphql-operations'

interface PartFileInfo {
    repoName: string
    repoID: GQL.Scalars['ID']
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
    hunks: FileDiffHunkFields[]

    /** Whether to show line numbers. */
    lineNumbers: boolean

    className: string
    location: H.Location
    history: H.History
    /** Reflect selected line in url */
    persistLines?: boolean
}

interface FileDiffHunksState {
    /**
     * Decorations for the file at the two revisions of the diff
     */
    decorations: Record<'head' | 'base', DecorationMapByLine>
}

/** Displays hunks in a unified file diff. */
export class FileDiffHunks extends React.Component<FileHunksProps, FileDiffHunksState> {
    /** Emits whenever the ref callback for the code element is called */
    private codeElements = new Subject<HTMLElement | null>()
    private nextCodeElement = (element: HTMLElement | null): void => this.codeElements.next(element)

    /** Emits whenever the ref callback for the blob element is called */
    private blobElements = new Subject<HTMLElement | null>()
    private nextBlobElement = (element: HTMLElement | null): void => this.blobElements.next(element)

    /** Emits with the latest Props on every componentDidUpdate and on componentDidMount */
    private componentUpdates = new Subject<FileHunksProps>()

    /** Subscriptions to be disposed on unmout */
    private subscriptions = new Subscription()

    constructor(props: FileHunksProps) {
        super(props)
        this.state = {
            decorations: { head: new Map(), base: new Map() },
        }

        if (this.props.extensionInfo) {
            this.subscriptions.add(
                this.props.extensionInfo.hoverifier.hoverify({
                    dom: diffDomFunctions,
                    positionEvents: this.codeElements.pipe(
                        filter(isDefined),
                        findPositionsFromEvents({ domFunctions: diffDomFunctions })
                    ),
                    positionJumps: NEVER, // TODO support diff URLs
                    resolveContext: hoveredToken => {
                        // if part is undefined, it doesn't matter whether we chose head or base, the line stayed the same
                        const { repoName, revision, filePath, commitID } = this.props.extensionInfo![
                            hoveredToken.part || 'head'
                        ]
                        // If a hover or go-to-definition was invoked on this part, we know the file path must exist
                        return { repoName, filePath: filePath!, revision, commitID }
                    },
                })
            )
        }

        // Listen to decorations from extensions and group them by line
        this.subscriptions.add(
            this.componentUpdates
                .pipe(
                    map(({ extensionInfo }) => extensionInfo),
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
                    this.setState({
                        decorations: {
                            head: groupDecorationsByLine(headDecorations),
                            base: groupDecorationsByLine(baseDecorations),
                        },
                    })
                })
        )
    }

    public componentDidMount(): void {
        this.componentUpdates.next(this.props)
    }

    public componentDidUpdate(): void {
        this.componentUpdates.next(this.props)
    }

    public shouldComponentUpdate(
        nextProps: Readonly<FileHunksProps>,
        nextState: Readonly<FileDiffHunksState>
    ): boolean {
        return !isEqual(this.props, nextProps) || !isEqual(this.state, nextState)
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        return (
            <div className={`file-diff-hunks ${this.props.className}`} ref={this.nextBlobElement}>
                {this.props.hunks.length === 0 ? (
                    <div className="text-muted m-2">No changes</div>
                ) : (
                    <div className="file-diff-hunks__container" ref={this.nextCodeElement}>
                        <table className="file-diff-hunks__table">
                            {this.props.lineNumbers && (
                                <colgroup>
                                    <col width="40" />
                                    <col width="40" />
                                    <col />
                                </colgroup>
                            )}
                            <tbody>
                                {this.props.hunks.map((hunk, index) => (
                                    <DiffHunk
                                        {...this.props}
                                        key={index}
                                        hunk={hunk}
                                        decorations={this.state.decorations}
                                    />
                                ))}
                            </tbody>
                        </table>
                    </div>
                )}
            </div>
        )
    }
}
