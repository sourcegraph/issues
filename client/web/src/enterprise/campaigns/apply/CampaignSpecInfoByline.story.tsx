import { storiesOf } from '@storybook/react'
import React from 'react'
import { CampaignSpecInfoByline } from './CampaignSpecInfoByline'
import { subDays } from 'date-fns'
import { EnterpriseWebStory } from '../../components/EnterpriseWebStory'

const { add } = storiesOf('web/campaigns/apply/CampaignSpecInfoByline', module).addDecorator(story => (
    <div className="p-3 container web-content">{story()}</div>
))

add('Default', () => (
    <EnterpriseWebStory>
        {() => (
            <CampaignSpecInfoByline
                createdAt={subDays(new Date(), 3).toISOString()}
                creator={{ url: 'http://test.test/alice', username: 'alice' }}
            />
        )}
    </EnterpriseWebStory>
))
