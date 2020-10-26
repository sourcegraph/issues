import { afterEachSaveScreenshotIfFailed } from '../../../shared/src/testing/screenshotReporter'
import { createDriverForTest, Driver } from '../../../shared/src/testing/driver'
import { testSingleFilePage } from './shared'
import { getConfig } from '../../../shared/src/testing/config'
import { ExternalServiceKind } from '../../../shared/src/graphql-operations'

const GHE_BASE_URL = process.env.GHE_BASE_URL || 'https://ghe.sgdev.org'
const GHE_USERNAME = process.env.GHE_USERNAME
if (!GHE_USERNAME) {
    throw new Error('GHE_USERNAME environment variable must be set')
}
const GHE_PASSWORD = process.env.GHE_PASSWORD
if (!GHE_PASSWORD) {
    throw new Error('GHE_PASSWORD environment variable must be set')
}
const GHE_TOKEN = process.env.GHE_TOKEN
if (!GHE_TOKEN) {
    throw new Error('GHE_TOKEN environment variable must be set')
}

const REPO_PREFIX = new URL(GHE_BASE_URL).hostname

const { sourcegraphBaseUrl, ...restConfig } = getConfig('sourcegraphBaseUrl')

/**
 * Logs into GitHub Enterprise enterprise.
 */
async function gheLogin({ page }: Driver): Promise<void> {
    await page.goto(GHE_BASE_URL)
    if (new URL(page.url()).pathname.endsWith('/login')) {
        await page.type('#login_field', GHE_USERNAME!)
        await page.type('#password', GHE_PASSWORD!)
        await Promise.all([page.click('input[name=commit]'), page.waitForNavigation()])
    }
}

describe('Sourcegraph browser extension on GitHub Enterprise', () => {
    let driver: Driver

    before(async function () {
        this.timeout(4 * 60 * 1000)
        driver = await createDriverForTest({ loadExtension: true, sourcegraphBaseUrl })

        if (sourcegraphBaseUrl !== 'https://sourcegraph.com') {
            if (restConfig.testUserPassword) {
                await driver.ensureLoggedIn({ username: 'test', password: restConfig.testUserPassword })
            }
            await gheLogin(driver)
            await driver.setExtensionSourcegraphUrl()
            await driver.ensureHasExternalService({
                kind: ExternalServiceKind.GITHUB,
                displayName: 'GitHub Enterprise (e2e)',
                config: JSON.stringify({
                    url: GHE_BASE_URL,
                    token: GHE_TOKEN,
                    repos: ['sourcegraph/jsonrpc2'],
                }),
                ensureRepos: [`${REPO_PREFIX}/sourcegraph/jsonrpc2`],
            })
        }

        // GHE doesn't allow cloning public repos through the UI (only GitHub.com has the GitHub importer).
        // These tests expect that sourcegraph/jsonrpc2 is cloned on the GHE instance.
        await driver.page.goto(`${GHE_BASE_URL}/sourcegraph/jsonrpc2`)
        if (await driver.page.evaluate(() => document.querySelector('#not-found-search') !== null)) {
            throw new Error('You must clone sourcegraph/jsonrpc2 to your GHE instance to run these tests')
        }
    })

    after(async () => {
        await driver.close()
    })

    // Take a screenshot when a test fails.
    afterEachSaveScreenshotIfFailed(() => driver.page)

    testSingleFilePage({
        getDriver: () => driver,
        url: `${GHE_BASE_URL}/sourcegraph/jsonrpc2/blob/4fb7cd90793ee6ab445f466b900e6bffb9b63d78/call_opt.go`,
        repoName: `${REPO_PREFIX}/sourcegraph/jsonrpc2`,
        sourcegraphBaseUrl,
        // Not using '.js-file-line' because it breaks the reliance on :nth-child() in testSingleFilePage()
        lineSelector: '.js-file-line-container tr',
        goToDefinitionURL: new URL(
            '/sourcegraph/jsonrpc2/blob/4fb7cd90793ee6ab445f466b900e6bffb9b63d78/call_opt.go#L5:6',
            GHE_BASE_URL
        ).toString(),
    })
})
