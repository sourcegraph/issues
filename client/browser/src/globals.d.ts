/** Set by the browser extension page and extension entry scripts. */
declare var SG_ENV: 'EXTENSION' | 'PAGE' | undefined

/** Set by the browser extension content, background and option page entry scripts. */
declare var EXTENSION_ENV: 'CONTENT' | 'BACKGROUND' | 'OPTIONS' | null | undefined

/** Set by native integrations. */
declare var SOURCEGRAPH_URL: string | undefined

/** Set by native integrations. */
declare var SOURCEGRAPH_INTEGRATION:
    | 'phabricator-integration'
    | 'bitbucket-integration'
    | 'gitlab-integration'
    | undefined

/**
 * Set by Gitlab native integration to load the assets from the Gitlab instance
 * instead of the Sourcegraph instance.
 */
declare var SOURCEGRAPH_ASSETS_URL: string | undefined

/** Global object with metadata available on Gitlab pages. */
declare var gon: {
    gitlab_url: string
}

/** Set from the Phabricator native integration. **/
declare var PHABRICATOR_CALLSIGN_MAPPINGS:
    | {
          callsign: string
          path: string
      }[]
    | undefined

/** Set from the Phabricator native integration. **/
declare var SOURCEGRAPH_PHABRICATOR_EXTENSION: boolean | undefined

/** Set from the Phabricator native integration. **/
declare var SOURCEGRAPH_BUNDLE_URL: string | undefined

/**
 * Set by shared/dev/jest-environment.js
 */
declare var jsdom: import('jsdom').JSDOM

declare module '*.scss' {
    const cssModule: string
    export default cssModule
}
