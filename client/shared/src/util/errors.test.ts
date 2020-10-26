import { asError, isErrorLike } from './errors'

describe('isErrorLike', () => {
    test('reports true for Error values', () => expect(isErrorLike(new Error('m'))).toBeTruthy())
    test('reports true for error-like values', () => expect(isErrorLike({ message: 'm' })).toBeTruthy())
    test('reports false for non-error-like values', () => expect(!isErrorLike('m')).toBeTruthy())
})

describe('asError', () => {
    test('preserves Error values', () => {
        const error = new Error('m')
        expect(asError(error)).toBe(error)
    })

    test('creates Error values from error-like values', () => {
        const error = asError({ message: 'm' })
        expect(isErrorLike(error)).toBeTruthy()
        expect(error.message).toBe('m')
    })

    test('creates Error values from strings', () => {
        const error = asError('m')
        expect(isErrorLike(error)).toBeTruthy()
        expect(error.message).toBe('m')
    })
})
