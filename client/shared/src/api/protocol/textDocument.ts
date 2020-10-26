import { Position } from '@sourcegraph/extension-api-types'
import { DocumentSelector } from 'sourcegraph'
import { TextDocumentIdentifier } from '../client/types/textDocument'

/**
 * A parameter literal used in requests to pass a text document and a position inside that
 * document.
 */
export interface TextDocumentPositionParameters {
    /**
     * The text document.
     */
    textDocument: TextDocumentIdentifier

    /**
     * The position inside the text document.
     */
    position: Position
}

/**
 * General text document registration options.
 */
export interface TextDocumentRegistrationOptions {
    /**
     * A document selector to identify the scope of the registration.
     */
    documentSelector: DocumentSelector
}
