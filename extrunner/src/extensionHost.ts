import { startExtensionHost } from '../../shared/src/api/extension/extensionHost'
import * as MessageChannelAdapter from '@sourcegraph/comlink/messagechanneladapter'

const proxyConn = new WebSocket('ws://localhost:11001/proxy')
proxyConn.onopen = () => {
    console.info('Proxy connection open')
    const exposeConn = new WebSocket('ws://localhost:11001/expose')
    exposeConn.onopen = () => {
        console.info('Expose connection open')
        const endpoints = {
            expose: MessageChannelAdapter.wrap(proxyConn as MessageChannelAdapter.StringMessageChannel),
            proxy: MessageChannelAdapter.wrap(exposeConn as MessageChannelAdapter.StringMessageChannel),
        }

        console.info('Starting extension host')
        startExtensionHost(endpoints)
    }
}
