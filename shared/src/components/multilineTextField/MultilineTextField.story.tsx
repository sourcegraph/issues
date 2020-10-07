import { action } from '@storybook/addon-actions'
import { storiesOf } from '@storybook/react'
import React, { useState } from 'react'
import { MultilineTextField } from './MultilineTextField'

// tslint:disable: jsx-no-lambda

const onSubmit = action('onSubmit (of containing form)')

const { add } = storiesOf('MultilineTextField', module)

const MultilineTextFieldInteractive: React.FunctionComponent<
    { initialValue?: string } & Pick<Parameters<typeof MultilineTextField>[0], 'newlineOnShiftEnterKeypress'>
> = ({ initialValue = '', ...props }) => {
    const [value, setValue] = useState(initialValue)
    return (
        // tslint:disable-next-line: jsx-ban-elements
        <form
            onSubmit={e => {
                e.preventDefault()
                onSubmit(e)
            }}
        >
            <MultilineTextField
                {...props}
                className="m-2"
                value={value}
                onChange={e => setValue(e.currentTarget.value)}
            />
        </form>
    )
}

add('1 line', () => <MultilineTextFieldInteractive initialValue="Hello, world!" />)

add('newlineOnShiftEnterKeypress', () => (
    <MultilineTextFieldInteractive initialValue="Hello, world!" newlineOnShiftEnterKeypress={true} />
))

add('multiple lines', () => (
    <MultilineTextFieldInteractive initialValue={'Hello, Alice!\nHello, Bob!\nHello, Carol!\nHello, David!'} />
))
