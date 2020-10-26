import { asObservable } from './asObservable'
import assert from 'assert'
import { of } from 'rxjs'

describe('asObservable', () => {
    it('accepts an Observable', async () => {
        assert.equal(await asObservable(() => of(1)).toPromise(), 1)
    })
    it('accepts a sync value', async () => {
        assert.equal(await asObservable(() => 1).toPromise(), 1)
    })
    it('catches errors', async () => {
        await assert.rejects(
            () =>
                asObservable(() => {
                    throw new Error('test')
                }).toPromise(),
            /test/
        )
    })
})
