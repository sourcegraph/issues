import { getPathExtension } from './languages'

describe('util/index', () => {
    describe('getPathExtension', () => {
        test('returns extension if normal path', () => {
            expect(getPathExtension('/foo/baz/bar.go')).toBe('go')
        })

        test('returns empty string if no extension', () => {
            expect(getPathExtension('README')).toBe('')
        })

        test('returns empty string if hidden file with no extension', () => {
            expect(getPathExtension('.gitignore')).toBe('')
        })

        test('returns extension for path with multiple dot separators', () => {
            expect(getPathExtension('.baz.bar.go')).toBe('go')
        })
    })
})
