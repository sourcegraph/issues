import { isErrorLike } from '../util/errors'
import { parseExtensionManifestOrError } from './extensionManifest'

describe('parseExtensionManifestOrError', () => {
    test('parses valid input', () =>
        expect(parseExtensionManifestOrError('{"url":"a","activationEvents":["*"]}')).toEqual({
            url: 'a',
            activationEvents: ['*'],
        }))
    test('returns an error value for invalid JSONC', () => {
        const value = parseExtensionManifestOrError('.')
        expect(isErrorLike(value)).toBeTruthy()
    })
    test('returns an error value for valid JSONC but invalid data', () => {
        const value = parseExtensionManifestOrError('{"url":"a"}')
        expect(isErrorLike(value)).toBeTruthy()
    })
    test('invalid categories', () => {
        const value = parseExtensionManifestOrError('{"categories":[1]}')
        expect(isErrorLike(value)).toBeTruthy()
    })
    test('invalid tags', () => {
        const value = parseExtensionManifestOrError('{"tags":[1]}')
        expect(isErrorLike(value)).toBeTruthy()
    })
})
