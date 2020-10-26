import { Observable, Subject, Subscription } from 'rxjs'

/**
 * commitIDFromPermalink finds the permalink element on the page and extracts
 * the 40 character commit ID from it. This will throw if the link doesn't exist
 * or doesn't match the provided regex.
 */
export function commitIDFromPermalink({ selector, hrefRegex }: { selector: string; hrefRegex: RegExp }): string {
    const permalinkElement = document.querySelector<HTMLAnchorElement>(selector)
    if (!permalinkElement) {
        throw new Error(
            `Unable to determine the commit ID (40 character hash) you're on because the permalink shortcut element (query selector ${selector}) which contains the commit ID does not exist in the DOM.`
        )
    }
    const href = permalinkElement.getAttribute('href')
    if (!href) {
        throw new Error(
            `Unable to determine the commit ID (40 character hash) you're on because the permalink shortcut element (query selector ${selector}) which contains the commit ID does not have an href attribute.`
        )
    }
    const commitIDMatch = hrefRegex.exec(href)
    if (!commitIDMatch || !commitIDMatch[1]) {
        throw new Error(
            `Unable to determine the commit ID (40 character hash) you're on because the permalink shortcut element's (query selector ${selector}) href is ${href}, which doesn't match the regex /${hrefRegex.source}/.`
        )
    }
    return commitIDMatch[1]
}

/**
 * Compatible with MutationRecord, but synthesizable.
 */
export interface MutationRecordLike {
    addedNodes: ArrayLike<Node> & Iterable<Node>
    removedNodes: ArrayLike<Node> & Iterable<Node>
}

/**
 * An Observable wrapper around `MutationObserver`.
 *
 * Instructs the user agent to observe a given `target` (a node) and report any mutations based on
 * the criteria given by `options` (an object).
 *
 * @param target The node to observe.
 * @param options Allows for setting mutation observation options via object members.
 * @param paused Allows pausing (via {@link MutationObserver#disconnect}) and resuming (via
 * {@link MutationObserver#observe}) of the mutation observer. This is useful if the caller is
 * itself mutating the DOM and doesn't want to receive events for its own mutations.
 */
export const observeMutations = (
    target: Node,
    options?: MutationObserverInit,
    paused?: Subject<boolean>
): Observable<MutationRecord[]> =>
    new Observable<MutationRecord[]>(subscriber => {
        const subscriptions = new Subscription()
        const mutationObserver = new MutationObserver(mutations => subscriber.next(mutations))
        mutationObserver.observe(target, options)
        subscriptions.add(() => mutationObserver.disconnect())
        if (paused) {
            subscriptions.add(
                paused.subscribe(paused => {
                    if (paused) {
                        mutationObserver.disconnect()
                    } else {
                        mutationObserver.observe(target, options)
                    }
                })
            )
        }
        return () => subscriptions.unsubscribe()
    })

/**
 * Like `element.querySelectorAll()`, but will return (only) the element itself if it matches the selector.
 */
export function querySelectorAllOrSelf<K extends keyof HTMLElementTagNameMap>(
    element: Element,
    selectors: K
): ArrayLike<HTMLElementTagNameMap[K]> & Iterable<HTMLElementTagNameMap[K]>
export function querySelectorAllOrSelf<K extends keyof SVGElementTagNameMap>(
    element: Element,
    selectors: K
): ArrayLike<SVGElementTagNameMap[K]> & Iterable<SVGElementTagNameMap[K]>
export function querySelectorAllOrSelf<E extends Element = Element>(
    element: Element,
    selectors: string
): ArrayLike<E> & Iterable<E>
export function querySelectorAllOrSelf(element: Element, selectors: string): ArrayLike<Element> & Iterable<Element> {
    return element.matches(selectors) ? [element] : element.querySelectorAll(selectors)
}

/**
 * Like `element.querySelector()`, but will return the element itself if it matches the selector.
 */
export function querySelectorOrSelf<K extends keyof HTMLElementTagNameMap>(
    element: Element,
    selectors: K
): HTMLElementTagNameMap[K] | null
export function querySelectorOrSelf<K extends keyof SVGElementTagNameMap>(
    element: Element,
    selectors: K
): SVGElementTagNameMap[K] | null
export function querySelectorOrSelf<E extends Element = Element>(element: Element, selectors: string): E | null
export function querySelectorOrSelf(element: Element, selectors: string): Element | null {
    return element.matches(selectors) ? element : element.querySelector(selectors)
}
