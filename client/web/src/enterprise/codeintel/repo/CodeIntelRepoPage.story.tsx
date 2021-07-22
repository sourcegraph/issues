import { storiesOf } from '@storybook/react'
import React from 'react'

import { RepositoryFields } from '../../../graphql-operations'
import { EnterpriseWebStory } from '../../components/EnterpriseWebStory'

import { CodeIntelRepoPage } from './CodeIntelRepoPage'

const { add } = storiesOf('web/codeintel/repo/CodeIntelRepoPage', module)
    .addDecorator(story => <div className="p-3 container web-content">{story()}</div>)
    .addParameters({
        chromatic: {
            viewports: [320, 576, 978, 1440],
        },
    })

const repoDefaults: RepositoryFields = {
    description: 'An awesome repo!',
    defaultBranch: null,
    viewerCanAdminister: false,
    externalURLs: [],
    id: 'repoid',
    name: 'github.com/sourcegraph/awesome',
    url: 'http://test.test/awesome',
}

add('Page', () => (
    <EnterpriseWebStory initialEntries={['/github.com/sourcegraph/awesome/-/code-intelligence']}>
        {props => <CodeIntelRepoPage {...props} repo={repoDefaults} />}
    </EnterpriseWebStory>
))
