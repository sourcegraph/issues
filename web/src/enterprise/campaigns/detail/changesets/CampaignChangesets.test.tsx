import React from 'react'
import { CampaignChangesets } from './CampaignChangesets'
import * as H from 'history'
import { of, Subject } from 'rxjs'
import { NOOP_TELEMETRY_SERVICE } from '../../../../../../shared/src/telemetry/telemetryService'
import { shallow } from 'enzyme'
import { ChangesetExternalState, ChangesetState } from '../../../../graphql-operations'

describe('CampaignChangesets', () => {
    const history = H.createMemoryHistory()
    test('renders', () =>
        expect(
            shallow(
                <CampaignChangesets
                    queryChangesets={() =>
                        of({
                            totalCount: 1,
                            nodes: [
                                {
                                    id: '0',
                                    __typename: 'HiddenExternalChangeset',
                                    createdAt: new Date('2020-01-03').toISOString(),
                                    externalState: ChangesetExternalState.OPEN,
                                    nextSyncAt: null,
                                    state: ChangesetState.SYNCED,
                                    updatedAt: new Date('2020-01-04').toISOString(),
                                },
                            ],
                        })
                    }
                    campaign={{ id: '123', closedAt: null, viewerCanAdminister: true }}
                    history={history}
                    location={history.location}
                    isLightTheme={true}
                    campaignUpdates={new Subject<void>()}
                    changesetUpdates={new Subject<void>()}
                    extensionsController={undefined as any}
                    platformContext={undefined as any}
                    telemetryService={NOOP_TELEMETRY_SERVICE}
                />
            )
        ).toMatchSnapshot())
})
