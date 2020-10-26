import { action } from '@storybook/addon-actions'
import { storiesOf } from '@storybook/react'
import React, { useState } from 'react'
import { Toggle } from './Toggle'
import webStyles from '../../../web/src/main.scss'
import { radios } from '@storybook/addon-knobs'

const onToggle = action('onToggle')

const { add } = storiesOf('branded/Toggle', module).addDecorator(story => {
    const theme = radios('Theme', { Light: 'light', Dark: 'dark' }, 'light')
    document.body.classList.toggle('theme-light', theme === 'light')
    document.body.classList.toggle('theme-dark', theme === 'dark')

    return (
        <>
            <div>{story()}</div>
            <style>{webStyles}</style>
        </>
    )
})

add(
    'Interactive',
    () => {
        const [value, setValue] = useState(false)

        const onToggle = (value: boolean) => setValue(value)

        return (
            <div className="d-flex align-items-center">
                <Toggle value={value} onToggle={onToggle} title="Hello" className="mr-2" /> Value is {String(value)}
            </div>
        )
    },
    {
        chromatic: {
            disable: true,
        },
    }
)

add('On', () => <Toggle value={true} onToggle={onToggle} />)

add('Off', () => <Toggle value={false} onToggle={onToggle} />)

add('Disabled & on', () => <Toggle value={true} disabled={true} onToggle={onToggle} />)

add('Disabled & off', () => <Toggle value={false} disabled={true} onToggle={onToggle} />)
