import { storiesOf } from '@storybook/react'
import React from 'react'
import { LoaderInput } from './LoaderInput'
import { BrandedStory } from './BrandedStory'
import { boolean } from '@storybook/addon-knobs'

const { add } = storiesOf('branded/LoaderInput', module).addDecorator(story => (
    <div className="container mt-3" style={{ width: 800 }}>
        {story()}
    </div>
))

add('Interactive', () => (
    <BrandedStory>
        {() => (
            <div>
                <LoaderInput loading={boolean('loading', false)}>
                    <input type="text" placeholder="Loader input" className="form-control" />
                </LoaderInput>
            </div>
        )}
    </BrandedStory>
))
