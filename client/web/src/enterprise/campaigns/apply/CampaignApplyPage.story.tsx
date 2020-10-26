import { storiesOf } from '@storybook/react'
import { boolean } from '@storybook/addon-knobs'
import React from 'react'
import { CampaignApplyPage } from './CampaignApplyPage'
import { of, Observable } from 'rxjs'
import { CampaignSpecChangesetSpecsResult, ChangesetSpecFields, CampaignSpecFields } from '../../../graphql-operations'
import { visibleChangesetSpecStories } from './VisibleChangesetSpecNode.story'
import { hiddenChangesetSpecStories } from './HiddenChangesetSpecNode.story'
import { fetchCampaignSpecById } from './backend'
import { addDays, subDays } from 'date-fns'
import { EnterpriseWebStory } from '../../components/EnterpriseWebStory'

const { add } = storiesOf('web/campaigns/apply/CampaignApplyPage', module)
    .addDecorator(story => <div className="p-3 container web-content">{story()}</div>)
    .addParameters({
        chromatic: {
            viewports: [320, 576, 978, 1440],
        },
    })

const nodes: ChangesetSpecFields[] = [
    ...Object.values(visibleChangesetSpecStories),
    ...Object.values(hiddenChangesetSpecStories),
]

const campaignSpec: CampaignSpecFields = {
    appliesToCampaign: null,
    createdAt: subDays(new Date(), 5).toISOString(),
    creator: {
        url: '/users/alice',
        username: 'alice',
    },
    description: {
        name: 'awesome-campaign',
        description: 'This is the description',
    },
    diffStat: {
        added: 10,
        changed: 8,
        deleted: 10,
    },
    expiresAt: addDays(new Date(), 7).toISOString(),
    id: 'specid',
    namespace: {
        namespaceName: 'alice',
        url: '/users/alice',
    },
    viewerCanAdminister: boolean('viewerCanAdminister', true),
}

const fetchCampaignSpecCreate: typeof fetchCampaignSpecById = () => of(campaignSpec)

const fetchCampaignSpecUpdate: typeof fetchCampaignSpecById = () =>
    of({
        ...campaignSpec,
        appliesToCampaign: {
            id: 'somecampaign',
            name: 'awesome-campaign',
            url: '/users/alice/campaigns/somecampaign',
        },
    })

const queryChangesetSpecs = (): Observable<
    (CampaignSpecChangesetSpecsResult['node'] & { __typename: 'CampaignSpec' })['changesetSpecs']
> =>
    of({
        pageInfo: {
            endCursor: null,
            hasNextPage: false,
        },
        totalCount: nodes.length,
        nodes,
    })

const queryEmptyChangesetSpecs = (): Observable<
    (CampaignSpecChangesetSpecsResult['node'] & { __typename: 'CampaignSpec' })['changesetSpecs']
> =>
    of({
        pageInfo: {
            endCursor: null,
            hasNextPage: false,
        },
        totalCount: 0,
        nodes: [],
    })

const queryEmptyFileDiffs = () =>
    of({ fileDiffs: { totalCount: 0, pageInfo: { endCursor: null, hasNextPage: false }, nodes: [] } })

add('Create', () => (
    <EnterpriseWebStory>
        {props => (
            <CampaignApplyPage
                {...props}
                expandChangesetDescriptions={true}
                specID="123123"
                fetchCampaignSpecById={fetchCampaignSpecCreate}
                queryChangesetSpecs={queryChangesetSpecs}
                queryChangesetSpecFileDiffs={queryEmptyFileDiffs}
            />
        )}
    </EnterpriseWebStory>
))

add('Update', () => (
    <EnterpriseWebStory>
        {props => (
            <CampaignApplyPage
                {...props}
                expandChangesetDescriptions={true}
                specID="123123"
                fetchCampaignSpecById={fetchCampaignSpecUpdate}
                queryChangesetSpecs={queryChangesetSpecs}
                queryChangesetSpecFileDiffs={queryEmptyFileDiffs}
            />
        )}
    </EnterpriseWebStory>
))

add('No changesets', () => (
    <EnterpriseWebStory>
        {props => (
            <CampaignApplyPage
                {...props}
                expandChangesetDescriptions={true}
                specID="123123"
                fetchCampaignSpecById={fetchCampaignSpecCreate}
                queryChangesetSpecs={queryEmptyChangesetSpecs}
                queryChangesetSpecFileDiffs={queryEmptyFileDiffs}
            />
        )}
    </EnterpriseWebStory>
))
