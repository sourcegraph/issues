import { parse, ParseError, ParseErrorCode } from '@sqs/jsonc-parser/lib/main'
import { asError, createAggregateError, ErrorLike } from './errors'

/**
 * Parses the JSON input using an error-tolerant "JSONC" parser. If an error occurs, it is returned as a value
 * instead of being thrown. This is useful when input is parsed in the background (not in response to any specific
 * user action), because it makes it easier to save the error and only show it to the user when it matters (for
 * some interactive user action).
 */
export function parseJSONCOrError<T>(input: string): T | ErrorLike {
    try {
        return parseJSON(input) as T
    } catch (error) {
        return asError(error)
    }
}

/**
 * Parses the JSON input using an error-tolerant "JSONC" parser.
 */
function parseJSON(input: string): any {
    const errors: ParseError[] = []
    const parsed = parse(input, errors, { allowTrailingComma: true, disallowComments: false })
    if (errors.length > 0) {
        throw createAggregateError(
            errors.map(error => ({
                ...error,
                code: ParseErrorCode[error.error],
                message: `parse error (code: ${error.error}, offset: ${error.offset}, length: ${error.length})`,
            }))
        )
    }
    return parsed
}
