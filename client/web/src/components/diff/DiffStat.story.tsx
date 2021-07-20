import { number } from '@storybook/addon-knobs'
import { storiesOf } from '@storybook/react'
import React from 'react'

import { WebStory } from '../WebStory'

import { DiffStat, DiffStatSquares, DiffStatStack } from './DiffStat'

const getSharedKnobs = () => ({
    added: number('Added', 10),
    changed: number('Changed', 4),
    deleted: number('Deleted', 8),
})

const { add } = storiesOf('web/diffs/DiffStat', module).addDecorator(story => (
    <div className="p-3 container">{story()}</div>
))

add('Collapsed counts', () => {
    const stats = getSharedKnobs()
    return <WebStory>{() => <DiffStat {...stats} />}</WebStory>
})

add('Expanded counts', () => {
    const stats = getSharedKnobs()
    return <WebStory>{() => <DiffStat {...stats} expandedCounts={true} />}</WebStory>
})

add('DiffStatSquares', () => {
    const stats = getSharedKnobs()
    return <WebStory>{() => <DiffStatSquares {...stats} />}</WebStory>
})
add('DiffStatStack', () => {
    const stats = getSharedKnobs()
    return <WebStory>{() => <DiffStatStack {...stats} />}</WebStory>
})
