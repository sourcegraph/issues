import { GraphQLClient } from './GraphQlClient'
import * as GQL from '../../../../shared/src/graphql/schema'
import { mutateSettings } from '../../../../shared/src/settings/edit'
import { getUser } from './api'
/**
 * Applies an edit to the user settings for the given username.
 */
export async function editUserSettings(
    username: string,
    edit: GQL.ISettingsEdit,
    graphQLClient: GraphQLClient
): Promise<void> {
    const user = await getUser(graphQLClient, username)
    if (!user) {
        throw new Error(`User not found: ${username}`)
    }
    const [{ latestSettings }] = user.settingsCascade.subjects.slice(-1)
    const lastID = latestSettings ? latestSettings.id : null
    await mutateSettings(graphQLClient, user.id, lastID, edit)
}
