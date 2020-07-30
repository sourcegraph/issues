import React from 'react'
import * as GQL from '../../../../../shared/src/graphql/schema'
import { CampaignDetails } from './CampaignDetails'
import * as H from 'history'
import { of } from 'rxjs'
import { NOOP_TELEMETRY_SERVICE } from '../../../../../shared/src/telemetry/telemetryService'
import { PageTitle } from '../../../components/PageTitle'
import { registerHighlightContributions } from '../../../../../shared/src/highlight/contributions'
import { shallow, mount } from 'enzyme'

// This is idempotent, so calling it in multiple tests is not a problem.
registerHighlightContributions()

jest.mock('./changesets/CampaignChangesets', () => ({ CampaignChangesets: 'CampaignChangesets' }))
jest.mock('../icons', () => ({ CampaignsIcon: 'CampaignsIcon' }))

const history = H.createMemoryHistory()

describe('CampaignDetails', () => {
    afterEach(() => {
        PageTitle.titleSet = false
    })

    test('creation form for empty manual campaign', () =>
        expect(
            shallow(
                <CampaignDetails
                    campaignID={undefined}
                    history={history}
                    location={history.location}
                    authenticatedUser={{ id: 'a', username: 'alice', avatarURL: null }}
                    isLightTheme={true}
                    extensionsController={undefined as any}
                    platformContext={undefined as any}
                    telemetryService={NOOP_TELEMETRY_SERVICE}
                />
            )
        ).toMatchSnapshot())

    test('creation form given existing patch set', () => {
        const component = mount(
            <CampaignDetails
                campaignID={undefined}
                history={history}
                location={{ ...history.location, search: 'patchSet=p' }}
                authenticatedUser={{ id: 'a', username: 'alice', avatarURL: null }}
                isLightTheme={true}
                extensionsController={undefined as any}
                platformContext={undefined as any}
                telemetryService={NOOP_TELEMETRY_SERVICE}
            />
        )
        expect(component).toMatchSnapshot()
    })

    const renderCampaignDetails = ({ viewerCanAdminister }: { viewerCanAdminister: boolean }) => (
        <CampaignDetails
            campaignID="c"
            history={history}
            location={history.location}
            authenticatedUser={{ id: 'a', username: 'alice', avatarURL: null }}
            isLightTheme={true}
            extensionsController={undefined as any}
            platformContext={undefined as any}
            telemetryService={NOOP_TELEMETRY_SERVICE}
            _fetchCampaignById={() =>
                of(({
                    __typename: 'Campaign' as const,
                    id: 'c',
                    name: 'n',
                    description: 'd',
                    // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
                    author: { username: 'alice' } as GQL.User,
                    changesets: { totalCount: 2, stats: { total: 10, closed: 0, merged: 0 } },
                    changesetCountsOverTime: [] as GQL.ChangesetCounts[],
                    viewerCanAdminister,
                    hasUnpublishedPatches: false,
                    branch: 'awesome-branch',
                    createdAt: '2020-01-01',
                    updatedAt: '2020-01-01',
                    closedAt: null,
                    diffStat: {
                        __typename: 'IDiffStat' as const,
                        added: 5,
                        changed: 3,
                        deleted: 2,
                    },
                } as any) as GQL.Campaign)
            }
        />
    )

    for (const viewerCanAdminister of [true, false]) {
        describe(`viewerCanAdminister: ${String(viewerCanAdminister)}`, () => {
            test('viewing existing', () => {
                expect(mount(renderCampaignDetails({ viewerCanAdminister }))).toMatchSnapshot()
            })
        })
    }
})
