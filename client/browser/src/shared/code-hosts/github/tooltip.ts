/** GitHub element classes for showing a tooltip above an element (n=north). */
const TOOLTIP_CLASSES = ['tooltipped', 'tooltipped-n']

/** Sets the GitHub native tooltip on the given element. */
export function setElementTooltip(element: HTMLElement, tooltip: string | null): void {
    if (typeof tooltip === 'string') {
        element.classList.add(...TOOLTIP_CLASSES)
        element.setAttribute('aria-label', tooltip)
    } else {
        element.classList.remove(...TOOLTIP_CLASSES)
        element.removeAttribute('aria-label')
    }
}
