import { combineLatest, Observable, ReplaySubject } from 'rxjs'
import { map, switchMap, take } from 'rxjs/operators'
import { PrivateRepoPublicSourcegraphComError } from '../../../../shared/src/backend/errors'
import { GraphQLResult } from '../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../shared/src/graphql/schema'
import { PlatformContext } from '../../../../shared/src/platform/context'
import { mutateSettings, updateSettings } from '../../../../shared/src/settings/edit'
import { EMPTY_SETTINGS_CASCADE, gqlToCascade } from '../../../../shared/src/settings/settings'
import { LocalStorageSubject } from '../../../../shared/src/util/LocalStorageSubject'
import { toPrettyBlobURL } from '../../../../shared/src/util/url'
import { ExtensionStorageSubject } from '../../browser-extension/web-extension-api/ExtensionStorageSubject'
import { background } from '../../browser-extension/web-extension-api/runtime'
import { isInPage } from '../context'
import { CodeHost } from '../code-hosts/shared/codeHost'
import { DEFAULT_SOURCEGRAPH_URL, observeSourcegraphURL } from '../util/context'
import { createExtensionHost } from './extensionHost'
import { editClientSettings, fetchViewerSettings, mergeCascades, storageSettingsCascade } from './settings'
import { requestGraphQlHelper } from '../backend/requestGraphQl'
import { isHTTPAuthError } from '../../../../shared/src/backend/fetch'
import { asError } from '../../../../shared/src/util/errors'
import { InlineExtensionsService, shouldUseInlineExtensions } from './inlineExtensionsService'
import { ModelService } from '../../../../shared/src/api/client/services/modelService'
import { IExtensionsService, ExtensionsService } from '../../../../shared/src/api/client/services/extensionsService'

export interface SourcegraphIntegrationURLs {
    /**
     * The URL of the configured Sourcegraph instance. Used for extensions, find-references, ...
     */
    sourcegraphURL: string

    /**
     * The base URL where assets will be fetched from (CSS, extension host
     * worker bundle, ...)
     *
     * This is the sourcegraph URL in most cases, but may be different for
     * native code hosts that self-host the integration bundle.
     */
    assetsURL: string
}

/**
 * The PlatformContext provided in the browser (for browser extensions and native integrations).
 */
export interface BrowserPlatformContext extends PlatformContext {
    /**
     * Refetches the settings cascade from the Sourcegraph instance.
     */
    refreshSettings(): Promise<void>
}

/**
 * Creates the {@link PlatformContext} for the browser (for browser extensions and native integrations)
 */
