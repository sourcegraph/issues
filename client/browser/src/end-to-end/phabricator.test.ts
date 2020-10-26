import expect from 'expect'
import { afterEachSaveScreenshotIfFailed } from '../../../shared/src/testing/screenshotReporter'
import { createDriverForTest, Driver } from '../../../shared/src/testing/driver'
import { PhabricatorMapping } from '../browser-extension/web-extension-api/types'
import { isEqual } from 'lodash'
import { getConfig } from '../../../shared/src/testing/config'
import { retry } from '../../../shared/src/testing/utils'
import { ExternalServiceKind } from '../../../shared/src/graphql-operations'

// By default, these tests run against a local Phabricator instance and a local Sourcegraph instance.
// To run them against phabricator.sgdev.org and umami.sgdev.org, set the below env vars in addition to SOURCEGRAPH_BASE_URL.

const PHABRICATOR_BASE_URL = process.env.PHABRICATOR_BASE_URL || 'http://127.0.0.1'
const PHABRICATOR_USERNAME = process.env.PHABRICATOR_USERNAME || 'admin'
const PHABRICATOR_PASSWORD = process.env.PHABRICATOR_PASSWORD || 'sourcegraph'
const TEST_NATIVE_INTEGRATION = Boolean(
    process.env.TEST_NATIVE_INTEGRATION && JSON.parse(process.env.TEST_NATIVE_INTEGRATION)
)
const { gitHubToken, sourcegraphBaseUrl, ...restConfig } = getConfig('gitHubToken', 'sourcegraphBaseUrl')

/**
 * Logs into Phabricator.
 */
async function phabricatorLogin({ page }: Driver): Promise<void> {
    await page.goto(PHABRICATOR_BASE_URL)
    await page.waitForSelector('.phabricator-wordmark')
    if (await page.$('input[name=username]')) {
        await page.type('input[name=username]', PHABRICATOR_USERNAME)
        await page.type('input[name=password]', PHABRICATOR_PASSWORD)
        await page.click('button[name="__submit__"]')
    }
    await page.waitForSelector('.phabricator-core-user-menu')
}

/**
 * Waits for the jrpc repository to finish cloning.
 */
async function waitUntilRepositoryCloned(driver: Driver): Promise<void> {
    await retry(
        async () => {
            await driver.page.goto(PHABRICATOR_BASE_URL + '/source/jrpc/manage/status/')
            expect(
                await driver.page.evaluate(() =>
                    [...document.querySelectorAll('.phui-status-item-target')].map(element =>
                        element.textContent!.trim()
                    )
                )
            ).toContain('Fully Imported')
        },
        { retries: 20 }
    )
}

/**
 * Adds sourcegraph/jsonrpc2 to this Phabricator instance.
 */
async function addPhabricatorRepo(driver: Driver): Promise<void> {
    // These steps are idempotent as they will error if the same repo already exists

    // Add new repo to Diffusion
    await driver.page.goto(PHABRICATOR_BASE_URL + '/diffusion/edit/?vcs=git')
    await driver.page.waitForSelector('input[name=shortName]')
    await driver.page.type('input[name=name]', 'sourcegraph/jsonrpc2')
    await driver.page.type('input[name=callsign]', 'JRPC')
    await driver.page.type('input[name=shortName]', 'jrpc')
    await driver.page.click('button[type=submit]')

    // Configure it to clone github.com/sourcegraph/jsonrpc2
    await driver.page.goto(PHABRICATOR_BASE_URL + '/source/jrpc/uri/edit/')
    await driver.page.waitForSelector('input[name=uri]')
    await driver.page.type('input[name=uri]', 'https://github.com/sourcegraph/jsonrpc2.git')
    await driver.page.select('select[name=io]', 'observe')
    await driver.page.select('select[name=display]', 'always')
    await driver.page.click('button[type="submit"][name="__submit__"]')

    // Activate the repo and wait for it to clone
    await driver.page.goto(PHABRICATOR_BASE_URL + '/source/jrpc/manage/')
    const activateButton = await driver.page.waitForSelector('a[href="/source/jrpc/edit/activate/"]')
    const buttonLabel = ((await (await activateButton.getProperty('textContent')).jsonValue()) as string).trim()
    // Don't click if it says "Deactivate Repository"
    if (buttonLabel === 'Activate Repository') {
        await activateButton.click()
        await driver.page.waitForSelector('form[action="/source/jrpc/edit/activate/"]')
        await (await driver.page.$x('//button[text()="Activate Repository"]'))[0].click()
        await driver.page.waitForNavigation()
        await waitUntilRepositoryCloned(driver)
    }
}

