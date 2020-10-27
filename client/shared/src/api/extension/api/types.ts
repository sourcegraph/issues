import { Position, Range } from '@sourcegraph/extension-api-classes'
import * as clientType from '@sourcegraph/extension-api-types'
import * as sourcegraph from 'sourcegraph'

/**
 * Firefox does not like URLs being transmitted, so convert them to strings.
 *
 * https://github.com/sourcegraph/sourcegraph/issues/8928
 */
export function fromDocumentSelector(selector: sourcegraph.DocumentSelector): sourcegraph.DocumentSelector {
    return selector.map(filter =>
        typeof filter === 'string' ? filter : { ...filter, baseUri: filter.baseUri?.toString() }
    )
}

/**
 * Converts from a plain object {@link clientType.Position} to an instance of {@link Position}.
 *
 * @internal
 */
export function toPosition(position: clientType.Position): Position {
    return new Position(position.line, position.character)
}

export function fromDocumentHighlight(highlight: sourcegraph.DocumentHighlight): clientType.DocumentHighlight {
    return {
        range: highlight.range && fromRange(highlight.range),
        kind: highlight.kind,
    }
}

/**
 * Converts from an instance of a badged {@link Location} to the plain object {@link clientType.Location}.
 *
 * @internal
 */
export function fromLocation(
    location: sourcegraph.Badged<sourcegraph.Location>
): sourcegraph.Badged<clientType.Location> {
    return {
        uri: location.uri.href,
        range: location.range && fromRange(location.range),
        badge: location.badge,
    }
}

/**
 * Converts from an instance of a badged {@link Hover} to the plain object {@link clientType.Hover}.
 *
 * @internal
 */
export function fromHover(hover: sourcegraph.Badged<sourcegraph.Hover>): sourcegraph.Badged<clientType.Hover> {
    return {
        contents: hover.contents,
        range: hover.range && fromRange(hover.range),
        badge: hover.badge,
        alerts: hover.alerts,
    }
}

/**
 * Converts from an instance of {@link Range} to the plain object {@link clientType.Range}.
 *
 * @internal
 */
function fromRange(range: Range | sourcegraph.Range): clientType.Range {
    return range instanceof Range ? range.toJSON() : range
}
