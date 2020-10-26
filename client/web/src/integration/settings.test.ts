import assert from 'assert'
import { retry } from '../../../shared/src/testing/utils'
import { createDriverForTest, Driver } from '../../../shared/src/testing/driver'
import { commonWebGraphQlResults } from './graphQlResults'
import { createWebIntegrationTestContext, WebIntegrationTestContext } from './context'
import { settingsID, testUserID } from '../../../shared/src/testing/integration/graphQlResults'
import { afterEachSaveScreenshotIfFailed } from '../../../shared/src/testing/screenshotReporter'

describe('Settings', () => {
    let driver: Driver
    before(async () => {
        driver = await createDriverForTest()
    })
    after(() => driver?.close())
    let testContext: WebIntegrationTestContext
    beforeEach(async function () {
        testContext = await createWebIntegrationTestContext({
            driver,
            currentTest: this.currentTest!,
            directory: __dirname,
        })
    })
    afterEachSaveScreenshotIfFailed(() => driver.page)
    afterEach(() => testContext?.dispose())

    describe('User settings page', () => {
        it('updates user settings', async () => {
            testContext.overrideGraphQL({
                ...commonWebGraphQlResults,
                SettingsCascade: () => ({
                    settingsSubject: {
                        settingsCascade: {
                            subjects: [
                                {
                                    latestSettings: {
                                        id: settingsID,
                                        contents: JSON.stringify({}),
                                    },
                                },
                            ],
                        },
                    },
                }),
                OverwriteSettings: () => ({
                    settingsMutation: {
                        overwriteSettings: {
                            empty: {
                                alwaysNil: null,
                            },
                        },
                    },
                }),
                UserArea: () => ({
                    user: {
                        __typename: 'User',
                        id: testUserID,
                        username: 'test',
                        displayName: null,
                        url: '/users/test',
                        settingsURL: '/users/test/settings',
                        avatarURL: null,
                        viewerCanAdminister: true,
                        viewerCanChangeUsername: true,
                        siteAdmin: true,
                        builtinAuth: true,
                        createdAt: '2020-03-02T11:52:15Z',
                        emails: [{ email: 'test@sourcegraph.test', verified: true }],
                        organizations: { nodes: [] },
                        permissionsInfo: null,
                        tags: [],
                    },
                }),
            })

            const getSettingsEditorContent = async (): Promise<string | null | undefined> => {
                await driver.page.waitForSelector('.test-settings-file .monaco-editor .view-lines')
                return driver.page.evaluate(
                    () =>
                        document
                            .querySelector<HTMLElement>('.test-settings-file .monaco-editor .view-lines')
                            ?.textContent?.replace(/\u00A0/g, ' ') // Monaco replaces all spaces with &nbsp;
                )
            }

            await driver.page.goto(driver.sourcegraphBaseUrl + '/users/test/settings')

            await driver.page.waitForSelector('.test-settings-file .monaco-editor')
            await driver.page.waitForSelector('.test-save-toolbar-save')

            assert.strictEqual(
                await driver.page.evaluate(
                    () => document.querySelector<HTMLButtonElement>('.test-save-toolbar-save')?.disabled
                ),
                true,
                'Expected save button to be disabled'
            )

            // Replace with new settings
            const newSettings = '{ /* These are new settings */}'
            await driver.replaceText({
                selector: '.test-settings-file .monaco-editor .view-lines',
                newText: newSettings,
                selectMethod: 'keyboard',
                enterTextMethod: 'type',
            })
            await retry(async () => {
                const currentSettings = await getSettingsEditorContent()
                assert.strictEqual(currentSettings, newSettings)
            })

            assert.strictEqual(
                await driver.page.evaluate(
                    () => document.querySelector<HTMLButtonElement>('.test-save-toolbar-save')?.disabled
                ),
                false,
                'Expected save button to not be disabled'
            )

            // Assert mutation is done when save button is clicked
            const overrideSettingsVariables = await testContext.waitForGraphQLRequest(async () => {
                await driver.findElementWithText('Save changes', { action: 'click' })
            }, 'OverwriteSettings')

            assert.deepStrictEqual(overrideSettingsVariables, {
                contents: newSettings,
                lastID: settingsID,
                subject: testUserID,
            })
        })
    })
})
