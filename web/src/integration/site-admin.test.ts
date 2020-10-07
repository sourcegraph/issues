import assert from 'assert'
import { createDriverForTest, Driver } from '../../../shared/src/testing/driver'
import { createWebIntegrationTestContext, WebIntegrationTestContext } from './context'
import { saveScreenshotsUponFailures } from '../../../shared/src/testing/screenshotReporter'
import { commonWebGraphQlResults } from './graphQlResults'
import { ExternalServiceKind } from '../graphql-operations'

describe('Site-admin', () => {
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
    saveScreenshotsUponFailures(() => driver.page)
    afterEach(() => testContext?.dispose())

    describe('Manage repositories page', () => {
        it('redirects to repository status page after updating', async () => {
            testContext.overrideGraphQL({
                ...commonWebGraphQlResults,
                ExternalServices: () => ({
                    externalServices: {
                        totalCount: 1,
                        nodes: [
                            {
                                id: 'RXh0ZXJuYWxTZXJ2aWNlOjQ=',
                                kind: ExternalServiceKind.GITLAB,
                                displayName: 'GITLAB #1',
                                config:
                                    '{ "projectQuery": [   "?membership=true" ], "token": "Qa_YqabmAm5DvbdWZ3rm", "url": "https://gitlab.com"}',
                            },
                        ],
                        pageInfo: { hasNextPage: false },
                    },
                }),
                Repositories: () => ({
                    repositories: {
                        nodes: [
                            {
                                id: 'UmVwb3NpdG9yeToxNDI=',
                                name: 'github.com/sourcegraph/sourcegraph',
                                createdAt: '2020-08-04T17:33:45-04:00',
                                viewerCanAdminister: true,
                                url: '/github.com/sourcegraph/sourcegraph',
                                mirrorInfo: {
                                    cloned: true,
                                    cloneInProgress: false,
                                    updatedAt: '2020-08-04T17:32:23-04:00',
                                },
                            },
                            {
                                id: 'UmVwb3NpdG9yeToxMzc=',
                                name: 'gitlab.com/beyang/thyme',
                                createdAt: '2020-08-04T17:33:45-04:00',
                                viewerCanAdminister: true,
                                url: '/gitlab.com/beyang/thyme',
                                mirrorInfo: {
                                    cloned: true,
                                    cloneInProgress: false,
                                    updatedAt: '2020-08-04T17:32:21-04:00',
                                },
                            },
                        ],
                        totalCount: 2,
                        pageInfo: {
                            hasNextPage: false,
                        },
                    },
                }),
                addExternalService: () => ({
                    addExternalService: {
                        displayName: 'GitLab',
                        id: 'RXh0ZXJuYWxTZXJ2aWNlOjIw',
                        kind: ExternalServiceKind.GITLAB,
                        warning: null,
                    },
                }),
            })

            await driver.page.goto(driver.sourcegraphBaseUrl + '/site-admin/external-services')

            await driver.findElementWithText('Add repositories', { action: 'click', wait: true })

            await driver.page.waitForSelector('[data-test-external-service-card-link="GITLAB"]')
            await driver.page.click('[data-test-external-service-card-link="GITLAB"]')

            const newSettings = JSON.stringify({
                url: 'https://gitlab.com',
                token: '<access token>',
                projectQuery: ['projects?membership=true&archived=no'],
            })

            await driver.replaceText({
                selector: '.site-admin-area .monaco-editor .view-lines',
                newText: newSettings,
                selectMethod: 'keyboard',
                enterTextMethod: 'type',
            })

            const addRepositoryLink = await driver.page.waitForSelector('.test-add-external-service-button')
            await addRepositoryLink.click()

            // assert that the user sees a banner that lets them know that repositories are being updated
            await driver.page.waitForSelector('.test-updating-repositories-alert')
            assert(
                await driver.page.evaluate(() =>
                    document
                        .querySelector<HTMLParagraphElement>('.test-updating-repositories-alert')
                        ?.textContent?.includes('Updating repositories')
                )
            )

            await driver.page.screenshot({ path: '../img.jpg', type: 'jpeg' })

            assert(true)
        })
    })
})
