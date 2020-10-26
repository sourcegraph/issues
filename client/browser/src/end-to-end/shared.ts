import expect from 'expect'
import { Driver } from '../../../shared/src/testing/driver'
import { retry } from '../../../shared/src/testing/utils'
import assert from 'assert'

/**
 * Defines e2e tests for a single-file page of a code host.
 */
export function testSingleFilePage({
    getDriver,
    url,
    sourcegraphBaseUrl,
    repoName,
    lineSelector,
    goToDefinitionURL,
}: {
    /** Called to get the driver */
    getDriver: () => Driver

    /** The URL to sourcegraph/jsonrpc2 call_opt.go at commit 4fb7cd90793ee6ab445f466b900e6bffb9b63d78 on the code host */
    url: string

    /** The base URL of the sourcegraph instance */
    sourcegraphBaseUrl: string

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
            await getDriver().page.bringToFront()

            await getDriver().page.waitForSelector('.code-view-toolbar .open-on-sourcegraph', { timeout: 10000 })
            expect(await getDriver().page.$$('.code-view-toolbar .open-on-sourcegraph')).toHaveLength(1)
            await getDriver().page.click('.code-view-toolbar .open-on-sourcegraph')

            // The button opens a new tab, so get the new page whose opener is the current page, and get its url.
            const currentPageTarget = getDriver().page.target()
            const newTarget = await getDriver().browser.waitForTarget(target => target.opener() === currentPageTarget)
            const newPage = await newTarget.page()
            expect(newPage.url()).toBe(
                `${sourcegraphBaseUrl}/${repoName}@4fb7cd90793ee6ab445f466b900e6bffb9b63d78/-/blob/call_opt.go?utm_source=chrome-extension`
            )
            await newPage.close()
        })

        it('shows hover tooltips when hovering a token', async () => {
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
                        `${sourcegraphBaseUrl}/${repoName}@4fb7cd90793ee6ab445f466b900e6bffb9b63d78/-/blob/call_opt.go#L5:6`
                )
            })
        })
    })
}
