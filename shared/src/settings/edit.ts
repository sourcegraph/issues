import { from, Observable } from 'rxjs'
import { first, map, switchMap } from 'rxjs/operators'
import { SettingsEdit } from '../api/client/services/settings'
import { dataOrThrowErrors, gql } from '../graphql/graphql'
import * as GQL from '../graphql/schema'
import { PlatformContext } from '../platform/context'
import { isErrorLike } from '../util/errors'
import { ViewerSettingsResult } from '../graphql-operations'

const settingsCascadeFragment = gql`
    fragment SettingsCascadeFields on SettingsCascade {
        subjects {
            __typename
            id
            ... on Org {
                name
                displayName
            }
            ... on User {
                username
                displayName
            }
            ... on Site {
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
export function fetchViewerSettings(
    requestGraphQL: PlatformContext['requestGraphQL']
): Observable<ViewerSettingsResult['viewerSettings']> {
    return requestGraphQL<ViewerSettingsResult>({
        request: gql`
            query ViewerSettings {
                viewerSettings {
                    ...SettingsCascadeFields
                }
            }
            ${settingsCascadeFragment}
        `,
        mightContainPrivateInfo: false,
        variables: {},
    }).pipe(
        map(dataOrThrowErrors),
        map(data => data.viewerSettings)
    )
}

/**
 * A helper function for performing an update to settings.
 *
 * @param applySettingsEdit A function that is called to actually apply and persist the update.
 */
export function updateSettings(
    { settings, requestGraphQL }: Pick<PlatformContext, 'settings' | 'requestGraphQL'>,
    subjectToUpdate: GQL.Scalars['ID'],
    args: SettingsEdit | string,
    applySettingsEdit: (
        { requestGraphQL }: Pick<PlatformContext, 'requestGraphQL'>,
        subject: GQL.Scalars['ID'],
        lastID: number | null,
        edit: GQL.SettingsEdit | string
    ) => Promise<void>
): Promise<void> {
    return from(settings)
        .pipe(
            first(),
            switchMap(settingsCascade => {
                if (!settingsCascade.subjects) {
                    throw new Error('settings not available')
                }
                if (isErrorLike(settingsCascade.subjects)) {
                    throw new Error(`settings not available due to error: ${settingsCascade.subjects.message}`)
                }
                const subjectSettings = settingsCascade.subjects.find(subject => subject.subject.id === subjectToUpdate)
                if (!subjectSettings) {
                    throw new Error(`no settings subject: ${subjectToUpdate}`)
                }
                if (isErrorLike(subjectSettings.settings)) {
                    throw new Error(`settings subject error: ${subjectSettings.settings.message}`)
                }
                const lastID = subjectSettings.settings ? subjectSettings.lastID : null

                return applySettingsEdit(
                    { requestGraphQL },
                    subjectToUpdate,
                    lastID,
                    typeof args === 'string'
                        ? args
                        : {
                              keyPath: toGQLKeyPath(args.path),
                              value: args.value,
                          }
                )
            })
        )
        .toPromise()
}

function toGQLKeyPath(keyPath: (string | number)[]): GQL.KeyPathSegment[] {
    return keyPath.map(member => (typeof member === 'string' ? { property: member } : { index: member }))
}

/**
 * Perform a mutation against the GraphQL API to modify the settings for a subject.
 *
 * @param edit An edit to a specific value in the settings, or a stringified JSON value to overwrite the entire
 * settings.
 */
export function mutateSettings(
    { requestGraphQL }: Pick<PlatformContext, 'requestGraphQL'>,
    subject: GQL.Scalars['ID'],
    lastID: number | null,
    edit: GQL.ConfigurationEdit | string
): Promise<void> {
    return typeof edit === 'string'
        ? overwriteSettings({ requestGraphQL }, subject, lastID, edit)
        : editSettings({ requestGraphQL }, subject, lastID, edit)
}

/**
 * Perform a mutation against the GraphQL API to edit the settings for a subject.
 *
 * This function uses configurationMutation (not settingsMutation) and editConfiguration (not editSettings) for
 * backcompat.
 *
 * @param edit An edit to a specific value in the settings.
 */
function editSettings(
    { requestGraphQL }: Pick<PlatformContext, 'requestGraphQL'>,
    subject: GQL.Scalars['ID'],
    lastID: number | null,
    edit: GQL.ConfigurationEdit
): Promise<void> {
    return from(
        requestGraphQL({
            request: gql`
                mutation EditSettings($subject: ID!, $lastID: Int, $edit: ConfigurationEdit!) {
                    configurationMutation(input: { subject: $subject, lastID: $lastID }) {
                        editConfiguration(edit: $edit) {
                            empty {
                                alwaysNil
                            }
                        }
                    }
                }
            `,
            variables: { subject, lastID, edit },
            mightContainPrivateInfo: false,
        })
    )
        .pipe(
            map(dataOrThrowErrors),
            map(() => undefined)
        )
        .toPromise()
}

/**
 * Perform a mutation against the GraphQL API to overwrite the settings for a subject.
 *
 * NOTE: This GraphQL query is only compatible with Sourcegraph 2.13 and newer (due to the use of
 * Mutation.settingsMutation and SettingsMutation.overwriteSettings).
 *
 * @param contents A stringified JSON value to overwrite the entire settings with.
 */
export function overwriteSettings(
    { requestGraphQL }: Pick<PlatformContext, 'requestGraphQL'>,
    subject: GQL.Scalars['ID'],
    lastID: number | null,
    contents: string
): Promise<void> {
    return from(
        requestGraphQL({
            request: gql`
                mutation OverwriteSettings($subject: ID!, $lastID: Int, $contents: String!) {
                    settingsMutation(input: { subject: $subject, lastID: $lastID }) {
                        overwriteSettings(contents: $contents) {
                            empty {
                                alwaysNil
                            }
                        }
                    }
                }
            `,
            variables: { subject, lastID, contents },
            mightContainPrivateInfo: false,
        })
    )
        .pipe(
            map(dataOrThrowErrors),
            map(() => undefined)
        )
        .toPromise()
}
