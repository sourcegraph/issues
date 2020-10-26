import { Key, ModifierKey } from '@slimsag/react-shortcuts'

/**
 * An action and its associated keybindings.
 */
export interface KeyboardShortcut {
    /** A unique ID for this keybinding. */
    id: string

    /** A descriptive title. */
    title: string

    /** The keybindings that trigger this shortcut. */
    keybindings: Keybinding[]

    /** If set, do not show this in the KeyboardShortcutsHelp modal. */
    hideInHelp?: boolean
}

/** A key sequence (that triggers a keyboard shortcut). */
export interface Keybinding {
    /** Keys that must be held down. */
    held?: ModifierKey[]

    /** Keys that must be pressed in order (when holding the `held` keys). */
    ordered: Key[]
}
