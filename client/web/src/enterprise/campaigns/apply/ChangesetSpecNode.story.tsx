import { storiesOf } from '@storybook/react'
import React from 'react'
import { ChangesetSpecNode } from './ChangesetSpecNode'
import { visibleChangesetSpecStories } from './VisibleChangesetSpecNode.story'
import { hiddenChangesetSpecStories } from './HiddenChangesetSpecNode.story'
import { of } from 'rxjs'
import { EnterpriseWebStory } from '../../components/EnterpriseWebStory'

const { add } = storiesOf('web/campaigns/apply/ChangesetSpecNode', module).addDecorator(story => (
    <div className="p-3 container web-content changeset-spec-list__grid">{story()}</div>
))

const queryEmptyFileDiffs = () =>
    of({ fileDiffs: { totalCount: 0, pageInfo: { endCursor: null, hasNextPage: false }, nodes: [] } })

add('Overview', () => {
    const nodes = [...Object.values(visibleChangesetSpecStories), ...Object.values(hiddenChangesetSpecStories)]
    return (
        <EnterpriseWebStory>
            {props => (
                <>
                    {nodes.map(node => (
                        <ChangesetSpecNode
                            {...props}
                            key={node.id}
                            node={node}
                            queryChangesetSpecFileDiffs={queryEmptyFileDiffs}
                        />
                    ))}
                </>
            )}
        </EnterpriseWebStory>
    )
})
