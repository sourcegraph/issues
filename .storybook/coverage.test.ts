import initStoryshots from '@storybook/addon-storyshots'
import { puppeteerTest } from '@storybook/addon-storyshots-puppeteer'
import * as path from 'path'
import { pathToFileURL } from 'url'
import { recordCoverage } from '../client/shared/src/testing/coverage'

// This test suite does not actually test anything.
// It just loads up the storybook in Puppeteer and records its coverage,
// so it can be tracked in Codecov.

initStoryshots({
    configPath: __dirname,
    suite: 'Storybook',
    test: puppeteerTest({
        storybookUrl: pathToFileURL(path.resolve(__dirname, '../storybook-static')).href,
        testBody: async page => {
            await recordCoverage(page.browser())
        },
    }),
})
