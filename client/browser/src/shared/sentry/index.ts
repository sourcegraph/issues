/* eslint rxjs/no-ignored-subscription: warn */
import * as Sentry from '@sentry/browser'
import { once } from 'lodash'
import { observeStorageKey } from '../../browser-extension/web-extension-api/storage'
import { featureFlagDefaults } from '../../browser-extension/web-extension-api/types'
import { isInPage } from '../context'
import { DEFAULT_SOURCEGRAPH_URL, getExtensionVersion, observeSourcegraphURL } from '../util/context'

const IS_EXTENSION = true

const isExtensionStackTrace = (stacktrace: Sentry.Stacktrace, extensionID: string): boolean =>
    !!stacktrace.frames?.some(({ filename }) => !!filename?.includes(extensionID))

const callSentryInit = once((extensionID: string) => {
    Sentry.init({
        dsn: 'https://32613b2b6a5b4da2aa50660a60297d79@sentry.io/1334031',
        beforeSend: event => {
            // Filter out events if we can tell from the stack trace that
            // they didn't originate from extension code.
            let keep = true
            if (event.exception?.values) {
                keep = event.exception.values.some(
                    ({ stacktrace }) => !!(stacktrace && isExtensionStackTrace(stacktrace, extensionID))
                )
            } else if (event.stacktrace) {
                keep = isExtensionStackTrace(event.stacktrace, extensionID)
            }
            return keep ? event : null
        },
    })
})

/** Initialize Sentry for error reporting. */
export function initSentry(script: 'content' | 'options' | 'background', codeHost?: string): void {
    if (process.env.NODE_ENV !== 'production') {
        return
    }

    observeStorageKey('sync', 'featureFlags').subscribe((flags = featureFlagDefaults) => {
        const allowed = flags.allowErrorReporting

        // Don't initialize if user hasn't allowed us to report errors or in Phabricator.
        if (!allowed || isInPage) {
            const client = Sentry.getCurrentHub().getClient()
            if (client) {
                client.getOptions().enabled = false
            }
            return
        }

        callSentryInit(browser.runtime.id)

        Sentry.configureScope(scope => {
            scope.setTag('script', script)
            scope.setTag('extension_version', getExtensionVersion())
            if (codeHost) {
                scope.setTag('code_host', codeHost)
            }
        })
    })

    observeSourcegraphURL(IS_EXTENSION).subscribe(url => {
        Sentry.configureScope(scope => {
            scope.setTag('using_dot_com', url === DEFAULT_SOURCEGRAPH_URL ? 'true' : 'false')
        })
    })
}
