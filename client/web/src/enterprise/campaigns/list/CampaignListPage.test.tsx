import * as H from 'history'
import React from 'react'
import { CampaignListPage } from './CampaignListPage'
import { NOOP_TELEMETRY_SERVICE } from '../../../../../shared/src/telemetry/telemetryService'
import { of } from 'rxjs'
import { shallow } from 'enzyme'
import { nodes } from './CampaignNode.story'
import { NOOP_BREADCRUMB_SETTERS } from '../../../components/Breadcrumbs'

const history = H.createMemoryHistory()

describe('CampaignListPage', () => {
    for (const totalCount of [0, 1]) {
        test(`renders for siteadmin and totalCount: ${totalCount}`, () => {
            expect(
                shallow(
                    <CampaignListPage
                        {...NOOP_BREADCRUMB_SETTERS}
                        history={history}
                        location={history.location}
                        queryCampaigns={() =>
                            of({
                                totalCount: Object.values(nodes).length,
                                nodes: Object.values(nodes),
                                pageInfo: { endCursor: null, hasNextPage: false },
                            })
                        }
                        telemetryService={NOOP_TELEMETRY_SERVICE}
                    />
                )
            ).toMatchSnapshot()
        })
        test(`renders for non-siteadmin and totalCount: ${totalCount}`, () => {
            expect(
                shallow(
                    <CampaignListPage
                        {...NOOP_BREADCRUMB_SETTERS}
                        history={history}
                        location={history.location}
                        queryCampaigns={() =>
                            of({
                                totalCount: Object.values(nodes).length,
                                nodes: Object.values(nodes),
                                pageInfo: { endCursor: null, hasNextPage: false },
                            })
                        }
                        telemetryService={NOOP_TELEMETRY_SERVICE}
                    />
                )
            ).toMatchSnapshot()
        })
    }
})
