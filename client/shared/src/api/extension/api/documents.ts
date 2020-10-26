import { ProxyMarked, proxyMarker } from 'comlink'
import { Subject } from 'rxjs'
import { TextDocument } from 'sourcegraph'
import { TextModelUpdate } from '../../client/services/modelService'
import { ExtensionDocument } from './textDocument'

/** @internal */
export interface ExtensionDocumentsAPI extends ProxyMarked {
    $acceptDocumentData(modelUpdates: readonly TextModelUpdate[]): void
}

const DOCUMENT_NOT_FOUND_ERROR_NAME = 'DocumentNotFoundError'
class DocumentNotFoundError extends Error {
    public readonly name = DOCUMENT_NOT_FOUND_ERROR_NAME
    constructor(resource: string) {
        super(`document not found: ${resource}`)
    }
}

/** @internal */
export class ExtensionDocuments implements ExtensionDocumentsAPI, ProxyMarked {
    public readonly [proxyMarker] = true

    private documents = new Map<string, ExtensionDocument>()

    constructor(private sync: () => Promise<void>) {}

    /**
     * Returns the known document with the given URI.
     *
     * @internal
     */
    public get(resource: string): ExtensionDocument {
        const textDocument = this.documents.get(resource)
        if (!textDocument) {
            throw new DocumentNotFoundError(resource)
        }
        return textDocument
    }

    /**
     * If needed, perform a sync with the client to ensure that its pending sends have been received before
     * retrieving this document.
     *
     * @todo This is necessary because hovers can be sent before the document is loaded, and it will cause a
     * "document not found" error.
     *
     * @deprecated `getSync()` makes no additional guarantees over `get()` anymore.
     */
    public async getSync(resource: string): Promise<ExtensionDocument> {
        const textDocument = this.documents.get(resource)
        if (textDocument) {
            return textDocument
        }
        await this.sync()
        return this.get(resource)
    }

    /**
     * Returns all known documents.
     *
     * @internal
     */
    public getAll(): ExtensionDocument[] {
        return [...this.documents.values()]
    }

    public openedTextDocuments = new Subject<TextDocument>()

    public $acceptDocumentData(modelUpdates: readonly TextModelUpdate[]): void {
        for (const update of modelUpdates) {
            switch (update.type) {
                case 'added': {
                    const { uri, languageId, text } = update
                    const textDocument = new ExtensionDocument({ uri, languageId, text })
                    this.documents.set(update.uri, textDocument)
                    this.openedTextDocuments.next(textDocument)
                    break
                }
                case 'updated': {
                    const textDocument = this.get(update.uri)
                    textDocument.update(update)
                    break
                }
                case 'deleted':
                    this.documents.delete(update.uri)
                    break
            }
        }
    }
}
