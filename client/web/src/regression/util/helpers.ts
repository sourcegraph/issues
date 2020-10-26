import * as GQL from '../../../../shared/src/graphql/schema'
import { GraphQLClient } from './GraphQlClient'
import { Driver } from '../../../../shared/src/testing/driver'
import { gql, dataOrThrowErrors } from '../../../../shared/src/graphql/graphql'
import { catchError, map } from 'rxjs/operators'
import { throwError } from 'rxjs'
import { Key } from 'ts-key-enum'
import { PlatformContext } from '../../../../shared/src/platform/context'
import {
    deleteUser,
    getUser,
    createUser,
    fetchAllOrganizations,
    createOrganization,
    deleteOrganization,
    getViewerSettings,
    fetchSiteConfiguration,
    updateSiteConfiguration,
} from './api'
import { Config } from '../../../../shared/src/testing/config'
import { ResourceDestructor } from './TestResourceManager'
import * as jsonc from '@sqs/jsonc-parser'
import * as jsoncEdit from '@sqs/jsonc-parser/lib/edit'
import {
    GitHubAuthProvider,
    GitLabAuthProvider,
    OpenIDConnectAuthProvider,
    SAMLAuthProvider,
    SiteConfiguration,
} from '../../schema/site.schema'
import { first } from 'lodash'
import { overwriteSettings } from '../../../../shared/src/settings/edit'
import { retry } from '../../../../shared/src/testing/utils'
import { asError } from '../../../../shared/src/util/errors'

/**
 * Create the user with the specified password. Returns a destructor that destroys the test user. Assumes basic auth.
 */
export async function ensureLoggedInOrCreateTestUser(
    driver: Driver,
    gqlClient: GraphQLClient,
    {
        username,
        deleteIfExists,
        testUserPassword,
    }: {
        username: string
        deleteIfExists?: boolean
    } & Pick<Config, 'testUserPassword'>
): Promise<ResourceDestructor> {
    const userDestructor = (): Promise<void> => deleteUser(gqlClient, username, false)

    if (!username.startsWith('test-')) {
        throw new Error(`Test username must start with "test-" (was ${JSON.stringify(username)})`)
    }

    if (deleteIfExists) {
        await deleteUser(gqlClient, username, false)
    } else {
        // Attempt to log in first
        try {
            await driver.ensureLoggedIn({ username, password: testUserPassword })
            return userDestructor
        } catch (error) {
            console.log(
                `Login failed (error: ${asError(error).message}), will attempt to create user ${JSON.stringify(
                    username
                )}`
            )
        }
    }

    await createTestUser(driver, gqlClient, { username, testUserPassword })
    await driver.ensureLoggedIn({ username, password: testUserPassword })
    return userDestructor
}

async function createTestUser(
    driver: Driver,
    gqlClient: GraphQLClient,
    { username, testUserPassword }: { username: string } & Pick<Config, 'testUserPassword'>
): Promise<void> {
    // If there's an error, try to create the user
    const passwordResetURL = await gqlClient
        .mutateGraphQL(
            gql`
                mutation CreateUser($username: String!, $email: String) {
                    createUser(username: $username, email: $email) {
                        resetPasswordURL
                    }
                }
            `,
            { username }
        )
        .pipe(
            map(dataOrThrowErrors),
            catchError(error =>
                throwError(
                    new Error(
                        `Could not create user ${JSON.stringify(
                            username
                        )} (you may need to update the sudo access token used by the test): ${asError(error).message})`
                    )
                )
            ),
            map(({ createUser }) => createUser.resetPasswordURL)
        )
        .toPromise()
    if (!passwordResetURL) {
        throw new Error('passwordResetURL was empty')
    }

    await driver.page.goto(passwordResetURL)
    await driver.page.waitForSelector('.reset-password-page__form')
    await driver.page.keyboard.type(testUserPassword)
    await driver.page.keyboard.down(Key.Enter)

    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
    await driver.page.waitForFunction(() => document.body.textContent!.includes('Your password was reset'))
}

export async function createAuthProvider(
    gqlClient: GraphQLClient,
    authProvider: GitHubAuthProvider | GitLabAuthProvider | OpenIDConnectAuthProvider | SAMLAuthProvider
): Promise<ResourceDestructor> {
    const siteConfig = await fetchSiteConfiguration(gqlClient).toPromise()
    const siteConfigParsed: SiteConfiguration = jsonc.parse(siteConfig.configuration.effectiveContents)
    const authProviders = siteConfigParsed['auth.providers']
    if (
        authProviders?.some(
            provider =>
                provider.type === authProvider.type && (provider as any).displayName === authProvider.displayName
        )
    ) {
        return () => Promise.resolve() // provider already exists
    }
    const editFns = [
        (contents: string) =>
            jsoncEdit.setProperty(contents, ['auth.providers', -1], authProvider, {
                eol: '\n',
                insertSpaces: true,
                tabSize: 2,
            }),
    ]
    const { destroy } = await editSiteConfig(gqlClient, ...editFns)
    return destroy
}

/**
 * Ensures a new user, deleting the existing one if it already exists.
 */
export async function ensureNewUser(
    { requestGraphQL }: Pick<PlatformContext, 'requestGraphQL'>,
    username: string,
    email: string | undefined
): Promise<ResourceDestructor> {
    try {
        const user = await getUser({ requestGraphQL }, username)
        if (user) {
            await deleteUser({ requestGraphQL }, username)
        }
    } catch (error) {
        if (!asError(error).message.includes('user not found')) {
            throw error
        }
    }
    await createUser({ requestGraphQL }, username, email).toPromise()
    return () => deleteUser({ requestGraphQL }, username, true)
}

/**
 * Ensures a new organization, deleting the existing one if it already exists.
 */
