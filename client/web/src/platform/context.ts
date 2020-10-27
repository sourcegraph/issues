import { concat, Observable, ReplaySubject } from 'rxjs'
import { map, publishReplay, refCount } from 'rxjs/operators'
import { createExtensionHostWorker } from '../../../shared/src/api/extension/worker'
import { gql } from '../../../shared/src/graphql/graphql'
import * as GQL from '../../../shared/src/graphql/schema'
import { PlatformContext } from '../../../shared/src/platform/context'
import { mutateSettings, updateSettings } from '../../../shared/src/settings/edit'
import { gqlToCascade } from '../../../shared/src/settings/settings'
import { createAggregateError, asError } from '../../../shared/src/util/errors'
import { LocalStorageSubject } from '../../../shared/src/util/LocalStorageSubject'
import {
    toPrettyBlobURL,
    RepoFile,
    UIPositionSpec,
    ViewStateSpec,
    RenderModeSpec,
    UIRangeSpec,
} from '../../../shared/src/util/url'
import { queryGraphQL, requestGraphQL } from '../backend/graphql'
import { Tooltip } from '../../../branded/src/components/tooltip/Tooltip'
import { eventLogger } from '../tracking/eventLogger'

/**
 * Creates the {@link PlatformContext} for the web app.
 */
export function createPlatformContext(): PlatformContext {
    const updatedSettings = new ReplaySubject<GQL.ISettingsCascade>(1)
    const context: PlatformContext = {
        settings: concat(fetchViewerSettings(), updatedSettings).pipe(map(gqlToCascade), publishReplay(1), refCount()),
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
                    updatedSettings.next(await fetchViewerSettings().toPromise())
                    await updateSettings(context, subject, edit, mutateSettings)
                } else {
                    throw error
                }
            }
            updatedSettings.next(await fetchViewerSettings().toPromise())
        },
        requestGraphQL: ({ request, variables }) => requestGraphQL(request, variables),
        forceUpdateTooltip: () => Tooltip.forceUpdate(),
        createExtensionHost: () => Promise.resolve(createExtensionHostWorker()),
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

const settingsCascadeFragment = gql`
    fragment SettingsCascadeFields on SettingsCascade {
        subjects {
            __typename
            ... on Org {
                id
                name
                displayName
            }
            ... on User {
                id
                username
                displayName
            }
            ... on Site {
                id
                siteID
            }
            latestSettings {
                id
                contents
            }
            settingsURL
            viewerCanAdminister
        }
        final
    }
`

/**
 * Fetches the viewer's settings from the server. Callers should use settingsRefreshes#next instead of calling
 * this function, to ensure that the result is propagated consistently throughout the app instead of only being
 * returned to the caller.
 *
 * @returns Observable that emits the settings
 */
function fetchViewerSettings(): Observable<GQL.ISettingsCascade> {
    return queryGraphQL(gql`
        query ViewerSettings {
            viewerSettings {
                ...SettingsCascadeFields
            }
        }
        ${settingsCascadeFragment}
    `).pipe(
        map(({ data, errors }) => {
            if (!data?.viewerSettings) {
                throw createAggregateError(errors)
            }
            return data.viewerSettings
        })
    )
}
