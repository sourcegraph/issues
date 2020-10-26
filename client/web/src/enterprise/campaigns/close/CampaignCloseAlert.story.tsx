import { storiesOf } from '@storybook/react'
import { boolean } from '@storybook/addon-knobs'
import React from 'react'
import { CampaignCloseAlert } from './CampaignCloseAlert'
import { EnterpriseWebStory } from '../../components/EnterpriseWebStory'
import { useState } from '@storybook/addons'

const { add } = storiesOf('web/campaigns/close/CampaignCloseAlert', module)
    .addDecorator(story => <div className="p-3 container web-content">{story()}</div>)
    .addParameters({
        chromatic: {
            viewports: [320, 576, 978, 1440],
        },
    })

add('Has open changesets', () => {
    const [closeChangesets, setCloseChangesets] = useState(false)
    return (
        <EnterpriseWebStory>
            {props => (
                <CampaignCloseAlert
                    {...props}
                    campaignID="campaign123"
                    campaignURL="/users/john/campaigns/campaign123"
                    totalCount={10}
                    closeChangesets={closeChangesets}
                    setCloseChangesets={setCloseChangesets}
                    viewerCanAdminister={boolean('viewerCanAdminister', true)}
                    closeCampaign={() => Promise.resolve()}
                />
            )}
        </EnterpriseWebStory>
    )
})
add('No open changesets', () => {
    const [closeChangesets, setCloseChangesets] = useState(false)
    return (
        <EnterpriseWebStory>
            {props => (
                <CampaignCloseAlert
                    {...props}
                    campaignID="campaign123"
                    campaignURL="/users/john/campaigns/campaign123"
                    totalCount={0}
                    closeChangesets={closeChangesets}
                    setCloseChangesets={setCloseChangesets}
                    viewerCanAdminister={boolean('viewerCanAdminister', true)}
                    closeCampaign={() => Promise.resolve()}
                />
            )}
        </EnterpriseWebStory>
    )
})
