import { Context, ContextProvider, ProviderProps, ShortcutManager } from '@slimsag/react-shortcuts'
import * as React from 'react'

/**
 * Describes the variable this file injects into the `global` object. It is
 * heavily prefixed to avoid collisions.
 */
interface GlobalContext {
    /** The singleton ShortcutManager object. */
    browserExtensionShortcutManager?: ShortcutManager
}

// This ShortcutProvider is derived from the default @shopify/react-shortcuts
// implementation:
//
// https://github.com/Shopify/quilt/blob/master/packages/react-shortcuts/src/ShortcutProvider/ShortcutProvider.tsx
//
// We cannot use the default implementation above because it assumes the
// application is rendered via a single React component in order to create the
// ShortcutManager singleton. In our case, there are multiple React components
// on the page and they each need to use a single ShortcutManager, so we must
// manage it ourselves here. If we did not do this, we would have multiple
// ShortcutManagers and each would register their own conflicting document
// event handlers.
export class ShortcutProvider extends React.Component<ProviderProps, never> {
    public componentDidMount(): void {
        const globals = global as GlobalContext
        if (!globals.browserExtensionShortcutManager) {
            globals.browserExtensionShortcutManager = new ShortcutManager()
            globals.browserExtensionShortcutManager.setup()
        }
    }

    public render(): JSX.Element | null {
        const globals = global as GlobalContext
        const context: Context = {
            shortcutManager: globals.browserExtensionShortcutManager,
        }

        return <ContextProvider value={context}>{this.props.children}</ContextProvider>
    }
}
