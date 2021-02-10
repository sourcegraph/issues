import { Driver, createDriverForTest, percySnapshot } from '../../../shared/src/testing/driver'
import { afterEachSaveScreenshotIfFailed } from '../../../shared/src/testing/screenshotReporter'
import { WebIntegrationTestContext, createWebIntegrationTestContext } from './context'
import { test } from 'mocha'

describe('Site admin area visual tests', () => {
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
    test('Repositories list', async () => {
        await driver.page.goto(driver.sourcegraphBaseUrl + '/site-admin/repositories?query=gorilla%2Fmux')
        await driver.page.waitForSelector('a[href="/github.com/gorilla/mux"]', { visible: true })
        await percySnapshot(driver.page, 'Repositories list')
    })

    test('Site admin overview', async () => {
        await driver.page.goto(driver.sourcegraphBaseUrl + '/site-admin')
        await driver.page.waitForSelector('.test-site-admin-overview-menu', { visible: true })
        await driver.page.waitForSelector('.test-product-certificate', { visible: true })
        await percySnapshot(driver.page, 'Site admin overview')
    })
})
