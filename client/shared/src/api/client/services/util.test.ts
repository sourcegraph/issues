import { flattenAndCompact } from './util'

describe('flattenAndCompact', () => {
    test('flattens and compacts 1 level deep', () => {
        expect(flattenAndCompact([null, [1, 2], [3]])).toEqual([1, 2, 3])
    })

    test('passes through null', () => {
        expect(flattenAndCompact(null)).toEqual(null)
    })

    test('converts an empty result to null', () => {
        expect(flattenAndCompact([null, [], []])).toEqual(null)
    })
})
