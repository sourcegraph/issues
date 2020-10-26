import { TextDocumentPositionParameters } from './textDocument'

export interface ReferenceContext {
    /**
     * Include the declaration of the current symbol.
     */
    includeDeclaration: boolean
}

/**
 * Parameters for a [ReferencesRequest](#ReferencesRequest).
 */
export interface ReferenceParameters extends TextDocumentPositionParameters {
    context: ReferenceContext
}
