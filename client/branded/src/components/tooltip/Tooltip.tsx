import * as React from 'react'
import * as Popper from 'popper.js'
import { Tooltip as BootstrapTooltip } from 'reactstrap'

interface Props {}

interface State {
    subject?: HTMLElement
    subjectSeq: number
    lastEventTarget?: HTMLElement
    content?: string
    placement?: string
}

/**
 * A global tooltip displayed for elements containing a `data-tooltip` attribute.
 */
export class Tooltip extends React.PureComponent<Props, State> {
    private static SUBJECT_ATTRIBUTE = 'data-tooltip'

    /**
     * Singleton instance, so that other components can call Tooltip.forceUpdate().
     */
    private static INSTANCE: Tooltip | undefined

    public state: State = { subjectSeq: 0 }

    /**
     * Forces an update of the tooltip content. Other components must call this if they modify their tooltip
     * content while the tooltip is still visible.
     */
    public static forceUpdate(): void {
        const instance = Tooltip.INSTANCE
        if (instance) {
            instance.setState(previousState => {
                const subject = previousState.lastEventTarget && instance.getSubject(previousState.lastEventTarget)
                return {
                    subject,
                    content: subject ? instance.getContent(subject) : undefined,
                }
            })
        }
    }

    public componentDidMount(): void {
        Tooltip.INSTANCE = this

        document.addEventListener('focusin', this.handleEvent)
        document.addEventListener('mouseover', this.handleEvent)
        document.addEventListener('touchend', this.handleEvent)
        document.addEventListener('click', this.handleEvent)
    }

    public componentWillUnmount(): void {
        Tooltip.INSTANCE = undefined

        document.removeEventListener('focusin', this.handleEvent)
        document.removeEventListener('mouseover', this.handleEvent)
        document.removeEventListener('touchend', this.handleEvent)
        document.removeEventListener('click', this.handleEvent)
    }

    public render(): React.ReactFragment | null {
        return this.state.subject && this.state.content ? (
            <BootstrapTooltip
                // Set key prop to work around a bug where quickly mousing between 2 elements with tooltips
                // displays the 2nd element's tooltip as still pointing to the first.
                key={this.state.subjectSeq}
                isOpen={true}
                target={this.state.subject}
                placement={(this.state.placement ?? 'auto') as Popper.Placement}
                // This is a workaround to an issue with tooltips in reactstrap that causes the entire page to freeze.
                // Remove when https://github.com/reactstrap/reactstrap/issues/1482 is fixed.
                modifiers={{
                    flip: {
                        enabled: false,
                    },
                }}
            >
                {this.state.content}
            </BootstrapTooltip>
        ) : null
    }

    private handleEvent = (event: Event): void => {
        // As a special case, don't show the tooltip for click events on submit buttons that are probably triggered
        // by the user pressing the enter button. It is not desirable for the tooltip to be shown in that case.
        if (
            event.type === 'click' &&
            (event.target as HTMLElement).tagName === 'BUTTON' &&
            (event.target as HTMLButtonElement).type === 'submit' &&
            (event as MouseEvent).pageX === 0 &&
            (event as MouseEvent).pageY === 0
        ) {
            this.setState({ subject: undefined, content: undefined })
            return
        }

        const eventTarget = event.target as HTMLElement
        const subject = this.getSubject(eventTarget)
        this.setState(previousState => ({
            subject,
            subjectSeq: previousState.subject === subject ? previousState.subjectSeq : previousState.subjectSeq + 1,
            content: subject ? this.getContent(subject) : undefined,
            placement: subject ? this.getPlacement(subject) : undefined,
        }))
    }

    /**
     * Find the nearest ancestor element to e that contains a tooltip.
     */
    private getSubject = (element: HTMLElement | null): HTMLElement | undefined => {
        while (element) {
            if (element === document.body) {
                break
            }
            if (element.hasAttribute(Tooltip.SUBJECT_ATTRIBUTE)) {
                // If e is not actually attached to the DOM, then abort.
                if (!document.body.contains(element)) {
                    return undefined
                }
                return element
            }
            element = element.parentElement
        }
        return undefined
    }

    private getContent = (subject: HTMLElement): string | undefined => {
        if (!document.body.contains(subject)) {
            return undefined
        }
        return subject.getAttribute(Tooltip.SUBJECT_ATTRIBUTE) || undefined
    }

    private getPlacement = (subject: HTMLElement): string | undefined => {
        if (!document.body.contains(subject)) {
            return undefined
        }
        return subject.getAttribute('data-placement') || undefined
    }
}

/**
 * Sets or removes a plain-text tooltip on the HTML element using the native style for Sourcegraph
 * web app.
 *
 * @param element The HTML element whose tooltip to set or remove.
 * @param tooltip The tooltip plain-text content (to add the tooltip) or `null` (to remove the
 * tooltip).
 */
export function setElementTooltip(element: HTMLElement, tooltip: string | null): void {
    if (tooltip) {
        element.dataset.tooltip = tooltip
    } else {
        element.removeAttribute('data-tooltip')
    }
}
