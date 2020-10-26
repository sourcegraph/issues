import { TestScheduler } from 'rxjs/testing'
import { from, defer } from 'rxjs'
import { repeatUntil } from './repeatUntil'

const scheduler = (): TestScheduler => new TestScheduler((a, b) => expect(a).toEqual(b))

describe('repeatUntil()', () => {
    it('completes if the emitted value matches select', () => {
        scheduler().run(({ cold, expectObservable }) => {
            expectObservable(
                from(
                    cold<number>('a|', { a: 5 }).pipe(repeatUntil(value => value === 5))
                )
            ).toBe('(a|)', { a: 5 })
        })
    })

    it('resubscribes until an emitted value matches select', () => {
        scheduler().run(({ cold, expectObservable }) => {
            let number = 0
            expectObservable(defer(() => cold('a|', { a: ++number })).pipe(repeatUntil(value => value === 3))).toBe(
                'ab(c|)',
                {
                    a: 1,
                    b: 2,
                    c: 3,
                }
            )
        })
    })

    it('delays resubscription if delay is provided', () => {
        scheduler().run(({ cold, expectObservable }) => {
            let number = 0
            expectObservable(
                defer(() => cold('a|', { a: ++number })).pipe(repeatUntil(value => value === 5, { delay: 5000 }))
            ).toBe('a 5s b 5s c 5s d 5s (e|)', {
                a: 1,
                b: 2,
                c: 3,
                d: 4,
                e: 5,
            })
        })
    })
})
