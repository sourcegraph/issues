import { dedupeWhitespace } from './strings'

describe('utils/string', () => {
    describe(`${dedupeWhitespace.name}()`, () => {
        it('deduplicates whitespace', () => {
            expect(dedupeWhitespace('    a    b   c   d   ')).toBe(' a b c d ')
        })
    })
})
