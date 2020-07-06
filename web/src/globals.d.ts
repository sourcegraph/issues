interface PageError {
    statusCode: number
    statusText: string
    error: string
    errorID: string
}

interface Window {
    pageError?: PageError
    context: SourcegraphContext
    MonacoEnvironment: {
        getWorkerUrl(moduleId: string, label: string): string
    }
}

/**
 * Represents user properties that are guaranteed to both (1) be set if the user is signed in,
 * and (2) not change over a user session
 */
interface ImmutableUser {
    readonly UID: number
}

type DeployType = 'kubernetes' | 'docker-container' | 'docker-compose' | 'pure-docker' | 'dev'

/**
 * Defined in cmd/frontend/internal/app/jscontext/jscontext.go JSContext struct
 */
interface SourcegraphContext
    extends Pick<Required<import('./schema/site.schema').SiteConfiguration>, 'experimentalFeatures'> {
    xhrHeaders: { [key: string]: string }
    csrfToken: string
    userAgentIsBot: boolean

    /**
     * Whether the user is authenticated. Use authenticatedUser in ./auth.ts to obtain information about the user.
     */
    readonly isAuthenticatedUser: boolean

    readonly sentryDSN: string | null

    /** Externally accessible URL for Sourcegraph (e.g., https://sourcegraph.com or http://localhost:3080). */
    externalURL: string

    /** URL path to image/font/etc. assets on server */
    assetsRoot: string

    version: string

    /**
     * Debug is whether debug mode is enabled.
     */
    debug: boolean

    sourcegraphDotComMode: boolean

    /**
     * siteID is the identifier of the Sourcegraph site.
     */
    siteID: string

    /** The GraphQL ID of the Sourcegraph site. */
    siteGQLID: GQL.ID

    /**
     * Whether the site needs to be initialized.
     */
    needsSiteInit: boolean

    /**
     * Emails support enabled
     */
    emailEnabled: boolean

    /**
     * A subset of the site configuration. Not all fields are set.
     */
    site: Pick<
        import('./schema/site.schema').SiteConfiguration,
        | 'auth.public'
        | 'update.channel'
        | 'campaigns.readAccess.enabled'
        | 'disableNonCriticalTelemetry'
        | 'permissions.backgroundSync'
    >

    /** Whether access tokens are enabled. */
    accessTokensAllow: 'all-users-create' | 'site-admin-create' | 'none'

    /** Whether the reset-password flow is enabled. */
    resetPasswordEnabled: boolean

    /**
     * Likely running within a Docker container under a Mac host OS.
     */
    likelyDockerOnMac: boolean

    /**
     * Whether or not the server needs to restart in order to apply a pending
     * configuration change.
     */
    needServerRestart: boolean

    /**
     * The kind of deployment.
     */
    deployType: DeployType

    /** Whether unauthenticated viewers can view the site. */
    readonly authPublic: boolean

    /** Whether signup is allowed on the site. */
    allowSignup: boolean

    /** Authentication provider instances in site config. */
    authProviders?: {
        displayName: string
        isBuiltin: boolean
        authenticationURL?: string
    }[]

    /** Custom branding for the homepage and search icon. */
    branding?: {
        /** The URL of the favicon to be used for your instance */
        favicon?: string

        /** Override style for light themes */
        light?: BrandAssets
        /** Override style for dark themes */
        dark?: BrandAssets

        /** Prevents the icon in the top-left corner of the screen from spinning. */
        disableSymbolSpin?: boolean

        brandName: string
    }

    /** The publishable key for the billing service (Stripe). */
    billingPublishableKey?: string
}

interface BrandAssets {
    /** The URL to the logo used on the homepage */
    logo?: string
    /** The URL to the symbol used as the search icon */
    symbol?: string
}

declare module '*.scss' {
    const cssModule: string
    export default cssModule
}
