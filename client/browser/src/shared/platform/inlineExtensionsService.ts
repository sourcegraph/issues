import { isExtension } from '../context'
import { isFirefox } from '../util/context'
import { IExtensionsService, ExecutableExtension } from '../../../../shared/src/api/client/services/extensionsService'
import { Subscribable, from } from 'rxjs'
import { checkOk } from '../../../../shared/src/backend/fetch'
import { ExtensionManifest } from '../../../../shared/src/extensions/extensionManifest'

/**
 * Determine if inline extensions should be loaded.
 *
 * This requires the browser extension to be built with inline extensions enabled.
 * At build time this is determined by `shouldBuildWithInlineExtensions`.
 */
export const shouldUseInlineExtensions = (): boolean => isExtension && isFirefox()

/**
 * Get the manifest URL and script URL for a Sourcegraph extension which is inline (bundled with the browser add-on).
 */
function getURLsForInlineExtension(extensionName: string): { manifestURL: string; scriptURL: string } {
    return {
        manifestURL: browser.extension.getURL(`extensions/${extensionName}/package.json`),
        scriptURL: browser.extension.getURL(`extensions/${extensionName}/extension.js`),
    }
}

export function getInlineExtensions(): Subscribable<ExecutableExtension[]> {
    const extensionName = 'template'
    const { manifestURL, scriptURL } = getURLsForInlineExtension('template')
    const requestPromise = fetch(manifestURL)
        .then(response => checkOk(response).json())
        .then(
            (manifest: ExtensionManifest) =>
                [
                    {
                        id: `sourcegraph/${extensionName}`,
                        manifest,
                        scriptURL,
                    },
                ] as ExecutableExtension[]
        )

    return from(requestPromise)
}

/**
 * Manages the set of inline (bundled) extensions that are loaded. Only applicable to the Firefox browser add-on.
 *
 * The inline extensions service loads extensions directly from the add-on assets and does not load any code remotely.
 */
export class InlineExtensionsService implements IExtensionsService {
    public get activeExtensions(): Subscribable<ExecutableExtension[]> {
        return getInlineExtensions()
    }
}
