import { storiesOf } from '@storybook/react'
import { boolean } from '@storybook/addon-knobs'
import React from 'react'
import { CampaignChangesets } from './CampaignChangesets'
import { addHours } from 'date-fns'
import { of } from 'rxjs'
import {
    ChangesetFields,
    ChangesetExternalState,
    ChangesetReconcilerState,
    ChangesetPublicationState,
    ChangesetCheckState,
    ChangesetReviewState,
} from '../../../../graphql-operations'
import { queryExternalChangesetWithFileDiffs } from '../backend'
import { EnterpriseWebStory } from '../../../components/EnterpriseWebStory'

const { add } = storiesOf('web/campaigns/CampaignChangesets', module).addDecorator(story => (
    <div className="p-3 container web-content">{story()}</div>
))

const now = new Date()
const nodes: ChangesetFields[] = [
    ...Object.values(ChangesetExternalState).map(
        (externalState): ChangesetFields => ({
            __typename: 'ExternalChangeset',
            id: 'somechangeset' + externalState,
            updatedAt: now.toISOString(),
            nextSyncAt: addHours(now, 1).toISOString(),
            externalState,
            title: 'Changeset title on code host',
            reconcilerState: ChangesetReconcilerState.COMPLETED,
            publicationState: ChangesetPublicationState.PUBLISHED,
            body: 'This changeset does the following things:\nIs awesome\nIs useful',
            checkState: ChangesetCheckState.PENDING,
            createdAt: now.toISOString(),
            externalID: '123',
            externalURL: {
                url: 'http://test.test/pr/123',
            },
            diffStat: {
                added: 10,
                changed: 20,
                deleted: 8,
            },
            labels: [],
            repository: {
                id: 'repoid',
                name: 'github.com/sourcegraph/sourcegraph',
                url: 'http://test.test/sourcegraph/sourcegraph',
            },
            reviewState: ChangesetReviewState.COMMENTED,
            error: null,
            currentSpec: { id: 'spec-rand-id-1' },
        })
    ),
    ...Object.values(ChangesetExternalState).map(
        (externalState): ChangesetFields => ({
            __typename: 'HiddenExternalChangeset' as const,
            id: 'somehiddenchangeset' + externalState,
            updatedAt: now.toISOString(),
            nextSyncAt: addHours(now, 1).toISOString(),
            externalState,
            createdAt: now.toISOString(),
            reconcilerState: ChangesetReconcilerState.COMPLETED,
            publicationState: ChangesetPublicationState.PUBLISHED,
        })
    ),
]
const queryChangesets = () => of({ totalCount: nodes.length, nodes, pageInfo: { endCursor: null, hasNextPage: false } })

const queryEmptyExternalChangesetWithFileDiffs: typeof queryExternalChangesetWithFileDiffs = () =>
    of({
        diff: {
            __typename: 'PreviewRepositoryComparison',
            fileDiffs: {
                nodes: [],
                totalCount: 0,
                pageInfo: {
                    endCursor: null,
                    hasNextPage: false,
                },
            },
        },
    })

add('List of changesets', () => (
    <EnterpriseWebStory>
        {props => (
            <CampaignChangesets
                {...props}
                queryChangesets={queryChangesets}
                queryExternalChangesetWithFileDiffs={queryEmptyExternalChangesetWithFileDiffs}
                extensionsController={undefined as any}
                platformContext={undefined as any}
                campaignID="campaignid"
                viewerCanAdminister={boolean('viewerCanAdminister', true)}
            />
        )}
    </EnterpriseWebStory>
))
