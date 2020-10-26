import { Shortcut } from '@slimsag/react-shortcuts'
import React, { useCallback, useState } from 'react'
import { Modal } from 'reactstrap'
import { KeyboardShortcut } from '../../../shared/src/keyboardShortcuts'
import { KeyboardShortcutsProps } from './keyboardShortcuts'

interface Props extends KeyboardShortcutsProps {
    /** The keyboard shortcut to show this modal. */
    keyboardShortcutForShow: KeyboardShortcut
}

/**
 * Keyboard shortcuts that are implemented in a legacy way, not using the central keyboard shortcuts
 * registry. These are shown in the help modal.
 */
const LEGACY_KEYBOARD_SHORTCUTS: KeyboardShortcut[] = [
    {
        id: 'canonicalURL',
        title: 'Expand URL to its canonical form (on file or tree page)',
        keybindings: [{ ordered: ['y'] }],
    },
]

export const KeyboardShortcutsHelp: React.FunctionComponent<Props> = ({
    keyboardShortcutForShow,
    keyboardShortcuts,
}) => {
    const [isOpen, setIsOpen] = useState(false)
    const toggleIsOpen = useCallback(() => setIsOpen(!isOpen), [isOpen])

    return (
        <>
            {keyboardShortcutForShow.keybindings.map((keybinding, index) => (
                <Shortcut key={index} {...keybinding} onMatch={toggleIsOpen} />
            ))}
            <Modal isOpen={isOpen} toggle={toggleIsOpen} centered={true} autoFocus={true} keyboard={true} fade={false}>
                <div className="modal-header">
                    <h4 className="modal-title">Keyboard shortcuts</h4>
                    <button
                        type="button"
                        className="btn btn-icon"
                        data-dismiss="modal"
                        aria-label="Close"
                        onClick={toggleIsOpen}
                    >
                        <span aria-hidden="true">&times;</span>
                    </button>
                </div>
                <div className="modal-body">
                    <ul className="list-group list-group-flush">
                        {[...keyboardShortcuts, ...LEGACY_KEYBOARD_SHORTCUTS]
                            .filter(({ hideInHelp }) => !hideInHelp)
                            .map(({ id, title, keybindings }) => (
                                <li
                                    key={id}
                                    className="list-group-item d-flex align-items-center justify-content-between"
                                >
                                    {title}
                                    <span>
                                        {keybindings.map((keybinding, index) => (
                                            <span key={index}>
                                                {index !== 0 && ' or '}
                                                {[...(keybinding.held || []), ...keybinding.ordered].map(
                                                    (key, index) => (
                                                        <kbd key={index}>{key}</kbd>
                                                    )
                                                )}
                                            </span>
                                        ))}
                                    </span>
                                </li>
                            ))}
                    </ul>
                </div>
            </Modal>
        </>
    )
}
