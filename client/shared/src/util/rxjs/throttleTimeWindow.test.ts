import { of } from 'rxjs'
import { mergeMap } from 'rxjs/operators'
import { TestScheduler } from 'rxjs/testing'
import { throttleTimeWindow } from './throttleTimeWindow'

const scheduler = (): TestScheduler => new TestScheduler((a, b) => expect(a).toEqual(b))

describe('throttleTimeWindow', () => {
    test('emit the first value (immediately) and last value (at the end) in each time window', () => {
        scheduler().run(({ hot, expectObservable, expectSubscriptions }) => {
            const observable = hot('-a-x-y----b---x-cx---|')
            const subscriptions = '^--------------------!'
            const expected = '-a----y---b----xc----(x|)'

            const result = observable.pipe(throttleTimeWindow(5, 1))

            expectObservable(result).toBe(expected)
            expectSubscriptions(observable.subscriptions).toBe(subscriptions)
        })
    })

    test('emit the first 2 values (immediately) and last value (at the end) in each time window', () => {
        scheduler().run(({ hot, expectObservable, expectSubscriptions }) => {
            const observable = hot('-a-x-y----b---x-cx---|')
            const subscriptions = '^--------------------!'
            const expected = '-a-x--y---b---x-cx---|'

            const result = observable.pipe(throttleTimeWindow(5, 2))

            expectObservable(result).toBe(expected)
            expectSubscriptions(observable.subscriptions).toBe(subscriptions)
        })
    })

    test('emit the first 3 values (immediately) and last value (at the end) in each time window', () => {
        scheduler().run(({ hot, expectObservable, expectSubscriptions }) => {
            const observable = hot('-a-x-y----bxy-z-cx---|')
            const subscriptions = '^--------------------!'
            const expected = '-a-x-y----bxy--zcx---|'

            const result = observable.pipe(throttleTimeWindow(5, 3))

            expectObservable(result).toBe(expected)
            expectSubscriptions(observable.subscriptions).toBe(subscriptions)
        })
    })

    test('simply mirror the source if values are not emitted often enough', () => {
        scheduler().run(({ hot, expectObservable, expectSubscriptions }) => {
            const observable = hot('-a--------b-----c----|')
            const subscriptions = '^--------------------!'
            const expected = '-a--------b-----c----|'
            expectObservable(observable.pipe(throttleTimeWindow(5, 2))).toBe(expected)
            expectSubscriptions(observable.subscriptions).toBe(subscriptions)
        })
    })

    test('handle a busy producer emitting a regular repeating sequence', () => {
        scheduler().run(({ hot, expectObservable, expectSubscriptions }) => {
            const observable = hot('abcdefabcdefabcdefabcdefa|')
            const subscriptions = '^------------------------!'
            const expected = 'ab---fab---fab---fab---fa|'

            expectObservable(observable.pipe(throttleTimeWindow(5, 2))).toBe(expected)
            expectSubscriptions(observable.subscriptions).toBe(subscriptions)
        })
    })

    test('complete when source does not emit', () => {
        scheduler().run(({ hot, expectObservable, expectSubscriptions }) => {
            const observable = hot('-----|')
            const subscriptions = '^----!'
            const expected = '-----|'

            expectObservable(observable.pipe(throttleTimeWindow(5, 2))).toBe(expected)
            expectSubscriptions(observable.subscriptions).toBe(subscriptions)
        })
    })

    test('raise error when source does not emit and raises error', () => {
        scheduler().run(({ hot, expectObservable, expectSubscriptions }) => {
            const observable = hot('-----#')
            const subscriptions = '^----!'
            const expected = '-----#'

            expectObservable(observable.pipe(throttleTimeWindow(1, 2))).toBe(expected)
            expectSubscriptions(observable.subscriptions).toBe(subscriptions)
        })
    })

    test('handle an empty source', () => {
        scheduler().run(({ cold, expectObservable, expectSubscriptions }) => {
            const observable = cold('|')
            const subscriptions = '(^!)'
            const expected = '|'

            expectObservable(observable.pipe(throttleTimeWindow(3, 2))).toBe(expected)
            expectSubscriptions(observable.subscriptions).toBe(subscriptions)
        })
    })

    test('handle a never source', () => {
        scheduler().run(({ cold, expectObservable, expectSubscriptions }) => {
            const observable = cold('-')
            const subscriptions = '^'
            const expected = '-'

            expectObservable(observable.pipe(throttleTimeWindow(3, 2))).toBe(expected)
            expectSubscriptions(observable.subscriptions).toBe(subscriptions)
        })
    })

    test('handle a throw source', () => {
        scheduler().run(({ cold, expectObservable, expectSubscriptions }) => {
            const observable = cold('#')
            const subscriptions = '(^!)'
            const expected = '#'

            expectObservable(observable.pipe(throttleTimeWindow(3, 2))).toBe(expected)
            expectSubscriptions(observable.subscriptions).toBe(subscriptions)
        })
    })

    test('throttle and does not complete when source does not completes', () => {
        scheduler().run(({ hot, expectObservable, expectSubscriptions }) => {
            const observable = hot('-a--(bc)-------d----------------')
            const unsub = '-------------------------------!'
            const subscriptions = '^------------------------------!'
            const expected = '-a----c--------d----------------'

            expectObservable(observable.pipe(throttleTimeWindow(5)), unsub).toBe(expected)
            expectSubscriptions(observable.subscriptions).toBe(subscriptions)
        })
    })

    test('not break unsubscription chains when result is unsubscribed explicitly', () => {
        scheduler().run(({ hot, expectObservable, expectSubscriptions }) => {
            const observable = hot('-a--(bc)-------d----------------')
            const subscriptions = '^------------------------------!'
            const expected = '-a----c--------d----------------'
            const unsub = '-------------------------------!'

            const result = observable.pipe(
                mergeMap(emission => of(emission)),
                throttleTimeWindow(5),
                mergeMap(emission => of(emission))
            )

            expectObservable(result, unsub).toBe(expected)
            expectSubscriptions(observable.subscriptions).toBe(subscriptions)
        })
    })

    test('throttle values until source raises error', () => {
        scheduler().run(({ hot, expectObservable, expectSubscriptions }) => {
            const observable = hot('-a--(bc)-------d---------------#')
            const subscriptions = '^------------------------------!'
            const expected = '-a----c--------d---------------#'

            expectObservable(observable.pipe(throttleTimeWindow(5))).toBe(expected)
            expectSubscriptions(observable.subscriptions).toBe(subscriptions)
        })
    })
})
