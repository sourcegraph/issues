import { describe, before, after, test } from 'mocha'
import { Key } from 'ts-key-enum'
import { getConfig } from '../../../shared/src/testing/config'
import { getTestTools } from './util/init'
import { Driver } from '../../../shared/src/testing/driver'
import { GraphQLClient, createGraphQLClient } from './util/GraphQlClient'
import { TestResourceManager } from './util/TestResourceManager'
import { ensureLoggedInOrCreateTestUser, ensureNewUser, ensureNewOrganization, editSiteConfig } from './util/helpers'
import { getUser, setUserSiteAdmin, fetchAllOrganizations, deleteOrganization, getViewerSettings } from './util/api'
import { PlatformContext } from '../../../shared/src/platform/context'
import * as GQL from '../../../shared/src/graphql/schema'
import { parseJSONCOrError } from '../../../shared/src/util/jsonc'
import { Settings, QuickLink } from '../schema/settings.schema'
import * as jsoncEdit from '@sqs/jsonc-parser/lib/edit'
import { retry } from '../../../shared/src/testing/utils'
import delay from 'delay'
import { saveScreenshotsUponFailures } from '../../../shared/src/testing/screenshotReporter'
import { isErrorLike } from '../../../shared/src/util/errors'

async function deleteOrganizationByName(
    { requestGraphQL }: Pick<PlatformContext, 'requestGraphQL'>,
    name: string
): Promise<void> {
    const orgs = await fetchAllOrganizations({ requestGraphQL }, { first: 1000, query: '' }).toPromise()
    const matches = orgs.nodes.filter((org: GQL.Org) => org.name === name)
    await Promise.all(matches.map(org => deleteOrganization({ requestGraphQL }, org.id).toPromise()))
}

