import { mkdir } from 'mz/fs'
import * as path from 'path'
import * as puppeteer from 'puppeteer'
import { afterEach } from 'mocha'

/**
 * Registers an `afterEach` hook (for use with Mocha) that takes a screenshot of
 * the browser when a test fails. It is used by e2e and integration tests.
 */
export function afterEachSaveScreenshotIfFailed(getPage: () => puppeteer.Page): void {
    afterEach('Save screenshot', async function () {
        if (this.currentTest && this.currentTest.state === 'failed') {
            await takeScreenshot({
                page: getPage(),
                repoRootDir: path.resolve(__dirname, '..', '..', '..', '..'),
                screenshotDir: path.resolve(__dirname, '..', '..', '..', '..', 'puppeteer'),
                testName: this.currentTest.fullTitle(),
            })
        }
    })
}

async function takeScreenshot({
    page,
    repoRootDir,
    screenshotDir,
    testName,
}: {
    page: puppeteer.Page
    repoRootDir: string
    screenshotDir: string
    testName: string
}): Promise<void> {
    await mkdir(screenshotDir, { recursive: true })
    const fileName = testName.replace(/\W/g, '_') + '.png'
    const filePath = path.join(screenshotDir, fileName)
    const screenshot = await page.screenshot({ path: filePath })
    if (process.env.CI) {
        // Print image with ANSI escape code for Buildkite: https://buildkite.com/docs/builds/images-in-log-output.
        console.log(`\u001B]1338;url="artifact://${path.relative(repoRootDir, filePath)}";alt="Screenshot"\u0007`)
    } else if (process.env.TERM_PROGRAM === 'iTerm.app') {
        // Print image inline for iTerm2
        const nameBase64 = Buffer.from(fileName).toString('base64')
        console.log(`\u001B]1337;File=name=${nameBase64};inline=1;width=500px:${screenshot.toString('base64')}\u0007`)
    } else {
        console.log(`📸  Saved screenshot of failure to ${path.relative(process.cwd(), filePath)}`)
    }
}
