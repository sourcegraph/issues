import { storiesOf } from '@storybook/react'
import React from 'react'
import { EnterpriseWebStory } from '../../components/EnterpriseWebStory'
import { CreateCampaignPage } from './CreateCampaignPage'

const { add } = storiesOf('web/campaigns/CreateCampaignPage', module).addDecorator(story => (
    <div className="p-3 container web-content">{story()}</div>
))

add('Page', () => (
    <EnterpriseWebStory>
        {props => <CreateCampaignPage {...props} authenticatedUser={{ username: 'alice' }} />}
    </EnterpriseWebStory>
))
