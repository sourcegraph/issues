import { describe, before, test } from 'mocha'
import { Driver } from '../../../shared/src/testing/driver'
import { createAndInitializeDriver } from './util/init'
import { getConfig } from '../../../shared/src/testing/config'
import { Key } from 'ts-key-enum'
import { retry } from '../../../shared/src/testing/utils'
import delay from 'delay'

describe('Initialize new instance', () => {
    const config = getConfig(
        'init',
        'sourcegraphBaseUrl',
        'sudoUsername',
        'testUserPassword',
        'noCleanup',
        'headless',
        'slowMo',
        'logBrowserConsole',
        'logStatusMessages',
        'keepBrowser'
    )

    let driver: Driver
    before(async () => {
        driver = await createAndInitializeDriver(config)
    })
    ;(config.init ? test : test.skip)('Initialize new Sourcegraph instance', async function () {
        this.timeout(30 * 1000)
        await driver.page.goto(config.sourcegraphBaseUrl)
        await driver.page.waitForSelector('input[placeholder="Email"]', { timeout: 5 * 1000 })
        await driver.replaceText({
            selector: 'input[name="email"]',
            newText: 'insecure-dev-bots+admin@sourcegraph.com',
        })
        await driver.replaceText({
            selector: 'input[name="username"]',
            newText: config.sudoUsername,
        })
        await driver.replaceText({
            selector: 'input[name="password"]',
            newText: config.testUserPassword,
        })
        await driver.page.keyboard.press(Key.Enter)
        await driver.waitUntilURL(`${config.sourcegraphBaseUrl}/site-admin`)

        await retry(
            async () => {
                await driver.page.reload()
                await driver.findElementWithText('Connect a code host', { wait: { timeout: 5 * 1000 } })
                await delay(1000)
            },
            { retries: 10 }
        )
    })
})
