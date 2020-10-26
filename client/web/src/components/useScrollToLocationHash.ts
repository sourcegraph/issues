import * as H from 'history'
import { useEffect, useState } from 'react'
import { tryCatch } from '../../../shared/src/util/errors'

/**
 * A React hook that scrolls the viewport to the element identified in the location hash (e.g., the
 * element with ID or name "foo" for the URL "https://example.com/a/b/#foo").
 *
 * This is needed because the browser's standard scroll-to-hash behavior doesn't work when using
 * react-router.
 *
 * If a React component needs the browser to scroll to elements that it renders asynchronously, the
 * React component must use this hook in such a way that it is invoked on each render. It is OK if
 * multiple components in a render tree use this hook.
 */
export const useScrollToLocationHash = (location: H.Location): void => {
    // Run on each render, because the element we need to scroll to might be derived from
    // asynchronously fetched data and not be present on the first render. But once we've found and
    // scrolled to an element for the location hash, don't keep trying to scroll to that element on
    // each render.
    const [scrolledTo, setScrolledTo] = useState<string>()
    // eslint-disable-next-line react-hooks/exhaustive-deps
    useEffect(() => {
        if (location.hash) {
            const idOrName = location.hash.slice(1)
            if (idOrName !== scrolledTo) {
                const element =
                    tryCatch(() => document.querySelector(`#${idOrName}`)) ||
                    document.getElementsByName(idOrName).item(0)
                if (element) {
                    element.scrollIntoView()
                    setScrolledTo(idOrName)
                } else {
                    setScrolledTo(undefined)
                }
            }
        }
    })
}
