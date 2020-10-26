import { Observable } from 'rxjs'

/**
 * Returns an Observable for a WebExtension API event listener.
 * The handler will always return `void`.
 */
export const fromBrowserEvent = <F extends (...args: any[]) => void>(
    emitter: browser.CallbackEventEmitter<F>
): Observable<Parameters<F>> =>
    // Do not use fromEventPattern() because of https://github.com/ReactiveX/rxjs/issues/4736
    new Observable(subscriber => {
        const handler: any = (...args: any) => subscriber.next(args)
        try {
            emitter.addListener(handler)
        } catch (error) {
            subscriber.error(error)
            return undefined
        }
        return () => emitter.removeListener(handler)
    })
