import { Remote } from 'comlink'
import {
    WorkspaceRoot,
    ContextValues,
    TextDocument,
    Unsubscribable,
    QueryTransformer,
    LinkPreviewProvider,
    Window as WindowS,
    DirectoryViewer,
    ViewComponent,
    NotificationType,
    InputBoxOptions,
    ProgressOptions,
    ProgressReporter,
    DocumentSelector,
    HoverProvider,
    DefinitionProvider,
    ReferenceProvider,
    LocationProvider,
    CompletionItemProvider,
} from 'sourcegraph'
import { ClientContextAPI } from '../client/api/context'
import * as clientType from '@sourcegraph/extension-api-types'

import { Subject, ReplaySubject, Observable, BehaviorSubject } from 'rxjs'

import { TextModelUpdate } from '../client/services/modelService'
import { ExtDocument } from './api/textDocument'
import { PanelViewData } from '../client/api/views'
import { SettingsCascade } from '../../settings/settings'
import { SettingsEdit } from '../client/services/settings'
import { CommandEntry } from '../client/services/command'
import { ViewerUpdate } from '../client/services/viewerService'
import { ExtCodeEditor } from './api/codeEditor'

// ExtLanguageFeatures

// --------------------------------------
// State held by Extension host
export interface State {
    // ExtDocuments
    documents: Map<string, ExtDocument>

    // ExtExtensions
    extensionDeactivate: Map<string, () => void | Promise<void>>

    // ExtRoots
    roots: readonly WorkspaceRoot[]

    // ExtViews
    // implicit panelViews: Map<string, ExtPanelView>

    // ExtConfiguration
    config: Readonly<SettingsCascade<object>>

    // ExtWindows
    activeWindow: {
        viewComponents: Map<string, ExtCodeEditor | DirectoryViewer>
    }
}

// --------------------------------------
// tracking changes (derived data)
export interface Updates {
    // ExtDocuments
    openedTextDocuments: Subject<TextDocument>

    // ExtRoots
    rootChanges: Subject<void>

    // ExtConfiguration
    configChanges: ReplaySubject<void>

    // ExtWindows
    activeWindowChanges: Observable<WindowS>
    activeViewComponentChanges: BehaviorSubject<ViewComponent | undefined>
}

// --------------------------------------
// Worker -> Main
export interface ToMainThread {
    // ExtContext
    updateContext: (proxy: Remote<ClientContextAPI>, updates: ContextValues) => void

    // ExtViews
    registerPanelViewProvider: (
        id: string
    ) => {
        // returns an object that remembers the id here
        // NOTE implicit state here
        updatePanel: (data: PanelViewData) => void // EXTRA proxy
    }
    registerDirectoryViewProvider: (id: string, provider: object) => Unsubscribable // EXTRA proxy
    registerGlobalPageViewProvider: (id: string, provider: object) => Unsubscribable // EXTRA proxy

    // ExtConfiguration
    requestConfigurationUpdate: (edit: SettingsEdit) => Promise<void>

    // ExtSearch
    registerQueryTransformer: (provider: QueryTransformer) => Unsubscribable // EXTRA proxy

    // ExtCommands
    executeCommand: (command: string, args: any[]) => Promise<any>
    registerCommand: (entry: CommandEntry) => Unsubscribable // EXTRA proxy

    // ExtContent
    registerLinkPreviewProvider: (urlMatchPattern: string, provider: LinkPreviewProvider) => Unsubscribable // EXTRA proxy

    // ExtWindows
    windows: {
        showNotification: (message: string, type: NotificationType) => void
        showMessage: (message: string) => Promise<void>
        showInputBox: (options?: InputBoxOptions) => Promise<string | undefined>
        showProgress: (options: ProgressOptions) => Promise<ProgressReporter> // EXTRA proxy
    }

    // ExtLanguageFeatures
    registerHoverProvider: (selector: DocumentSelector, provider: HoverProvider) => Unsubscribable // EXTRA proxy
    registerDefinitionProvider: (selector: DocumentSelector, provider: DefinitionProvider) => Unsubscribable // EXTRA proxy
    registerReferenceProvider: (selector: DocumentSelector, provider: ReferenceProvider) => Unsubscribable // EXTRA proxy
    registerLocationProvider: (id: string, selector: DocumentSelector, provider: LocationProvider) => Unsubscribable // EXTRA proxy
    registerCompletionItemProvider: (selector: DocumentSelector, provider: CompletionItemProvider) => Unsubscribable // EXTRA proxy
}

// --------------------------------------
// Main -> Worker
export interface FromMainThread {
    // ExtDocuments
    updateDocumentData: (modelUpdates: readonly TextModelUpdate[]) => void

    // ExtExtensions
    activateExtension: (extensionID: string, bundleURL: string) => Promise<void>
    deactivateExtension: (extensionID: string) => Promise<void>

    // ExtRoots
    acceptRoots(roots: clientType.WorkspaceRoot[]): void

    // ExtConfiguration
    updateConfigurationData: (data: Readonly<SettingsCascade<object>>) => void

    // ExtWindows
    acceptWindowData: (viewerUpdates: ViewerUpdate[]) => void
}
