import { Position, Range } from '@sourcegraph/extension-api-classes'
import * as extensionApiTypes from '@sourcegraph/extension-api-types'
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
 * Converts from a plain object {@link extensionApiTypes.Position} to an instance of {@link Position}.
 *
 * @internal
 */
export function toPosition(position: extensionApiTypes.Position): Position {
    return new Position(position.line, position.character)
}

/**
 * Converts from an instance of a badged {@link Location} to the plain object {@link extensionApiTypes.Location}.
 *
 * @internal
 */
export function fromLocation(
    location: sourcegraph.Badged<sourcegraph.Location>
): sourcegraph.Badged<extensionApiTypes.Location> {
    return {
        uri: location.uri.href,
        range: fromRange(location.range),
        badge: location.badge,
    }
}

/**
 * Converts from an instance of a badged {@link Hover} to the plain object {@link extensionApiTypes.Hover}.
 *
 * @internal
 */
export function fromHover(hover: sourcegraph.Badged<sourcegraph.Hover>): sourcegraph.Badged<extensionApiTypes.Hover> {
    return {
        contents: hover.contents,
        range: fromRange(hover.range),
        badge: hover.badge,
        alerts: hover.alerts,
    }
}

/**
 * Converts from an instance of {@link Range} to the plain object {@link extensionApiTypes.Range}.
 *
 * @internal
 */
function fromRange(range: Range | sourcegraph.Range | undefined): extensionApiTypes.Range | undefined {
    if (!range) {
        return undefined
    }
    return range instanceof Range ? range.toJSON() : range
}
