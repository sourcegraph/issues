import { storiesOf } from '@storybook/react'
import { boolean } from '@storybook/addon-knobs'
import React from 'react'
import { GlobalCampaignsArea } from './GlobalCampaignsArea'
import { AuthenticatedUser } from '../../../auth'
import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { EnterpriseWebStory } from '../../components/EnterpriseWebStory'

const { add } = storiesOf('web/campaigns/GlobalCampaignsArea', module).addDecorator(story => (
    <React.Suspense fallback={<LoadingSpinner />}>
        <div className="p-3 container">{story()}</div>
    </React.Suspense>
))

add('Dotcom', () => (
    <EnterpriseWebStory>
        {props => (
            <GlobalCampaignsArea
                {...props}
                isSourcegraphDotCom={true}
                platformContext={undefined as any}
                extensionsController={undefined as any}
                authenticatedUser={
                    boolean('isAuthenticated', false) ? ({ username: 'alice' } as AuthenticatedUser) : null
                }
                match={{ isExact: true, path: '/campaigns', url: 'http://test.test/campaigns', params: {} }}
            />
        )}
    </EnterpriseWebStory>
))
