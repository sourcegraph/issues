import { MarkupKind } from '@sourcegraph/extension-api-classes'
import { Hover as PlainHover, Range } from '@sourcegraph/extension-api-types'
import { Badged, Hover, MarkupContent, HoverAlert } from 'sourcegraph'

/** A hover that is merged from multiple Hover results and normalized. */
export interface HoverMerged {
    contents: Badged<MarkupContent>[]
    alerts?: Badged<HoverAlert>[]
    range?: Range
}

/** Create a merged hover from the given individual hovers. */
export function fromHoverMerged(
    values: readonly (Badged<Hover | PlainHover> | null | undefined)[]
): HoverMerged | null {
    const contents: HoverMerged['contents'] = []
    const alerts: HoverMerged['alerts'] = []
    let range: Range | undefined
    for (const result of values) {
        if (result) {
            if (result.contents?.value) {
                contents.push({
                    value: result.contents.value,
                    kind: result.contents.kind || MarkupKind.PlainText,
                    badge: result.badge,
                })
            }
            if (result.alerts) {
                alerts.push(...result.alerts)
            }
            if (result.range && !range) {
                range = result.range
            }
        }
    }

    if (contents.length === 0) {
        return null
    }
    return range ? { contents, alerts, range } : { contents, alerts }
}
