// causes false positive on act()
/* eslint-disable @typescript-eslint/no-floating-promises */

import { renderHook, act } from '@testing-library/react-hooks'
import { useObservable, useEventObservable } from './useObservable'
import { Observable, Subscriber } from 'rxjs'
import * as sinon from 'sinon'
import { map } from 'rxjs/operators'
import { useMemo, useCallback } from 'react'

describe('useObservable()', () => {
    it('should return the latest value of the given Observable', () => {
        const subscribe = sinon.spy((subscriber: Subscriber<number>) => {
            subscriber.next(1)
        })
        const { result } = renderHook(() => useObservable(useMemo(() => new Observable<number>(subscribe), [])))
        expect(result.current).toBe(1)
        sinon.assert.calledOnce(subscribe)
        const [subscriber] = subscribe.args[0]
        act(() => {
            subscriber.next(2)
        })
        expect(result.current).toBe(2)
        sinon.assert.calledOnce(subscribe)
    })

    it('should return undefined if the Observable did not emit anything yet', () => {
        const subscribe = sinon.spy((_subscriber: Subscriber<number>) => {
            /* noop */
        })
        const { result } = renderHook(() => useObservable(useMemo(() => new Observable<number>(subscribe), [])))
        expect(result.current).toBe(undefined)
        sinon.assert.calledOnce(subscribe)
        const [subscriber] = subscribe.args[0]
        act(() => {
            subscriber.next(1)
        })
        expect(result.current).toBe(1)
        sinon.assert.calledOnce(subscribe)
    })

    it('should throw if the Observable errored', () => {
        const error = new Error('fail')
        const subscribe = sinon.spy((subscriber: Subscriber<number>) => {
            subscriber.error(error)
        })
        const { result } = renderHook(() => useObservable(useMemo(() => new Observable<number>(subscribe), [])))
        expect(result.error).toBe(error)
        sinon.assert.calledOnce(subscribe)
    })

    it('should subscribe if component rerenders and observable changed', () => {
        const subscribe = sinon.spy((subscriber: Subscriber<number>) => {
            subscriber.next(1)
            return sinon.spy(() => {
                /* noop */
            })
        })
        let dep = 'dep'
        const { result, rerender } = renderHook(() =>
            // Intentional extra dependency
            // eslint-disable-next-line react-hooks/exhaustive-deps
            useObservable(useMemo(() => new Observable<number>(subscribe), [dep]))
        )
        expect(result.current).toBe(1)
        sinon.assert.calledOnce(subscribe)
        dep = 'changed'
        rerender()
        expect(result.current).toBe(1)
        sinon.assert.calledTwice(subscribe)
        const unsubscribe = subscribe.returnValues[0]
        sinon.assert.calledOnce(unsubscribe)
    })

    it('should not subscribe if component rerenders and observable did not change', () => {
        const subscribe = sinon.spy((subscriber: Subscriber<number>) => {
            subscriber.next(1)
            return sinon.spy(() => {
                /* noop */
            })
        })
        const { result, rerender } = renderHook(() =>
            useObservable(useMemo(() => new Observable<number>(subscribe), []))
        )
        expect(result.current).toBe(1)
        sinon.assert.calledOnce(subscribe)
        rerender()
        expect(result.current).toBe(1)
        sinon.assert.calledOnce(subscribe)
        const unsubscribe = subscribe.returnValues[0]
        sinon.assert.notCalled(unsubscribe)
    })

    it('should unsubscribe when the component unmounts', () => {
        const subscribe = sinon.spy((subscriber: Subscriber<number>) => {
            subscriber.next(1)
            return sinon.spy(() => {
                /* noop */
            })
        })
        const { result, unmount } = renderHook(() =>
            useObservable(useMemo(() => new Observable<number>(subscribe), []))
        )
        expect(result.current).toBe(1)
        sinon.assert.calledOnce(subscribe)
        unmount()
        sinon.assert.calledOnce(subscribe)
        const unsubscribe = subscribe.returnValues[0]
        sinon.assert.calledOnce(unsubscribe)
    })
})

describe('useEventObservable()', () => {
    it('should emit on the transform function whenever an event was triggered', () => {
        const spy = sinon.spy((number: number) => number + 10)
        const { result } = renderHook(() =>
            useEventObservable(useCallback((events: Observable<number>) => events.pipe(map(spy)), []))
        )
        let [onEvent, value] = result.current
        act(() => {
            onEvent(1)
        })
        ;[onEvent, value] = result.current
        expect(value).toBe(11)
        sinon.assert.calledOnce(spy)
        sinon.assert.calledWith(spy, 1)
        act(() => {
            onEvent(2)
        })
        ;[onEvent, value] = result.current
        expect(value).toBe(12)
        sinon.assert.calledTwice(spy)
        sinon.assert.calledWith(spy, 2)
    })
})
