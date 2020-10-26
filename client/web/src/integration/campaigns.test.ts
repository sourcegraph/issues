import assert from 'assert'
import { createDriverForTest, Driver } from '../../../shared/src/testing/driver'
import { commonWebGraphQlResults } from './graphQlResults'
import { createWebIntegrationTestContext, WebIntegrationTestContext } from './context'
import { afterEachSaveScreenshotIfFailed } from '../../../shared/src/testing/screenshotReporter'
import { subDays, addDays } from 'date-fns'
import {
    ChangesetCheckState,
    ChangesetExternalState,
    ChangesetPublicationState,
    ChangesetReconcilerState,
    ChangesetReviewState,
    ChangesetCountsOverTimeVariables,
    ChangesetCountsOverTimeResult,
    ExternalChangesetFileDiffsVariables,
    ExternalChangesetFileDiffsResult,
    CampaignChangesetsVariables,
    CampaignChangesetsResult,
    WebGraphQlOperations,
    ExternalChangesetFileDiffsFields,
    DiffHunkLineType,
    ChangesetSpecType,
    ListCampaign,
    CampaignByNamespaceResult,
} from '../graphql-operations'
import { SharedGraphQlOperations } from '../../../shared/src/graphql-operations'

const campaignListNode: ListCampaign = {
    id: 'campaign123',
    url: '/users/alice/campaigns/test-campaign',
    name: 'test-campaign',
    createdAt: subDays(new Date(), 5).toISOString(),
    changesets: { stats: { closed: 4, merged: 10, open: 5 } },
    closedAt: null,
    description: null,
    namespace: {
        namespaceName: 'alice',
        url: '/users/alice',
    },
}

const mockDiff: NonNullable<ExternalChangesetFileDiffsFields['diff']> = {
    __typename: 'RepositoryComparison',
    fileDiffs: {
        nodes: [
            {
                internalID: 'intid123',
                oldPath: '/somefile.md',
                newPath: '/somefile.md',
                oldFile: {
                    __typename: 'GitBlob',
                    binary: false,
                    byteSize: 0,
                },
                newFile: {
                    __typename: 'GitBlob',
                    binary: false,
                    byteSize: 0,
                },
                mostRelevantFile: {
                    __typename: 'GitBlob',
                    url: 'http://test.test/fileurl',
                },
                hunks: [
                    {
                        section: "@@ -70,33 +81,154 @@ super('awesome', () => {",
                        oldRange: {
                            startLine: 70,
                            lines: 33,
                        },
                        newRange: {
                            startLine: 81,
                            lines: 154,
                        },
                        oldNoNewlineAt: false,
                        highlight: {
                            aborted: false,
                            lines: [
                                {
                                    html: 'some fiel content',
                                    kind: DiffHunkLineType.DELETED,
                                },
                                {
                                    html: 'some file content',
                                    kind: DiffHunkLineType.ADDED,
                                },
                            ],
                        },
                    },
                ],
                stat: {
                    added: 10,
                    changed: 3,
                    deleted: 8,
                },
            },
        ],
        pageInfo: {
            endCursor: null,
            hasNextPage: false,
        },
        totalCount: 1,
    },
    range: {
        base: {
            __typename: 'GitRef',
            target: {
                oid: 'abc123base',
            },
        },
        head: {
            __typename: 'GitRef',
            target: {
                oid: 'abc123head',
            },
        },
    },
}

const ChangesetCountsOverTime: (variables: ChangesetCountsOverTimeVariables) => ChangesetCountsOverTimeResult = () => ({
    node: {
        __typename: 'Campaign',
        changesetCountsOverTime: [
            {
                closed: 12,
                date: subDays(new Date(), 2).toISOString(),
                merged: 10,
                openApproved: 3,
                openChangesRequested: 1,
                openPending: 81,
                total: 130,
                draft: 10,
            },
            {
                closed: 12,
                date: subDays(new Date(), 1).toISOString(),
                merged: 10,
                openApproved: 23,
                openChangesRequested: 1,
                openPending: 66,
                total: 130,
                draft: 5,
            },
        ],
    },
})

const ExternalChangesetFileDiffs: (
    variables: ExternalChangesetFileDiffsVariables
) => ExternalChangesetFileDiffsResult = () => ({
    node: {
        __typename: 'ExternalChangeset',
        diff: mockDiff,
    },
})

