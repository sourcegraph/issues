import { extensionsQuery, splitExtensionID, urlToExtensionsQuery, validCategories } from './extension'

describe('splitExtensionID', () => {
    it('splits extensionID with host', () => {
        expect(splitExtensionID('sourcegraph.example.com/bob/myextension')).toStrictEqual({
            host: 'sourcegraph.example.com',
            publisher: 'bob',
            name: 'myextension',
        })
    })
    it('splits extensionID without host', () => {
        expect(splitExtensionID('alice/myextension')).toStrictEqual({
            publisher: 'alice',
            name: 'myextension',
            isSourcegraphExtension: false,
        })
    })
})

describe('validCategories', () => {
    test('selects only known categories, sorts, and dedupes', () =>
        expect(validCategories(['Programming languages', 'Other', 'x', 'Other'])).toEqual([
            'Other',
            'Programming languages',
        ]))

    test('returns undefined for undefined', () => expect(validCategories(undefined)).toEqual(undefined))

    test('returns undefined for empty', () => expect(validCategories([])).toEqual(undefined))

    test('returns undefined when no categories are known', () => expect(validCategories(['x'])).toEqual(undefined))
})

describe('extensionsQuery', () => {
    test('category (unquoted)', () => expect(extensionsQuery({ category: 'c' })).toBe('category:c'))
    test('category (quoted)', () => expect(extensionsQuery({ category: 'c c' })).toBe('category:"c c"'))
    test('tag (unquoted)', () => expect(extensionsQuery({ tag: 't' })).toBe('tag:t'))
    test('tag (quoted)', () => expect(extensionsQuery({ tag: 't t' })).toBe('tag:"t t"'))
    test('none', () => expect(extensionsQuery({})).toBe(''))
})

describe('urlToExtensionsQuery', () => {
    test('generates', () => expect(urlToExtensionsQuery('foo bar')).toBe('/extensions?query=foo+bar'))
})
