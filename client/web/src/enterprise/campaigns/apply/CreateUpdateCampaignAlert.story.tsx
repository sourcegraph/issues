import { storiesOf } from '@storybook/react'
import { boolean } from '@storybook/addon-knobs'
import React from 'react'
import { CreateUpdateCampaignAlert } from './CreateUpdateCampaignAlert'
import { EnterpriseWebStory } from '../../components/EnterpriseWebStory'

const { add } = storiesOf('web/campaigns/apply/CreateUpdateCampaignAlert', module)
    .addDecorator(story => <div className="p-3 container web-content">{story()}</div>)
    .addParameters({
        chromatic: {
            viewports: [320, 576, 978, 1440],
        },
    })

add('Create', () => (
    <EnterpriseWebStory>
        {props => (
            <CreateUpdateCampaignAlert
                {...props}
                specID="123"
                campaign={null}
                viewerCanAdminister={boolean('viewerCanAdminister', true)}
            />
        )}
    </EnterpriseWebStory>
))
add('Update', () => (
    <EnterpriseWebStory>
        {props => (
            <CreateUpdateCampaignAlert
                {...props}
                specID="123"
                campaign={{ id: '123', name: 'awesome-campaign', url: 'http://test.test/awesome' }}
                viewerCanAdminister={boolean('viewerCanAdminister', true)}
            />
        )}
    </EnterpriseWebStory>
))
