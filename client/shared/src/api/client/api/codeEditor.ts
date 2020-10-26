import { ProxyMarked, proxyMarker } from 'comlink'
import { TextDocumentDecoration } from '@sourcegraph/extension-api-types'
import { flatten, values } from 'lodash'
import { BehaviorSubject, Observable, Subscription } from 'rxjs'
import { ProvideTextDocumentDecorationSignature } from '../services/decoration'
import { FeatureProviderRegistry } from '../services/registry'
import { TextDocumentIdentifier } from '../types/textDocument'

/** @internal */
export interface ClientCodeEditorAPI extends ProxyMarked {
    $setDecorations(resource: string, decorationType: string, decorations: TextDocumentDecoration[]): void
}

interface PreviousDecorations {
    [resource: string]: {
        [decorationType: string]: TextDocumentDecoration[]
    }
}

/** @internal */
export class ClientCodeEditor implements ClientCodeEditorAPI {
    public readonly [proxyMarker] = true

    private subscriptions = new Subscription()

    /** Map of document URI to its decorations (last published by the server). */
    private decorations = new Map<string, BehaviorSubject<TextDocumentDecoration[]>>()

    private previousDecorations: PreviousDecorations = {}

    constructor(private registry: FeatureProviderRegistry<undefined, ProvideTextDocumentDecorationSignature>) {
        this.subscriptions.add(
            this.registry.registerProvider(
                undefined,
                (textDocument: TextDocumentIdentifier): Observable<TextDocumentDecoration[]> =>
                    this.getDecorationsSubject(textDocument.uri)
            )
        )
    }

    public $setDecorations(resource: string, decorationType: string, decorations: TextDocumentDecoration[]): void {
        // eslint-disable-next-line rxjs/no-ignored-observable
        this.getDecorationsSubject(resource, decorationType, decorations)
    }

    private getDecorationsSubject(
        resource: string,
        decorationType?: string,
        decorations?: TextDocumentDecoration[]
    ): BehaviorSubject<TextDocumentDecoration[]> {
        let subject = this.decorations.get(resource)
        if (!subject) {
            subject = new BehaviorSubject<TextDocumentDecoration[]>(decorations || [])
            this.decorations.set(resource, subject)
            this.previousDecorations[resource] = {}
        }
        if (decorations !== undefined) {
            // Replace previous decorations for this resource + decorationType
            this.previousDecorations[resource][decorationType!] = decorations

            // Merge decorations for all types for this resource, and emit them
            const nextDecorations = flatten(values(this.previousDecorations[resource]))
            subject.next(nextDecorations)
        }
        return subject
    }

    public unsubscribe(): void {
        // Clear decorations.
        for (const subject of this.decorations.values()) {
            subject.next([])
        }

        this.subscriptions.unsubscribe()
    }
}
