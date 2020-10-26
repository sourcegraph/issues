import React from 'react'
import renderer from 'react-test-renderer'
import { Modal } from 'reactstrap'
import { KeyboardShortcutsHelp } from './KeyboardShortcutsHelp'

describe('KeyboardShortcutsHelp', () => {
    test('', () => {
        const output = renderer.create(
            <KeyboardShortcutsHelp
                keyboardShortcuts={[
                    {
                        id: 'x',
                        title: 't',
                        keybindings: [{ held: ['Alt'], ordered: ['x'] }],
                    },
                ]}
                keyboardShortcutForShow={{
                    id: 'x',
                    title: 't',
                    keybindings: [{ held: ['Alt'], ordered: ['x'] }],
                }}
            />
        )
        // Modal is hidden by default and uses portal, so we can't easily test its contents. Grab
        // its inner .modal-body and snapshot that instead.
        expect(renderer.create(output.root.findByType(Modal).props.children[1]).toJSON()).toMatchSnapshot()
    })
})
