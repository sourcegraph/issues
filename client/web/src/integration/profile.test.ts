import assert from 'assert'
import { createDriverForTest, Driver } from '../../../shared/src/testing/driver'
import { commonWebGraphQlResults } from './graphQlResults'
import { createWebIntegrationTestContext, WebIntegrationTestContext } from './context'
import { afterEachSaveScreenshotIfFailed } from '../../../shared/src/testing/screenshotReporter'
import { UserAreaUserFields } from '../graphql-operations'

describe('User profile page', () => {
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

    it('updates display name', async () => {
        const USER: UserAreaUserFields = {
            __typename: 'User',
            id: 'VXNlcjoxODkyNw==',
            username: 'test',
            displayName: 'Test',
            url: '/users/test',
            settingsURL: '/users/test/settings',
            avatarURL: '',
            viewerCanAdminister: true,
            viewerCanChangeUsername: true,
            siteAdmin: true,
            builtinAuth: true,
            createdAt: '2020-04-10T21:11:42Z',
            emails: [{ email: 'test@example.com', verified: true }],
            organizations: { nodes: [] },
            permissionsInfo: null,
            tags: [],
        }
        testContext.overrideGraphQL({
            ...commonWebGraphQlResults,
            UserArea: () => ({
                user: USER,
            }),
            UpdateUser: () => ({ updateUser: { ...USER, displayName: 'Test2' } }),
        })
        await driver.page.goto(driver.sourcegraphBaseUrl + '/users/test/settings/profile')
        await driver.page.waitForSelector('.user-profile-form-fields')
        await driver.replaceText({
            selector: '.test-UserProfileFormFields__displayName',
            newText: 'Test2',
            selectMethod: 'selectall',
        })

        const requestVariables = await testContext.waitForGraphQLRequest(async () => {
            await driver.page.click('#test-EditUserProfileForm__save')
        }, 'UpdateUser')

        assert.strictEqual(requestVariables.displayName, 'Test2')
    })
})
