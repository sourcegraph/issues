import { getExtensionVersion, getPlatformName } from '../util/context'

/**
 * getHeaders emits the required headers for making requests to Sourcegraph server instances.
 * Requests can be blocked for various reasons and therefore the HTTP request MUST use the headers returned here.
 */
export function getHeaders(): { [name: string]: string } | undefined {
    // This is required for requests to be allowed by Sourcegraph's CORS rules.
    return {
        'X-Requested-With': `Sourcegraph - ${getPlatformName() as string} v${getExtensionVersion()}`,
    }
}
