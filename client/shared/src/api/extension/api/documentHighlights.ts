import type * as sourcegraph from 'sourcegraph'

/**
 * The type of a document highlight.
 * This is needed because if sourcegraph.DocumentHighlightKind enum values are referenced,
 * the `sourcegraph` module import at the top of the file is emitted in the generated code.
 */
export const DocumentHighlightKind: typeof sourcegraph.DocumentHighlightKind = {
    Text: 'text' as sourcegraph.DocumentHighlightKind.Text,
    Read: 'read' as sourcegraph.DocumentHighlightKind.Read,
    Write: 'write' as sourcegraph.DocumentHighlightKind.Write,
}
