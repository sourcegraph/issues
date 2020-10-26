import * as comlink from 'comlink'
import { isEqual, omit } from 'lodash'
import { combineLatest, from, ReplaySubject, Unsubscribable, ObservableInput, Subscription } from 'rxjs'
import { distinctUntilChanged, map, switchMap } from 'rxjs/operators'
import { PanelView, View } from 'sourcegraph'
import { ContributableViewContainer } from '../../protocol'
import { ViewerService, getActiveCodeEditorPosition } from '../services/viewerService'
import { PanelViewWithComponent, PanelViewProviderRegistry } from '../services/panelViews'
import { Location } from '@sourcegraph/extension-api-types'
import { MaybeLoadingResult } from '@sourcegraph/codeintellify'
import { ProxySubscribable } from '../../extension/api/common'
import { wrapRemoteObservable, ProxySubscription } from './common'
import { ViewService, ViewContexts } from '../services/viewService'
import { FlatExtHostAPI } from '../../contract'

/** @internal */
export interface PanelViewData extends Pick<PanelView, 'title' | 'content' | 'priority' | 'component'> {}

export interface PanelUpdater extends Unsubscribable, comlink.ProxyMarked {
    update(data: PanelViewData): void
}

/** @internal */
export interface ClientViewsAPI extends comlink.ProxyMarked {
    $registerPanelViewProvider(provider: { id: string }): PanelUpdater

    $registerDirectoryViewProvider(
        id: string,
        provider: comlink.Remote<
            ((context: ViewContexts[typeof ContributableViewContainer.Directory]) => ProxySubscribable<View | null>) &
                comlink.ProxyMarked
        >
    ): Unsubscribable & comlink.ProxyMarked

    $registerHomepageViewProvider(
        id: string,
        provider: comlink.Remote<
            ((context: ViewContexts[typeof ContributableViewContainer.Homepage]) => ProxySubscribable<View | null>) &
                comlink.ProxyMarked
        >
    ): Unsubscribable & comlink.ProxyMarked

    $registerInsightsPageViewProvider(
        id: string,
        provider: comlink.Remote<
            ((
                context: ViewContexts[typeof ContributableViewContainer.InsightsPage]
            ) => ProxySubscribable<View | null>) &
                comlink.ProxyMarked
        >
    ): Unsubscribable & comlink.ProxyMarked

    $registerGlobalPageViewProvider(
        id: string,
        provider: comlink.Remote<
            ((context: ViewContexts[typeof ContributableViewContainer.GlobalPage]) => ProxySubscribable<View | null>) &
                comlink.ProxyMarked
        >
    ): Unsubscribable & comlink.ProxyMarked
}

/** @internal */
export class ClientViews implements ClientViewsAPI {
    public readonly [comlink.proxyMarker] = true

    constructor(
        private panelViewRegistry: PanelViewProviderRegistry,
        private extensionHostAPI: comlink.Remote<FlatExtHostAPI>,
        private viewerService: ViewerService,
        private viewService: ViewService
    ) {}

    public $registerPanelViewProvider(provider: { id: string }): PanelUpdater {
        // TODO(sqs): This will probably hang forever if an extension neglects to set any of the fields on a
        // PanelView because this subject will never emit.
        const panelView = new ReplaySubject<PanelViewData>(1)
        const registryUnsubscribable = this.panelViewRegistry.registerProvider(
            { ...provider, container: ContributableViewContainer.Panel },
            combineLatest([
                panelView.pipe(
                    map(data => omit(data, 'component')),
                    distinctUntilChanged((a, b) => isEqual(a, b))
                ),
                panelView.pipe(
                    map(({ component }) => component),
                    distinctUntilChanged((a, b) => isEqual(a, b)),
                    map(component => {
                        if (!component) {
                            return undefined
                        }

                        return from(this.viewerService.activeViewerUpdates).pipe(
                            map(getActiveCodeEditorPosition),
                            switchMap(
                                (parameters): ObservableInput<MaybeLoadingResult<Location[]>> => {
                                    if (!parameters) {
                                        return [{ isLoading: false, result: [] }]
                                    }
                                    return wrapRemoteObservable(
                                        this.extensionHostAPI.getLocations(component.locationProvider, parameters)
                                    )
                                }
                            )
                        )
                    })
                ),
            ]).pipe(
                map(([{ title, content, priority }, locationProvider]) => {
                    const panelView: PanelViewWithComponent = {
                        title,
                        content,
                        priority,
                        locationProvider,
                    }
                    return panelView
                })
            )
        )
        return comlink.proxy({
            update: (data: PanelViewData) => {
                panelView.next(data)
            },
            unsubscribe: () => {
                registryUnsubscribable.unsubscribe()
            },
        })
    }

    public $registerDirectoryViewProvider(
        id: string,
        provider: comlink.Remote<
            (
                context: ViewContexts[typeof ContributableViewContainer.Directory]
            ) => ProxySubscribable<View | null> & comlink.ProxyMarked
        >
    ): Unsubscribable & comlink.ProxyMarked {
        const subscription = new Subscription()
        subscription.add(
            this.viewService.register(id, ContributableViewContainer.Directory, context =>
                wrapRemoteObservable(provider(context), subscription)
            )
        )
        subscription.add(new ProxySubscription(provider))
        return comlink.proxy(subscription)
    }

    public $registerHomepageViewProvider(
        id: string,
        provider: comlink.Remote<
            (
                context: ViewContexts[typeof ContributableViewContainer.Homepage]
            ) => ProxySubscribable<View | null> & comlink.ProxyMarked
        >
    ): Unsubscribable & comlink.ProxyMarked {
        const subscription = new Subscription()
        subscription.add(
            this.viewService.register(id, ContributableViewContainer.Homepage, context =>
                wrapRemoteObservable(provider(context), subscription)
            )
        )
        subscription.add(new ProxySubscription(provider))
        return comlink.proxy(subscription)
    }

    public $registerInsightsPageViewProvider(
        id: string,
        provider: comlink.Remote<
            (
                context: ViewContexts[typeof ContributableViewContainer.InsightsPage]
            ) => ProxySubscribable<View | null> & comlink.ProxyMarked
        >
    ): Unsubscribable & comlink.ProxyMarked {
        const subscription = new Subscription()
        subscription.add(
            this.viewService.register(id, ContributableViewContainer.InsightsPage, context =>
                wrapRemoteObservable(provider(context), subscription)
            )
        )
        subscription.add(new ProxySubscription(provider))
        return comlink.proxy(subscription)
    }

    public $registerGlobalPageViewProvider(
        id: string,
        provider: comlink.Remote<
            (
                context: ViewContexts[typeof ContributableViewContainer.GlobalPage]
            ) => ProxySubscribable<View | null> & comlink.ProxyMarked
        >
    ): Unsubscribable & comlink.ProxyMarked {
        const subscription = new Subscription()
        subscription.add(
            this.viewService.register(id, ContributableViewContainer.GlobalPage, context =>
                wrapRemoteObservable(provider(context), subscription)
            )
        )
        subscription.add(new ProxySubscription(provider))
        return comlink.proxy(subscription)
    }
}
