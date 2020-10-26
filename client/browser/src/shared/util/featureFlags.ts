import { storage } from '../../browser-extension/web-extension-api/storage'
import { featureFlagDefaults, FeatureFlags } from '../../browser-extension/web-extension-api/types'
import { isInPage } from '../context'

interface FeatureFlagsStorage {
    /**
     * Checks to see if the feature flag is set enabled.
     */
    isEnabled<K extends keyof FeatureFlags>(key: K): Promise<boolean>
    /**
     * Enable a feature flag.
     */
    enable<K extends keyof FeatureFlags>(key: K): Promise<void>
    /**
     * Disable a feature flag.
     */
    disable<K extends keyof FeatureFlags>(key: K): Promise<void>
    /**
     * Set a feature flag.
     */
    set<K extends keyof FeatureFlags>(key: K, enabled: boolean): Promise<void>
    /** Toggle a feature flag. */
    toggle<K extends keyof FeatureFlags>(key: K): Promise<boolean>
}

interface FeatureFlagUtilities {
    get(key: keyof FeatureFlags): Promise<boolean | undefined>
    set(key: keyof FeatureFlags, enabled: boolean): Promise<void>
}

const createFeatureFlagStorage = ({ get, set }: FeatureFlagUtilities): FeatureFlagsStorage => ({
    set,
    enable: key => set(key, true),
    disable: key => set(key, false),
    async isEnabled<K extends keyof FeatureFlags>(key: K): Promise<boolean> {
        const value = await get(key)
        return typeof value === 'boolean' ? value : featureFlagDefaults[key]
    },
    async toggle<K extends keyof FeatureFlags>(key: K): Promise<boolean> {
        const value = await get(key)
        await set(key, !value)
        return !value
    },
})

async function bextGet<K extends keyof FeatureFlags>(key: K): Promise<boolean | undefined> {
    const { featureFlags = {} } = await storage.sync.get()
    return featureFlags[key]
}

async function bextSet<K extends keyof FeatureFlags>(key: K, value: FeatureFlags[K]): Promise<void> {
    const { featureFlags } = await storage.sync.get('featureFlags')
    await storage.sync.set({ featureFlags: { ...featureFlags, [key]: value } })
}

const browserExtensionFeatureFlags = createFeatureFlagStorage({
    get: bextGet,
    set: bextSet,
})

const inPageFeatureFlags = createFeatureFlagStorage({
    // eslint-disable-next-line @typescript-eslint/require-await
    get: async key => {
        const value = localStorage.getItem(key)
        return value === null ? undefined : value === 'true'
    },
    // eslint-disable-next-line @typescript-eslint/require-await
    set: async (key, value) => {
        localStorage.setItem(key, String(value))
    },
})

export const featureFlags: FeatureFlagsStorage = isInPage ? inPageFeatureFlags : browserExtensionFeatureFlags
