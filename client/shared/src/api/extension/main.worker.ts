import '../../polyfills'

import { fromEvent } from 'rxjs'
import { take } from 'rxjs/operators'
import { isEndpointPair } from '../../platform/context'
import { startExtensionHost } from './extensionHost'
import { hasProperty } from '../../util/types'

interface InitMessage {
    endpoints: {
        proxy: MessagePort
        expose: MessagePort
    }
}

const isInitMessage = (value: unknown): value is InitMessage =>
    typeof value === 'object' && value !== null && hasProperty('endpoints')(value) && isEndpointPair(value.endpoints)

/**
 * The entrypoint for the JavaScript context that runs the extension host (and all extensions).
 *
 * To initialize the extension host, the parent sends it an {@link InitMessage}
 */
async function extensionHostMain(): Promise<void> {
    try {
        const event = await fromEvent<MessageEvent>(self, 'message').pipe(take(1)).toPromise()
        if (!isInitMessage(event.data)) {
            throw new Error('First message event in extension host worker was not a well-formed InitMessage')
        }
        const { endpoints } = event.data
        const extensionHost = startExtensionHost(endpoints)
        self.addEventListener('unload', () => extensionHost.unsubscribe())
    } catch (error) {
        console.error('Error starting the extension host:', error)
        self.close()
    }
}

// eslint-disable-next-line @typescript-eslint/no-floating-promises
extensionHostMain()
