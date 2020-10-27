import { Subscription } from 'rxjs'
import * as uuid from 'uuid'
import { EndpointPair, ClosableEndpointPair } from '../../../../shared/src/platform/context'
import { isInPage } from '../context'
import { SourcegraphIntegrationURLs } from './context'
import { browserPortToMessagePort } from './ports'

function createInPageExtensionHost({
    assetsURL,
}: Pick<SourcegraphIntegrationURLs, 'assetsURL'>): Promise<ClosableEndpointPair> {
    return new Promise(resolve => {
        // Create an iframe pointing to extensionHostFrame.html,
        // which will load the extension host worker, and forward it
        // the client endpoints.
        const frame: HTMLIFrameElement = document.createElement('iframe')
        frame.setAttribute('src', new URL('extensionHostFrame.html', assetsURL).href)
        frame.setAttribute('style', 'display: none;')
        document.body.append(frame)
        const clientAPIChannel = new MessageChannel()
        const extensionHostAPIChannel = new MessageChannel()
        const workerEndpoints: EndpointPair = {
            proxy: clientAPIChannel.port2,
            expose: extensionHostAPIChannel.port2,
        }
        const clientEndpoints = {
            proxy: extensionHostAPIChannel.port1,
            expose: clientAPIChannel.port1,
        }
        const subscription = new Subscription(() => {
            clientAPIChannel.port1.close()
            clientAPIChannel.port2.close()
            extensionHostAPIChannel.port1.close()
            extensionHostAPIChannel.port2.close()
        })
        // Subscribe to the load event on the frame
        frame.addEventListener(
            'load',
            () => {
                frame.contentWindow!.postMessage(
                    {
                        type: 'workerInit',
                        payload: {
                            endpoints: clientEndpoints,
                            wrapEndpoints: false,
                        },
                    },
                    new URL(assetsURL).origin,
                    Object.values(clientEndpoints)
                )
                resolve({ ...workerEndpoints, subscription })
            },
            {
                once: true,
            }
        )
    })
}

/**
 * Returns a promise of a communication channel to an extension host and a Subscription to cleanup
 *
 * When executing in-page (for example as a Phabricator plugin), this simply
 * creates an extension host worker and emits the returned EndpointPair + Subscription to cleanup.
 *
 * When executing in the browser extension, we create pair of browser.runtime.Port objects,
 * named 'expose-{uuid}' and 'proxy-{uuid}', and return the ports wrapped using ${@link endpointFromPort}.
 *
 * The background script will listen to newly created ports, create an extension host
 * worker per pair of ports, and forward messages between the port objects and
 * the extension host worker's endpoints.
 */
export async function createExtensionHost(
    urls: Pick<SourcegraphIntegrationURLs, 'assetsURL'>
): Promise<ClosableEndpointPair> {
    if (isInPage) {
        return createInPageExtensionHost(urls)
    }
    const id = uuid.v4()

    // This is run in the content script
    const subscription = new Subscription()
    const setup = (role: keyof EndpointPair): MessagePort => {
        const port = browser.runtime.connect({ name: `${role}-${id}` })
        subscription.add(() => port.disconnect())

        const link = browserPortToMessagePort(port, `comlink-${role}-`, name => browser.runtime.connect({ name }))
        subscription.add(link.subscription)
        return link.messagePort
    }

    return {
        proxy: setup('proxy'),
        expose: setup('expose'),
        subscription,
    }
}
