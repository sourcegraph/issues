/**
 * A position in a document.
 *
 * @see module:sourcegraph.Position
 */
export interface Position {
    /** Zero-based line number. */
    readonly line: number

    /** Zero-based character on a line. */
    readonly character: number
}

/**
 * A range represents an ordered pair of two positions. The {@link Range#start} is always before the
 * {@link Range#end}.
 *
 * @see module:sourcegraph.Range
 */
export interface Range {
    /**
     * The start position. It is before or equal to [end](#Range.end).
     */
    readonly start: Position

    /**
     * The end position. It is after or equal to [start](#Range.start).
     */
    readonly end: Position
}

/**
 * A selection is a pair of two positions.
 *
 * @see module:sourcegraph.Selection
 */
export interface Selection extends Range {
    /**
     * The position at which the selection starts. This position might be before or after {@link Selection#active}.
     */
    readonly anchor: Position

    /**
     * The position of the cursor. This position might be before or after {@link Selection#anchor}.
     */
    readonly active: Position

    /**
     * Whether the selection is reversed. The selection is reversed if {@link Selection#active} is before
     * {@link Selection#anchor}.
     */
    readonly isReversed: boolean
}

/**
 * A location refers to a document (or a range within a document).
 *
 * @see module:sourcegraph.Location
 */
export interface Location {
    /** The URI of the document. */
    readonly uri: string

    /** An optional range within the document. */
    readonly range?: Range
}
