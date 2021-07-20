import { action } from '@storybook/addon-actions'
import { storiesOf } from '@storybook/react'
import { noop } from 'lodash'
import React from 'react'

import { EnterpriseWebStory } from '../../../components/EnterpriseWebStory'

import { PublishChangesetsModal } from './PublishChangesetsModal'

const { add } = storiesOf('web/batches/details/PublishChangesetsModal', module).addDecorator(story => (
    <div className="p-3 container">{story()}</div>
))

const changesetIDsFunction = () => Promise.resolve(['test-123', 'test-234'])
const publishChangesets = () => {
    action('PublishChangesets')
    return Promise.resolve()
}

add('Confirmation', () => (
    <EnterpriseWebStory>
        {props => (
            <PublishChangesetsModal
                {...props}
                afterCreate={noop}
                batchChangeID="test-123"
                changesetIDs={changesetIDsFunction}
                onCancel={noop}
                publishChangesets={publishChangesets}
            />
        )}
    </EnterpriseWebStory>
))
