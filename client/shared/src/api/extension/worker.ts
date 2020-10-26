// eslint-disable-next-line import/extensions
import ExtensionHostWorker from './main.worker.ts'
import { EndpointPair, ClosableEndpointPair } from '../../platform/context'
import { Subscription } from 'rxjs'

/**
 * Creates a web worker with the extension host and sets up a bidirectional MessageChannel-based communication channel.
 *
 * If a `workerBundleURL` is provided, it is used to create a new Worker(), instead of using the ExtensionHostWorker
 * returned by worker-loader. This is useful to load the worker bundle from a different path.
 */
export function createExtensionHostWorker(workerBundleURL?: string): ClosableEndpointPair {
    const clientAPIChannel = new MessageChannel()
    const extensionHostAPIChannel = new MessageChannel()
    const worker = workerBundleURL ? new Worker(workerBundleURL) : new ExtensionHostWorker()
    const workerEndpoints: EndpointPair = {
        proxy: clientAPIChannel.port2,
        expose: extensionHostAPIChannel.port2,
    }
    worker.postMessage({ endpoints: workerEndpoints }, Object.values(workerEndpoints))
    const clientEndpoints = {
        proxy: extensionHostAPIChannel.port1,
        expose: clientAPIChannel.port1,
    }
    const subscription = new Subscription(() => {
        extensionHostAPIChannel.port1.close()
        extensionHostAPIChannel.port2.close()
        clientAPIChannel.port1.close()
        clientAPIChannel.port2.close()
    })
    return { ...clientEndpoints, subscription }
}
