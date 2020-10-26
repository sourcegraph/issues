export interface ErrorLike {
    message: string
    name?: string
}

export const isErrorLike = (value: unknown): value is ErrorLike =>
    typeof value === 'object' && value !== null && ('stack' in value || 'message' in value) && !('__typename' in value)

/**
 * Converts an ErrorLike to a proper Error if needed, copying all properties
 *
 * @param value An Error, object with ErrorLike properties, or other value.
 */
export const asError = (value: unknown): Error => {
    if (value instanceof Error) {
        return value
    }
    if (isErrorLike(value)) {
        return Object.assign(new Error(value.message), value)
    }
    return new Error(String(value))
}

const AGGREGATE_ERROR_NAME = 'AggregateError'

/**
 * An Error that aggregates multiple errors
 */
interface AggregateError extends Error {
    name: typeof AGGREGATE_ERROR_NAME
    errors: Error[]
}

/**
 * A type guard checking whether the given value is an {@link AggregateError}
 */
export const isAggregateError = (value: unknown): value is AggregateError =>
    isErrorLike(value) && value.name === AGGREGATE_ERROR_NAME

/**
 * DEPRECATED: use dataOrThrowErrors instead
 * Creates an aggregate error out of multiple provided error likes
 *
 * @param errors The errors or ErrorLikes to aggregate
 */
export const createAggregateError = (errors: ErrorLike[] = []): Error =>
    errors.length === 1
        ? asError(errors[0])
        : Object.assign(new Error(errors.map(error => error.message).join('\n')), {
              name: AGGREGATE_ERROR_NAME,
              errors: errors.map(asError),
          })

/**
 * Run the passed function and return `undefined` if it throws an error.
 */
export function tryCatch<T>(function_: () => T): T | undefined {
    try {
        return function_()
    } catch {
        return undefined
    }
}
