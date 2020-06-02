import * as comlink from 'comlink'
import * as sourcegraph from 'sourcegraph'
import { ClientViewsAPI, PanelUpdater, PanelViewData } from '../../client/api/views'
import { synchronousSubscription } from '../../util'
import { Unsubscribable } from 'rxjs'
import { toProxyableSubscribable } from './common'
import { ContributableViewContainer } from '../../protocol'
import { ViewContexts } from '../../client/services/viewService'

/**
 * @internal
 */
class ExtPanelView implements sourcegraph.PanelView {
    private data: PanelViewData = {
        title: '',
        content: '',
        priority: 0,
        component: null,
    }

    constructor(private proxyPromise: Promise<comlink.Remote<PanelUpdater>>) {}

    public get title(): string {
        return this.data.title
    }
    public set title(value: string) {
        this.data.title = value
        this.sendData()
    }

    public get content(): string {
        return this.data.content
    }
    public set content(value: string) {
        this.data.content = value
        this.sendData()
    }

    public get priority(): number {
        return this.data.priority
    }
    public set priority(value: number) {
        this.data.priority = value
        this.sendData()
    }

    public get component(): sourcegraph.PanelView['component'] {
        return this.data.component
    }
    public set component(value: sourcegraph.PanelView['component']) {
        this.data.component = value
        this.sendData()
    }

    private sendData(): void {
        // eslint-disable-next-line @typescript-eslint/no-floating-promises
        this.proxyPromise.then(proxy => proxy.update(this.data))
    }

    public unsubscribe(): void {
        // eslint-disable-next-line @typescript-eslint/no-floating-promises
        this.proxyPromise.then(proxy => proxy.unsubscribe())
    }
}

/** @internal */
export class ExtViews implements comlink.ProxyMarked {
    public readonly [comlink.proxyMarker] = true

    constructor(private proxy: comlink.Remote<ClientViewsAPI>) {}

    public createPanelView(id: string): ExtPanelView {
        const panelProxyPromise = this.proxy.$registerPanelViewProvider({ id })
        return new ExtPanelView(panelProxyPromise)
    }

    public registerViewProvider(id: string, provider: sourcegraph.ViewProvider): Unsubscribable {
        switch (provider.where) {
            case ContributableViewContainer.Directory: {
                return synchronousSubscription(
                    this.proxy.$registerDirectoryViewProvider(
                        id,
                        comlink.proxy((context: ViewContexts[typeof ContributableViewContainer.Directory]) =>
                            toProxyableSubscribable(
                                provider.provideView({
                                    viewer: {
                                        ...context.viewer,
                                        directory: {
                                            ...context.viewer.directory,
                                            uri: new URL(context.viewer.directory.uri),
                                        },
                                    },
                                    workspace: {
                                        uri: new URL(context.workspace.uri),
                                    },
                                }),
                                result => result || null
                            )
                        )
                    )
                )
            }
            case ContributableViewContainer.GlobalPage: {
                return synchronousSubscription(
                    this.proxy.$registerGlobalPageViewProvider(
                        id,
                        comlink.proxy((context: ViewContexts[typeof ContributableViewContainer.GlobalPage]) =>
                            toProxyableSubscribable(provider.provideView(context), result => result || null)
                        )
                    )
                )
            }
        }
    }
}
