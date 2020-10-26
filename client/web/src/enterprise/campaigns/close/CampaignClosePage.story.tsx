import React from 'react'
import { storiesOf } from '@storybook/react'
import { boolean } from '@storybook/addon-knobs'
import { CampaignClosePage } from './CampaignClosePage'
import {
    queryChangesets as _queryChangesets,
    queryExternalChangesetWithFileDiffs,
    fetchCampaignByNamespace,
} from '../detail/backend'
import { of } from 'rxjs'
import { subDays } from 'date-fns'
import {
    ChangesetExternalState,
    ChangesetPublicationState,
    ChangesetReconcilerState,
    ChangesetCheckState,
    ChangesetReviewState,
    CampaignFields,
} from '../../../graphql-operations'
import { useMemo, useCallback } from '@storybook/addons'
import { EnterpriseWebStory } from '../../components/EnterpriseWebStory'

const { add } = storiesOf('web/campaigns/close/CampaignClosePage', module)
    .addDecorator(story => <div className="p-3 container web-content">{story()}</div>)
    .addParameters({
        chromatic: {
            viewports: [320, 576, 978, 1440],
        },
    })

const now = new Date()

const campaignDefaults: CampaignFields = {
    __typename: 'Campaign',
    changesets: {
        stats: {
            closed: 1,
            deleted: 1,
            merged: 2,
            draft: 1,
            open: 2,
            total: 10,
            unpublished: 4,
        },
    },
    createdAt: subDays(now, 5).toISOString(),
    initialApplier: {
        url: '/users/alice',
        username: 'alice',
    },
    diffStat: {
        added: 10,
        changed: 8,
        deleted: 10,
    },
    id: 'specid',
    url: '/users/alice/campaigns/specid',
    namespace: {
        namespaceName: 'alice',
        url: '/users/alice',
    },
    viewerCanAdminister: true,
    closedAt: null,
    description: '## What this campaign does\n\nTruly awesome things for example.',
    name: 'awesome-campaign',
    updatedAt: subDays(now, 5).toISOString(),
    lastAppliedAt: subDays(now, 5).toISOString(),
    lastApplier: {
        url: '/users/bob',
        username: 'bob',
    },
    currentSpec: {
        originalInput: 'name: awesome-campaign\ndescription: somestring',
    },
}

