import { Subject } from 'rxjs'
import { isPromiseLike, isSubscribable, tryCatchPromise } from './util'

describe('tryCatchPromise', () => {
    test('returns a resolved promise with the synchronous result', async () =>
        expect(await tryCatchPromise(() => 1)).toBe(1))

    test('returns a resolved promise with the asynchronous result', async () =>
        expect(await tryCatchPromise(() => Promise.resolve(1))).toBe(1))

    const ERROR = new Error('x')
    test('returns a rejected promise with the synchronous exception', () => {
        const promise = tryCatchPromise(() => {
            throw ERROR
        })
        let rejected: any
        return promise
            .then(undefined, error => {
                rejected = error
            })
            .then(() => {
                expect(rejected).toBe(ERROR)
            })
    })

    test('returns a rejected promise with the asynchronous error', () => {
        const promise = tryCatchPromise(() => Promise.reject(ERROR))
        let rejected: any
        return promise
            .then(undefined, error => {
                rejected = error
            })
            .then(() => {
                expect(rejected).toBe(ERROR)
            })
    })
})

describe('isPromise', () => {
    test('returns true for promises', () =>
        expect(
            isPromiseLike(
                new Promise<any>(() => {
                    /* noop */
                })
            )
        ).toBe(true))
    test('returns true for non-promises', () => {
        expect(isPromiseLike(1)).toBe(false)
        expect(isPromiseLike({ then: 1 })).toBe(false)
        expect(isPromiseLike(new Subject<any>())).toBe(false)
    })
})

describe('isSubscribable', () => {
    test('returns true for subscribables', () => expect(isSubscribable(new Subject<any>())).toBe(true))
    test('returns true for non-subscribables', () => {
        expect(isSubscribable(1)).toBe(false)
        expect(isSubscribable({ subscribe: 1 })).toBe(false)
        expect(
            isSubscribable(
                new Promise<any>(() => {
                    /* noop */
                })
            )
        ).toBe(false)
    })
})
