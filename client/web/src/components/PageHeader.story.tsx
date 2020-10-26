import { storiesOf } from '@storybook/react'
import React from 'react'
import webStyles from '../SourcegraphWebApp.scss'
import { PageHeader } from './PageHeader'
import PuzzleOutlineIcon from 'mdi-react/PuzzleOutlineIcon'
import { text } from '@storybook/addon-knobs'
import ClockOutlineIcon from 'mdi-react/ClockOutlineIcon'

const { add } = storiesOf('web/PageHeader', module).addDecorator(story => (
    <>
        <style>{webStyles}</style>
        <div className="layout__app-router-container">
            <div className="container mt-3">{story()}</div>
        </div>
    </>
))

add('Basic header', () => {
    const title = text('Title', 'Page name')
    return <PageHeader title={title} icon={PuzzleOutlineIcon} />
})

add(
    'Complex header',
    () => {
        const title = text('Title', 'Page name')
        return <PageHeader title={title} icon={ClockOutlineIcon} />
    },
    {
        design: {
            type: 'figma',
            url: 'https://www.figma.com/file/BkY8Ak997QauG0Iu2EqArv/Sourcegraph-Components?node-id=206%3A71',
        },
    }
)
