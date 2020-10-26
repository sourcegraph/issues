import { Selection } from '@sourcegraph/extension-api-types'
import { BehaviorSubject, Subscribable, throwError, Observable, Subject } from 'rxjs'
import { map, filter, takeWhile, startWith, switchMap } from 'rxjs/operators'
import { TextDocumentPositionParameters } from '../../protocol'
import { ModelService, TextModel, PartialModel } from './modelService'
import { ReferenceCounter } from '../../../util/ReferenceCounter'

export type Viewer = CodeEditor | DirectoryViewer
export type ViewerData = CodeEditorData | DirectoryViewerData

/**
 * ViewerId exposes the unique ID of a viewer.
 */
export interface ViewerId {
    /** The unique ID of the viewer. */
    readonly viewerId: string
}

export interface BaseViewerData {
    readonly isActive: boolean
}

export interface DirectoryViewerData extends BaseViewerData {
    readonly type: 'DirectoryViewer'
    /** The URI of the directory that this viewer is displaying. */
    readonly resource: string
}

/**
 * Describes a code viewer to be created.
 */
export interface CodeEditorData extends BaseViewerData {
    readonly type: 'CodeEditor'

    /** The URI of the model that this viewer is displaying. */
    readonly resource: string

    readonly selections: Selection[]
}

/**
 * Describes a code editor that has been added to the {@link ViewerService}.
 */
export interface CodeEditor extends ViewerId, CodeEditorData {}

/**
 * Describes a directory viewer that has been added to the {@link ViewerService}.
 */
export interface DirectoryViewer extends ViewerId, DirectoryViewerData {}

export type ViewerWithPartialModel = CodeEditorWithPartialModel | DirectoryViewer // Directories don't have a model

/**
 * A code editor with a partial model.
 *
 * To get the editor's full model, look up the model in the {@link ModelService}.
 */
export interface CodeEditorWithPartialModel extends CodeEditor {
    model: PartialModel
}

/**
 * A code editor with its full model, including the model text.
 */
export interface CodeEditorWithModel extends CodeEditor {
    /** The code editor's model. */
    model: TextModel
}

export type ViewerUpdate =
    | ({ type: 'added'; viewerData: ViewerData } & ViewerId)
    | ({ type: 'updated'; viewerData: Pick<CodeEditorData, 'selections'> } & ViewerId)
    | ({ type: 'deleted' } & ViewerId)

/**
 * The viewer service manages viewers.
 */
export interface ViewerService {
    /**
     * A map of all known viewers, indexed by viewerId.
     *
     * This is mostly used for testing, most consumers should use
     * {@link ViewerService#viewerUpdates} or {@link ViewerService#activeViewerUpdates}
     */
    readonly viewers: ReadonlyMap<string, Viewer>

    /**
     * An observable of all viewer updates.
     *
     * Emits when a viewer is added, updated or removed.
     */
    readonly viewerUpdates: Subscribable<ViewerUpdate[]>

    /**
     * An observable of updates to the active viewer.
     *
     * Emits the active viewer if there is one, or `undefined` otherwise.
     */
    readonly activeViewerUpdates: Subscribable<Viewer | undefined>

    /**
     * Add a viewer.
     *
     * @param viewer The description of the viewer to add.
     * @returns The added code viewer (which must be passed as the first argument to other
     * {@link ViewerService} methods to operate on this viewer).
     */
    addViewer(viewer: ViewerData): ViewerId

    /**
     * Observe a viewer for changes.
     *
     * @param viewer The viewer to observe.
     * @returns An observable that emits when the viewer changes,
     * and completes when the viewer is removed.
     * If no such viewer exists, it emits an error.
     */
    observeViewer(viewer: ViewerId): Observable<ViewerData>

    /**
     * Sets the selections for a CodeEditor.
     *
     * @param codeEditor The editor for which to set the selections.
     * @param selections The new selections to apply.
     * @throws if no editor exists with the given editor ID.
     * @throws if the editor ID is not a CodeEditor.
     */
    setSelections(codeEditor: ViewerId, selections: Selection[]): void

    /**
     * Removes a viewer.
     * Also removes the corresponding model if no other viewer is referencing it.
     *
     * @param viewer The viewer to remove.
     */
    removeViewer(viewer: ViewerId): void

    /**
     * Remove all viewers.
     */
    removeAllViewers(): void
}

const VIEWER_NOT_FOUND_ERROR_NAME = 'ViewerNotFoundError'
class ViewerNotFoundError extends Error {
    public readonly name = VIEWER_NOT_FOUND_ERROR_NAME
    constructor(viewerId: string) {
        super(`Viewer not found: ${viewerId}`)
    }
}

/**
 * Creates a {@link ViewerService} instance.
 */
