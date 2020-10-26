import { storiesOf } from '@storybook/react'
import React from 'react'
import { HiddenExternalChangesetNode } from './HiddenExternalChangesetNode'
import { addHours } from 'date-fns'
import {
    ChangesetExternalState,
    ChangesetReconcilerState,
    ChangesetPublicationState,
} from '../../../../graphql-operations'
import { EnterpriseWebStory } from '../../../components/EnterpriseWebStory'

const { add } = storiesOf('web/campaigns/HiddenExternalChangesetNode', module).addDecorator(story => (
    <div className="p-3 container web-content campaign-changesets__grid">{story()}</div>
))

add('All external states', () => {
    const now = new Date()
    return (
        <EnterpriseWebStory>
            {props => (
                <>
                    {Object.values(ChangesetExternalState).map((externalState, index) => (
                        <HiddenExternalChangesetNode
                            key={index}
                            {...props}
                            node={{
                                id: 'somechangeset',
                                updatedAt: now.toISOString(),
                                nextSyncAt: addHours(now, 1).toISOString(),
                                externalState,
                                publicationState: ChangesetPublicationState.PUBLISHED,
                                reconcilerState: ChangesetReconcilerState.COMPLETED,
                            }}
                        />
                    ))}
                </>
            )}
        </EnterpriseWebStory>
    )
})
