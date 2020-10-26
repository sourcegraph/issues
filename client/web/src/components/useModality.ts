import React, { useEffect, useRef } from 'react'
import { Key } from 'ts-key-enum'

function getFocusableElements(element: HTMLElement): HTMLElement[] {
    return [
        ...element.querySelectorAll('a, button, input, textarea, select, details, [tabindex]:not([tabindex="-1"])'),
    ].filter((element): element is HTMLElement => !element.hasAttribute('disabled') && element instanceof HTMLElement)
}

/**
 * Utility hook for modal-type components (e.g. ModalContainer, PopoverContainer).
 * `useModality` adds focus trapping and intuitive close logic to modal containers.
 */
export function useModality(
    onClose?: () => void,
    targetID?: string
): {
    modalContainerReference: React.MutableRefObject<HTMLDivElement | null>
    modalBodyReference: React.MutableRefObject<HTMLDivElement | null>
} {
    const modalContainerReference = useRef<HTMLDivElement>(null)
    const modalBodyReference = useRef<HTMLDivElement>(null)

    // On first render, close over the element that was focused to open it. on unmount, refocus that element.
    // Add keydown event listener for: 1) focus trapping, 2) `esc` to close
    useEffect(() => {
        const focusedElement = document.activeElement

        // TODO: use body ref instead?
        const containerElement = modalContainerReference.current
        containerElement?.focus()

        function onKeyDownContainer(event: KeyboardEvent): void {
            if (event.key === Key.Escape) {
                onClose?.()
            }

            // focus trapping
            if (containerElement) {
                const focusableElements = getFocusableElements(containerElement)

                const firstFocusable = focusableElements[0]
                const lastFocusable = focusableElements[focusableElements.length - 1]

                if (event.key === Key.Tab) {
                    if (event.shiftKey) {
                        if (document.activeElement === containerElement) {
                            // don't let the user escape (container is focused when modal is opened)
                            event.preventDefault()
                            return
                        }
                        if (document.activeElement === firstFocusable) {
                            // if this is the first focusable element, focus the last focusable element
                            event.preventDefault()
                            lastFocusable.focus()
                        }
                    } else if (document.activeElement === lastFocusable) {
                        // if this is the last focusable element, focus the first focusable element
                        event.preventDefault()
                        firstFocusable.focus()
                    }
                }
            }
        }

        containerElement?.addEventListener('keydown', onKeyDownContainer)

        return () => {
            if (focusedElement && focusedElement instanceof HTMLElement) {
                focusedElement.focus()
            }
            containerElement?.removeEventListener('keydown', onKeyDownContainer)
        }
    }, [modalBodyReference, modalContainerReference, onClose])

    // Close modal when user clicks outside of modal body, or clicks the target element again (for popovers)
    // (optional behavior: user opts in by using `bodyReference` as the ref attribute to the body element and/or id of target)
    useEffect(() => {
        function handleMouseDownOutside(event: MouseEvent): void {
            const modalBody = modalBodyReference.current
            const targetElement = targetID ? document.querySelector(`#${targetID}`) : null

            if (modalBody || targetElement) {
                const isNotModalBody =
                    !modalBody || (modalBody !== event.target && !modalBody.contains(event.target as Node))
                const isNotTargetElement =
                    !targetElement || (targetElement !== event.target && !targetElement.contains(event.target as Node))

                if (onClose && isNotModalBody && isNotTargetElement) {
                    document.addEventListener('mouseup', handleMouseUp)
                }
            }
        }

        // Only called when mousedown was outside of the modal body
        function handleMouseUp(event: MouseEvent): void {
            document.removeEventListener('mouseup', handleMouseUp)

            const modalBody = modalBodyReference.current
            const targetElement = targetID ? document.querySelector(`#${targetID}`) : null

            // if mouse is still outside of modal body, close the modal
            const isNotModalBody =
                !modalBody || (modalBody !== event.target && !modalBody.contains(event.target as Node))
            const isNotTargetElement =
                !targetElement || (targetElement !== event.target && !targetElement.contains(event.target as Node))

            if (onClose && isNotModalBody && isNotTargetElement) {
                onClose()
            }
        }

        document.addEventListener('mousedown', handleMouseDownOutside)

        return () => {
            document.removeEventListener('mousedown', handleMouseDownOutside)
            // just in case (e.g. modal could close from a timeout between mousedown and mouseup)
            document.removeEventListener('mouseup', handleMouseUp)
        }
    }, [modalBodyReference, onClose, targetID])

    return {
        modalBodyReference,
        modalContainerReference,
    }
}
