import { storiesOf } from '@storybook/react'
import React from 'react'
import { WebStory } from '../components/WebStory'
import { OrDivider } from './OrDivider'

const { add } = storiesOf('web/OrDivider', module).addDecorator(story => (
    <div className="p-3 container web-content">{story()}</div>
))

add('Alone', () => (
    <WebStory>
        {() => (
            <div className="card border-0">
                <OrDivider />
            </div>
        )}
    </WebStory>
))
