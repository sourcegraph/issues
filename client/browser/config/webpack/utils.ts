import extensionInfo from '../../src/browser-extension/manifest.spec.json'

/**
 * Generates a unique bundle ID that is used to prevent the Phabricator extension
 * from returning cached contents after upgrading.
 *
 * @returns The current extension version from extension.info.json.
 */
export function generateBundleUID(): string {
    if (!extensionInfo?.version) {
        throw new Error('Could not resolve extension version from manifest.')
    }
    return extensionInfo.version
}
