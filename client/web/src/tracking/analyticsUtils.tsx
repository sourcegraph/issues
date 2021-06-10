import { fromEvent, concat, Observable, of } from 'rxjs'
import { fromFetch } from 'rxjs/fetch'
import { catchError, filter, map, mapTo, publishReplay, refCount, take, timeout } from 'rxjs/operators'

import { asError } from '@sourcegraph/shared/src/util/errors'

import { IS_CHROME } from '../marketing/util'

import { eventLogger } from './eventLogger'

interface EventQueryParameters {
    utm_campaign?: string
    utm_source?: string
    utm_medium?: string
}

const extensionMarker = document.querySelector<HTMLDivElement>('#sourcegraph-app-background')

/**
 * Indicates if the webapp ever receives a message from the user's Sourcegraph browser extension,
 * either in the form of a DOM marker element, or from a CustomEvent.
 *
 * You should likely use browserExtensionInstalled, rather than _browserExtensionMessageReceived,
 * which may never emit or complete.
 */
export const browserExtensionMessageReceived = (extensionMarker
    ? // If the marker exists, the extension is installed
      of({ platform: extensionMarker.dataset?.platform })
    : // If not, listen for a registration event
      fromEvent<CustomEvent>(document, 'sourcegraph:browser-extension-registration').pipe(
          take(1),
          map(({ detail }) => ({
              platform: detail?.platform,
          }))
      )
).pipe(
    // Replay the same latest value for every subscriber
    publishReplay(1),
    refCount()
)

const CHROME_EXTENSION_ID = 'dgjhfomjieaadpoljlnidmbgkdffpack'

/**
 * A better way to check if the Chrome extension is installed that doesn't depend on the extension having permissions to the page.
 * This is not possible on Firefox though.
 */
const checkChromeExtensionInstalled = (): Observable<boolean> => {
    if (!IS_CHROME) {
        return of(false)
    }
    return fromFetch(`chrome-extension://${CHROME_EXTENSION_ID}/img/icon-16.png`, {
        selector: response => of(response.ok),
    }).pipe(catchError(() => of(false)))
}

/**
 * Indicates if the current user has the browser extension installed. It waits 500ms for the browser
 * extension to fire a registration event, and if it doesn't, emits false
 */
export const browserExtensionInstalled: Observable<boolean> = concat(
    checkChromeExtensionInstalled().pipe(filter(isInstalled => isInstalled)),
    browserExtensionMessageReceived.pipe(
        mapTo(true),
        timeout(500),
        catchError(error => {
            if (asError(error).name === 'TimeoutError') {
                return [false]
            }
            throw error
        }),
        catchError(() => [false])
    )
).pipe(
    take(1),
    // Replay the same latest value for every subscriber
    publishReplay(1),
    refCount()
)

/**
 * Get pageview-specific event properties from URL query string parameters
 */
export function pageViewQueryParameters(url: string): EventQueryParameters {
    const parsedUrl = new URL(url)

    const utmSource = parsedUrl.searchParams.get('utm_source')
    if (utmSource === 'saved-search-email') {
        eventLogger.log('SavedSearchEmailClicked')
    } else if (utmSource === 'saved-search-slack') {
        eventLogger.log('SavedSearchSlackClicked')
    } else if (utmSource === 'code-monitoring-email') {
        eventLogger.log('CodeMonitorEmailLinkClicked')
    }

    return {
        utm_campaign: parsedUrl.searchParams.get('utm_campaign') || undefined,
        utm_source: parsedUrl.searchParams.get('utm_source') || undefined,
        utm_medium: parsedUrl.searchParams.get('utm_medium') || undefined,
    }
}
