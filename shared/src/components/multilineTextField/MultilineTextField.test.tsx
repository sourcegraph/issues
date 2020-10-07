import React from 'react'
import { cleanup, fireEvent, render } from 'react-testing-library'
import sinon from 'sinon'
import { Key } from 'ts-key-enum'
import { MultilineTextField } from './MultilineTextField'

describe('MultilineTextField', () => {
    afterAll(cleanup)

    const renderComponentInForm = (
        props: Pick<Parameters<typeof MultilineTextField>[0], 'newlineOnShiftEnterKeypress' | 'value'>
    ) => {
        const onSubmit = sinon.spy()
        const { container } = render(
            // tslint:disable-next-line: jsx-ban-elements
            <form onSubmit={onSubmit}>
                <MultilineTextField {...props} />
            </form>
        )
        // tslint:disable-next-line: no-unnecessary-type-assertion
        return { textArea: container.querySelector('textarea')!, onSubmit }
    }

    const fireKeyPress = ({ textArea, shiftKey }: { textArea: HTMLTextAreaElement; shiftKey?: boolean }) =>
        fireEvent.keyPress(textArea, {
            key: Key.Enter,
            shiftKey,
            // Need to set code and charCode (see
            // https://github.com/testing-library/react-testing-library/issues/269#issuecomment-455854112).
            code: 13,
            charCode: 13,
        })

    test('Enter submits', () => {
        const { textArea, onSubmit } = renderComponentInForm({ value: 'a' })
        fireKeyPress({ textArea })
        expect(textArea.value).toBe('a')
        expect(onSubmit.calledOnce).toBe(true)
    })

    describe('newlineOnShiftEnterKeypress', () => {
        test('== false: Shift+Enter submits', () => {
            const { textArea, onSubmit } = renderComponentInForm({
                newlineOnShiftEnterKeypress: false,
                value: 'a',
            })
            fireKeyPress({ textArea, shiftKey: true })
            expect(textArea.value).toBe('a')
            expect(onSubmit.calledOnce).toBe(true)
        })

        test('== true: Shift+Enter adds newline', async () => {
            const { textArea, onSubmit } = renderComponentInForm({
                newlineOnShiftEnterKeypress: true,
                value: 'a',
            })
            fireKeyPress({ textArea, shiftKey: true })
            expect(onSubmit.callCount).toBe(0)
        })

        test('== true: Enter submits', () => {
            const { textArea, onSubmit } = renderComponentInForm({
                newlineOnShiftEnterKeypress: true,
                value: 'a',
            })
            fireKeyPress({ textArea })
            expect(textArea.value).toBe('a')
            expect(onSubmit.calledOnce).toBe(true)
        })
    })
})
