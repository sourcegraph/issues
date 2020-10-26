import { PlatformName, getPlatformName } from '../../util/context'

export const EXTENSION_MARKER_ID = 'sourcegraph-app-background'

/**
 * A custom native integration <-> browser extension event used to free
 * browser extension subscriptions when the native integration gets activated
 * on the page, so as to avoid conflicts such as duplicate UI elements.
 */
export const NATIVE_INTEGRATION_ACTIVATED = 'sourcegraph:native-integration-activated'

/**
 * Injects a `#sourcegraph-app-background` hidden element.
 *
 * This element is checked for in the webapp to know if the browser extension
 * is installed, and in the browser extension to determine whether a native integration
 * is already running on the page.
 *
 * Not idempotent.
 */
export function injectExtensionMarker(): void {
    const extensionMarker = document.createElement('div')
    extensionMarker.id = EXTENSION_MARKER_ID
    extensionMarker.dataset.platform = getPlatformName()
    extensionMarker.style.display = 'none'
    document.body.append(extensionMarker)
}

/**
 * Dispatches a custom event to signal to Sourcegraph web app
 * that the browser extension is installed.
 */
export function signalBrowserExtensionInstalled(): void {
    if (document.readyState === 'complete' || document.readyState === 'interactive') {
        dispatchSourcegraphEvents()
    } else {
        window.addEventListener('load', dispatchSourcegraphEvents, { once: true })
    }
}

function dispatchSourcegraphEvents(): void {
    // Send custom webapp <-> extension registration event in case webapp listener is attached first.
    document.dispatchEvent(
        new CustomEvent<{ platform: PlatformName }>('sourcegraph:browser-extension-registration', {
            detail: { platform: getPlatformName() },
        })
    )
}

export const checkIsSourcegraph = (
    sourcegraphServerUrl: string,
    { origin, href }: Pick<Location, 'origin' | 'href'> = window.location
): boolean =>
    origin === sourcegraphServerUrl ||
    /^https?:\/\/(www\.)?sourcegraph\.com/.test(href) ||
    !!document.querySelector('#sourcegraph-chrome-webstore-item')
