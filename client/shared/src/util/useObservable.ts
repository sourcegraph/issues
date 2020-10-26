import { useEffect, useState, useMemo } from 'react'
import { Observable, Observer, Subject } from 'rxjs'

/**
 * Returns a function that will trigger an error on the next render,
 * which can be caught by an ErrorBoundary higher up in the component tree.
 */
export function useError(): (error: any) => void {
    const [error, setError] = useState<any>()
    if (error) {
        throw error
    }
    return setError
}

/**
 * React hook to get the latest value of an Observable.
 * Will return `undefined` if the Observable didn't emit yet.
 * If the Observable errors, will throw an error that can be caught with `try`/`catch` or with a React error boundary.
 * The Observable is subscribed on the first render and unsubscribed on unmount or whenever it changes (wrap it in `useMemo()` to prevent this).
 *
 * @param observable The Observable to subscribe to. If this is the return value of a function, you should use `useMemo()` to make sure it is not resubscribed on every render.
 * @throws If the Observable pipeline errors.
 */
export function useObservable<T>(observable: Observable<T>): T | undefined {
    const [error, setError] = useState<any>()
    const [currentValue, setCurrentValue] = useState<T>()

    useEffect(() => {
        setCurrentValue(undefined)
        const subscription = observable.subscribe({ next: setCurrentValue, error: setError })
        return () => subscription.unsubscribe()
    }, [observable])

    if (error) {
        throw error
    }

    return currentValue
}

/**
 * A React hook to handle a React event with an RxJS pipeline.
 *
 * @template T The event type.
 * @template R The result type.
 * @param transform An RxJS operator pipeline that is passed an Observable of
 * the events triggered on the next function and can transform it to values of
 * which the latest one is returned. **You need to wrap this callback with `useCallback()`**.
 * @returns A next function to be used in JSX as an event handler, and the latest result of the Observable pipeline.
 * @throws If the Observable pipeline errors.
 */
export function useEventObservable<T, R>(
    transform: (events: Observable<T>) => Observable<R>
): [Observer<T>['next'], R | undefined]
export function useEventObservable<R>(
    transform: (events: Observable<void>) => Observable<R>
): [() => void, R | undefined]
export function useEventObservable<T, R>(
    transform: (events: Observable<T>) => Observable<R>
): [Observer<T>['next'], R | undefined] {
    const events = useMemo(() => new Subject<T>(), [])
    const observable = useMemo(() => events.pipe(transform), [events, transform])
    const nextEvent = useMemo(() => events.next.bind(events), [events])
    const value = useObservable(observable)
    return [nextEvent, value]
}
