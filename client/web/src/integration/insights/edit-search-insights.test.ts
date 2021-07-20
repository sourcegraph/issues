import assert from 'assert'

import { createDriverForTest, Driver } from '@sourcegraph/shared/src/testing/driver'
import { emptyResponse } from '@sourcegraph/shared/src/testing/integration/graphQlResults'
import { afterEachSaveScreenshotIfFailed } from '@sourcegraph/shared/src/testing/screenshotReporter'

import { createWebIntegrationTestContext, WebIntegrationTestContext } from '../context'
import { percySnapshotWithVariants } from '../utils'

import {
    INSIGHT_TYPES_MIGRATION_BULK_SEARCH,
    INSIGHT_TYPES_MIGRATION_COMMITS,
    INSIGHT_VIEW_TEAM_SIZE,
    INSIGHT_VIEW_TYPES_MIGRATION,
} from './utils/insight-mock-data'
import { overrideGraphQLExtensions } from './utils/override-graphql-with-extensions'

interface InsightValues {
    series: {
        name?: string | null
        query?: string | null
        stroke?: string | null
    }[]
    repositories?: string
    title?: string
    visibility?: string
    step: Record<string, string | number | undefined>
}

function getInsightFormValues(): InsightValues {
    const repositories = document.querySelector<HTMLInputElement>('[name="repositories"]')?.value
    const title = document.querySelector<HTMLInputElement>('input[name="title"]')?.value
    const visibility = document.querySelector<HTMLInputElement>('input[name="visibility"]:checked')?.value
    const granularityType = document.querySelector<HTMLInputElement>('input[name="step"]:checked')?.value ?? ''
    const granularityValue = document.querySelector<HTMLInputElement>('input[name="stepValue"]')?.value

    const series = [...document.querySelectorAll('[data-testid="form-series"] [data-testid="series-card"]')].map(
        card => ({
            name: card.querySelector('[data-testid="series-name"]')?.textContent,
            query: card.querySelector('[data-testid="series-query"]')?.textContent,
            stroke: card.querySelector<HTMLElement>('[data-testid="series-color-mark"]')?.style.color,
        })
    )

    return {
        series,
        repositories,
        title,
        visibility,
        step: { [granularityType]: granularityValue },
    }
}

