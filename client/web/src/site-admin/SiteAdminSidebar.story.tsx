import { storiesOf } from '@storybook/react'
import React from 'react'

import { WebStory } from '../components/WebStory'

import { siteAdminSidebarGroups } from './sidebaritems'
import { SiteAdminSidebar } from './SiteAdminSidebar'

const { add } = storiesOf('web/site-admin/AdminSidebar', module).addDecorator(story => (
    <div style={{ width: '192px' }}>{story()}</div>
))

add(
    'Admin Sidebar Items',
    () => (
        <WebStory>
            {webProps => (
                <SiteAdminSidebar
                    {...webProps}
                    className="site-admin-sidebar"
                    groups={siteAdminSidebarGroups}
                    isSourcegraphDotCom={false}
                    batchChangesEnabled={false}
                />
            )}
        </WebStory>
    ),
    {
        design: {
            type: 'figma',
            url:
                'https://www.figma.com/file/cT23UcGqbdjdV4H2yxweIu/%2311775-Map-the-current-information-architecture-%5BApproved%5D?node-id=68%3A1',
        },
    }
)
