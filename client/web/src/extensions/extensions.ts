import { RegistryExtensionFieldsForList } from '../graphql-operations'
import { ExtensionCategory, EXTENSION_CATEGORIES } from '../../../shared/src/schema/extensionSchema'
import {
    ConfiguredRegistryExtension,
    toConfiguredRegistryExtension,
    isExtensionEnabled,
} from '../../../shared/src/extensions/extension'
import { validCategories } from './extension/extension'
import { isErrorLike, ErrorLike } from '../../../shared/src/util/errors'
import { ConfiguredExtensionCache, ExtensionsEnablement } from './ExtensionRegistry'
import { Settings } from '../../../shared/src/settings/settings'
import { createRecord } from '../../../shared/src/util/createRecord'

export interface ConfiguredRegistryExtensions {
    [id: string]: ConfiguredRegistryExtension<RegistryExtensionFieldsForList>
}

export interface ConfiguredExtensionRegistry {
    /** Maps categories to ids of extensions  */
    extensionIDsByCategory: Record<
        ExtensionCategory,
        {
            /** IDs of all extensions for which this is the primary category */
            primaryExtensionIDs: string[]
            /** IDs of all extensions that fall into this category */
            allExtensionIDs: string[]
        }
    >

    /** All extensions returned by the query indexed by id */
    extensions: ConfiguredRegistryExtensions
}

const NO_VALID_CATEGORIES: 'Other'[] = ['Other']

/**
 * Configures extensions for the registry.
 *
 * `extensionIDsByCategory`: Object mapping category name to array of extension ids in that category
 * `extensions`: Object mapping extension id to the configured extension with that id
 *
 * `configureExtensionRegistry` is passed a cache of configured extensions to avoid
 * parsing manifests multiple times during the lifecycle of the extension registry.
 */
export function configureExtensionRegistry(
    nodes: RegistryExtensionFieldsForList[],
    configuredExtensionCache: ConfiguredExtensionCache
): ConfiguredExtensionRegistry {
    const extensions: ConfiguredRegistryExtensions = {}

    const extensionIDsByCategory: ConfiguredExtensionRegistry['extensionIDsByCategory'] = createRecord(
        EXTENSION_CATEGORIES,
        () => ({ primaryExtensionIDs: [], allExtensionIDs: [] })
    )

    for (const node of nodes) {
        // cache parsed extension manifests
        let configuredRegistryExtension = configuredExtensionCache.get(node.id)
        if (!configuredRegistryExtension) {
            configuredRegistryExtension = toConfiguredRegistryExtension(node)
            configuredExtensionCache.set(node.id, configuredRegistryExtension)
        }

        let categories: ExtensionCategory[]
        if (!isErrorLike(configuredRegistryExtension.manifest) && configuredRegistryExtension.manifest?.categories) {
            categories = validCategories(configuredRegistryExtension.manifest.categories) || NO_VALID_CATEGORIES
        } else {
            categories = NO_VALID_CATEGORIES
        }

        // TODO: Add `primaryCategory` to extension schema
        // Primary category is either specified or inferred by array position
        const primaryCategory = categories[0]

        extensionIDsByCategory[primaryCategory].primaryExtensionIDs.push(configuredRegistryExtension.id)
        for (const category of categories) {
            extensionIDsByCategory[category].allExtensionIDs.push(configuredRegistryExtension.id)
        }

        extensions[configuredRegistryExtension.id] = configuredRegistryExtension
    }

    return { extensions, extensionIDsByCategory }
}

/** Groups extensions by category */
export function applyCategoryFilter(
    extensionIDsByCategory: ConfiguredExtensionRegistry['extensionIDsByCategory'],
    categories: ExtensionCategory[],
    selectedCategories: ExtensionCategory[]
): Record<ExtensionCategory, string[]> {
    if (selectedCategories.length === 0) {
        // Primary categories
        return createRecord(categories, category => [...extensionIDsByCategory[category].primaryExtensionIDs])
    }

    // Categorize in toggle order, make sure the same extension doesn't appear twice.
    const filteredCategorizedExtensions = createRecord<ExtensionCategory, string[]>(selectedCategories, () => [])

    // To "blacklist" extension ID after it has been used
    const takenIDs = new Set<string>()

    for (const category of selectedCategories) {
        for (const extensionID of extensionIDsByCategory[category].allExtensionIDs) {
            if (!takenIDs.has(extensionID)) {
                filteredCategorizedExtensions[category].push(extensionID)

                takenIDs.add(extensionID)
            }
        }
    }

    return filteredCategorizedExtensions
}

/**
 * Filters categorized registry extensions by enablement (enabled | disabled | all)
 */
export function applyExtensionsEnablement(
    categorizedExtensions: Record<ExtensionCategory, string[]>,
    filteredCategoryIDs: ExtensionCategory[],
    enablement: ExtensionsEnablement,
    settings: Settings | ErrorLike | null
): Record<ExtensionCategory, string[]> {
    if (enablement === 'all') {
        return categorizedExtensions
    }

    return createRecord(filteredCategoryIDs, category =>
        categorizedExtensions[category].filter(
            extensionID => (enablement === 'enabled') === isExtensionEnabled(settings, extensionID)
        )
    )
}
