import { SettingsCascade } from '../settings/settings'
import { SettingsEdit } from './client/services/settings'
import * as extensionApiTypes from '@sourcegraph/extension-api-types'
import { Remote, ProxyMarked } from 'comlink'
import { Unsubscribable, DocumentHighlight } from 'sourcegraph'
import { ProxySubscribable } from './extension/api/common'
import { TextDocumentPositionParameters } from './protocol'
import { MaybeLoadingResult } from '@sourcegraph/codeintellify'
import { HoverMerged } from './client/types/hover'

/**
 * This is exposed from the extension host thread to the main thread
 * e.g. for communicating  direction "main -> ext host"
 * Note this API object lives in the extension host thread
 */
export interface FlatExtensionHostAPI {
    /**
     * Updates the settings exposed to extensions.
     */
    syncSettingsData: (data: Readonly<SettingsCascade<object>>) => void

    // Workspace
    addWorkspaceRoot: (root: extensionApiTypes.WorkspaceRoot) => void
    getWorkspaceRoots: () => extensionApiTypes.WorkspaceRoot[]
    removeWorkspaceRoot: (uri: string) => void
    setVersionContext: (versionContext: string | undefined) => void
    getVersionContext: () => string | undefined

    // Search
    transformSearchQuery: (query: string) => ProxySubscribable<string>

    // Languages
    getHover: (parameters: TextDocumentPositionParameters) => ProxySubscribable<MaybeLoadingResult<HoverMerged | null>>
    getDocumentHighlights: (parameters: TextDocumentPositionParameters) => ProxySubscribable<DocumentHighlight[]>
    getDefinition: (
        parameters: TextDocumentPositionParameters
    ) => ProxySubscribable<MaybeLoadingResult<extensionApiTypes.Location[]>>
}

/**
 * This is exposed from the main thread to the extension host thread"
 * e.g. for communicating  direction "ext host -> main"
 * Note this API object lives in the main thread
 */
export interface MainThreadAPI {
    /**
     * Applies a settings update from extensions.
     */
    applySettingsEdit: (edit: SettingsEdit) => Promise<void>

    // Commands
    executeCommand: (command: string, args: any[]) => Promise<any>
    registerCommand: (
        name: string,
        command: Remote<((...args: any) => any) & ProxyMarked>
    ) => Unsubscribable & ProxyMarked
}
