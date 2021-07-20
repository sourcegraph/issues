import { InsightsDashboardType } from '../../core/types'

import { ALL_INSIGHTS_DASHBOARD, getInsightsDashboards } from './use-dashboards'

describe('getInsightsDashboards', () => {
    describe('should return empty custom list', () => {
        test('with null subject value', () => {
            expect(getInsightsDashboards(null, {})).toStrictEqual([])
        })

        test('with error like settings value', () => {
            expect(
                getInsightsDashboards(
                    [
                        {
                            subject: {
                                __typename: 'User',
                                id: '101',
                                username: 'emirkusturica',
                                displayName: 'Emir Kusturica',
                                viewerCanAdminister: true,
                            },
                            settings: new Error(),
                            lastID: null,
                        },
                    ],
                    {}
                )
            ).toStrictEqual([ALL_INSIGHTS_DASHBOARD])
        })

        test('with unsupported types of settings cascade subject', () => {
            expect(
                getInsightsDashboards(
                    [
                        {
                            subject: {
                                __typename: 'Client',
                                id: '101',
                                displayName: 'Emir Kusturica',
                                viewerCanAdminister: true,
                            },
                            settings: new Error(),
                            lastID: null,
                        },
                    ],
                    {}
                )
            ).toStrictEqual([ALL_INSIGHTS_DASHBOARD])
        })
    })

    describe('should return dashboard list', () => {
        test('with built in dashboard only if dashboard settings are empty', () => {
            expect(
                getInsightsDashboards(
                    [
                        {
                            subject: {
                                __typename: 'Org',
                                id: '102',
                                name: 'sourcegraph',
                                displayName: 'Sourcegraph',
                                viewerCanAdminister: true,
                            },
                            settings: {},
                            lastID: null,
                        },
                        {
                            subject: {
                                __typename: 'User',
                                id: '101',
                                username: 'emirkusturica',
                                displayName: 'Emir Kusturica',
                                viewerCanAdminister: true,
                            },
                            settings: {},
                            lastID: null,
                        },
                    ],
                    {}
                )
            ).toStrictEqual([
                ALL_INSIGHTS_DASHBOARD,
                {
                    type: InsightsDashboardType.Organization,
                    builtIn: true,
                    id: '102',
                    title: 'Sourcegraph',
                    insightIds: [],
                    owner: {
                        id: '102',
                        name: 'Sourcegraph',
                    },
                },
                {
                    type: InsightsDashboardType.Personal,
                    builtIn: true,
                    title: 'Emir Kusturica',
                    id: '101',
                    insightIds: [],
                    owner: {
                        id: '101',
                        name: 'Emir Kusturica',
                    },
                },
            ])
        })

        test('with personal (user-wide) dashboards only', () => {
            expect(
                getInsightsDashboards(
                    [
                        {
                            subject: {
                                __typename: 'User',
                                id: '101',
                                username: 'emirkusturica',
                                displayName: 'Emir Kusturica',
                                viewerCanAdminister: true,
                            },
                            settings: {
                                'insights.dashboards': {
                                    'insights.dashboard.testDashboard': {
                                        id: '001',
                                        title: 'Test Dashboard',
                                        insightIds: ['insightID1', 'insightID2'],
                                    },
                                    'insights.dashboard.anotherTestDashboard': {
                                        id: '002',
                                        title: 'Another Test Dashboard',
                                        insightIds: ['insightID3', 'insightID4'],
                                    },
                                },
                            },
                            lastID: null,
                        },
                    ],
                    {}
                )
            ).toStrictEqual([
                ALL_INSIGHTS_DASHBOARD,
                {
                    type: InsightsDashboardType.Personal,
                    title: 'Emir Kusturica',
                    builtIn: true,
                    id: '101',
                    insightIds: [],
                    owner: {
                        id: '101',
                        name: 'Emir Kusturica',
                    },
                },
                {
                    type: InsightsDashboardType.Personal,
                    id: '001',
                    title: 'Test Dashboard',
                    settingsKey: 'insights.dashboard.testDashboard',
                    insightIds: ['insightID1', 'insightID2'],
                    owner: {
                        id: '101',
                        name: 'Emir Kusturica',
                    },
                },
                {
                    type: InsightsDashboardType.Personal,
                    id: '002',
                    title: 'Another Test Dashboard',
                    settingsKey: 'insights.dashboard.anotherTestDashboard',
                    insightIds: ['insightID3', 'insightID4'],
                    owner: {
                        id: '101',
                        name: 'Emir Kusturica',
                    },
                },
            ])
        })

        test('with org-wide and personal dashboards', () => {
            expect(
                getInsightsDashboards(
                    [
                        {
                            subject: {
                                __typename: 'Org',
                                id: '102',
                                name: 'sourcegraph',
                                displayName: 'Sourcegraph',
                                viewerCanAdminister: true,
                            },
                            settings: {
                                'insights.dashboards': {
                                    'insights.dashboard.testDashboard': {
                                        id: '001',
                                        title: 'Test Dashboard',
                                        insightIds: ['insightID1', 'insightID2'],
                                    },
                                },
                            },
                            lastID: null,
                        },
                        {
                            subject: {
                                __typename: 'User',
                                id: '101',
                                username: 'emirkusturica',
                                displayName: 'Emir Kusturica',
                                viewerCanAdminister: true,
                            },
                            settings: {
                                'insights.dashboards': {
                                    'insights.dashboard.anotherTestDashboard': {
                                        id: '002',
                                        title: 'Another Test Dashboard',
                                        insightIds: ['insightID3', 'insightID4'],
                                    },
                                },
                            },
                            lastID: null,
                        },
                    ],
                    {}
                )
            ).toStrictEqual([
                ALL_INSIGHTS_DASHBOARD,
                {
                    type: InsightsDashboardType.Organization,
                    builtIn: true,
                    id: '102',
                    title: 'Sourcegraph',
                    insightIds: [],
                    owner: {
                        id: '102',
                        name: 'Sourcegraph',
                    },
                },
                {
                    type: InsightsDashboardType.Organization,
                    id: '001',
                    title: 'Test Dashboard',
                    settingsKey: 'insights.dashboard.testDashboard',
                    insightIds: ['insightID1', 'insightID2'],
                    owner: {
                        id: '102',
                        name: 'Sourcegraph',
                    },
                },
                {
                    type: InsightsDashboardType.Personal,
                    id: '101',
                    title: 'Emir Kusturica',
                    builtIn: true,
                    insightIds: [],
                    owner: {
                        id: '101',
                        name: 'Emir Kusturica',
                    },
                },
                {
                    type: InsightsDashboardType.Personal,
                    id: '002',
                    title: 'Another Test Dashboard',
                    settingsKey: 'insights.dashboard.anotherTestDashboard',
                    insightIds: ['insightID3', 'insightID4'],
                    owner: {
                        id: '101',
                        name: 'Emir Kusturica',
                    },
                },
            ])
        })
    })
})
