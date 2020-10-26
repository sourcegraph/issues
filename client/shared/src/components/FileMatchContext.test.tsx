import { mergeContext } from './FileMatchContext'

describe('components/FileMatchContext', () => {
    describe('mergeContext', () => {
        test('handles empty input', () => {
            expect(mergeContext(1, [])).toEqual([])
        })
        test('does not merge context when there is only one line', () => {
            expect(mergeContext(1, [{ line: 5 }])).toEqual([[{ line: 5 }]])
        })
        test('merges overlapping context', () => {
            expect(mergeContext(1, [{ line: 5 }, { line: 6 }])).toEqual([[{ line: 5 }, { line: 6 }]])
        })
        test('merges adjacent context', () => {
            expect(mergeContext(1, [{ line: 5 }, { line: 8 }])).toEqual([[{ line: 5 }, { line: 8 }]])
        })
        test('does not merge context when far enough apart', () => {
            expect(mergeContext(1, [{ line: 5 }, { line: 9 }])).toEqual([[{ line: 5 }], [{ line: 9 }]])
        })
    })
})
