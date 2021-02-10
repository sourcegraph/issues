import expect from 'expect'
import { Driver, createDriverForTest } from '../../../shared/src/testing/driver'
import { retry } from '../../../shared/src/testing/utils'
import { afterEachSaveScreenshotIfFailed } from '../../../shared/src/testing/screenshotReporter'
import { WebIntegrationTestContext, createWebIntegrationTestContext } from './context'
import { test } from 'mocha'

describe('directory page', () => {
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
    it('shows a row for each file in the directory', async () => {
        await driver.page.goto(
            driver.sourcegraphBaseUrl + '/github.com/gorilla/securecookie@e59506cc896acb7f7bf732d4fdf5e25f7ccd8983'
        )
        await driver.page.waitForSelector('.test-tree-entries', { visible: true })
        await retry(async () =>
            expect(
                await driver.page.evaluate(() => document.querySelectorAll('.test-tree-entry-directory').length)
            ).toStrictEqual(1)
        )
        await retry(async () =>
            expect(
                await driver.page.evaluate(() => document.querySelectorAll('.test-tree-entry-file').length)
            ).toStrictEqual(7)
        )
    })

    test('shows commit information on a row', async () => {
        await driver.page.goto(
            driver.sourcegraphBaseUrl + '/github.com/sourcegraph/go-diff@3f415a150aec0685cb81b73cc201e762e075006d',
            {
                waitUntil: 'domcontentloaded',
            }
        )
        await driver.page.waitForSelector('.test-tree-page-no-recent-commits')
        await driver.page.click('.test-tree-page-show-all-commits')
        await driver.page.waitForSelector('.git-commit-node__message', { visible: true })
        await retry(async () =>
            expect(
                await driver.page.evaluate(() => document.querySelectorAll('.git-commit-node__message')[3].textContent)
            ).toContain('Add support for new/removed binary files.')
        )
        await retry(async () =>
            expect(
                await driver.page.evaluate(() =>
                    document.querySelectorAll('.git-commit-node-byline')[3].textContent!.trim()
                )
            ).toContain('Dmitri Shuralyov')
        )
        await retry(async () =>
            expect(
                await driver.page.evaluate(() => document.querySelectorAll('.git-commit-node__oid')[3].textContent)
            ).toEqual('2083912')
        )
    })

    it('navigates when clicking on a row', async () => {
        await driver.page.goto(
            driver.sourcegraphBaseUrl + '/github.com/sourcegraph/jsonrpc2@c6c7b9aa99fb76ee5460ccd3912ba35d419d493d'
        )
        // click on directory
        await driver.page.waitForSelector('.tree-entry', { visible: true })
        await driver.page.click('.tree-entry')
        await driver.assertWindowLocation(
            '/github.com/sourcegraph/jsonrpc2@c6c7b9aa99fb76ee5460ccd3912ba35d419d493d/-/tree/websocket'
        )
    })
})
