import expect from 'expect'
import { describe, test, before, beforeEach, after } from 'mocha'
import { afterEachSaveScreenshotIfFailed } from '../../../shared/src/testing/screenshotReporter'
import { afterEachRecordCoverage } from '../../../shared/src/testing/coverage'
import { createDriverForTest, Driver } from '../../../shared/src/testing/driver'
import { ExternalServiceKind } from '../../../shared/src/graphql/schema'
import { getConfig } from '../../../shared/src/testing/config'

const { gitHubToken, sourcegraphBaseUrl } = getConfig('gitHubToken', 'sourcegraphBaseUrl')

describe('End-to-end test suite', () => {
    let driver: Driver
    const config = getConfig('headless', 'slowMo', 'testUserPassword')
    before(async function () {
        this.timeout(2 * 60 * 1000)
        // Start browser
        driver = await createDriverForTest({
            sourcegraphBaseUrl,
            logBrowserConsole: true,
            ...config,
        })
        const clonedRepoSlugs = ['sourcegraph/jsonrpc2', 'sourcegraph/go-diff']
        await driver.ensureLoggedIn({ username: 'test', password: config.testUserPassword, email: 'test@test.com' })
        await driver.resetUserSettings()
        await driver.ensureHasExternalService({
            kind: ExternalServiceKind.GITHUB,
            displayName: 'test-test-github',
            config: JSON.stringify({
                url: 'https://github.com',
                token: gitHubToken,
                repos: clonedRepoSlugs,
            }),
            ensureRepos: clonedRepoSlugs.map(slug => `github.com/${slug}`),
        })
    })

    after('Close browser', () => driver?.close())
    afterEachSaveScreenshotIfFailed(() => driver.page)
    afterEachRecordCoverage(() => driver)
    beforeEach(() => driver?.newPage())

    test('Failed login', async () => {
        await driver.newPage()
        await driver.page.goto(driver.sourcegraphBaseUrl)
        expect(new URL(driver.page.url()).pathname).toStrictEqual('/sign-in')
        await driver.page.waitForSelector('.test-signin-form')
        await driver.page.type('input', 'invalid-username')
        await driver.page.type('input[name=password]', 'invalid-password')
        await driver.page.click('button[type=submit]')
        expect(new URL(driver.page.url()).pathname).toStrictEqual('/sign-in')
        await driver.page.waitForSelector('.test-auth-error')
    })

    test('Search', async () => {
        await driver.page.goto(driver.sourcegraphBaseUrl + '/search?q=fmt.Sprintf')
        await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length > 0)
    })
})
