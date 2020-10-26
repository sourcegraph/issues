import React from 'react'
import { CampaignNode } from './CampaignNode'
import { parseISO } from 'date-fns'
import { createMemoryHistory } from 'history'
import { mount } from 'enzyme'
import { ListCampaign } from '../../../graphql-operations'

const now = parseISO('2019-01-01T23:15:01Z')

describe('CampaignNode', () => {
    const node: ListCampaign = {
        id: '123',
        url: '/users/alice/campaigns/123',
        name: 'Upgrade lodash to v4',
        description: `
# Removes lodash

- and renders in markdown
        `,
        changesets: { stats: { merged: 0, open: 1, closed: 3 } },
        createdAt: '2019-12-04T23:15:01Z',
        closedAt: null,
        namespace: {
            namespaceName: 'alice',
            url: '/users/alice',
        },
    }

    test('open campaign', () => {
        expect(
            mount(<CampaignNode displayNamespace={true} node={node} now={() => now} history={createMemoryHistory()} />)
        ).toMatchSnapshot()
    })
    test('open campaign on user page', () => {
        expect(
            mount(<CampaignNode displayNamespace={false} node={node} now={() => now} history={createMemoryHistory()} />)
        ).toMatchSnapshot()
    })
    test('closed campaign', () => {
        expect(
            mount(
                <CampaignNode
                    node={{ ...node, closedAt: '2019-12-04T23:19:01Z' }}
                    displayNamespace={true}
                    now={() => now}
                    history={createMemoryHistory()}
                />
            )
        ).toMatchSnapshot()
    })
    test('campaign without description', () => {
        expect(
            mount(
                <CampaignNode
                    node={{
                        ...node,
                        description: null,
                    }}
                    displayNamespace={true}
                    now={() => now}
                    history={createMemoryHistory()}
                />
            )
        ).toMatchSnapshot()
    })
})
