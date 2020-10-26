import * as sourcegraph from 'sourcegraph'
import { Range } from './location'

/**
 * A decoration to apply to a text document.
 *
 * @see module:sourcgraph.TextDocumentDecoration
 */
export interface TextDocumentDecoration
    extends Pick<sourcegraph.TextDocumentDecoration, Exclude<keyof sourcegraph.TextDocumentDecoration, 'range'>> {
    /** The range that the decoration applies to. */
    range: Range
}
