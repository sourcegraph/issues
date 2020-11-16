import 'message-port-polyfill'

import { BehaviorSubject, from, throwError, of, Subscription } from 'rxjs'
import { filter, first, switchMap, take } from 'rxjs/operators'
import * as sourcegraph from 'sourcegraph'
import { EndpointPair, PlatformContext } from '../../platform/context'
import { isDefined } from '../../util/types'
import { createExtensionHostClientConnection } from '../client/connection'
import { Services } from '../client/services'
import { ViewerData } from '../client/services/viewerService'
import { TextModel } from '../client/services/modelService'
import { InitData, startExtensionHost } from '../extension/extensionHost'
import { FlatExtensionHostAPI } from '../contract'
import { Remote } from 'comlink'
import * as extensionApiTypes from '@sourcegraph/extension-api-types'

export function assertToJSON(a: any, expected: any): void {
    const raw = JSON.stringify(a)
    const actual = JSON.parse(raw)
    expect(actual).toEqual(expected)
}

interface TestInitData {
    roots: readonly extensionApiTypes.WorkspaceRoot[]
    models?: readonly TextModel[]
    viewers: readonly ViewerData[]
}

const FIXTURE_INIT_DATA: TestInitData = {
    roots: [{ uri: 'file:///' }],
    models: [{ uri: 'file:///f', text: 't', languageId: 'l' }],
    viewers: [
        {
            type: 'CodeEditor',
            resource: 'file:///f',
            selections: [],
            isActive: true,
        },
    ],
}

interface Mocks
    extends Pick<
        PlatformContext,
        | 'settings'
        | 'updateSettings'
        | 'requestGraphQL'
        | 'getScriptURLForExtension'
        | 'clientApplication'
        | 'sideloadedExtensionURL'
    > {}

const NOOP_MOCKS: Mocks = {
    settings: of({ final: {}, subjects: [] }),
    updateSettings: () => Promise.reject(new Error('Mocks#updateSettings not implemented')),
    requestGraphQL: () => throwError(new Error('Mocks#queryGraphQL not implemented')),
    getScriptURLForExtension: scriptURL => scriptURL,
    clientApplication: 'sourcegraph',
    sideloadedExtensionURL: new BehaviorSubject<string | null>(null),
}

/**
 * Set up a new client-extension integration test.
 *
 * @internal
 */
export async function integrationTestContext(
    partialMocks: Partial<Mocks> = NOOP_MOCKS,
    initModel: TestInitData = FIXTURE_INIT_DATA
): Promise<{
    extensionAPI: typeof sourcegraph
    services: Services
    extensionHost: Remote<FlatExtensionHostAPI>
}> {
    const mocks = partialMocks ? { ...NOOP_MOCKS, ...partialMocks } : NOOP_MOCKS

    const clientAPIChannel = new MessageChannel()
    const extensionHostAPIChannel = new MessageChannel()
    const extensionHostEndpoints: EndpointPair = {
        proxy: clientAPIChannel.port2,
        expose: extensionHostAPIChannel.port2,
    }
    const clientEndpoints: EndpointPair = {
        proxy: extensionHostAPIChannel.port1,
        expose: clientAPIChannel.port1,
    }

    const extensionHost = startExtensionHost(extensionHostEndpoints)

    const services = new Services(mocks)
    const initData: Omit<InitData, 'initialSettings'> = {
        sourcegraphURL: 'https://example.com/',
        clientApplication: 'sourcegraph',
    }

    const { api } = await createExtensionHostClientConnection(
        Promise.resolve({
            endpoints: clientEndpoints,
            subscription: new Subscription(),
        }),
        services,
        initData,
        mocks
    )

    const extensionAPI = await extensionHost.extensionAPI
    if (initModel.models) {
        for (const model of initModel.models) {
            services.model.addModel(model)
        }
    }
    for (const editor of initModel.viewers) {
        services.viewer.addViewer(editor)
    }

    for (const root of initModel.roots) {
        await api.addWorkspaceRoot(root)
    }

    // Wait for initModel to be initialized
    if (initModel.viewers.length > 0) {
        await Promise.all([
            from(extensionAPI.workspace.openedTextDocuments).pipe(take(initModel.viewers.length)).toPromise(),
            from(extensionAPI.app.activeWindowChanges)
                .pipe(
                    first(isDefined),
                    switchMap(activeWindow =>
                        from(activeWindow.activeViewComponentChanges).pipe(
                            filter(isDefined),
                            take(initModel.viewers.length)
                        )
                    )
                )
                .toPromise(),
        ])
    }

    return {
        extensionAPI,
        services,
        extensionHost: api,
    }
}

/**
 * Returns a {@link Promise} and a function. The {@link Promise} blocks until the returned function is called.
 *
 * @internal
 */
export function createBarrier(): { wait: Promise<void>; done: () => void } {
    let done!: () => void
    const wait = new Promise<void>(resolve => (done = resolve))
    return { wait, done }
}

export function collectSubscribableValues<T>(subscribable: sourcegraph.Subscribable<T>): T[] {
    const values: T[] = []
    subscribable.subscribe(value => values.push(value))
    return values
}
