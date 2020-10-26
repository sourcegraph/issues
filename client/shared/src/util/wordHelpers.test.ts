import { getWordAtText } from './wordHelpers'

describe('getWordAtText', () => {
    test('finds words', () => {
        expect(getWordAtText(0, '')).toBe(null)
        expect(getWordAtText(0, 'a')).toEqual({ word: 'a', startColumn: 0, endColumn: 1 })
        expect(getWordAtText(1, 'a')).toEqual({ word: 'a', startColumn: 0, endColumn: 1 })
        expect(getWordAtText(0, 'aa')).toEqual({ word: 'aa', startColumn: 0, endColumn: 2 })
        expect(getWordAtText(1, 'aa')).toEqual({ word: 'aa', startColumn: 0, endColumn: 2 })
        expect(getWordAtText(2, 'aa')).toEqual({ word: 'aa', startColumn: 0, endColumn: 2 })
        expect(getWordAtText(3, 'aa bb cc')).toEqual({ word: 'bb', startColumn: 3, endColumn: 5 })
        expect(getWordAtText(0, ' a')).toBe(null)
        expect(getWordAtText(3, 'a   b')).toBe(null)
    })
})
