import { startCase } from 'lodash'
import assert from 'assert'
import { afterEachSaveScreenshotIfFailed } from '../../../shared/src/testing/screenshotReporter'
import { Driver, createDriverForTest } from '../../../shared/src/testing/driver'
import { testSingleFilePage } from './shared'
import { retry } from '../../../shared/src/testing/utils'
import { getConfig } from '../../../shared/src/testing/config'
import { fromEvent } from 'rxjs'
import { first, filter, timeout, mergeMap } from 'rxjs/operators'
import { Target, Page } from 'puppeteer'
import { isDefined } from '../../../shared/src/util/types'

describe('Sourcegraph browser extension on github.com', function () {
    this.slow(8000)

    const { browser, sourcegraphBaseUrl, ...restConfig } = getConfig('browser', 'sourcegraphBaseUrl')

    let driver: Driver

    before('Open browser', async function () {
        this.timeout(90 * 1000)
        driver = await createDriverForTest({ loadExtension: true, browser, sourcegraphBaseUrl, ...restConfig })
        if (sourcegraphBaseUrl !== 'https://sourcegraph.com') {
            if (restConfig.testUserPassword) {
                await driver.ensureLoggedIn({ username: 'test', password: restConfig.testUserPassword })
            }
            await driver.setExtensionSourcegraphUrl()
        }
    })

    // Take a screenshot when a test fails
    afterEachSaveScreenshotIfFailed(() => driver.page)

    after('Close browser', () => driver?.close())

    testSingleFilePage({
        getDriver: () => driver,
        url: 'https://github.com/sourcegraph/jsonrpc2/blob/4fb7cd90793ee6ab445f466b900e6bffb9b63d78/call_opt.go',
        repoName: 'github.com/sourcegraph/jsonrpc2',
        sourcegraphBaseUrl,
        // Not using '.js-file-line' because it breaks the reliance on :nth-child() in testSingleFilePage()
        lineSelector: '.js-file-line-container tr',
        goToDefinitionURL:
            'https://github.com/sourcegraph/jsonrpc2/blob/4fb7cd90793ee6ab445f466b900e6bffb9b63d78/call_opt.go#L5:6',
    })

    const headGoToDefinitionUrl = new URL(
        '/github.com/gorilla/mux@e73f183699f8ab7d54609771e1fa0ab7ffddc21b/-/blob/regexp.go#L247:24&tab=def',
        sourcegraphBaseUrl
    ).toString()
    const tokens = {
        // https://github.com/gorilla/mux/pull/117/files#diff-9ef8a22c4ce5141c30a501c542fb1adeL244
        base: {
            token: 'varsN',
            lineId: 'diff-a609417fa264c6aed88fb8cfe2d9b4fb24226ffdf7db1f685e344d5239783d46L244',
            goToDefinitionURL:
                'https://github.com/gorilla/mux/blob/f15e0c49460fd49eebe2bcc8486b05d1bef68d3a/regexp.go#L139:2',
        },
        // https://github.com/gorilla/mux/pull/117/files#diff-9ef8a22c4ce5141c30a501c542fb1adeR247
        head: {
            token: 'host',
            lineId: 'diff-a609417fa264c6aed88fb8cfe2d9b4fb24226ffdf7db1f685e344d5239783d46R247',
            goToDefinitionURL: headGoToDefinitionUrl,
        },
    }

    describe('Pull request pages', () => {
        for (const diffType of ['unified', 'split']) {
            describe(`${startCase(diffType)} view`, () => {
                for (const side of ['base', 'head'] as const) {
                    const { token, lineId, goToDefinitionURL } = tokens[side]
                    it(`provides hover tooltips on token "${token}" in the ${side} part`, async () => {
                        await driver.page.goto(`https://github.com/gorilla/mux/pull/117/files?diff=${diffType}`)
                        await driver.page.bringToFront()
                        // The browser extension takes a bit to initialize and register all event listeners.
                        // Waiting here saves one retry cycle below in the common case.
                        // If it's not enough, the retry will catch it.
                        await driver.page.waitFor(1500)
                        const tokenElement = await retry(async () => {
                            const lineNumberElement = await driver.page.waitForSelector(`#${lineId}`, {
                                timeout: 10000,
                            })
                            const row = (
                                await driver.page.evaluateHandle(
                                    (element: Element) => element.closest('tr'),
                                    lineNumberElement
                                )
                            ).asElement()!
                            assert(row, 'Expected row to exist')
                            const tokenElement = (
                                await driver.page.evaluateHandle(
                                    (row: Element, token: string) =>
                                        [...row.querySelectorAll('span')].find(
                                            element => element.textContent === token
                                        ),
                                    row,
                                    token
                                )
                            ).asElement()
                            assert(tokenElement, 'Expected token element to exist')
                            return tokenElement
                        })
                        // Retry is here to wait for listeners to be registered
                        await retry(async () => {
                            await tokenElement.hover()
                            await driver.page.waitForSelector('.test-tooltip-go-to-definition', { timeout: 5000 })
                        })

                        // Check go-to-definition jumps to the right place
                        await retry(async () => {
                            const href = await driver.page.evaluate(
                                () => document.querySelector<HTMLAnchorElement>('.test-tooltip-go-to-definition')?.href
                            )
                            assert.strictEqual(href, goToDefinitionURL)
                        })
                        let page: Page = driver.page
                        if (new URL(goToDefinitionURL).hostname !== 'github.com') {
                            ;[page] = await Promise.all([
                                fromEvent<Target>(driver.browser, 'targetcreated')
                                    .pipe(
                                        timeout(5000),
                                        mergeMap(target => target.page()),
                                        filter(isDefined),
                                        first()
                                    )
                                    .toPromise(),
                                driver.page.click('.test-tooltip-go-to-definition'),
                            ])
                        } else {
                            await Promise.all([
                                driver.page.waitForNavigation(),
                                driver.page.click('.test-tooltip-go-to-definition'),
                            ])
                        }
                        await retry(async () => {
                            assert.strictEqual(await page.evaluate(() => location.href), goToDefinitionURL)
                        })
                        // If an additional page was opened, close it so we return to the original `driver.page`.
                        if (page !== driver.page) {
                            await page.close()
                        }
                    })
                }
            })
        }
    })
})