export function createViewerService(modelService: Pick<ModelService, 'removeModel'>): ViewerService {
    // Don't use lodash.uniqueId because that makes it harder to hard-code expected ID values in
    // test code (because the IDs change depending on test execution order).
    let id = 0
    const nextId = (): string => `viewer#${id++}`

    /** A map of viewer ids to code viewers. */
    const viewers = new Map<string, Viewer>()
    const viewerUpdates = new Subject<ViewerUpdate[]>()
    const activeViewerUpdates = new BehaviorSubject<Viewer | undefined>(undefined)
    /**
     * Returns the Viewer with the given viewerId.
     * Throws if no viewer exists with the given viewerId.
     */
    const getViewer = (viewerId: ViewerId['viewerId']): Viewer => {
        const viewer = viewers.get(viewerId)
        if (!viewer) {
            throw new ViewerNotFoundError(viewerId)
        }
        return viewer
    }

    const modelReferences = new ReferenceCounter()
    return {
        viewers,
        viewerUpdates,
        activeViewerUpdates,
        addViewer: viewerData => {
            const viewerId = nextId()
            if (viewerData.type === 'CodeEditor') {
                modelReferences.increment(viewerData.resource)
            }
            const viewer: Viewer = {
                ...viewerData,
                viewerId,
            }
            viewers.set(viewerId, viewer)
            viewerUpdates.next([{ type: 'added', viewerId, viewerData }])
            if (viewerData.isActive) {
                activeViewerUpdates.next(viewer)
            }
            return viewer
        },
        observeViewer: ({ viewerId }) => {
            try {
                const viewer = getViewer(viewerId)
                return viewerUpdates.pipe(
                    filter(updates => updates.some(update => update.viewerId === viewerId)),
                    takeWhile(updates =>
                        updates.every(update => update.viewerId !== viewerId || update.type !== 'deleted')
                    ),
                    map(() => getViewer(viewerId)),
                    startWith(viewer)
                )
            } catch (error) {
                return throwError(error)
            }
        },
        setSelections({ viewerId }: ViewerId, selections: Selection[]): void {
            const viewer = getViewer(viewerId)
            if (viewer.type !== 'CodeEditor') {
                throw new Error(`Editor ID ${viewerId} is type ${String(viewer.type)}, expected CodeEditor`)
            }
            viewers.set(viewerId, { ...viewer, selections })
            viewerUpdates.next([{ type: 'updated', viewerId, viewerData: { selections } }])
        },
        removeViewer({ viewerId }: ViewerId): void {
            const viewer = getViewer(viewerId)
            viewers.delete(viewerId)
            viewerUpdates.next([{ type: 'deleted', viewerId }])
            // Check if this was the active viewer
            if (activeViewerUpdates.value && activeViewerUpdates.value.viewerId === viewerId) {
                activeViewerUpdates.next(undefined)
            }
            if (viewer.type === 'CodeEditor' && modelReferences.decrement(viewer.resource)) {
                modelService.removeModel(viewer.resource)
            }
        },
        removeAllViewers(): void {
            const updates: ViewerUpdate[] = [...viewers.keys()].map(viewerId => ({ type: 'deleted', viewerId }))
            viewers.clear()
            viewerUpdates.next(updates)
        },
    }
}

/**
 * Helper function to get the active viewer's {@link TextDocumentPositionParams} from
 * {@link ViewerService#viewers}. If there is no active viewer or it has no position, it returns
 * null.
 */
export function getActiveCodeEditorPosition(activeViewer: Viewer | undefined): TextDocumentPositionParameters | null {
    if (!activeViewer || activeViewer.type !== 'CodeEditor') {
        return null
    }
    const sel = activeViewer.selections[0]
    if (!sel) {
        return null
    }
    // TODO(sqs): Return null for empty selections (but currently all selected tokens are treated as an empty
    // selection at the beginning of the token, so this would break a lot of things, so we only do this for empty
    // selections when the start character is -1). HACK(sqs): Character === -1 means that the whole line is
    // selected (this is a bug in the caller, but it is useful here).
    const isEmpty =
        sel.start.line === sel.end.line && sel.start.character === sel.end.character && sel.start.character === -1
    if (isEmpty) {
        return null
    }
    return {
        textDocument: { uri: activeViewer.resource },
        position: sel.start,
    }
}

/**
 * Observe a viewer and its model for changes.
 *
 * @param viewerId The ID of a **CodeEditor**.
 */
export function observeEditorAndModel(
    { viewerId }: ViewerId,
    { observeViewer }: Pick<ViewerService, 'observeViewer'>,
    { observeModel }: Pick<ModelService, 'observeModel'>
): Observable<CodeEditorWithModel> {
    return observeViewer({ viewerId }).pipe(
        map(viewer => {
            if (viewer.type !== 'CodeEditor') {
                throw new Error(`Editor ID ${viewerId} is type ${String(viewer.type)}, expected CodeEditor`)
            }
            return viewer
        }),
        switchMap(viewer => observeModel(viewer.resource).pipe(map(model => ({ viewerId, ...viewer, model }))))
    )
}
