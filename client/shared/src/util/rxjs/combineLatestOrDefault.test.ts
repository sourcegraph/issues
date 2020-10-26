import { of } from 'rxjs'
import { TestScheduler } from 'rxjs/testing'
import { combineLatestOrDefault } from './combineLatestOrDefault'

const scheduler = (): TestScheduler => new TestScheduler((a, b) => expect(a).toEqual(b))

describe('combineLatestOrDefault', () => {
    describe('with 0 source observables', () => {
        test('emits an empty array and completes', () =>
            scheduler().run(({ expectObservable }) =>
                expectObservable(combineLatestOrDefault([], 'x')).toBe('(a|)', {
                    a: [],
                })
            ))
    })

    describe('with 1 source observable', () => {
        test('waits to emit/complete until the source observable emits/completes', () =>
            scheduler().run(({ cold, expectObservable }) =>
                expectObservable(combineLatestOrDefault([cold('-b-|', { b: 1 })])).toBe('-b-|', {
                    b: [1],
                })
            ))
    })

    describe('of()', () => {
        test('handles 1 of() source observable', () =>
            scheduler().run(({ expectObservable }) =>
                expectObservable(combineLatestOrDefault([of(1)])).toBe('(a|)', {
                    a: [1],
                })
            ))

        test('handles 2 of() source observables', () =>
            scheduler().run(({ expectObservable }) =>
                expectObservable(combineLatestOrDefault([of(1), of(2)])).toBe('(a|)', {
                    a: [1, 2],
                })
            ))
    })

    test('handles source observables with staggered emissions', () =>
        scheduler().run(({ cold, expectObservable }) =>
            expectObservable(combineLatestOrDefault([cold('-b-|', { b: 1 }), cold('--c|', { c: 2 })])).toBe('-bc|', {
                b: [1],
                c: [1, 2],
            })
        ))

    test('handles source observables with staggered completions', () =>
        scheduler().run(({ cold, expectObservable }) =>
            expectObservable(combineLatestOrDefault([cold('a|', { a: 1 }), cold('a-|', { a: 2 })])).toBe('a-|', {
                a: [1, 2],
            })
        ))

    test('handles source observables with concurrent emissions', () =>
        scheduler().run(({ cold, expectObservable }) =>
            expectObservable(combineLatestOrDefault([cold('-b-|', { b: 1 }), cold('-b-|', { b: 2 })])).toBe('-b-|', {
                b: [1, 2],
            })
        ))

    test('handles observables with staggered and concurrent emissions', () =>
        scheduler().run(({ cold, expectObservable }) =>
            expectObservable(
                combineLatestOrDefault([
                    cold('a-|', { a: 1 }),
                    cold('ab|', { a: 2, b: 3 }),
                    cold('-bc-|', { b: 4, c: 5 }),
                ])
            ).toBe('abc-|', {
                a: [1, 2],
                b: [1, 3, 4],
                c: [1, 3, 5],
            })
        ))

    test('fills in the default value if provided', () =>
        scheduler().run(({ cold, expectObservable }) =>
            expectObservable(
                combineLatestOrDefault([cold('a--|', { a: 1 }), cold('-b-|', { b: 2 }), cold('--c|', { c: 3 })], 42)
            ).toBe('abc|', {
                a: [1, 42, 42],
                b: [1, 2, 42],
                c: [1, 2, 3],
            })
        ))
})
