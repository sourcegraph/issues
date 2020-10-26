import { Subscribable, Subject, BehaviorSubject, Observable, throwError } from 'rxjs'
import { TextDocument } from 'sourcegraph'
import { ReferenceCounter } from '../../../util/ReferenceCounter'
import { filter, takeWhile, map, startWith } from 'rxjs/operators'

/**
 * A text model is a text document and associated metadata.
 *
 * How does this relate to editors (in {@link ViewerService}? A model is the file, an editor is the
 * window that the file is shown in. Things like the content and language are properties of the
 * model; things like decorations and the selection ranges are properties of the editor.
 */
export interface TextModel extends Pick<TextDocument, 'uri' | 'languageId' | 'text'> {}

/**
 * A partial {@link TextModel}, containing only the fields that are
 * guaranteed to never be updated.
 */
export interface PartialModel extends Pick<TextModel, 'languageId'> {}

export type TextModelUpdate =
    | ({ type: 'added' } & TextModel)
    | ({ type: 'updated'; text: string } & Pick<TextModel, 'uri'>)
    | ({ type: 'deleted' } & Pick<TextModel, 'uri'>)

/**
 * The model service manages document contents and metadata.
 *
 * See {@link Model} for an explanation of the difference between a model and an editor.
 */
export interface ModelService {
    /**
     * A map of all known models, indexed by URI.
     *
     * This is mostly used for testing, most consumers should use
     * {@link ModelService#modelUpdates}, {@link ModelService#activeLanguages} or {@link ModelService#observeModel} instead.
     */
    models: ReadonlyMap<string, TextModel>

    /**
     * An observable of all model updates.
     *
     * Emits when a model is added, updated or removed.
     */
    modelUpdates: Subscribable<readonly TextModelUpdate[]>

    /**
     * An observable of unique languageIds across all models.
     */
    activeLanguages: Subscribable<ReadonlySet<string>>

    /**
     * Observe a model for changes.
     *
     * @param uri The uri of the model to observe.
     * @returns An observable that emits when the model changes,
     * and completes when the model is removed.
     * If no such model exists, it emits an error.
     */
    observeModel(uri: string): Observable<TextModel>

    /**
     * Returns the {@link PartialModel} for the given uri.
     *
     */
    getPartialModel(uri: string): PartialModel

    /**
     * Adds a model.
     *
     * @param model The model to add.
     */
    addModel(model: TextModel): void

    /**
     * Updates a model's text content.
     *
     * @param uri The URI of the model whose content to update.
     * @param text The new text content (which will overwrite the model's previous content).
     * @throws if the model does not exist.
     */
    updateModel(uri: string, text: string): void

    /**
     * Reports whether a model with a given URI has already been added.
     *
     * @param uri The model URI to check.
     */
    hasModel(uri: string): boolean

    /**
     * Removes a model.
     *
     * @param uri The URI of the model to remove.
     */
    removeModel(uri: string): void
}

/**
 * Creates a new instance of {@link ModelService}.
 */
export function createModelService(): ModelService {
    /** A map of URIs to TextModels */
    const models = new Map<string, TextModel>()
    const modelUpdates = new Subject<TextModelUpdate[]>()
    const activeLanguages = new BehaviorSubject<ReadonlySet<string>>(new Set())
    const languageReferences = new ReferenceCounter<string>()
    const getModel = (uri: string): TextModel => {
        const model = models.get(uri)
        if (!model) {
            throw new Error(`model does not exist with URI ${uri}`)
        }
        return model
    }
    return {
        models,
        modelUpdates,
        activeLanguages,
        getPartialModel: uri => {
            const { languageId } = getModel(uri)
            return {
                languageId,
            }
        },
        observeModel: uri => {
            try {
                const model = getModel(uri)
                return modelUpdates.pipe(
                    filter(updates => updates.some(update => update.uri === uri)),
                    takeWhile(updates => updates.every(update => update.uri !== uri || update.type !== 'deleted')),
                    map(() => getModel(uri)),
                    startWith(model)
                )
            } catch (error) {
                return throwError(error)
            }
        },
        addModel: model => {
            if (models.has(model.uri)) {
                throw new Error(`model already exists with URI ${model.uri}`)
            }
            models.set(model.uri, model)
            modelUpdates.next([{ type: 'added', ...model }])
            // Update activeLanguages if no other existing model has the same language.
            if (languageReferences.increment(model.languageId)) {
                activeLanguages.next(new Set<string>(languageReferences.keys()))
            }
        },
        updateModel: (uri, text) => {
            const model = getModel(uri)
            models.set(uri, {
                ...model,
                text,
            })
            modelUpdates.next([{ type: 'updated', uri, text }])
        },
        hasModel: uri => models.has(uri),
        removeModel: uri => {
            const model = getModel(uri)
            models.delete(uri)
            modelUpdates.next([{ type: 'deleted', uri }])
            if (languageReferences.decrement(model.languageId)) {
                activeLanguages.next(new Set<string>(languageReferences.keys()))
            }
        },
    }
}
