import expect from 'expect'
import { afterEachSaveScreenshotIfFailed } from '../../../shared/src/testing/screenshotReporter'
import { createDriverForTest, Driver } from '../../../shared/src/testing/driver'
import { retry } from '../../../shared/src/testing/utils'
import { testSingleFilePage } from './shared'
import { getConfig } from '../../../shared/src/testing/config'
import { ExternalServiceKind } from '../../../shared/src/graphql-operations'

// By default, these tests run against a local Bitbucket instance and a local Sourcegraph instance.
// You can run them against other instances by setting the below env vars in addition to SOURCEGRAPH_BASE_URL.

const BITBUCKET_BASE_URL = process.env.BITBUCKET_BASE_URL || 'http://localhost:7990'
const BITBUCKET_USERNAME = process.env.BITBUCKET_USERNAME || 'test'
const BITBUCKET_PASSWORD = process.env.BITBUCKET_PASSWORD || 'test'
const TEST_NATIVE_INTEGRATION = Boolean(process.env.TEST_NATIVE_INTEGRATION)
const REPO_PATH_PREFIX = new URL(BITBUCKET_BASE_URL).hostname

const BITBUCKET_INTEGRATION_JAR_URL = 'https://storage.googleapis.com/sourcegraph-for-bitbucket-server/latest.jar'

const { sourcegraphBaseUrl, ...restConfig } = getConfig('sourcegraphBaseUrl')

/**
 * Logs into Bitbucket.
 */
async function bitbucketLogin({ page }: Driver): Promise<void> {
    await page.goto(new URL('/login', BITBUCKET_BASE_URL).toString())
    if (new URL(page.url()).pathname.endsWith('/login')) {
        await page.type('#j_username', BITBUCKET_USERNAME)
        await page.type('#j_password', BITBUCKET_PASSWORD)
        await Promise.all([page.waitForNavigation(), page.click('#submit')])
    }
    if (new URL(page.url()).pathname.endsWith('/login')) {
        throw new Error('Failed to authenticate to bitbucket server')
    }
}

async function createProject(driver: Driver): Promise<void> {
    await driver.page.goto(BITBUCKET_BASE_URL + '/projects')
    await driver.page.waitForSelector('.entity-table')
    const existingProject = await driver.page.evaluate(() =>
        [...document.querySelectorAll('span.project-name')].some(project => project.textContent === 'SOURCEGRAPH')
    )
    if (existingProject) {
        return
    }
    await driver.page.goto(BITBUCKET_BASE_URL + '/projects?create')
    await driver.page.type('form.project-settings input[name="key"]', 'SOURCEGRAPH')
    await driver.page.type('form.project-settings input[name="name"]', 'SOURCEGRAPH')
    await Promise.all([
        driver.page.waitForNavigation(),
        driver.page.click('form.project-settings input.aui-button-primary[type="submit"]'),
    ])
    if (new URL(driver.page.url()).search.endsWith('?create')) {
        throw new Error('Failed to authenticate to bitbucket server')
    }
}

/**
 * Adds sourcegraph/jsonrpc2 to this Bitbucket instance.
 */
async function importBitbucketRepo(driver: Driver): Promise<void> {
    await createProject(driver)
    // Import repo (idempotent)
    await driver.page.goto(BITBUCKET_BASE_URL + '/plugins/servlet/import-repository/SOURCEGRAPH')
    await driver.page.waitForSelector('button[data-source="GIT"]')
    await driver.page.click('button[data-source="GIT"]')
    await driver.page.waitForSelector('input[name="url"]')
    await driver.page.type('.source-form.git-specific input[name="url"]', 'https://github.com/sourcegraph/jsonrpc2')
    // Need to focus the next input field to trigger validation and have the submit button be enabled
    await driver.page.focus('.source-form.git-specific input[name="username"]')
    await driver.page.click('.next-step [name="connect"]')
    await retry(async () => {
        const browsePage = '/projects/SOURCEGRAPH/repos/jsonrpc2/browse'
        await driver.page.goto(BITBUCKET_BASE_URL + browsePage)
        // Retry until not redirected to the "import in progress" page anymore
        expect(new URL(driver.page.url()).pathname).toBe(new URL(browsePage, BITBUCKET_BASE_URL).pathname)
        // Ensure this is not a 404 page
        expect(await driver.page.$('.filebrowser-content')).toBeTruthy()
    })
}

/**
 * Configures the Sourcegraph for Bitbucket Server integration on the Bitbucket instance.
 */
