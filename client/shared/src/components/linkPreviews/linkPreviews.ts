import { LinkPreviewMerged, renderMarkupContents } from '../../api/client/services/linkPreview'

/** Options for {@link applyLinkPreview}. */
export interface ApplyLinkPreviewOptions {
    /**
     * CSS classes for the link preview content element to customize styling.
     */
    linkPreviewContentClass?: string

    /**
     * Sets or removes a plain-text tooltip on the HTML element using the native style.
     *
     * @param element The HTML element whose tooltip to set or remove.
     * @param tooltip The tooltip plain-text content (to add the tooltip) or `null` (to remove the
     * tooltip).
     */
    setElementTooltip?: (element: HTMLElement, tooltip: string | null) => void
}

/**
 * Updates the DOM surrounding {@link} to reflect the link preview for a single link. This operation
 * is idempotent.
 *
 * @param link The <a> element in the content view.
 * @param linkPreview The link preview to display.
 */
export function applyLinkPreview(
    { setElementTooltip, linkPreviewContentClass }: ApplyLinkPreviewOptions,
    link: HTMLAnchorElement,
    linkPreview: LinkPreviewMerged | null
): void {
    const LINK_PREVIEW_CONTENT_ELEMENT_CLASS_NAME = 'sg-link-preview-content'
    let afterElement: HTMLElement | undefined
    if (
        link.nextSibling instanceof HTMLElement &&
        link.nextSibling.classList.contains(LINK_PREVIEW_CONTENT_ELEMENT_CLASS_NAME)
    ) {
        afterElement = link.nextSibling
    }

    if (linkPreview?.content && linkPreview.content.length > 0) {
        if (afterElement) {
            afterElement.innerHTML = '' // clear for updated content
        } else {
            afterElement = document.createElement('span')
            afterElement.classList.add(LINK_PREVIEW_CONTENT_ELEMENT_CLASS_NAME)
            if (linkPreviewContentClass) {
                afterElement.classList.add(...linkPreviewContentClass.split(' '))
            }
            link.after(afterElement)
        }
        for (const content of renderMarkupContents(linkPreview.content)) {
            if (typeof content === 'string') {
                afterElement.append(content)
            } else {
                const span = document.createElement('span')
                span.innerHTML = content.html
                // Use while-loop instead of iterating over span.childNodes because the loop body
                // mutates span.childNodes, so nodes would be skipped.
                while (span.hasChildNodes()) {
                    afterElement.append(span.childNodes[0])
                }
            }
        }
    } else if (afterElement) {
        afterElement.remove()
        afterElement = undefined
    }

    if (setElementTooltip) {
        setElementTooltip(link, getHoverText(linkPreview))
        if (afterElement) {
            setElementTooltip(afterElement, getHoverText(linkPreview))
        }
    }
}

function getHoverText(linkPreview: LinkPreviewMerged | null): string | null {
    if (!linkPreview) {
        return null
    }
    const hoverValues = linkPreview.hover.map(({ value }) => value).filter(value => !!value)
    return hoverValues.length > 0 ? hoverValues.join(' ') : null
}
