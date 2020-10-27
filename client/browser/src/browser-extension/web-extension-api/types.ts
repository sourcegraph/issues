import { GraphQLResult } from '../../../../shared/src/graphql/graphql'
import { OptionFlagValues } from '../../shared/util/optionFlags'

export interface PhabricatorMapping {
    callsign: string
    path: string
}

/**
 * The feature flags available.
 */
export interface FeatureFlags {
    /**
     * Allow error reporting.
     *
     * @todo Since this is not really a feature flag, just unnest it into settings (and potentially get rid of the feature flags abstraction completely)
     */
    allowErrorReporting: boolean

    /**
     * Send telemetry
     */
    sendTelemetry: boolean

    /**
     * Support link previews from extensions in content views (such as GitHub issues).
     */
    experimentalLinkPreviews: boolean
}

export const featureFlagDefaults: FeatureFlags = {
    allowErrorReporting: false,
    sendTelemetry: true,
    experimentalLinkPreviews: false,
}

interface SourcegraphURL {
    sourcegraphURL: string
}

export interface SyncStorageItems extends SourcegraphURL {
    sourcegraphAnonymousUid: string
    /**
     * Temporarily disable the browser extension features.
     */
    disableExtension: boolean
    /**
     * Storage for feature flags.
     */
    featureFlags: Partial<OptionFlagValues>
    /**
     * Overrides settings from Sourcegraph.
     */
    clientSettings: string
    dismissedHoverAlerts: Record<string, boolean | undefined>
}

export interface LocalStorageItems {
    sideloadedExtensionURL: string | null
}

export interface ManagedStorageItems extends SourcegraphURL {
    phabricatorMappings: PhabricatorMapping[]
}

/**
 * Functions in the background page that can be invoked from content scripts.
 */
export interface BackgroundPageApi {
    openOptionsPage(): Promise<void>
    createBlobURL(bundleUrl: string): Promise<string>
    requestGraphQL<T, V = object>(options: {
        request: string
        variables: V
        sourcegraphURL?: string
    }): Promise<GraphQLResult<T>>
    notifyPrivateRepository(isPrivateRepository: boolean): Promise<void>
    checkPrivateRepository(tabId: number): Promise<boolean>
}

/**
 * Shape of the handler object in the background page.
 * The handlers get access to the sender tab of the message as a parameter.
 */
export type BackgroundPageApiHandlers = {
    [M in keyof BackgroundPageApi]: (
        payload: Parameters<BackgroundPageApi[M]>[0],
        sender: browser.runtime.MessageSender
    ) => ReturnType<BackgroundPageApi[M]>
}
