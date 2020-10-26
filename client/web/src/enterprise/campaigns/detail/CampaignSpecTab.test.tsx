import React from 'react'
import { CampaignSpecTab } from './CampaignSpecTab'
import { mount } from 'enzyme'
import { CampaignFields } from '../../../graphql-operations'
import { registerHighlightContributions } from '../../../../../shared/src/highlight/contributions'

// This is idempotent, so calling it in multiple tests is not a problem.
registerHighlightContributions()

const ALICE: CampaignFields['initialApplier'] | CampaignFields['lastApplier'] = {
    username: 'alice',
    url: 'https://example.com/alice',
}

describe('CampaignSpecTab', () => {
    test('last apply was the (initial) creation', () => {
        expect(
            mount(
                <CampaignSpecTab
                    campaign={{
                        name: 'c',
                        createdAt: '2020-01-01T15:00:00Z',
                        lastAppliedAt: '2020-01-01T15:00:00Z',
                        lastApplier: ALICE,
                    }}
                    originalInput="x"
                />
            )
        ).toMatchSnapshot()
    })

    test('last apply was an update (after creation)', () => {
        expect(
            mount(
                <CampaignSpecTab
                    campaign={{
                        name: 'c',
                        createdAt: '2020-01-01T15:00:00Z',
                        lastAppliedAt: '2020-02-03T16:07:08Z',
                        lastApplier: ALICE,
                    }}
                    originalInput="x"
                />
            )
        ).toMatchSnapshot()
    })

    test('input spec is JSON', () => {
        expect(
            mount(
                <CampaignSpecTab
                    campaign={{
                        name: 'c',
                        createdAt: '2020-01-01T15:00:00Z',
                        lastAppliedAt: '2020-01-01T15:00:00Z',
                        lastApplier: ALICE,
                    }}
                    originalInput='{"foo":"bar"}'
                />
            )
        ).toMatchSnapshot()
    })
})