describe('Organizations regression test suite', () => {
    describe('Organizations GUI', () => {
        const testUsername = 'test-org'
        const testOrg = {
            name: 'test-org-1',
            displayName: 'Test Org 1',
        }
        const config = getConfig(
            'sudoToken',
            'sudoUsername',
            'headless',
            'slowMo',
            'keepBrowser',
            'noCleanup',
            'sourcegraphBaseUrl',
            'testUserPassword',
            'logBrowserConsole'
        )

        let driver: Driver
        let gqlClient: GraphQLClient
        let resourceManager: TestResourceManager
        before(async () => {
            ;({ driver, gqlClient, resourceManager } = await getTestTools(config))
            resourceManager.add(
                'User',
                testUsername,
                await ensureLoggedInOrCreateTestUser(driver, gqlClient, {
                    username: testUsername,
                    deleteIfExists: true,
                    ...config,
                })
            )
            const user = await getUser(gqlClient, testUsername)
            if (!user) {
                throw new Error(`test user ${testUsername} does not exist`)
            }
            await setUserSiteAdmin(gqlClient, user.id, true)
            await deleteOrganizationByName(gqlClient, testOrg.name)
        })

        saveScreenshotsUponFailures(() => driver.page)

        after(async () => {
            if (!config.noCleanup) {
                await resourceManager.destroyAll()
            }
            if (driver) {
                await driver.close()
            }
        })

        test('Create an organization and test settings cascade', async () => {
            const expectedQuicklink = { name: 'Test Org 1 Quicklink', url: 'http://test-org-1-link.local' }
            const userGQLClient = createGraphQLClient({
                baseUrl: config.sourcegraphBaseUrl,
                token: config.sudoToken,
                sudoUsername: testUsername,
            })
            const getQuickLinks = async (): Promise<QuickLink[] | undefined> => {
                const rawSettings = await getViewerSettings(userGQLClient)
                const settingsOrError = parseJSONCOrError<Settings>(rawSettings.final)
                if (isErrorLike(settingsOrError)) {
                    // TODO harmonize Error and ErrorLike interfaces
                    // eslint-disable-next-line rxjs/throw-error, etc/throw-error
                    throw settingsOrError
                }
                return settingsOrError.quicklinks
            }

            await driver.page.goto(config.sourcegraphBaseUrl + '/site-admin/organizations')
            await driver.findElementWithText('Create organization', { action: 'click', wait: { timeout: 2000 } })
            await driver.replaceText({
                selector: '.test-new-org-name-input',
                newText: testOrg.name,
            })
            await driver.replaceText({
                selector: '.test-new-org-display-name-input',
                newText: testOrg.displayName,
            })
            await driver.findElementWithText('Create organization', { action: 'click' })
            resourceManager.add('Organization', testOrg.name, () => deleteOrganizationByName(gqlClient, testOrg.name))
            await driver.page.waitForSelector('.test-settings-file .monaco-editor')
            await driver.replaceText({
                selector: '.test-settings-file .monaco-editor',
                newText: `{"quicklinks": [${JSON.stringify(expectedQuicklink)}]}`,
                selectMethod: 'keyboard',
                enterTextMethod: 'paste',
            })

            await driver.page.keyboard.down(Key.Control)
            await driver.page.keyboard.down(Key.Shift)
            await driver.page.keyboard.press('i')
            await driver.page.keyboard.up(Key.Shift)
            await driver.page.keyboard.up(Key.Control)
            await driver.findElementWithText('Save changes', { action: 'click' })
            await delay(500) // Wait for save
            await driver.findElementWithText('Save changes')

            {
                const quicklinks = await getQuickLinks()
                if (!quicklinks) {
                    throw new Error('No quicklinks found')
                }
                if (
                    !quicklinks.some(
                        quicklink =>
                            quicklink.name === expectedQuicklink.name && quicklink.url === expectedQuicklink.url
                    )
                ) {
                    throw new Error(
                        `Did not find quicklink found ${JSON.stringify(
                            expectedQuicklink
                        )} in quicklinks: ${JSON.stringify(quicklinks)}`
                    )
                }
            }

            // Remove user from org
            await driver.findElementWithText('Members', { action: 'click' })
            // eslint-disable-next-line @typescript-eslint/no-misused-promises
            driver.page.once('dialog', async dialog => {
                await dialog.accept()
            })
            await driver.findElementWithText('Leave organization', { action: 'click', wait: { timeout: 1000 } })

            await driver.page.waitForFunction(
                () => document.body.textContent && !document.body.textContent.includes('Leave organization')
            )

            {
                const quicklinks = await getQuickLinks()
                if (
                    quicklinks?.some(
                        quicklink =>
                            quicklink.name === expectedQuicklink.name && quicklink.url === expectedQuicklink.url
                    )
                ) {
                    throw new Error(
                        `Found quicklink ${JSON.stringify(expectedQuicklink)} in quicklinks: ${JSON.stringify(
                            quicklinks
                        )}`
                    )
                }
            }
        })
    })

    describe('Organizations API', () => {
        const resourceManager = new TestResourceManager()
        const config = getConfig(
            'sudoToken',
            'sudoUsername',
            'headless',
            'slowMo',
            'keepBrowser',
            'noCleanup',
            'sourcegraphBaseUrl',
            'testUserPassword',
            'logBrowserConsole'
        )
        after(async () => {
            if (!config.noCleanup) {
                await resourceManager.destroyAll()
            }
        })

        test('auth.userOrgMap', async function () {
            this.timeout(120 * 1000)
            if (process.env.NODE_TLS_REJECT_UNAUTHORIZED !== '0') {
                throw new Error(
                    'You must set environment variable NODE_TLS_REJECT_UNAUTHORIZED=0 when running this test.'
                )
            }

            const testUser1 = {
                username: 'test-org-user-1',
                email: 'beyang+test-org-user-1@sourcegraph.com',
            }
            const testOrg = {
                name: 'test-org-2',
                displayName: 'Test Org 2',
            }
            const gqlClient = createGraphQLClient({
                baseUrl: config.sourcegraphBaseUrl,
                token: config.sudoToken,
                sudoUsername: config.sudoUsername,
            })
            const formattingOptions = { eol: '\n', insertSpaces: true, tabSize: 2 }

            // Initial state: no auth.userOrgMap property
            resourceManager.add(
                'Configuration',
                'auth.userOrgMap',
                await editSiteConfig(gqlClient, contents =>
                    jsoncEdit.removeProperty(contents, ['auth.userOrgMap'], formattingOptions)
                )
            )

            // Retry, because the configuration update endpoint is eventually consistent
            let lastCreatedOrg: GQL.Org
            await retry(
                async () => {
                    // Create org
                    const createdOrg = resourceManager.add(
                        'Organization',
                        testOrg.name,
                        await ensureNewOrganization(gqlClient, testOrg)
                    )
                    lastCreatedOrg = createdOrg

                    // Create user
                    resourceManager.add(
                        'User',
                        testUser1.username,
                        await ensureNewUser(gqlClient, testUser1.username, testUser1.email)
                    )

                    // Check that user is not part of org
                    {
                        const user = await getUser(gqlClient, testUser1.username)
                        if (!user) {
                            throw new Error(`user ${testUser1.username} wasn't created`)
                        }
                        if (user.organizations.nodes.some(org => org.id === createdOrg.id)) {
                            throw new Error(`user ${testUser1.username} should not be part of org ${testOrg.name}`)
                        }
                    }
                },
                { retries: 3 }
            )

            // Set auth.userOrgMap
            await editSiteConfig(gqlClient, contents =>
                jsoncEdit.setProperty(contents, ['auth.userOrgMap'], { '*': [testOrg.name] }, formattingOptions)
            )

            await retry(
                async () => {
                    // Re-create user
                    await ensureNewUser(gqlClient, testUser1.username, testUser1.email)

                    // Check that user is part of organization
                    {
                        const user = await getUser(gqlClient, testUser1.username)
                        if (!user) {
                            throw new Error(`user ${testUser1.username} wasn't created`)
                        }
                        if (!user.organizations.nodes.some(org => org.id === lastCreatedOrg.id)) {
                            throw new Error(`user ${testUser1.username} should be part of org ${testOrg.name}`)
                        }
                    }
                },
                { retries: 3 }
            )
        })
    })
})