async function configureSourcegraphIntegration(driver: Driver): Promise<void> {
    // Abort if plugin is not installed
    await driver.page.goto(PHABRICATOR_BASE_URL + '/config/application/')
    try {
        await driver.page.waitForSelector('a[href="/config/group/sourcegraph/"]', { timeout: 2000 })
    } catch {
        throw new Error(
            `The Sourcegraph native integration is not installed on ${PHABRICATOR_BASE_URL}. Please see https://docs.sourcegraph.com/dev/phabricator_gitolite#install-the-sourcegraph-phabricator-extension`
        )
    }

    // Configure the Sourcegraph URL
    await driver.page.goto(PHABRICATOR_BASE_URL + '/config/edit/sourcegraph.url/')
    await driver.replaceText({
        selector: '[name="value"]',
        newText: sourcegraphBaseUrl,
    })
    await Promise.all([driver.page.waitForNavigation(), driver.page.click('button[type="submit"]')])

    // Configure the repository mappings
    await driver.page.goto(PHABRICATOR_BASE_URL + '/config/edit/sourcegraph.callsignMappings/')
    const callSignConfigString = await driver.page.evaluate(() =>
        document.querySelector<HTMLTextAreaElement>('textarea[name="value"]')!.value.trim()
    )
    const callSignConfig: PhabricatorMapping[] = (callSignConfigString && JSON.parse(callSignConfigString)) || []
    const jsonRpc2Mapping: PhabricatorMapping = {
        path: 'github.com/sourcegraph/jsonrpc2',
        callsign: 'JRPC',
    }
    if (!callSignConfig.some(mapping => isEqual(jsonRpc2Mapping, mapping))) {
        await driver.replaceText({
            selector: 'textarea[name=value]',
            newText: JSON.stringify([...callSignConfig, jsonRpc2Mapping]),
        })
        await Promise.all([driver.page.waitForNavigation(), driver.page.click('button[type="submit"]')])
    }

    // Enable Sourcegraph native integration
    await driver.page.goto(PHABRICATOR_BASE_URL + '/config/edit/sourcegraph.enabled/')
    await driver.page.select('select[name="value"]', 'true')
    await Promise.all([driver.page.waitForNavigation(), driver.page.click('button[type="submit"]')])
}

/**
 * Runs initial setup for the Phabricator instance.
 */
async function init(driver: Driver): Promise<void> {
    if (restConfig.testUserPassword) {
        await driver.ensureLoggedIn({ username: 'test', password: restConfig.testUserPassword })
    }
    // TODO test with a Gitolite external service
    await driver.ensureHasExternalService({
        kind: ExternalServiceKind.GITHUB,
        displayName: 'GitHub (phabricator)',
        config: JSON.stringify({
            url: 'https://github.com',
            token: gitHubToken,
            repos: ['sourcegraph/jsonrpc2'],
            repositoryQuery: ['none'],
        }),
        ensureRepos: ['github.com/sourcegraph/jsonrpc2'],
    })
    await driver.ensureHasCORSOrigin({ corsOriginURL: PHABRICATOR_BASE_URL })
    await phabricatorLogin(driver)
    await addPhabricatorRepo(driver)
    if (TEST_NATIVE_INTEGRATION) {
        await configureSourcegraphIntegration(driver)
    } else {
        await driver.setExtensionSourcegraphUrl()
    }
}

describe('Sourcegraph Phabricator extension', () => {
    let driver: Driver

    before(async function () {
        this.timeout(4 * 60 * 1000)
        driver = await createDriverForTest({ loadExtension: !TEST_NATIVE_INTEGRATION, sourcegraphBaseUrl })
        await init(driver)
    })

    // Take a screenshot when a test fails.
    afterEachSaveScreenshotIfFailed(() => driver.page)

    it('adds "View on Sourcegraph" buttons to files', async () => {
        await driver.page.goto(
            PHABRICATOR_BASE_URL + '/source/jrpc/browse/master/call_opt.go;35a74f039c6a54af5bf0402d8f7da046c3f63ba2'
        )
        await driver.page.waitForSelector('.code-view-toolbar .open-on-sourcegraph')
        expect(await driver.page.$$('.code-view-toolbar .open-on-sourcegraph')).toHaveLength(1)
        await Promise.all([
            driver.page.waitForNavigation(),
            driver.page.click('.code-view-toolbar .open-on-sourcegraph'),
        ])
        expect(driver.page.url()).toBe(
            sourcegraphBaseUrl +
                '/github.com/sourcegraph/jsonrpc2@35a74f039c6a54af5bf0402d8f7da046c3f63ba2/-/blob/call_opt.go'
        )
    })

    it('shows hovers when clicking a token', async () => {
        await driver.page.goto(
            PHABRICATOR_BASE_URL + '/source/jrpc/browse/master/call_opt.go;35a74f039c6a54af5bf0402d8f7da046c3f63ba2'
        )
        await driver.page.waitForSelector('.code-view-toolbar .open-on-sourcegraph')

        // Pause to give codeintellify time to register listeners for
        // tokenization (only necessary in CI, not sure why).
        await driver.page.waitFor(1000)

        // Trigger tokenization of the line.
        const lineNumber = 5
        const codeLine = await driver.page.waitForSelector(
            `.diffusion-source > tbody > tr:nth-child(${lineNumber}) > td`
        )
        await codeLine.hover()

        // Once the line is tokenized, we can click on the individual token we want a hover for.
        const codeElement = await driver.page.waitForXPath(`//tbody/tr[${lineNumber}]//span[text()="CallOption"]`)
        await codeElement.click()
        await driver.page.waitForSelector('.test-tooltip-go-to-definition')
    })

    after(async () => {
        await driver.close()
    })
})
