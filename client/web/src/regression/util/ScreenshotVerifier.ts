import { Driver } from '../../../../shared/src/testing/driver'
import { BoundingBox } from 'puppeteer'

interface ExpectedScreenshot {
    screenshotFile: string
    description: string
}

/**
 * Utility class to verify screenshots match a particular description
 */
export class ScreenshotVerifier {
    public screenshots: ExpectedScreenshot[]
    constructor(public driver: Driver) {
        this.screenshots = []
    }

    public async verifyScreenshot({
        filename,
        description,
        clip,
    }: {
        /**
         * The filename to which to save the screenshot. It should be a decsriptive name that captures both
         * what should be happening in the screenshot and the context around what's happening. E.g.,
         * "progress-bar-after-initial-search-is-half-green.png"
         */
        filename: string

        /**
         * A short description of what should happen in the screenshot.
         */
        description: string
        clip?: BoundingBox
    }): Promise<void> {
        await this.driver.page.screenshot({
            path: filename,
            clip,
        })
        this.screenshots.push({ screenshotFile: filename, description })
    }

    public async verifySelector(
        filename: string,
        description: string,
        selector: string,
        waitForSelectorToBeVisibleTimeout: number = 0
    ): Promise<void> {
        if (waitForSelectorToBeVisibleTimeout > 0) {
            await this.driver.page.waitForFunction(
                selector => {
                    const element = document.querySelector<Element>(selector)
                    if (!element) {
                        return false
                    }
                    const { width, height } = element.getBoundingClientRect()
                    return width > 0 && height > 0
                },
                { timeout: waitForSelectorToBeVisibleTimeout },
                selector
            )
        }

        const clip: BoundingBox | undefined = await this.driver.page.evaluate(selector => {
            const element = document.querySelector<Element>(selector)
            if (!element) {
                throw new Error(`element with selector ${JSON.stringify(selector)} not found`)
            }
            const { left, top, width, height } = element.getBoundingClientRect()
            return { x: left, y: top, width, height }
        }, selector)
        await this.verifyScreenshot({
            filename,
            description,
            clip,
        })
    }

    /**
     * Returns instructions to manually verify each screenshot stored in the to-verify list.
     */
    public verificationInstructions(): string {
        return this.screenshots.length === 0
            ? ''
            : `

        @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
        @@@@ Manual verification steps required!!! @@@@
        @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@

        Please verify the following screenshots match the corresponding descriptions:

        ${this.screenshots
            .map(screenshot => `${screenshot.screenshotFile}:\t${JSON.stringify(screenshot.description)}`)
            .join('\n        ')}

        `
    }
}
