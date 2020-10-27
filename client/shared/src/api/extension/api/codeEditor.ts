import { ProxyMarked, proxyMarker } from 'comlink'
import { Range, Selection } from '@sourcegraph/extension-api-classes'
import * as clientType from '@sourcegraph/extension-api-types'
import { BehaviorSubject, Observable } from 'rxjs'
import * as sourcegraph from 'sourcegraph'
import { createDecorationType } from './decorations'
import { ExtensionDocument } from './textDocument'
import { CodeEditorData, ViewerId } from '../../viewerTypes'

const DEFAULT_DECORATION_TYPE = createDecorationType()

/**
 * Returns true if all of the objects properties are empty null, undefined, empty strings or objects that are also empty.
 */
const isEmptyObjectDeep = (value: any): boolean =>
    Array.isArray(value)
        ? value.every(isEmptyObjectDeep)
        : typeof value === 'object' && value !== null
        ? Object.values(value).every(isEmptyObjectDeep)
        : !value

const isDecorationEmpty = ({ range, isWholeLine, ...contents }: clientType.TextDocumentDecoration): boolean =>
    isEmptyObjectDeep(contents)

/** @internal */
export class ExtensionCodeEditor implements sourcegraph.CodeEditor, ProxyMarked {
    public readonly [proxyMarker] = true

    /** The internal ID of this code editor. */
    public readonly viewerId: string

    /** The URI of this editor's document. */
    public readonly resource: string

    constructor(data: CodeEditorData & ViewerId, public document: ExtensionDocument) {
        this.resource = data.resource
        this.viewerId = data.viewerId
        this.update(data)
    }

    public readonly selectionsChanges = new BehaviorSubject<sourcegraph.Selection[]>([])

    public readonly type = 'CodeEditor'

    public get selection(): sourcegraph.Selection | null {
        return this.selectionsChanges.value.length > 0 ? this.selectionsChanges.value[0] : null
    }

    public get selections(): sourcegraph.Selection[] {
        return this.selectionsChanges.value
    }

    private _decorationsByType = new Map<sourcegraph.TextDocumentDecorationType, clientType.TextDocumentDecoration[]>()

    private _mergedDecorations = new BehaviorSubject<clientType.TextDocumentDecoration[]>([])
    public get mergedDecorations(): Observable<clientType.TextDocumentDecoration[]> {
        return this._mergedDecorations
    }

    public setDecorations(
        decorationType: sourcegraph.TextDocumentDecorationType | null,
        decorations: sourcegraph.TextDocumentDecoration[]
    ): void {
        // Backcompat: extensions developed against an older version of the API
        // may not supply a decorationType
        decorationType = decorationType || DEFAULT_DECORATION_TYPE
        // Replace previous decorations for this decorationType
        this._decorationsByType.set(decorationType, decorations.map(fromTextDocumentDecoration))
        this._mergedDecorations.next(
            [...this._decorationsByType.values()].flat().filter(decoration => !isDecorationEmpty(decoration))
        )
    }

    public update(data: Pick<CodeEditorData, 'selections'>): void {
        this.selectionsChanges.next(data.selections.map(selection => Selection.fromPlain(selection)))
    }

    public toJSON(): any {
        return { type: this.type, document: this.document }
    }
}

function fromTextDocumentDecoration(decoration: sourcegraph.TextDocumentDecoration): clientType.TextDocumentDecoration {
    return {
        ...decoration,
        range: (decoration.range as Range).toJSON(),
    }
}
