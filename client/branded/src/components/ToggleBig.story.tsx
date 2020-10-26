import { action } from '@storybook/addon-actions'
import { storiesOf } from '@storybook/react'
import React, { useState } from 'react'
import { ToggleBig } from './ToggleBig'
import webStyles from '../../../web/src/main.scss'
import { radios } from '@storybook/addon-knobs'

const onToggle = action('onToggle')

const { add } = storiesOf('branded/ToggleBig', module).addDecorator(story => {
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
                <ToggleBig value={value} onToggle={onToggle} title="Hello" className="mr-2" /> Value is {String(value)}
            </div>
        )
    },
    {
        chromatic: {
            disable: true,
        },
    }
)

add('On', () => <ToggleBig value={true} onToggle={onToggle} />)

add('Off', () => <ToggleBig value={false} onToggle={onToggle} />)

add('Disabled & on', () => <ToggleBig value={true} disabled={true} onToggle={onToggle} />)

add('Disabled & off', () => <ToggleBig value={false} disabled={true} onToggle={onToggle} />)