const CampaignChangesets: (variables: CampaignChangesetsVariables) => CampaignChangesetsResult = () => ({
    node: {
        __typename: 'Campaign',
        changesets: {
            totalCount: 1,
            pageInfo: {
                endCursor: null,
                hasNextPage: false,
            },
            nodes: [
                {
                    __typename: 'ExternalChangeset',
                    body: 'body123',
                    checkState: ChangesetCheckState.PASSED,
                    createdAt: subDays(new Date(), 5).toISOString(),
                    updatedAt: subDays(new Date(), 5).toISOString(),
                    diffStat: {
                        added: 100,
                        changed: 10,
                        deleted: 23,
                    },
                    error: null,
                    externalID: '123',
                    externalState: ChangesetExternalState.OPEN,
                    externalURL: {
                        url: 'http://test.test/123',
                    },
                    id: 'changeset123',
                    labels: [
                        {
                            color: '93ba13',
                            description: null,
                            text: 'Abc label',
                        },
                    ],
                    nextSyncAt: null,
                    publicationState: ChangesetPublicationState.PUBLISHED,
                    reconcilerState: ChangesetReconcilerState.COMPLETED,
                    repository: {
                        id: 'repo123',
                        name: 'github.com/sourcegraph/repo',
                        url: 'http://test.test/repo',
                    },
                    reviewState: ChangesetReviewState.APPROVED,
                    title: 'The changeset title',
                    currentSpec: { id: 'spec-rand-id-1' },
                },
            ],
        },
    },
})

function mockCommonGraphQLResponses(
    entityType: 'user' | 'org',
    campaignOverrides?: Partial<NonNullable<CampaignByNamespaceResult['campaign']>>
): Partial<WebGraphQlOperations & SharedGraphQlOperations> {
    const namespaceURL = entityType === 'user' ? '/users/alice' : '/organizations/test-org'
    return {
        Organization: () => ({
            organization: {
                __typename: 'Org',
                createdAt: '2020-08-07T00:00',
                displayName: 'test-org',
                settingsURL: `${namespaceURL}/settings`,
                id: 'TestOrg',
                name: 'test-org',
                url: namespaceURL,
                viewerCanAdminister: true,
                viewerIsMember: false,
                viewerPendingInvitation: null,
            },
        }),
        UserArea: () => ({
            user: {
                __typename: 'User',
                id: 'user123',
                username: 'alice',
                displayName: 'alice',
                url: namespaceURL,
                settingsURL: `${namespaceURL}/settings`,
                avatarURL: '',
                viewerCanAdminister: true,
                viewerCanChangeUsername: true,
                siteAdmin: true,
                builtinAuth: true,
                createdAt: '2020-04-10T21:11:42Z',
                emails: [{ email: 'alice@example.com', verified: true }],
                organizations: { nodes: [] },
                permissionsInfo: null,
                tags: [],
            },
        }),
        CampaignByNamespace: () => ({
            campaign: {
                __typename: 'Campaign',
                id: 'campaign123',
                changesets: {
                    stats: { closed: 2, deleted: 1, merged: 3, open: 8, total: 19, unpublished: 3, draft: 2 },
                },
                closedAt: null,
                createdAt: subDays(new Date(), 5).toISOString(),
                updatedAt: subDays(new Date(), 5).toISOString(),
                description: '### Very cool campaign',
                initialApplier: {
                    url: '/users/alice',
                    username: 'alice',
                },
                name: 'test-campaign',
                namespace: {
                    namespaceName: entityType === 'user' ? 'alice' : 'test-org',
                    url: namespaceURL,
                },
                url: `${namespaceURL}/campaigns/test-campaign`,
                diffStat: {
                    added: 1000,
                    changed: 29,
                    deleted: 817,
                },
                viewerCanAdminister: true,
                lastAppliedAt: subDays(new Date(), 5).toISOString(),
                lastApplier: {
                    url: '/users/bob',
                    username: 'bob',
                },
                currentSpec: {
                    originalInput: 'name: awesome-campaign\ndescription: somesttring',
                },
                ...campaignOverrides,
            },
        }),
        CampaignsByUser: () => ({
            node: {
                __typename: 'User',
                campaigns: {
                    nodes: [campaignListNode],
                    pageInfo: {
                        endCursor: null,
                        hasNextPage: false,
                    },
                    totalCount: 1,
                },
            },
        }),
        CampaignsByOrg: () => ({
            node: {
                __typename: 'Org',
                campaigns: {
                    nodes: [
                        {
                            ...campaignListNode,
                            url: '/organizations/test-org/campaigns/test-campaign',
                            namespace: {
                                namespaceName: 'test-org',
                                url: '/organizations/test-org',
                            },
                        },
                    ],
                    pageInfo: {
                        endCursor: null,
                        hasNextPage: false,
                    },
                    totalCount: 1,
                },
            },
        }),
    }
}

