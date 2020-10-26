import { storiesOf } from '@storybook/react'
import { PrivateCodeCta } from './PrivateCodeCta'
import React from 'react'
import { WebStory } from '../../components/WebStory'

const { add } = storiesOf('web/PrivateCodeCta', module).addDecorator(story => <div className="p-4">{story()}</div>)

add('PrivateCodeCta', () => <WebStory>{() => <PrivateCodeCta />}</WebStory>, {
    design: {
        type: 'figma',
        url: 'https://www.figma.com/file/BkY8Ak997QauG0Iu2EqArv/Sourcegraph-Components?node-id=420%3A10',
    },
})
