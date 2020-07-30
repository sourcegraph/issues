import React from 'react'
import { ChangesetLabel } from './ChangesetLabel'
import { mount } from 'enzyme'

describe('ChangesetLabel', () => {
    test('renders a light label with dark text', () => {
        expect(
            mount(
                <ChangesetLabel
                    label={{
                        text: 'bug',
                        description: 'Something is wrong',
                        color: 'acfc99',
                    }}
                />
            )
        ).toMatchSnapshot()
    })
    test('renders a dark label with white text', () => {
        expect(
            mount(
                <ChangesetLabel
                    label={{
                        text: 'bug',
                        description: 'Something is wrong',
                        color: '330912',
                    }}
                />
            )
        ).toMatchSnapshot()
    })
})
