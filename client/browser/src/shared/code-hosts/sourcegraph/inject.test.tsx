import { checkIsSourcegraph } from './inject'

describe('checkIsSourcegraph()', () => {
    it('returns true when the location matches the provided sourcegraphServerURL', () => {
        expect(checkIsSourcegraph('https://sourcegraph.test:3443', new URL('https://sourcegraph.test:3443'))).toBe(true)
        expect(
            checkIsSourcegraph('https://sourcegraph.test:3443', new URL('https://sourcegraph.test:3443/search?q=test'))
        ).toBe(true)
    })
    it('returns true for sourcegraph.com', () => {
        expect(checkIsSourcegraph('https://sourcegraph.test:3443', new URL('https://sourcegraph.com'))).toBe(true)
    })
    it('returns false when the location attempts to impersonate sourcegraph.com', () => {
        expect(checkIsSourcegraph('https://sourcegraph.test:3443', new URL('https://wwwwsourcegraph.com'))).toBe(false)
    })
})
