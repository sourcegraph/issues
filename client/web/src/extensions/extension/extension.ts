import { uniq } from 'lodash'
import * as GQL from '../../../../shared/src/graphql/schema'
import {
    EXTENSION_CATEGORIES,
    ExtensionCategory,
    ExtensionManifest,
} from '../../../../shared/src/schema/extensionSchema'
import { Settings } from '../../../../shared/src/settings/settings'
import { ErrorLike, isErrorLike } from '../../../../shared/src/util/errors'
import { quoteIfNeeded } from '../../search'

/** Pattern for valid extension names. */
export const EXTENSION_NAME_VALID_PATTERN = '^[a-zA-Z0-9](?:[a-zA-Z0-9]|[_.-](?=[a-zA-Z0-9]))*$'

/** Maximum allowed length for an extension name. */
export const EXTENSION_NAME_MAX_LENGTH = 128

/** A useful minimal type for a registry extension's publisher. */
export type RegistryPublisher = (
    | Pick<GQL.IUser, '__typename' | 'id' | 'username'>
    | Pick<GQL.IOrg, '__typename' | 'id' | 'name'>
) & {
    /** The prefix for extension IDs published by this publisher (with the registry's host), if any. */
    extensionIDPrefix?: string
}

/** Returns the extension ID prefix (excluding the trailing "/") for a registry extension's publisher. */
export function extensionIDPrefix(publisher: RegistryPublisher): string {
    return `${publisher.extensionIDPrefix ? `${publisher.extensionIDPrefix}/` : ''}${publisherName(publisher)}`
}

export function publisherName(publisher: RegistryPublisher): string {
    switch (publisher.__typename) {
        case 'User':
            return publisher.username
        case 'Org':
            return publisher.name
    }
}

/** Returns the extension ID (in "publisher/name" format). */
export function toExtensionID(publisher: string | RegistryPublisher, name: string): string {
    return `${typeof publisher === 'string' ? publisher : extensionIDPrefix(publisher)}/${name}`
}

/**
 * Mirrors `registry.SplitExtensionID` from `frontend`:
 *
 * `splitExtensionID` splits an extension ID of the form [host/]publisher/name (where [host/] is the
 * optional registry prefix), such as "alice/myextension" or
 * "sourcegraph.example.com/bob/myextension". It returns the components in an object.
 *
 * @param extensionID The extension ID (string)
 */
export function splitExtensionID(
    extensionID: string
): { publisher: string; name: string; host?: string; isSourcegraphExtension?: boolean } {
    const parts = extensionID.split('/')
    if (parts.length === 3) {
        return {
            host: parts[0],
            publisher: parts[1],
            name: parts[2],
        }
    }

    return {
        publisher: parts[0] ?? '',
        name: parts[1] ?? '',
        isSourcegraphExtension: parts[0] === 'sourcegraph',
    }
}

/** Reports whether the given extension is mentioned (enabled or disabled) in the settings. */
export function isExtensionAdded(settings: Settings | ErrorLike | null, extensionID: string): boolean {
    return !!settings && !isErrorLike(settings) && !!settings.extensions && extensionID in settings.extensions
}

/**
 * @param categories The (unvalidated) categories defined on the extension.
 * @returns An array that is the subset of {@link categories} consisting only of known categories, sorted and with
 * duplicates removed.
 */
export function validCategories(categories: ExtensionManifest['categories']): ExtensionCategory[] | undefined {
    if (!categories || categories.length === 0) {
        return undefined
    }
    const validCategories = uniq(
        categories
            .filter((category): category is ExtensionCategory =>
                EXTENSION_CATEGORIES.includes(category as ExtensionCategory)
            )
            .sort()
    )
    return validCategories.length === 0 ? undefined : validCategories
}

/**
 * Constructs the extensions query given the options (for use in the query input field in the extension registry
 * list page).
 */
export function extensionsQuery({
    category,
    tag,
    installed,
    enabled,
    disabled,
}: {
    category?: string
    tag?: string
    installed?: boolean
    enabled?: boolean
    disabled?: boolean
}): string {
    const parts: string[] = []
    if (category) {
        parts.push(`category:${quoteIfNeeded(category)}`)
    }
    if (tag) {
        parts.push(`tag:${quoteIfNeeded(tag)}`)
    }
    if (installed) {
        parts.push('#installed')
    }
    if (enabled) {
        parts.push('#enabled')
    }
    if (disabled) {
        parts.push('#disabled')
    }
    return parts.join(' ')
}

/**
 * Constructs the URL to the extensions list with the given query string (which can be constructed using
 * {@link extensionsQuery}).
 */
export function urlToExtensionsQuery(query: string): string {
    const parameters = new URLSearchParams()
    parameters.set('query', query)
    return `/extensions?${parameters.toString()}`
}