export async function ensureNewOrganization(
    { requestGraphQL }: Pick<PlatformContext, 'requestGraphQL'>,
    variables: {
        /** The name of the organization. */
        name: string
        /** The new organization's display name (e.g. full name) in the organization profile. */
        displayName?: string
    }
): Promise<{ destroy: ResourceDestructor; result: GQL.IOrg }> {
    const matchingOrgs = (await fetchAllOrganizations({ requestGraphQL }, { first: 1000 }).toPromise()).nodes.filter(
        org => org.name === variables.name
    )
    if (matchingOrgs.length > 1) {
        throw new Error(`More than one organization name exists with name ${variables.name}`)
    }
    if (matchingOrgs.length === 1) {
        await deleteOrganization({ requestGraphQL }, matchingOrgs[0].id).toPromise()
    }
    const createdOrg = await createOrganization({ requestGraphQL }, variables).toPromise()
    return {
        destroy: () => deleteOrganization({ requestGraphQL }, createdOrg.id).toPromise(),
        result: createdOrg,
    }
}

export async function getGlobalSettings(
    gqlClient: GraphQLClient
): Promise<{ subjectID: GQL.ID; settingsID: number | null; contents: string }> {
    const settings = await getViewerSettings(gqlClient)
    const globalSettingsSubject = first(settings.subjects.filter(subject => subject.__typename === 'Site'))
    if (!globalSettingsSubject) {
        throw new Error('Could not get global settings')
    }
    return {
        subjectID: globalSettingsSubject.id,
        settingsID: globalSettingsSubject.latestSettings?.id ?? null,
        contents: globalSettingsSubject.latestSettings?.contents || '',
    }
}

export async function editGlobalSettings(
    gqlClient: GraphQLClient,
    ...edits: ((contents: string) => jsonc.Edit[])[]
): Promise<{ destroy: ResourceDestructor; result: string }> {
    const { subjectID, settingsID, contents: origContents } = await getGlobalSettings(gqlClient)
    let newContents = origContents
    for (const editFn of edits) {
        newContents = jsonc.applyEdits(newContents, editFn(newContents))
    }
    await overwriteSettings(gqlClient, subjectID, settingsID, newContents)
    return {
        destroy: async () => {
            const { subjectID, settingsID } = await getGlobalSettings(gqlClient)
            await overwriteSettings(gqlClient, subjectID, settingsID, origContents || '')
        },
        result: newContents,
    }
}

export async function editSiteConfig(
    gqlClient: GraphQLClient,
    ...edits: ((contents: string) => jsonc.Edit[])[]
): Promise<{ destroy: ResourceDestructor; result: boolean }> {
    const origConfig = await fetchSiteConfiguration(gqlClient).toPromise()
    let newContents = origConfig.configuration.effectiveContents
    for (const editFn of edits) {
        newContents = jsonc.applyEdits(newContents, editFn(newContents))
    }
    return {
        result: await updateSiteConfiguration(gqlClient, origConfig.configuration.id, newContents).toPromise(),
        destroy: async () => {
            const site = await fetchSiteConfiguration(gqlClient).toPromise()
            await updateSiteConfiguration(
                gqlClient,
                site.configuration.id,
                origConfig.configuration.effectiveContents
            ).toPromise()
        },
    }
}

export async function login(
    driver: Driver,
    {
        sourcegraphBaseUrl,
        authProviderDisplayName,
    }: Pick<Config, 'sourcegraphBaseUrl'> & { authProviderDisplayName: string },
    loginToAuthProvider: () => Promise<void>
): Promise<void> {
    await driver.page.goto(sourcegraphBaseUrl + '/-/sign-out')
    await driver.newPage()
    await driver.page.goto(sourcegraphBaseUrl)
    await retry(async () => {
        await driver.page.reload()
        await driver.findElementWithText('Continue with ' + authProviderDisplayName, {
            action: 'click',
            selector: 'a',
            wait: { timeout: 5000 },
        })
        await driver.page.waitForNavigation({ timeout: 3000 })
    })
    if (driver.page.url() !== sourcegraphBaseUrl + '/search') {
        await loginToAuthProvider()
        try {
            await driver.page.waitForFunction(
                url => document.location.href === url,
                { timeout: 5 * 1000 },
                sourcegraphBaseUrl + '/search'
            )
        } catch {
            throw new Error('unsuccessful login')
        }
    }
}

export async function loginToOkta(driver: Driver, username: string, password: string): Promise<void> {
    await driver.page.waitForSelector('#okta-signin-username')
    await driver.replaceText({
        selector: '#okta-signin-username',
        newText: username,
    })
    await driver.replaceText({
        selector: '#okta-signin-password',
        newText: password,
    })
    await driver.page.waitForSelector('#okta-signin-submit')
    await driver.page.click('#okta-signin-submit')
}

export async function loginToGitHub(driver: Driver, username: string, password: string): Promise<void> {
    await driver.page.waitForSelector('#login_field')
    await driver.replaceText({
        selector: '#login_field',
        newText: username,
        selectMethod: 'keyboard',
        enterTextMethod: 'paste',
    })
    await driver.replaceText({
        selector: '#password',
        newText: password,
        selectMethod: 'keyboard',
        enterTextMethod: 'paste',
    })
    await driver.page.keyboard.press('Enter')
}

export async function loginToGitLab(driver: Driver, username: string, password: string): Promise<void> {
    await driver.page.waitForSelector('input[name="user[login]"]', { timeout: 10000 })
    await driver.replaceText({
        selector: '#user_login',
        newText: username,
    })
    await driver.replaceText({
        selector: '#user_password',
        newText: password,
    })
    await driver.page.waitForSelector('input[data-qa-selector="sign_in_button"]')
    await driver.page.click('input[data-qa-selector="sign_in_button"]')
}
