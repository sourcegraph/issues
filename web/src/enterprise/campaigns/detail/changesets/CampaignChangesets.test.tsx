import React from 'react'
import { CampaignChangesets } from './CampaignChangesets'
import * as H from 'history'
import { of, Subject } from 'rxjs'
import { NOOP_TELEMETRY_SERVICE } from '../../../../../../shared/src/telemetry/telemetryService'
import { shallow } from 'enzyme'

describe('CampaignChangesets', () => {
    const history = H.createMemoryHistory()
    test('renders', () =>
        expect(
            shallow(
                <CampaignChangesets
                    queryChangesets={() =>
                        of({ nodes: [{ id: '0' }] } as any)
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
