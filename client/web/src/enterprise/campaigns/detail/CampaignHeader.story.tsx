import { storiesOf } from '@storybook/react'
import React from 'react'
import { CampaignHeader } from './CampaignHeader'
import { Link } from '../../../../../shared/src/components/Link'
import { EnterpriseWebStory } from '../../components/EnterpriseWebStory'

const { add } = storiesOf('web/campaigns/CampaignHeader', module).addDecorator(story => (
    <div className="p-3 container web-content">{story()}</div>
))

add('Full', () => (
    <EnterpriseWebStory>
        {props => (
            <CampaignHeader
                {...props}
                namespace={{ namespaceName: 'alice', url: 'https://test.test/alice' }}
                name="awesome-campaign"
                actionSection={
                    <Link to="/" className="btn btn-secondary">
                        Some button
                    </Link>
                }
            />
        )}
    </EnterpriseWebStory>
))
