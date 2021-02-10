import expect from 'expect'
import { Driver, createDriverForTest } from '../../../shared/src/testing/driver'
import { afterEachSaveScreenshotIfFailed } from '../../../shared/src/testing/screenshotReporter'
import { WebIntegrationTestContext, createWebIntegrationTestContext } from './context'
import { test } from 'mocha'

describe('Theme switcher', () => {
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
    test('changes the theme', async () => {
        await driver.page.goto(driver.sourcegraphBaseUrl + '/github.com/gorilla/mux/-/blob/mux.go')
        await driver.page.waitForSelector('.theme.theme-dark, .theme.theme-light', { visible: true })

        const getActiveThemeClasses = (): Promise<string[]> =>
            driver.page.evaluate(() =>
                [...document.querySelector('.theme')!.classList].filter(className => className.startsWith('theme-'))
            )

        expect(await getActiveThemeClasses()).toHaveLength(1)
        await driver.page.waitForSelector('.test-user-nav-item-toggle')
        await driver.page.click('.test-user-nav-item-toggle')

        // Switch to dark
        await driver.page.select('.test-theme-toggle', 'dark')
        expect(await getActiveThemeClasses()).toEqual(['theme-dark'])

        // Switch to light
        await driver.page.select('.test-theme-toggle', 'light')
        expect(await getActiveThemeClasses()).toEqual(['theme-light'])
    })
})
