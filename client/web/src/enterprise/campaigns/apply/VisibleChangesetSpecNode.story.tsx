import { storiesOf } from '@storybook/react'
import React from 'react'
import { VisibleChangesetSpecNode } from './VisibleChangesetSpecNode'
import { addDays } from 'date-fns'
import { VisibleChangesetSpecFields, ChangesetSpecType, Scalars } from '../../../graphql-operations'
import { of } from 'rxjs'
import { EnterpriseWebStory } from '../../components/EnterpriseWebStory'

const { add } = storiesOf('web/campaigns/apply/VisibleChangesetSpecNode', module).addDecorator(story => (
    <div className="p-3 container web-content changeset-spec-list__grid">{story()}</div>
))

const baseChangeset: (published: Scalars['PublishedValue']) => VisibleChangesetSpecFields = published => ({
    __typename: 'VisibleChangesetSpec',
    id: 'someidv2',
    expiresAt: addDays(new Date(), 7).toISOString(),
    type: ChangesetSpecType.EXISTING,
    description: {
        __typename: 'GitBranchChangesetDescription',
        baseRepository: { name: 'github.com/sourcegraph/testrepo', url: 'https://test.test/repo' },
        baseRef: 'master',
        headRef: 'cool-branch',
        body: 'Body text',
        commits: [
            {
                subject: 'This is the first line of the commit message',
                body: `And the more explanatory body. And the more explanatory body.
And the more explanatory body. And the more explanatory body.
And the more explanatory body. And the more explanatory body.
And the more explanatory body. And the more explanatory body. And the more explanatory body.
And the more explanatory body. And the more explanatory body. And the more explanatory body.`,
                author: {
                    avatarURL: null,
                    displayName: 'john',
                    email: 'john@test.not',
                    user: { displayName: 'lejohn', url: '/users/lejohn', username: 'john' },
                },
            },
        ],
        diffStat: {
            added: 10,
            changed: 8,
            deleted: 2,
        },
        title: 'Add prettier to repository',
        published,
    },
})

export const visibleChangesetSpecStories: Record<string, VisibleChangesetSpecFields> = {
    'Import changeset': {
        __typename: 'VisibleChangesetSpec',
        id: 'someidv1',
        expiresAt: addDays(new Date(), 7).toISOString(),
        type: ChangesetSpecType.EXISTING,
        description: {
            __typename: 'ExistingChangesetReference',
            baseRepository: { name: 'github.com/sourcegraph/testrepo', url: 'https://test.test/repo' },
            externalID: '123',
        },
    },
    'Create changeset published': baseChangeset(true),
    'Create changeset draft': baseChangeset('draft'),
    'Create changeset not published': baseChangeset(false),
}

const queryEmptyFileDiffs = () =>
    of({ fileDiffs: { totalCount: 0, pageInfo: { endCursor: null, hasNextPage: false }, nodes: [] } })

for (const storyName of Object.keys(visibleChangesetSpecStories)) {
    add(storyName, () => (
        <EnterpriseWebStory>
            {props => (
                <VisibleChangesetSpecNode
                    {...props}
                    node={visibleChangesetSpecStories[storyName]}
                    queryChangesetSpecFileDiffs={queryEmptyFileDiffs}
                />
            )}
        </EnterpriseWebStory>
    ))
}