export function createPlatformContext(
    { urlToFile, getContext }: Pick<CodeHost, 'urlToFile' | 'getContext'>,
    { sourcegraphURL, assetsURL }: SourcegraphIntegrationURLs,
    isExtension: boolean
): BrowserPlatformContext {
    const updatedViewerSettings = new ReplaySubject<Pick<GQL.ISettingsCascade, 'subjects' | 'final'>>(1)
    const requestGraphQL: PlatformContext['requestGraphQL'] = <T, V = object>({
        request,
        variables,
        mightContainPrivateInfo,
    }: {
        request: string
        variables: V
        mightContainPrivateInfo: boolean
    }): Observable<GraphQLResult<T>> =>
        observeSourcegraphURL(isExtension).pipe(
            take(1),
            switchMap(sourcegraphURL => {
                if (mightContainPrivateInfo && sourcegraphURL === DEFAULT_SOURCEGRAPH_URL) {
                    // If we can't determine the code host context, assume the current repository is private.
                    const privateRepository = getContext ? getContext().privateRepository : true
                    if (privateRepository) {
                        const nameMatch = request.match(/^\s*(?:query|mutation)\s+(\w+)/)
                        throw new PrivateRepoPublicSourcegraphComError(nameMatch ? nameMatch[1] : '')
                    }
                }
                return requestGraphQlHelper(isExtension, sourcegraphURL)<T, V>({ request, variables })
            })
        )

    const context: BrowserPlatformContext = {
        /**
         * The active settings cascade.
         *
         * - For unauthenticated users, this is the GraphQL settings plus client settings (which are stored locally in the browser extension).
         * - For authenticated users, this is just the GraphQL settings (client settings are ignored to simplify the UX).
         */
        settings: combineLatest([updatedViewerSettings, storageSettingsCascade]).pipe(
            map(([gqlCascade, storageCascade]) =>
                gqlCascade
                    ? mergeCascades(
                          gqlToCascade(gqlCascade),
                          gqlCascade.subjects.some(subject => subject.__typename === 'User')
                              ? EMPTY_SETTINGS_CASCADE
                              : storageCascade
                      )
                    : EMPTY_SETTINGS_CASCADE
            )
        ),
        refreshSettings: async () => {
            try {
                const settings = await fetchViewerSettings(requestGraphQL).toPromise()
                updatedViewerSettings.next(settings)
            } catch (error) {
                if (isHTTPAuthError(error)) {
                    // User is not signed in
                    console.warn(
                        `Could not fetch Sourcegraph settings from ${sourcegraphURL} because user is not signed into Sourcegraph`
                    )
                    updatedViewerSettings.next({ final: '{}', subjects: [] })
                } else {
                    throw error
                }
            }
        },
        updateSettings: async (subject, edit) => {
            if (subject === 'Client') {
                // Support storing settings on the client (in the browser extension) so that unauthenticated
                // Sourcegraph viewers can update settings.
                await updateSettings(context, subject, edit, () => editClientSettings(edit))
                return
            }

            try {
                await updateSettings(context, subject, edit, mutateSettings)
            } catch (error) {
                if (asError(error).message.includes('version mismatch')) {
                    // The user probably edited the settings in another tab, so
                    // try once more.
                    await context.refreshSettings()
                    await updateSettings(context, subject, edit, mutateSettings)
                } else {
                    throw error
                }
            }
            // TODO: We shouldn't need to make another HTTP request to get the latest state
            await context.refreshSettings()
        },
        requestGraphQL,
        forceUpdateTooltip: () => {
            // TODO(sqs): implement tooltips on the browser extension
        },
        createExtensionHost: () => createExtensionHost({ assetsURL }),
        getScriptURLForExtension: async bundleURL => {
            if (isInPage) {
                return bundleURL
            }
            // We need to import the extension's JavaScript file (in importScripts in the Web Worker) from a blob:
            // URI, not its original http:/https: URL, because Chrome extensions are not allowed to be published
            // with a CSP that allowlists https://* in script-src (see
            // https://developer.chrome.com/extensions/contentSecurityPolicy#relaxing-remote-script). (Firefox
            // add-ons have an even stricter restriction.)
            const blobURL = await background.createBlobURL(bundleURL)
            return blobURL
        },
        urlToFile: ({ rawRepoName, ...target }, context) => {
            // We don't always resolve the rawRepoName, e.g. if there are multiple definitions.
            // Construct URL to file on code host, if possible.
            if (rawRepoName && urlToFile) {
                return urlToFile(sourcegraphURL, { rawRepoName, ...target }, context)
            }
            // Otherwise fall back to linking to Sourcegraph (with an absolute URL).
            return `${sourcegraphURL}${toPrettyBlobURL(target)}`
        },
        sourcegraphURL,
        clientApplication: 'other',
        sideloadedExtensionURL: isInPage
            ? new LocalStorageSubject<string | null>('sideloadedExtensionURL', null)
            : new ExtensionStorageSubject('sideloadedExtensionURL', null),
        createExtensionsService(modelService: Pick<ModelService, 'activeLanguages'>): IExtensionsService {
            // On Firefox extension, use the inline extensions service.
            if (shouldUseInlineExtensions()) {
                return new InlineExtensionsService()
            }

            // On all other platforms, use the standard extensions service.
            return new ExtensionsService(this, modelService)
        },
    }
    return context
}
