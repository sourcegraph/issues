import { isMatch } from 'lodash'
import { BehaviorSubject, Subscribable } from 'rxjs'
import { PlatformContext } from '../../../platform/context'
import { Context } from './context'

/**
 * The context service owns the context data for the context, which consists of arbitrary key-value pairs that
 * describe application state.
 */
export interface ContextService {
    /**
     * The context data.
     */
    readonly data: Subscribable<Context> & { value: Context }

    /**
     * Sets the given context keys and values.
     * If a value is `null`, the context key is removed.
     *
     * @param update Object with context keys as values
     */
    updateContext(update: object): void
}

/** Create a {@link ContextService} instance. */
export function createContextService({
    clientApplication,
}: Pick<PlatformContext, 'clientApplication'>): ContextService {
    const data = new BehaviorSubject<Context>({
        'clientApplication.isSourcegraph': clientApplication === 'sourcegraph',

        // Arbitrary, undocumented versioning for extensions that need different behavior for different
        // Sourcegraph versions.
        //
        // TODO: Make this more advanced if many extensions need this (although we should try to avoid
        // extensions needing this).
        'clientApplication.extensionAPIVersion.major': 3,
    })
    return {
        data,
        updateContext(update: { [k: string]: unknown }): void {
            if (isMatch(this.data.value, update)) {
                return
            }
            const result: any = {}
            for (const [key, oldValue] of Object.entries(data.value)) {
                if (update[key] !== null) {
                    result[key] = oldValue
                }
            }
            for (const [key, value] of Object.entries(update)) {
                if (value !== null) {
                    result[key] = value
                }
            }
            data.next(result)
        },
    }
}
