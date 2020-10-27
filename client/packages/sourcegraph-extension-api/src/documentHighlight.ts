import { Range } from './location'

/**
 * A document highlight is a range inside a text document which deserves special attention.
 * Usually a document highlight is visualized by changing the background color of its range.
 */
export interface DocumentHighlight {
    /**
     * The range this highlight applies to.
     */
    range: Range

    /**
     * The highlight kind, default is text.
     */
    kind?: DocumentHighlightKind
}

export type DocumentHighlightKind = 'text' | 'read' | 'write'
