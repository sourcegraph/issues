import * as H from 'history'
import React from 'react'
import { CampaignBurndownChart } from './BurndownChart'
import { mount } from 'enzyme'
import { of } from 'rxjs'

describe('CampaignBurndownChart', () => {
    test('renders the chart', () => {
        const history = H.createMemoryHistory({ keyLength: 0 })
        expect(
            mount(
                <CampaignBurndownChart
                    campaignID="123"
                    queryChangesetCountsOverTime={() =>
                        of([
                            {
                                closed: 1,
                                date: '2019-11-13T12:00:00Z',
                                merged: 1,
                                openApproved: 1,
                                openChangesRequested: 1,
                                openPending: 1,
                                total: 10,
                                draft: 0,
                            },
                            {
                                closed: 1,
                                date: '2019-11-14T12:00:00Z',
                                merged: 5,
                                openApproved: 1,
                                openChangesRequested: 1,
                                openPending: 1,
                                total: 10,
                                draft: 0,
                            },
                        ])
                    }
                    history={history}
                    width={500}
                />
            )
        ).toMatchSnapshot()
    })
    test('renders an alert when data for less than 2 days is present', () => {
        const history = H.createMemoryHistory({ keyLength: 0 })
        expect(
            mount(
                <CampaignBurndownChart
                    campaignID="123"
                    queryChangesetCountsOverTime={() =>
                        of([
                            {
                                closed: 1,
                                date: '2019-11-13T12:00:00Z',
                                merged: 1,
                                openApproved: 1,
                                openChangesRequested: 1,
                                openPending: 1,
                                total: 10,
                                draft: 0,
                            },
                        ])
                    }
                    history={history}
                    width={500}
                />
            )
        ).toMatchSnapshot()
    })
    test('renders an alert when no datapoints are in a set of more than 1 day of data', () => {
        const history = H.createMemoryHistory({ keyLength: 0 })
        expect(
            mount(
                <CampaignBurndownChart
                    campaignID="123"
                    queryChangesetCountsOverTime={() =>
                        of([
                            {
                                closed: 0,
                                date: '2019-11-13T12:00:00Z',
                                merged: 0,
                                openApproved: 0,
                                openChangesRequested: 0,
                                openPending: 0,
                                total: 0,
                                draft: 0,
                            },
                            {
                                closed: 0,
                                date: '2019-11-14T12:00:00Z',
                                merged: 0,
                                openApproved: 0,
                                openChangesRequested: 0,
                                openPending: 0,
                                total: 0,
                                draft: 0,
                            },
                        ])
                    }
                    history={history}
                    width={500}
                />
            )
        ).toMatchSnapshot()
    })
})
