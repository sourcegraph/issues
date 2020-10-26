import { PlatformContext } from '../../platform/context'
<<<<<<< HEAD:shared/src/api/client/services.ts
=======
import { ReferenceParameters } from '../protocol'
>>>>>>> main:client/shared/src/api/client/services.ts
import { createContextService } from './context/contextService'
import { CommandRegistry } from './services/command'
import { ContributionRegistry } from './services/contribution'
import { LinkPreviewProviderRegistry } from './services/linkPreview'
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
    ) {}

    public readonly commands = new CommandRegistry()
    public readonly context = createContextService(this.platformContext)
    public readonly notifications = new NotificationsService()
    public readonly contribution = new ContributionRegistry(
        this.viewer,
        this.model,
        this.platformContext.settings,
        this.context.data
    )
    public readonly linkPreviews = new LinkPreviewProviderRegistry()
<<<<<<< HEAD:shared/src/api/client/services.ts
=======
    public readonly textDocumentDefinition = new TextDocumentLocationProviderRegistry()
    public readonly textDocumentReferences = new TextDocumentLocationProviderRegistry<ReferenceParameters>()
    public readonly textDocumentLocations = new TextDocumentLocationProviderIDRegistry()
    public readonly textDocumentDecoration = new TextDocumentDecorationProviderRegistry()
>>>>>>> main:client/shared/src/api/client/services.ts
    public readonly panelViews = new PanelViewProviderRegistry()
    public readonly view = createViewService()
}