async function configureSourcegraphIntegration(driver: Driver, enable: boolean): Promise<void> {
    await driver.page.goto(BITBUCKET_BASE_URL + '/plugins/servlet/upm?source=side_nav_manage_addons')
    await driver.page.waitForSelector('#upm-manage-plugins-user-installed')
    const sourcegraphPluginSelector = '.upm-plugin[data-key="com.sourcegraph.plugins.sourcegraph-bitbucket"]'
    if (await driver.page.$(sourcegraphPluginSelector)) {
        // Expand plugin menu
        await driver.page.click(`${sourcegraphPluginSelector} .upm-plugin-row`)
        // Enable if needed
        if (await driver.page.$(`${sourcegraphPluginSelector}.disabled`)) {
            if (enable) {
                await driver.page.waitForSelector(`${sourcegraphPluginSelector} [data-action="ENABLE"]`)
                await driver.page.click(`${sourcegraphPluginSelector} [data-action="ENABLE"]`)
                await driver.page.waitForSelector(`${sourcegraphPluginSelector} [data-action="DISABLE"]`)
            }
        } else if (!enable) {
            await driver.page.waitForSelector(`${sourcegraphPluginSelector} [data-action="DISABLE"]`)
            await driver.page.click(`${sourcegraphPluginSelector} [data-action="DISABLE"]`)
            await driver.page.waitForSelector(`${sourcegraphPluginSelector} [data-action="ENABLE"]`)
        }
    } else if (enable) {
        // Install
        await driver.page.click('#upm-upload')
        await driver.page.waitForSelector('#upm-upload-url')
        await driver.page.type('#upm-upload-url', BITBUCKET_INTEGRATION_JAR_URL)
        await driver.page.click('#upm-upload-dialog button.confirm')
        await driver.page.waitForSelector(sourcegraphPluginSelector)
    }
    await driver.page.reload()
    if (enable) {
        await driver.page.waitForSelector('#sourcegraph-admin-link')
        await driver.page.click('#sourcegraph-admin-link')
        await driver.page.waitForSelector('form#admin')
        // The Sourcegraph URL input field is disabled until the Sourcegraph URL has been fetched.
        await retry(async () => {
            expect(
                await driver.page.evaluate(
                    () => document.querySelector<HTMLInputElement>('form#admin input#url')!.disabled
                )
            ).toBe(false)
        })
        await driver.replaceText({ selector: 'form#admin input#url', newText: sourcegraphBaseUrl })
        await driver.page.click('form#admin input#submit')
        await driver.page.waitForSelector('.aui-message-success')
    }
}

describe('Sourcegraph browser extension on Bitbucket Server', () => {
    let driver: Driver

    before(async function () {
        this.timeout(4 * 60 * 1000)
        driver = await createDriverForTest({ loadExtension: !TEST_NATIVE_INTEGRATION, sourcegraphBaseUrl })
        if (sourcegraphBaseUrl !== 'https://sourcegraph.com' && restConfig.testUserPassword) {
            await driver.ensureLoggedIn({ username: 'test', password: restConfig.testUserPassword })
        }

        await bitbucketLogin(driver)

        await configureSourcegraphIntegration(driver, TEST_NATIVE_INTEGRATION)

        if (!TEST_NATIVE_INTEGRATION) {
            await driver.setExtensionSourcegraphUrl()
        }

        await importBitbucketRepo(driver)

        if (sourcegraphBaseUrl !== 'https://sourcegraph.com') {
            if (restConfig.testUserPassword) {
                await driver.ensureLoggedIn({ username: 'test', password: restConfig.testUserPassword })
            }
            await driver.ensureHasExternalService({
                kind: ExternalServiceKind.BITBUCKETSERVER,
                displayName: `Bitbucket ${BITBUCKET_BASE_URL}`,
                config: JSON.stringify({
                    url: BITBUCKET_BASE_URL,
                    username: BITBUCKET_USERNAME,
                    password: BITBUCKET_PASSWORD,
                    repos: ['SOURCEGRAPH/jsonrpc2'],
                }),
                ensureRepos: [REPO_PATH_PREFIX + '/SOURCEGRAPH/jsonrpc2'],
            })

            const bbsUrl = new URL(BITBUCKET_BASE_URL)
            // On localhost, allow all.
            const corsOrigin = bbsUrl.hostname === 'localhost' ? '*' : bbsUrl.origin
            await driver.ensureHasCORSOrigin({ corsOriginURL: corsOrigin })
        }
    })

    after(async () => {
        await driver.close()
    })

    // Take a screenshot when a test fails.
    afterEachSaveScreenshotIfFailed(() => driver.page)

    testSingleFilePage({
        getDriver: () => driver,
        url: `${BITBUCKET_BASE_URL}/projects/SOURCEGRAPH/repos/jsonrpc2/browse/call_opt.go?until=4fb7cd90793ee6ab445f466b900e6bffb9b63d78&untilPath=call_opt.go`,
        repoName: `${REPO_PATH_PREFIX}/SOURCEGRAPH/jsonrpc2`,
        sourcegraphBaseUrl,
        lineSelector: '.line',
    })
})
