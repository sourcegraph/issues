import { isErrorLike } from './errors'
import { parseJSONCOrError } from './jsonc'

describe('parseJSONCOrError', () => {
    test('parses valid JSON', () => expect(parseJSONCOrError('{"a":1}')).toEqual({ a: 1 }))
    test('parses valid JSONC', () => expect(parseJSONCOrError('{/*x*/"a":1,}')).toEqual({ a: 1 }))
    test('returns an error value for invalid input', () => {
        const value = parseJSONCOrError('.')
        expect(isErrorLike(value)).toBeTruthy()
    })
})
