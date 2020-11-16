import { PlatformContext } from '../../platform/context'
import { ReferenceParameters } from '../protocol'
import { createContextService } from './context/contextService'
import { CommandRegistry } from './services/command'
import { CompletionItemProviderRegistry } from './services/completion'
import { ContributionRegistry } from './services/contribution'
import { TextDocumentDecorationProviderRegistry } from './services/decoration'
import { createViewerService } from './services/viewerService'
import { IExtensionsService, ExtensionsService } from './services/extensionsService'
import { LinkPreviewProviderRegistry } from './services/linkPreview'
import { TextDocumentLocationProviderIDRegistry, TextDocumentLocationProviderRegistry } from './services/location'
import { createModelService } from './services/modelService'
import { NotificationsService } from './services/notifications'
import { PanelViewProviderRegistry } from './services/panelViews'
import { createViewService } from './services/viewService'

/**
 * Services is a container for all services used by the client application.
 */
export class Services {
    constructor(
        private platformContext: Pick<
            PlatformContext,
            | 'settings'
            | 'updateSettings'
            | 'requestGraphQL'
            | 'getScriptURLForExtension'
            | 'clientApplication'
            | 'sideloadedExtensionURL'
            | 'createExtensionsService'
        >
    ) {
        if (platformContext.createExtensionsService) {
            this.extensions = platformContext.createExtensionsService(this.model)
        } else {
            this.extensions = new ExtensionsService(platformContext, this.model)
        }
    }

    public readonly commands = new CommandRegistry()
    public readonly context = createContextService(this.platformContext)
    public readonly model = createModelService()
    public readonly viewer = createViewerService(this.model)
    public readonly notifications = new NotificationsService()
    public readonly contribution = new ContributionRegistry(
        this.viewer,
        this.model,
        this.platformContext.settings,
        this.context.data
    )
    public readonly extensions: IExtensionsService
    public readonly linkPreviews = new LinkPreviewProviderRegistry()
    public readonly textDocumentReferences = new TextDocumentLocationProviderRegistry<ReferenceParameters>()
    public readonly textDocumentLocations = new TextDocumentLocationProviderIDRegistry()
    public readonly textDocumentDecoration = new TextDocumentDecorationProviderRegistry()
    public readonly panelViews = new PanelViewProviderRegistry()
    public readonly completionItems = new CompletionItemProviderRegistry()
    public readonly view = createViewService()
}
