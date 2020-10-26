import expect from 'expect'
import { describe, before, beforeEach, afterEach, test } from 'mocha'
import { getTestTools } from './util/init'
import { getConfig } from '../../../shared/src/testing/config'
import { editSiteConfig } from './util/helpers'
import * as jsoncEdit from '@sqs/jsonc-parser/lib/edit'
import * as jsonc from '@sqs/jsonc-parser'
import { Driver } from '../../../shared/src/testing/driver'
import { TestResourceManager } from './util/TestResourceManager'
import { retry } from '../../../shared/src/testing/utils'
import { BuiltinAuthProvider, SiteConfiguration } from '../schema/site.schema'
import { fetchSiteConfiguration } from './util/api'
import { GraphQLClient } from './util/GraphQlClient'
import { afterEachSaveScreenshotIfFailed } from '../../../shared/src/testing/screenshotReporter'

describe('Site config test suite', () => {
    const formattingOptions = { eol: '\n', insertSpaces: true, tabSize: 2 }
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
    let resourceManager: TestResourceManager
    let gqlClient: GraphQLClient
    before(async () => {
        ;({ driver, resourceManager, gqlClient } = await getTestTools(config))
    })

    afterEachSaveScreenshotIfFailed(() => driver.page)

    beforeEach(() => {
        resourceManager = new TestResourceManager()
    })
    afterEach(async () => {
        await resourceManager.destroyAll()
    })
    test('htmlBodyTop', async function () {
        this.timeout(10 * 1000)
        resourceManager.add(
            'Configuration',
            'htmlBodyTop',
            await editSiteConfig(gqlClient, contents =>
                jsoncEdit.setProperty(
                    contents,
                    ['htmlBodyTop'],
                    '<div id="htmlBodyTopContent">TEST</div>',
                    formattingOptions
                )
            )
        )

        await retry(
            async () => {
                await driver.page.goto(config.sourcegraphBaseUrl)
                await driver.page.reload()
                await driver.page.waitForSelector('#htmlBodyTopContent', { timeout: 1000 })
            },
            { retries: 10 }
        )
    })

    test('builtin auth provider: allowSignup', async function () {
        this.timeout(20 * 1000)
        const siteConfig = await fetchSiteConfiguration(gqlClient).toPromise()
        const siteConfigParsed: SiteConfiguration = jsonc.parse(siteConfig.configuration.effectiveContents)
        const setBuiltinAuthProvider = async (provider: BuiltinAuthProvider) => {
            const authProviders = siteConfigParsed['auth.providers']
            const foundIndices =
                authProviders
                    ?.map((provider, index) => (provider.type === 'builtin' ? index : -1))
                    .filter(index => index !== -1) || []
            const builtinAuthProviderIndex = foundIndices.length > 0 ? foundIndices[0] : -1
            const editFns = []
            if (builtinAuthProviderIndex !== -1) {
                editFns.push((contents: string) => {
                    const parsed: SiteConfiguration = jsonc.parse(contents)
                    const found =
                        parsed['auth.providers']
                            ?.map((provider, index) => (provider.type === 'builtin' ? index : -1))
                            .filter(index => index !== -1) || []
                    const foundIndex = found.length > 0 ? found[0] : -1
                    if (foundIndex !== -1) {
                        return jsoncEdit.setProperty(
                            contents,
                            ['auth.providers', foundIndex],
                            undefined,
                            formattingOptions
                        )
                    }
                    return []
                })
            }
            editFns.push((contents: string) =>
                jsoncEdit.setProperty(contents, ['auth.providers', -1], provider, formattingOptions)
            )
            console.log('editFns', editFns)
            return editSiteConfig(gqlClient, ...editFns)
        }

        resourceManager.add(
            'Configuration',
            'builtin auth provider: allowSignup',
            await setBuiltinAuthProvider({ type: 'builtin', allowSignup: false })
        )

        await retry(
            async () => {
                await driver.page.goto(config.sourcegraphBaseUrl)
                await driver.page.reload()
                await driver.findElementWithText('Sign in', { wait: { timeout: 2000 } })
                expect(await driver.page.evaluate(() => document.body.textContent?.includes('Sign up'))).toBeFalsy()
            },
            { retries: 5 } // configuration propagation is eventually consistent
        )

        await setBuiltinAuthProvider({ type: 'builtin', allowSignup: true })
        await retry(
            async () => {
                await driver.page.goto(config.sourcegraphBaseUrl)
                await driver.page.reload()
                await driver.findElementWithText('Sign in', { wait: { timeout: 2000 } })
                expect(await driver.page.evaluate(() => document.body.textContent?.includes('Sign up'))).toBeTruthy()
            },
            { retries: 5 } // configuration propagation is eventually consistent
        )
    })
})