const queryChangesets: typeof _queryChangesets = () =>
    of({
        pageInfo: {
            endCursor: null,
            hasNextPage: false,
        },
        totalCount: 6,
        nodes: [
            {
                __typename: 'HiddenExternalChangeset',
                createdAt: subDays(now, 5).toISOString(),
                externalState: ChangesetExternalState.OPEN,
                id: 'someh1',
                nextSyncAt: null,
                publicationState: ChangesetPublicationState.UNPUBLISHED,
                reconcilerState: ChangesetReconcilerState.QUEUED,
                updatedAt: subDays(now, 5).toISOString(),
            },
            {
                __typename: 'HiddenExternalChangeset',
                createdAt: subDays(now, 5).toISOString(),
                externalState: ChangesetExternalState.OPEN,
                id: 'someh2',
                nextSyncAt: null,
                publicationState: ChangesetPublicationState.PUBLISHED,
                reconcilerState: ChangesetReconcilerState.PROCESSING,
                updatedAt: subDays(now, 5).toISOString(),
            },
            {
                __typename: 'HiddenExternalChangeset',
                createdAt: subDays(now, 5).toISOString(),
                externalState: ChangesetExternalState.OPEN,
                id: 'someh3',
                nextSyncAt: null,
                publicationState: ChangesetPublicationState.UNPUBLISHED,
                reconcilerState: ChangesetReconcilerState.ERRORED,
                updatedAt: subDays(now, 5).toISOString(),
            },
            {
                __typename: 'HiddenExternalChangeset',
                createdAt: subDays(now, 5).toISOString(),
                externalState: ChangesetExternalState.OPEN,
                id: 'someh4',
                nextSyncAt: null,
                publicationState: ChangesetPublicationState.PUBLISHED,
                reconcilerState: ChangesetReconcilerState.COMPLETED,
                updatedAt: subDays(now, 5).toISOString(),
            },
            {
                __typename: 'ExternalChangeset',
                body: 'body',
                checkState: ChangesetCheckState.PASSED,
                diffStat: {
                    added: 10,
                    changed: 9,
                    deleted: 1,
                },
                externalID: '123',
                externalURL: {
                    url: 'http://test.test/123',
                },
                labels: [{ color: '93ba13', description: 'Very awesome description', text: 'Some label' }],
                repository: {
                    id: 'repoid',
                    name: 'github.com/sourcegraph/awesome',
                    url: 'http://test.test/awesome',
                },
                reviewState: ChangesetReviewState.COMMENTED,
                title: 'Add prettier to all projects',
                createdAt: subDays(now, 5).toISOString(),
                updatedAt: subDays(now, 5).toISOString(),
                externalState: ChangesetExternalState.OPEN,
                nextSyncAt: null,
                id: 'somev1',
                reconcilerState: ChangesetReconcilerState.COMPLETED,
                publicationState: ChangesetPublicationState.PUBLISHED,
                error: null,
                currentSpec: { id: 'spec-rand-id-1' },
            },
            {
                __typename: 'ExternalChangeset',
                body: 'body',
                checkState: null,
                diffStat: {
                    added: 10,
                    changed: 9,
                    deleted: 1,
                },
                externalID: null,
                externalURL: null,
                labels: [],
                repository: {
                    id: 'repoid',
                    name: 'github.com/sourcegraph/awesome',
                    url: 'http://test.test/awesome',
                },
                reviewState: null,
                title: 'Add prettier to all projects',
                createdAt: subDays(now, 5).toISOString(),
                updatedAt: subDays(now, 5).toISOString(),
                externalState: null,
                nextSyncAt: null,
                id: 'somev2',
                reconcilerState: ChangesetReconcilerState.ERRORED,
                publicationState: ChangesetPublicationState.UNPUBLISHED,
                error: 'Cannot create PR, insufficient token scope.',
                currentSpec: { id: 'spec-rand-id-2' },
            },
        ],
    })

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

add('Overview', () => {
    const viewerCanAdminister = boolean('viewerCanAdminister', true)
    const campaign: CampaignFields = useMemo(
        () => ({
            ...campaignDefaults,
            viewerCanAdminister,
        }),
        [viewerCanAdminister]
    )
    const fetchCampaign: typeof fetchCampaignByNamespace = useCallback(() => of(campaign), [campaign])
    return (
        <EnterpriseWebStory>
            {props => (
                <CampaignClosePage
                    {...props}
                    queryChangesets={queryChangesets}
                    queryExternalChangesetWithFileDiffs={queryEmptyExternalChangesetWithFileDiffs}
                    namespaceID="n123"
                    campaignName="c123"
                    fetchCampaignByNamespace={fetchCampaign}
                    extensionsController={{} as any}
                    platformContext={{} as any}
                />
            )}
        </EnterpriseWebStory>
    )
})

add('No open changesets', () => {
    const campaign: CampaignFields = useMemo(() => campaignDefaults, [])
    const fetchCampaign: typeof fetchCampaignByNamespace = useCallback(() => of(campaign), [campaign])
    const queryEmptyChangesets = useCallback(
        () =>
            of({
                pageInfo: {
                    endCursor: null,
                    hasNextPage: false,
                },
                totalCount: 0,
                nodes: [],
            }),
        []
    )
    return (
        <EnterpriseWebStory>
            {props => (
                <CampaignClosePage
                    {...props}
                    queryChangesets={queryEmptyChangesets}
                    queryExternalChangesetWithFileDiffs={queryEmptyExternalChangesetWithFileDiffs}
                    namespaceID="n123"
                    campaignName="c123"
                    fetchCampaignByNamespace={fetchCampaign}
                    extensionsController={{} as any}
                    platformContext={{} as any}
                />
            )}
        </EnterpriseWebStory>
    )
})
