import { useRef, useEffect, useMemo } from 'react'

export interface TimeoutManager {
    setTimeout: (callback: (...args: unknown[]) => void, timeout: number | undefined, ...args: unknown[]) => void
    cancelTimeout: () => void
}

/**
 * React hook that returns a TimeoutManager object that makes timeout management
 * sane in function components.
 *
 * It clears previously queued timeouts when called and on component unmount, and
 * provides `setTimeout` and `cancelTimeout` methods
 *
 * NOTE: Each instance of TimeoutSetter is meant to manage one timeout at a time.
 * Create a new TimeoutSetter instance when you want to set concurrent timeouts
 *
 * @returns A object with a `setTimeout` method with the same parameters
 * as `window.setTimeout`, and a `cancelTimeout` method
 */
export function useTimeoutManager(): TimeoutManager {
    const timeoutIDReference = useRef<{ timeoutID: number | undefined }>({ timeoutID: undefined })

    useEffect(() => {
        // https://reactjs.org/blog/2020/08/10/react-v17-rc.html#potential-issues
        const current = timeoutIDReference.current
        return () => clearTimeout(current.timeoutID)
    }, [])

    return useMemo(
        () => ({
            setTimeout(callback, timeout, ...args) {
                if (timeoutIDReference.current) {
                    this.cancelTimeout()
                }

                timeoutIDReference.current.timeoutID = window.setTimeout(callback, timeout, ...args)
            },
            cancelTimeout() {
                clearTimeout(timeoutIDReference.current.timeoutID)
                timeoutIDReference.current.timeoutID = undefined
            },
        }),
        []
    )
}
