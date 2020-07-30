import { storiesOf } from '@storybook/react'
import { radios } from '@storybook/addon-knobs'
import React from 'react'
import { CampaignNode } from './CampaignNode'
import { createMemoryHistory } from 'history'
import webStyles from '../../../enterprise.scss'
import { Tooltip } from '../../../components/tooltip/Tooltip'
import isChromatic from 'chromatic/isChromatic'
import { ListCampaign } from '../../../graphql-operations'

export const nodes: Record<string, ListCampaign> = {
    'Open campaign': {
        id: 'test',
        name: 'Awesome campaign',
        description: `# Description

This is my thorough explanation. And it can also get very long, in that case the UI doesn't break though, which is good. And one more line to finally be longer than the viewport.`,
        createdAt: new Date('2020-05-05').toISOString(),
        closedAt: null,
        changesets: {
            stats: {
                open: 10,
                closed: 0,
                merged: 5,
            },
        },
        author: {
            username: 'alice',
        },
    },
    'No description': {
        id: 'test2',
        name: 'Awesome campaign',
        description: null,
        createdAt: new Date('2020-05-05').toISOString(),
        closedAt: null,
        changesets: {
            stats: {
                open: 10,
                closed: 0,
                merged: 5,
            },
        },
        author: {
            username: 'alice',
        },
    },
    'Closed campaign': {
        id: 'test3',
        name: 'Awesome campaign',
        description: `# Description

        This is my thorough explanation.`,
        createdAt: new Date('2020-05-05').toISOString(),
        closedAt: new Date('2020-06-05').toISOString(),
        changesets: {
            stats: {
                open: 0,
                closed: 10,
                merged: 5,
            },
        },
        author: {
            username: 'alice',
        },
    },
}

const { add } = storiesOf('web/campaigns/CampaignNode', module).addDecorator(story => {
    const theme = radios('Theme', { Light: 'light', Dark: 'dark' }, 'light')
    document.body.classList.toggle('theme-light', theme === 'light')
    document.body.classList.toggle('theme-dark', theme === 'dark')
    return (
        <>
            <Tooltip />
            <style>{webStyles}</style>
            <div className="p-3 container">{story()}</div>
        </>
    )
})

for (const key of Object.keys(nodes)) {
    add(key, () => (
        <CampaignNode
            node={nodes[key]}
            now={isChromatic() ? new Date('2020-05-05') : undefined}
            history={createMemoryHistory()}
        />
    ))
}
