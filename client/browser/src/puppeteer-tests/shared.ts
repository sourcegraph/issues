import expect from 'expect'
import { Driver } from '../../../shared/src/testing/driver'
import { retry } from '../../../shared/src/testing/utils'
import puppeteer from 'puppeteer'
import assert from 'assert'

/**
 * Defines e2e tests for a single-file page of a code host.
 */
export function testSingleFilePage({
    getDriver,
    url,
    repoName,
    lineSelector,
    goToDefinitionURL,
}: {
    /** Called to get the driver */
    getDriver: () => Driver

    /** The URL to sourcegraph/jsonrpc2 call_opt.go at commit 4fb7cd90793ee6ab445f466b900e6bffb9b63d78 on the code host */
    url: string

    /** The repo name of sourcgraph/jsonrpc2 on the Sourcegraph instance */
    repoName: string

    /** The CSS selector for a line in the code view */
    lineSelector: string
    /** The expected URL for the "Go to Definition" button */
    goToDefinitionURL?: string
}): void {
    describe('File views', () => {
        it('adds "View on Sourcegraph" buttons to files', async () => {
            await getDriver().page.goto(url)

            // Make sure the tab is active, because it might not be active if the install page has opened.
            await closeInstallPageTab(getDriver().page.browser())

            await getDriver().page.waitForSelector('.code-view-toolbar .open-on-sourcegraph', { timeout: 10000 })
            expect(await getDriver().page.$$('.code-view-toolbar .open-on-sourcegraph')).toHaveLength(1)

            // TODO: Uncomment this portion of the test once we migrate from puppeteer-firefox to puppeteer
            // We want to assert that Sourcegraph is opened in a new tab, but the old version of Firefox used
            // by puppeteer-firefox doesn't support the latest version of Sourcegraph. For now,
            // simply assert on the link.

            // await getDriver().page.click('.code-view-toolbar .open-on-sourcegraph')

            // // The button opens a new tab, so get the new page whose opener is the current page, and get its url.
            // const currentPageTarget = getDriver().page.target()
            // const newTarget = await getDriver().browser.waitForTarget(target => target.opener() === currentPageTarget)
            // const newPage = await newTarget.page()
            // expect(newPage.url()).toBe(
            //     `${sourcegraphBaseUrl}/${repoName}@4fb7cd90793ee6ab445f466b900e6bffb9b63d78/-/blob/call_opt.go?utm_source=chrome-extension`
            // )
            // await newPage.close()

            await retry(async () => {
                assert.strictEqual(
                    await getDriver().page.evaluate(
                        () => document.querySelector<HTMLAnchorElement>('.code-view-toolbar .open-on-sourcegraph')?.href
                    ),
                    `${
                        getDriver().sourcegraphBaseUrl
                    }/${repoName}@4fb7cd90793ee6ab445f466b900e6bffb9b63d78/-/blob/call_opt.go?utm_source=${
                        getDriver().browserType
                    }-extension`
                )
            })
        })

        // TODO(tj): this doesn't work without a mock code intel extension
        it.skip('shows hover tooltips when hovering a token', async () => {
            await getDriver().page.goto(url)
            await getDriver().page.waitForSelector('.code-view-toolbar .open-on-sourcegraph')

            // Pause to give codeintellify time to register listeners for
            // tokenization (only necessary in CI, not sure why).
            await getDriver().page.waitFor(1000)
            // Trigger tokenization of the line.
            const lineNumber = 16
            const line = await getDriver().page.waitForSelector(`${lineSelector}:nth-child(${lineNumber})`, {
                timeout: 10000,
            })
            const [token] = await line.$x('//span[text()="CallOption"]')
            await token.hover()
            await getDriver().page.waitForSelector('.test-tooltip-go-to-definition')
            await retry(async () => {
                assert.strictEqual(
                    await getDriver().page.evaluate(
                        () => document.querySelector<HTMLAnchorElement>('.test-tooltip-go-to-definition')?.href
                    ),
                    goToDefinitionURL ||
                        `${
                            getDriver().sourcegraphBaseUrl
                        }/${repoName}@4fb7cd90793ee6ab445f466b900e6bffb9b63d78/-/blob/call_opt.go#L5:6`
                )
            })
        })
    })
}

/**
 * Find a tab that contains the browser extension's after-install page (url
 * ending in `/after_install.html`) and, if found, close it.
 *
 * The after-install page is opened automatically when the browser extension is
 * installed. In tests, this means that it's opened automatically every time we
 * start the browser (with the browser extension loaded).
 */
export async function closeInstallPageTab(browser: puppeteer.Browser): Promise<void> {
    const pages = await browser.pages()
    const installPage = pages.find(page => page.url().endsWith('/after_install.html'))
    if (installPage) {
        await installPage.close()
    }
}
