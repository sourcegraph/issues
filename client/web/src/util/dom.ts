import { Observable } from 'rxjs'

/**
 * An Observable wrapper around ResizeObserver
 */
export const observeResize = (target: HTMLElement): Observable<void> =>
    new Observable(observer => {
        const resizeObserver = new ResizeObserver(() => {
            observer.next()
        })
        resizeObserver.observe(target)
        return () => resizeObserver.disconnect()
    })