describe('Campaigns', () => {
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

    describe('Campaigns list', () => {
        it('lists global campaigns', async () => {
            testContext.overrideGraphQL({
                ...commonWebGraphQlResults,
                Campaigns: () => ({
                    campaigns: {
                        nodes: [campaignListNode],
                        pageInfo: {
                            endCursor: null,
                            hasNextPage: false,
                        },
                        totalCount: 1,
                    },
                }),
            })
            await driver.page.goto(driver.sourcegraphBaseUrl + '/campaigns')
            await driver.page.waitForSelector('.test-campaign-list-page')
            await driver.page.waitForSelector('.test-campaign-namespace-link')
            await driver.page.waitForSelector('.test-campaign-link')
            assert.strictEqual(
                await driver.page.evaluate(
                    () => document.querySelector<HTMLAnchorElement>('.test-campaign-namespace-link')?.href
                ),
                testContext.driver.sourcegraphBaseUrl + '/users/alice/campaigns'
            )
            assert.strictEqual(
                await driver.page.evaluate(
                    () => document.querySelector<HTMLAnchorElement>('.test-campaign-link')?.href
                ),
                testContext.driver.sourcegraphBaseUrl + '/users/alice/campaigns/test-campaign'
            )
        })

        it('lists user campaigns', async () => {
            testContext.overrideGraphQL({
                ...commonWebGraphQlResults,
                ...mockCommonGraphQLResponses('user'),
            })
            await driver.page.goto(driver.sourcegraphBaseUrl + '/users/alice/campaigns')

            await driver.page.waitForSelector('.test-campaign-list-page')
            await driver.page.waitForSelector('.test-campaign-link')
            assert.strictEqual(
                await driver.page.evaluate(
                    () => document.querySelector<HTMLAnchorElement>('.test-campaign-link')?.href
                ),
                testContext.driver.sourcegraphBaseUrl + '/users/alice/campaigns/test-campaign'
            )
            assert.strictEqual(await driver.page.$('.test-campaign-namespace-link'), null)
        })

        it('lists org campaigns', async () => {
            testContext.overrideGraphQL({
                ...commonWebGraphQlResults,
                ...mockCommonGraphQLResponses('org'),
            })
            await driver.page.goto(driver.sourcegraphBaseUrl + '/campaigns')

            await driver.page.goto(driver.sourcegraphBaseUrl + '/organizations/test-org/campaigns')
            await driver.page.waitForSelector('.test-campaign-list-page')
            await driver.page.waitForSelector('.test-campaign-link')
            assert.strictEqual(
                await driver.page.evaluate(
                    () => document.querySelector<HTMLAnchorElement>('.test-campaign-link')?.href
                ),
                testContext.driver.sourcegraphBaseUrl + '/organizations/test-org/campaigns/test-campaign'
            )
            assert.strictEqual(await driver.page.$('.test-campaign-namespace-link'), null)
        })
    })

    describe('Campaign details', () => {
        for (const entityType of ['user', 'org'] as const) {
            it(`displays a single campaign for ${entityType}`, async () => {
                testContext.overrideGraphQL({
                    ...commonWebGraphQlResults,
                    ...mockCommonGraphQLResponses(entityType),
                    CampaignChangesets,
                    ChangesetCountsOverTime,
                    ExternalChangesetFileDiffs,
                })
                const namespaceURL = entityType === 'user' ? '/users/alice' : '/organizations/test-org'

                await driver.page.goto(driver.sourcegraphBaseUrl + namespaceURL + '/campaigns/test-campaign')
                // View overview page.
                await driver.page.waitForSelector('.test-campaign-details-page')

                // Expand one changeset.
                await driver.page.click('.test-campaigns-expand-changeset')
                // Expect one diff to be rendered.
                await driver.page.waitForSelector('.test-file-diff-node')

                // Switch to view burndown chart.
                await driver.page.click('.test-campaigns-chart-tab')
                await driver.page.waitForSelector('.test-campaigns-chart')

                // Switch to view spec file.
                await driver.page.click('.test-campaigns-spec-tab')
                await driver.page.waitForSelector('.test-campaigns-spec')

                // Go to close page via button.
                await Promise.all([driver.page.waitForNavigation(), driver.page.click('.test-campaigns-close-btn')])
                assert.strictEqual(
                    await driver.page.evaluate(() => window.location.href),
                    testContext.driver.sourcegraphBaseUrl + namespaceURL + '/campaigns/test-campaign/close'
                )
                await driver.page.waitForSelector('.test-campaign-close-page')
                // Change overrides to make campaign appear closed.
                testContext.overrideGraphQL({
                    ...commonWebGraphQlResults,
                    ...mockCommonGraphQLResponses(entityType, { closedAt: subDays(new Date(), 1).toISOString() }),
                    CampaignChangesets,
                    ChangesetCountsOverTime,
                    ExternalChangesetFileDiffs,
                    DeleteCampaign: () => ({
                        deleteCampaign: {
                            alwaysNil: null,
                        },
                    }),
                })

                // Return to details page.
                await Promise.all([
                    driver.page.waitForNavigation(),
                    driver.page.click('.test-campaigns-close-abort-btn'),
                ])
                await driver.page.waitForSelector('.test-campaign-details-page')
                assert.strictEqual(
                    await driver.page.evaluate(() => window.location.href),
                    testContext.driver.sourcegraphBaseUrl + namespaceURL + '/campaigns/test-campaign'
                )

                // Delete the closed campaign.
                await Promise.all([
                    driver.page.waitForNavigation(),
                    driver.acceptNextDialog(),
                    driver.page.click('.test-campaigns-delete-btn'),
                ])
                assert.strictEqual(
                    await driver.page.evaluate(() => window.location.href),
                    testContext.driver.sourcegraphBaseUrl + namespaceURL + '/campaigns'
                )

                // Test read tab from location.
                await driver.page.goto(driver.sourcegraphBaseUrl + namespaceURL + '/campaigns/test-campaign?tab=chart')
                await driver.page.waitForSelector('.test-campaigns-chart')
                await driver.page.goto(driver.sourcegraphBaseUrl + namespaceURL + '/campaigns/test-campaign?tab=spec')
                await driver.page.waitForSelector('.test-campaigns-spec')
            })
        }
    })

    describe('Campaign spec preview', () => {
        for (const entityType of ['user', 'org'] as const) {
            it(`displays a preview of a campaign spec in ${entityType} namespace`, async () => {
                const namespaceURL = entityType === 'user' ? '/users/alice' : '/organizations/test-org'
                testContext.overrideGraphQL({
                    ...commonWebGraphQlResults,
                    ...mockCommonGraphQLResponses(entityType),
                    CampaignSpecByID: () => ({
                        node: {
                            __typename: 'CampaignSpec',
                            id: 'spec123',
                            appliesToCampaign: null,
                            createdAt: subDays(new Date(), 2).toISOString(),
                            creator: {
                                username: 'alice',
                                url: '/users/alice',
                                avatarURL: null,
                            },
                            description: {
                                name: 'test-campaign',
                                description: '### Very great campaign',
                            },
                            diffStat: {
                                added: 1000,
                                changed: 100,
                                deleted: 182,
                            },
                            expiresAt: addDays(new Date(), 3).toISOString(),
                            namespace:
                                entityType === 'user'
                                    ? {
                                          namespaceName: 'alice',
                                          url: '/users/alice',
                                      }
                                    : {
                                          namespaceName: 'test-org',
                                          url: '/organizations/test-org',
                                      },
                            viewerCanAdminister: true,
                        },
                    }),
                    CampaignSpecChangesetSpecs: () => ({
                        node: {
                            __typename: 'CampaignSpec',
                            changesetSpecs: {
                                nodes: [
                                    {
                                        __typename: 'VisibleChangesetSpec',
                                        description: {
                                            __typename: 'GitBranchChangesetDescription',
                                            baseRef: 'main',
                                            headRef: 'head-ref',
                                            baseRepository: {
                                                name: 'github.com/sourcegraph/repo',
                                                url: 'http://test.test/repo',
                                            },
                                            published: true,
                                            body: 'Body',
                                            commits: [
                                                {
                                                    subject: 'Commit message',
                                                    body: 'And the more explanatory body.',
                                                    author: {
                                                        avatarURL: null,
                                                        displayName: 'john',
                                                        email: 'john@test.not',
                                                        user: {
                                                            displayName: 'lejohn',
                                                            url: '/users/lejohn',
                                                            username: 'john',
                                                        },
                                                    },
                                                },
                                            ],
                                            diffStat: {
                                                added: 10,
                                                changed: 2,
                                                deleted: 9,
                                            },
                                            title: 'Changeset title',
                                        },
                                        expiresAt: addDays(new Date(), 3).toISOString(),
                                        id: 'changesetspec123',
                                        type: ChangesetSpecType.BRANCH,
                                    },
                                ],
                                pageInfo: {
                                    endCursor: null,
                                    hasNextPage: false,
                                },
                                totalCount: 1,
                            },
                        },
                    }),
                    ChangesetSpecFileDiffs: () => ({
                        node: {
                            __typename: 'VisibleChangesetSpec',
                            description: {
                                __typename: 'GitBranchChangesetDescription',
                                diff: mockDiff,
                            },
                        },
                    }),
                    CreateCampaign: () => ({
                        createCampaign: {
                            id: 'campaign123',
                            url: namespaceURL + '/campaigns/test-campaign',
                        },
                    }),
                })

                await driver.page.goto(driver.sourcegraphBaseUrl + namespaceURL + '/campaigns/apply/spec123')
                // View overview page.
                await driver.page.waitForSelector('.test-campaign-apply-page')

                // Expand one changeset.
                await driver.page.click('.test-campaigns-expand-changeset-spec')
                // Expect one diff to be rendered.
                await driver.page.waitForSelector('.test-file-diff-node')

                // Apply campaign.
                await Promise.all([
                    driver.page.waitForNavigation(),
                    driver.acceptNextDialog(),
                    driver.page.click('.test-campaigns-confirm-apply-btn'),
                ])
                // Expect to be back at campaign overview page.
                assert.strictEqual(
                    await driver.page.evaluate(() => window.location.href),
                    testContext.driver.sourcegraphBaseUrl + namespaceURL + '/campaigns/test-campaign'
                )
            })
        }
    })

    describe('Campaign close preview', () => {
        for (const entityType of ['user', 'org'] as const) {
            it(`displays a preview for closing a campaign in ${entityType} namespace`, async () => {
                testContext.overrideGraphQL({
                    ...commonWebGraphQlResults,
                    ...mockCommonGraphQLResponses(entityType),
                    CampaignChangesets,
                    ExternalChangesetFileDiffs,
                    CloseCampaign: () => ({
                        closeCampaign: {
                            id: 'campaign123',
                        },
                    }),
                })
                const namespaceURL = entityType === 'user' ? '/users/alice' : '/organizations/test-org'

                await driver.page.goto(driver.sourcegraphBaseUrl + namespaceURL + '/campaigns/test-campaign/close')
                // View overview page.
                await driver.page.waitForSelector('.test-campaign-close-page')

                // Check close changesets box.
                assert.strictEqual(await driver.page.$('.test-campaigns-close-willclose-header'), null)
                await driver.page.click('.test-campaigns-close-changesets-checkbox')
                await driver.page.waitForSelector('.test-campaigns-close-willclose-header')

                // Expand one changeset.
                await driver.page.click('.test-campaigns-expand-changeset')
                // Expect one diff to be rendered.
                await driver.page.waitForSelector('.test-file-diff-node')

                // Close campaign.
                await Promise.all([
                    driver.page.waitForNavigation(),
                    driver.page.click('.test-campaigns-confirm-close-btn'),
                ])
                // Expect to be back at campaign overview page.
                assert.strictEqual(
                    await driver.page.evaluate(() => window.location.href),
                    testContext.driver.sourcegraphBaseUrl + namespaceURL + '/campaigns/test-campaign'
                )
            })
        }
    })
})
