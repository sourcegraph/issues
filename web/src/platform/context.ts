import { concat, ReplaySubject } from 'rxjs'
import { map, publishReplay, refCount } from 'rxjs/operators'
import { createExtensionHost } from '../../../shared/src/api/extension/worker'
import { PlatformContext } from '../../../shared/src/platform/context'
import { mutateSettings, updateSettings, fetchViewerSettings } from '../../../shared/src/settings/edit'
import { gqlToCascade } from '../../../shared/src/settings/settings'
import { asError } from '../../../shared/src/util/errors'
import { LocalStorageSubject } from '../../../shared/src/util/LocalStorageSubject'
import {
    toPrettyBlobURL,
    RepoFile,
    UIPositionSpec,
    ViewStateSpec,
    RenderModeSpec,
    UIRangeSpec,
} from '../../../shared/src/util/url'
import { requestGraphQL as webRequestGraphQL } from '../backend/graphql'
import { Tooltip } from '../components/tooltip/Tooltip'
import { eventLogger } from '../tracking/eventLogger'
import { SettingsCascadeFields } from '../../../shared/src/graphql-operations'

const requestGraphQL: PlatformContext['requestGraphQL'] = ({ request, variables }) =>
    webRequestGraphQL(request, variables)

/**
 * Creates the {@link PlatformContext} for the web app.
 */
export function createPlatformContext(): PlatformContext {
    const updatedSettings = new ReplaySubject<SettingsCascadeFields>(1)
    const context: PlatformContext = {
        settings: concat(fetchViewerSettings(requestGraphQL), updatedSettings).pipe(
            map(gqlToCascade),
            publishReplay(1),
            refCount()
        ),
        updateSettings: async (subject, edit) => {
            // Unauthenticated users can't update settings. (In the browser extension, they can update client
            // settings even when not authenticated. The difference in behavior in the web app vs. browser
            // extension is why this logic lives here and not in shared/.)
            if (!window.context.isAuthenticatedUser) {
                const editDescription =
                    typeof edit === 'string' ? 'edit settings' : 'update setting `' + edit.path.join('.') + '`'
                const url = new URL(window.context.externalURL)
                throw new Error(
                    `Unable to ${editDescription} because you are not signed in.` +
                        '\n\n' +
                        `[**Sign into Sourcegraph${
                            url.hostname === 'sourcegraph.com' ? '' : ` on ${url.host}`
                        }**](${`${url.href.replace(/\/$/, '')}/sign-in`})`
                )
            }

            try {
                await updateSettings(context, subject, edit, mutateSettings)
            } catch (error) {
                if (asError(error).message.includes('version mismatch')) {
                    // The user probably edited the settings in another tab, so
                    // try once more.
                    updatedSettings.next(await fetchViewerSettings(requestGraphQL).toPromise())
                    await updateSettings(context, subject, edit, mutateSettings)
                } else {
                    throw error
                }
            }
            updatedSettings.next(await fetchViewerSettings(requestGraphQL).toPromise())
        },
        requestGraphQL,
        forceUpdateTooltip: () => Tooltip.forceUpdate(),
        createExtensionHost: () => Promise.resolve(createExtensionHost()),
        urlToFile: toPrettyWebBlobURL,
        getScriptURLForExtension: bundleURL => bundleURL,
        sourcegraphURL: window.context.externalURL,
        clientApplication: 'sourcegraph',
        sideloadedExtensionURL: new LocalStorageSubject<string | null>('sideloadedExtensionURL', null),
        telemetryService: eventLogger,
    }
    return context
}

function toPrettyWebBlobURL(
    context: RepoFile &
        Partial<UIPositionSpec> &
        Partial<ViewStateSpec> &
        Partial<UIRangeSpec> &
        Partial<RenderModeSpec>
): string {
    const url = new URL(toPrettyBlobURL(context), location.href)
    url.searchParams.set('subtree', 'true')
    return url.pathname + url.search + url.hash
}
