import assert from 'assert'

import { createDriverForTest, Driver } from '@sourcegraph/shared/src/testing/driver'
import { emptyResponse } from '@sourcegraph/shared/src/testing/integration/graphQlResults'
import { afterEachSaveScreenshotIfFailed } from '@sourcegraph/shared/src/testing/screenshotReporter'

import { createWebIntegrationTestContext, WebIntegrationTestContext } from '../context'

import { INSIGHT_VIEW_TEAM_SIZE, INSIGHT_VIEW_TYPES_MIGRATION } from './utils/insight-mock-data'
import { overrideGraphQLExtensions } from './utils/override-graphql-with-extensions'

describe('Code insights page', () => {
    let driver: Driver
    let testContext: WebIntegrationTestContext

    before(async () => {
        driver = await createDriverForTest()
    })

    after(() => driver?.close())

    beforeEach(async function () {
        testContext = await createWebIntegrationTestContext({
            driver,
            currentTest: this.currentTest!,
            directory: __dirname,
        })
    })

    afterEachSaveScreenshotIfFailed(() => driver.page)
    afterEach(() => testContext?.dispose())

    it('should update user/org settings if insight delete happened', async () => {
        const settings = {
            'searchInsights.insight.graphQLTypesMigration': {},
            'searchInsights.insight.teamSize': {},
        }

        overrideGraphQLExtensions({
            testContext,

            /**
             * Since search insight and code stats insight are working via user/org
             * settings. We have to mock them by mocking user settings and provide
             * mock data - mocking extension work.
             * */
            userSettings: settings,
            insightExtensionsMocks: {
                'searchInsights.insight.graphQLTypesMigration': {
                    ...INSIGHT_VIEW_TYPES_MIGRATION,
                    title: 'Migration to new GraphQL TS types',
                },
                'searchInsights.insight.teamSize': INSIGHT_VIEW_TEAM_SIZE,
            },
            overrides: {
                /**
                 * Mock back-end insights with standard gql API handler.
                 * */
                Insights: () => ({ insights: { nodes: [] } }),
                OverwriteSettings: () => ({
                    settingsMutation: {
                        overwriteSettings: {
                            empty: emptyResponse,
                        },
                    },
                }),

                SubjectSettings: () => ({
                    settingsSubject: {
                        latestSettings: {
                            id: 310,
                            contents: JSON.stringify(settings),
                        },
                    },
                }),
            },
        })

        await driver.page.goto(driver.sourcegraphBaseUrl + '/insights')
        await driver.page.waitForSelector('[data-testid="line-chart__content"] svg circle')

        const variables = await testContext.waitForGraphQLRequest(async () => {
            await driver.page.click(
                '[data-testid="insight-card.searchInsights.insight.graphQLTypesMigration"] [data-testid="InsightContextMenuButton"]'
            )
            await driver.page.click('[data-testid="insight-context-menu-delete-button"]')
        }, 'OverwriteSettings')

        assert.deepStrictEqual(JSON.parse(variables.contents), {
            'searchInsights.insight.teamSize': {},
        })
    })
})
