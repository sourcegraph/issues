import * as H from 'history'
import React from 'react'
import { GlobalCampaignListPage } from './GlobalCampaignListPage'
import { NOOP_TELEMETRY_SERVICE } from '../../../../../../shared/src/telemetry/telemetryService'
import { of } from 'rxjs'
import { shallow } from 'enzyme'
import { nodes } from '../../list/CampaignNode.story'

const history = H.createMemoryHistory()

describe('GlobalCampaignListPage', () => {
    for (const totalCount of [0, 1]) {
        test(`renders for siteadmin and totalCount: ${totalCount}`, () => {
            expect(
                shallow(
                    <GlobalCampaignListPage
                        history={history}
                        location={history.location}
                        authenticatedUser={{ siteAdmin: true }}
                        queryCampaigns={() =>
                            of({ totalCount: Object.values(nodes).length, nodes: Object.values(nodes) })
                        }
                        telemetryService={NOOP_TELEMETRY_SERVICE}
                    />
                )
            ).toMatchSnapshot()
        })
        test(`renders for non-siteadmin and totalCount: ${totalCount}`, () => {
            expect(
                shallow(
                    <GlobalCampaignListPage
                        history={history}
                        location={history.location}
                        authenticatedUser={{ siteAdmin: false }}
                        queryCampaigns={() =>
                            of({ totalCount: Object.values(nodes).length, nodes: Object.values(nodes) })
                        }
                        telemetryService={NOOP_TELEMETRY_SERVICE}
                    />
                )
            ).toMatchSnapshot()
        })
    }
})