describe('Code insight edit insight page', () => {
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

    async function clearAndType(driver: Driver, selector: string, value: string): Promise<void> {
        await driver.page.click(selector, { clickCount: 3 })
        await driver.page.type(selector, value)
    }

    it('should update user/org settings if insight has been updated', async () => {
        const userSettings = {
            'searchInsights.insight.teamSize': {},
            'searchInsights.insight.graphQLTypesMigration': {
                title: 'Migration to new GraphQL TS types',
                repositories: ['github.com/sourcegraph/sourcegraph'],
                series: [
                    {
                        name: 'Imports of old GQL.* types',
                        query: 'patternType:regex case:yes \\*\\sas\\sGQL',
                        stroke: 'var(--oc-red-7)',
                    },
                    {
                        name: 'Imports of new graphql-operations types',
                        query: "patternType:regexp case:yes /graphql-operations'",
                        stroke: 'var(--oc-blue-7)',
                    },
                ],
                step: {
                    weeks: 6,
                },
            },
        }

        const orgSettings = {
            extensions: {},
            'searchInsights.insight.orgTeamSize': {},
        }

        // Mock `Date.now` to stabilize timestamps
        await driver.page.evaluateOnNewDocument(() => {
            // Number of ms between Unix epoch and June 31, 2021
            const mockMs = new Date('June 1, 2021 00:00:00 UTC').getTime()
            Date.now = () => mockMs
        })

        overrideGraphQLExtensions({
            testContext,
            /**
             * Since search insight and code stats insight are working via user/org
             * settings. We have to mock them by mocking user settings and provide
             * mock data - mocking extension work.
             * */
            userSettings,
            insightExtensionsMocks: {
                'searchInsights.insight.graphQLTypesMigration': INSIGHT_VIEW_TYPES_MIGRATION,
                'searchInsights.insight.TestInsightTitle': {
                    ...INSIGHT_VIEW_TYPES_MIGRATION,
                    title: 'Test insight',
                },
                'searchInsights.insight.teamSize': INSIGHT_VIEW_TEAM_SIZE,
                'searchInsights.insight.orgTeamSize': INSIGHT_VIEW_TEAM_SIZE,
            },
            overrides: {
                OverwriteSettings: () => ({
                    settingsMutation: {
                        overwriteSettings: {
                            empty: emptyResponse,
                        },
                    },
                }),

                SubjectSettings: ({ id }) => {
                    if (id === 'TestUserID') {
                        return {
                            settingsSubject: {
                                latestSettings: {
                                    id: 310,
                                    contents: JSON.stringify(userSettings),
                                },
                            },
                        }
                    }

                    if (id === 'Org_test_id') {
                        return {
                            settingsSubject: {
                                latestSettings: {
                                    id: 320,
                                    contents: JSON.stringify(orgSettings),
                                },
                            },
                        }
                    }

                    return {
                        settingsSubject: {
                            latestSettings: {
                                id: 100,
                                contents: '{ "a": "b" }',
                            },
                        },
                    }
                },

                /**
                 * Mock for async repositories field validation.
                 * */
                BulkRepositoriesSearch: () => ({
                    repoSearch0: { name: 'github.com/sourcegraph/sourcegraph' },
                    repoSearch1: { name: 'github.com/sourcegraph/about' },
                }),

                /**
                 * Mocks of commits searching and data search itself for live preview chart
                 * */
                BulkSearchCommits: () => INSIGHT_TYPES_MIGRATION_COMMITS,
                BulkSearch: () => INSIGHT_TYPES_MIGRATION_BULK_SEARCH,

                /** Mock for repository suggest component. */
                RepositorySearchSuggestions: () => ({
                    repositories: { nodes: [] },
                }),
            },
        })

        await driver.page.goto(
            driver.sourcegraphBaseUrl + '/insights/edit/searchInsights.insight.graphQLTypesMigration'
        )

        // Waiting for all important part of creation form will be rendered.
        await driver.page.waitForSelector('[data-testid="line-chart__content"] svg circle')

        // Edit all insight form fields ↓

        // Move user cursor at the end of the input
        await driver.page.click('[name="repositories"]', { clickCount: 3 })
        await driver.page.keyboard.press('ArrowRight')

        // Add new repo to repositories field
        await driver.page.keyboard.type(', github.com/sourcegraph/about')

        // Change insight title
        await clearAndType(driver, 'input[name="title"]', 'Test insight title')

        // Edit first insight series
        await driver.page.click(
            '[data-testid="form-series"] [data-testid="series-card"]:nth-child(1) [data-testid="series-edit-button"]'
        )
        await clearAndType(
            driver,
            '[data-testid="series-form"]:nth-child(1) input[name="seriesName"]',
            'test edited series title'
        )
        await clearAndType(
            driver,
            '[data-testid="series-form"]:nth-child(1) input[name="seriesQuery"]',
            'test edited series query'
        )
        await driver.page.click('[data-testid="series-form"]:nth-child(1) label[title="Cyan"]')

        // Remove second insight series
        await driver.page.click(
            '[data-testid="form-series"] [data-testid="series-card"] [data-testid="series-delete-button"]'
        )

        // Add new series
        await driver.page.click('[data-testid="form-series"] [data-testid="add-series-button"]')
        await clearAndType(
            driver,
            '[data-testid="series-form"]:nth-child(2) input[name="seriesName"]',
            'new test series title'
        )
        await clearAndType(
            driver,
            '[data-testid="series-form"]:nth-child(2) input[name="seriesQuery"]',
            'new test series query'
        )

        // Change visibility to test org by org ID mock - 'Org_test_id'
        await driver.page.click('input[name="visibility"][value="Org_test_id"]')

        // Change insight Granularity
        await driver.page.type('input[name="stepValue"]', '2')
        await driver.page.click('input[name="step"][value="days"]')

        const deleteFromUserConfigRequest = await testContext.waitForGraphQLRequest(async () => {
            await driver.page.click('[data-testid="insight-save-button"]')
        }, 'OverwriteSettings')

        const addToOrgConfigRequest = await testContext.waitForGraphQLRequest(() => {}, 'OverwriteSettings')

        // Check that old user settings config doesn't have edited insight
        assert.deepStrictEqual(JSON.parse(deleteFromUserConfigRequest.contents), {
            'searchInsights.insight.teamSize': {},
        })

        // Check that new org settings config has edited insight
        assert.deepStrictEqual(JSON.parse(addToOrgConfigRequest.contents), {
            extensions: {
                'sourcegraph/search-insights': true,
            },
            'searchInsights.insight.orgTeamSize': {},
            'searchInsights.insight.testInsightTitle': {
                title: 'Test insight title',
                repositories: ['github.com/sourcegraph/sourcegraph', 'github.com/sourcegraph/about'],
                series: [
                    {
                        name: 'test edited series title',
                        query: 'test edited series query',
                        stroke: 'var(--oc-cyan-7)',
                    },
                    {
                        name: 'new test series title',
                        query: 'new test series query',
                        stroke: 'var(--oc-grape-7)',
                    },
                ],
                step: {
                    days: 62,
                },
            },
        })
    })

    it('should open the edit page with pre-filled fields with values from user/org settings', async () => {
        const settings = {
            'searchInsights.insight.graphQLTypesMigration': {
                title: 'Migration to new GraphQL TS types',
                repositories: ['github.com/sourcegraph/sourcegraph'],
                series: [
                    {
                        name: 'Imports of old GQL.* types',
                        query: 'patternType:regex case:yes \\*\\sas\\sGQL',
                        stroke: 'var(--oc-red-7)',
                    },
                    {
                        name: 'Imports of new graphql-operations types',
                        query: "patternType:regexp case:yes /graphql-operations'",
                        stroke: 'var(--oc-blue-7)',
                    },
                ],
                step: {
                    weeks: 6,
                },
            },
        }

        // Mock `Date.now` to stabilize timestamps
        await driver.page.evaluateOnNewDocument(() => {
            // Number of ms between Unix epoch and June 31, 2021
            const mockMs = new Date('June 1, 2021 00:00:00 UTC').getTime()
            Date.now = () => mockMs
        })

        overrideGraphQLExtensions({
            testContext,

            /**
             * Since search insight and code stats insight are working via user/org
             * settings. We have to mock them by mocking user settings and provide
             * mock data - mocking extension work.
             * */
            userSettings: settings,
            insightExtensionsMocks: {
                'searchInsights.insight.graphQLTypesMigration': INSIGHT_VIEW_TYPES_MIGRATION,
                'searchInsights.insight.teamSize': INSIGHT_VIEW_TEAM_SIZE,
            },
            overrides: {
                /**
                 * Mock for async repositories field validation.
                 * */
                BulkRepositoriesSearch: () => ({
                    repoSearch0: { name: 'github.com/sourcegraph/sourcegraph' },
                }),

                /**
                 * Mocks of commits searching and data search itself for live preview chart
                 * */
                BulkSearchCommits: () => INSIGHT_TYPES_MIGRATION_COMMITS,
                BulkSearch: () => INSIGHT_TYPES_MIGRATION_BULK_SEARCH,

                /** Mock for repository suggest component. */
                RepositorySearchSuggestions: () => ({
                    repositories: { nodes: [] },
                }),
            },
        })

        await driver.page.goto(driver.sourcegraphBaseUrl + '/insights')
        await driver.page.waitForSelector('[data-testid="line-chart__content"] svg circle')

        // Click on edit button of insight context menu (three dots-menu)
        await driver.page.click(
            '[data-testid="insight-card.searchInsights.insight.graphQLTypesMigration"] [data-testid="InsightContextMenuButton"]'
        )
        await driver.page.click(
            '[data-testid="context-menu.searchInsights.insight.graphQLTypesMigration"] [data-testid="InsightContextMenuEditLink"]'
        )

        // Check redirect URL for edit insight page
        assert.strictEqual(
            driver.page.url().endsWith('/insights/edit/searchInsights.insight.graphQLTypesMigration'),
            true
        )

        // Waiting for all important part of creation form will be rendered.
        await driver.page.waitForSelector('[data-testid="search-insight-edit-page-content"]')
        await driver.page.waitForSelector('[data-testid="line-chart__content"] svg circle')

        await percySnapshotWithVariants(driver.page, 'Code insights edit page with search-based insight creation UI')

        // Gather all filled inputs within a creation UI form.
        const grabbedInsightInfo = await driver.page.evaluate(getInsightFormValues)

        assert.deepStrictEqual(grabbedInsightInfo, {
            title: 'Migration to new GraphQL TS types',
            repositories: 'github.com/sourcegraph/sourcegraph',
            visibility: 'TestUserID',
            series: [
                {
                    name: 'Imports of old GQL.* types',
                    query: 'patternType:regex case:yes \\*\\sas\\sGQL',
                    stroke: 'var(--oc-red-7)',
                },
                {
                    name: 'Imports of new graphql-operations types',
                    query: "patternType:regexp case:yes /graphql-operations'",
                    stroke: 'var(--oc-blue-7)',
                },
            ],
            step: {
                weeks: '6',
            },
        })
    })
})
