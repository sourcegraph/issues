import { uniq } from 'lodash'

/**
 * Creates a record object given an array of string literal union types.
 * Return initial value from provided function so that non-primitives are not shared.
 */
export function createRecord<K extends string, V>(
    array: K[] | Readonly<K[]>,
    initializeValue: (key: K) => V
): Record<K, V> {
    const record: Record<string, V> = {}
    for (const key of uniq(array)) {
        record[key] = initializeValue(key)
    }
    return record
}
