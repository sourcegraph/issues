import React, { useState } from 'react'
import { storiesOf } from '@storybook/react'
import brandedStyles from '../../branded.scss'

import { OptionsPageAdvancedSettings } from './OptionsPageAdvancedSettings'

const optionFlags = [
    { key: 'one', label: 'Flag One', value: true },
    { key: 'two', label: 'Flag Two', value: false },
    { key: 'three', label: 'Flag Three', value: false },
]

storiesOf('browser/Options/OptionsPageAdvancedSettings', module)
    .addDecorator(story => (
        <>
            <style>{brandedStyles}</style>
            <div className="options-page options-page--full">{story()}</div>
        </>
    ))
    .add('Default', () => {
        const [optionFlagValues, setOptionFlagValues] = useState(optionFlags)
        const setOptionFlag = (key: string, value: boolean) => {
            setOptionFlagValues(optionFlagValues.map(flag => (flag.key === key ? { ...flag, value } : flag)))
        }

        return <OptionsPageAdvancedSettings optionFlags={optionFlagValues} onChangeOptionFlag={setOptionFlag} />
    })
