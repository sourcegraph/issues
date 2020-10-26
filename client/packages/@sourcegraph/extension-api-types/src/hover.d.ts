import * as sourcegraph from 'sourcegraph'
import { Range } from './location'

/**
 * A hover message.
 *
 * @see module:sourcegraph.Hover
 */
export interface Hover extends Pick<sourcegraph.Hover, 'contents' | 'alerts'> {
    /** The range that the hover applies to. */
    readonly range?: Range
}
