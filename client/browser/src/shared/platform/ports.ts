import { isObject } from 'lodash'
import * as uuid from 'uuid'
import type { Message, MessageType } from 'comlink/dist/esm/protocol'
import { Subscription } from 'rxjs'

/** Comlink enum value of release messages. */
const RELEASE_MESSAGE_TYPE: MessageType.RELEASE = 5

/**
 * Returns a `MessagePort` that was connected to the given `browser.runtime.Port` so that it can be used as an endpoint
 * with comlink over browser extension script boundaries.
 *
 * `browser.runtime.Port` objects do not support transfering `MessagePort` objects, which comlink relies on.
 * A new `browser.runtime.Port`, with an associated unique ID, will be created for each `MessagePort` transfer.
 * The ID is added to the message and the original `MessagePort` is removed from it.
 *
 * @param browserPort The browser extension Port to link
 * @param prefix A prefix unique to this call of `browserPortToMessagePort` (but the same on both sides) to prefix Port names. Incoming ports not matching this prefix will be ignored.
 * @param connect A callback that is called to create a new connection to the other side of `browserPort` (e.g. through `browser.runtime.connect()` or `browser.tabs.connect()`).
 */
export function browserPortToMessagePort(
    browserPort: browser.runtime.Port,
    prefix: string,
    connect: (name: string) => browser.runtime.Port
): {
    messagePort: MessagePort
    subscription: Subscription
} {
    const rootSubscription = new Subscription()

    /** Browser ports waiting for a message referencing them, by their ID */
    const connectedBrowserPorts = new Map<string, browser.runtime.Port>()

    /** Callbacks from messages that referenced a port ID, waiting for that port to connect */
    const waitingForPorts = new Map<string, (port: browser.runtime.Port) => void>()

    // Listen to all incoming connections matching the prefix and memorize them
    // to have them available when a message arrives that references them.
    const connectListener = (incomingPort: browser.runtime.PortWithSender): void => {
        if (!incomingPort.name.startsWith(prefix)) {
            return
        }
        const id = incomingPort.name.slice(prefix.length)
        const waitingFn = waitingForPorts.get(id)
        if (waitingFn) {
            waitingForPorts.delete(id)
            waitingFn(incomingPort)
        } else {
            connectedBrowserPorts.set(id, incomingPort)
        }
    }
    browser.runtime.onConnect.addListener(connectListener)
    rootSubscription.add(() => browser.runtime.onConnect.removeListener(connectListener))

    /** Run the given callback as soon as the given port ID is connected. */
    const whenConnected = (id: string, callback: (port: browser.runtime.Port) => void): void => {
        const browserPort = connectedBrowserPorts.get(id)
        if (browserPort) {
            // We can delete the port from the Map after we set up a connection.
            // There can never be the same ID twice and it is impossible to transfer the same MessagePort twice.
            connectedBrowserPorts.delete(id)
            callback(browserPort)
            return
        }
        waitingForPorts.set(id, callback)
    }

    /**
     * Set up bi-directional listeners between the two ports that open new `browser.runtime.Port`s for transferred `MessagePort`s.
     *
     * @param browserPort The browser port used for communication with the other thread.
     * @param adapterPort The `MessagePort` used to communicate with comlink.
     */
    function link(browserPort: browser.runtime.Port, adapterPort: MessagePort): void {
        const subscription = new Subscription(() => {
            const browserPortId = browserPort.name.slice(prefix.length)
            connectedBrowserPorts.delete(browserPortId)
            waitingForPorts.delete(browserPortId)
            // Close both ports.
            adapterPort.close()
            browserPort.disconnect()
        })
        rootSubscription.add(subscription)

        const adapterListener = (event: MessageEvent): void => {
            const data: Message = event.data
            // Message from comlink needing to be forwarded to browser port with MessagePorts removed
            const portReferences: PortReference[] = []
            // Find message port references and connect to the other side with the found IDs.
            for (const { value, path, key, parent } of iteratePropertiesDeep(data)) {
                if (value instanceof MessagePort) {
                    // Remove MessagePort from message
                    parent[key!] = null
                    // Open browser port in place of MessagePort
                    const id = uuid.v4()
                    const browserPort = connect(prefix + id)
                    link(browserPort, value)
                    // Include the ID of the browser port in the message
                    portReferences.push({ path, id })
                }
            }
            // Wrap message for the browser port to include all port IDs
            const browserPortMessage: BrowserPortMessage = { message: data, portRefs: portReferences }

            browserPort.postMessage(browserPortMessage)

            // Handle release messages (sent before the other end is closed)
            // by cleaning up all Ports we control
            if (data.type === RELEASE_MESSAGE_TYPE) {
                subscription.unsubscribe()
            }
        }
        adapterPort.addEventListener('message', adapterListener)
        subscription.add(() => adapterPort.removeEventListener('message', adapterListener))

        const browserPortListener = ({ message, portRefs }: BrowserPortMessage): void => {
            const transfer: MessagePort[] = []
            for (const portReference of portRefs) {
                const { port1: comlinkMessagePort, port2: intermediateMessagePort } = new MessageChannel()

                // Replace the port reference at the given path with a MessagePort that will be transferred.
                replaceValueAtPath(message, portReference.path, comlinkMessagePort)
                transfer.push(comlinkMessagePort)

                // Once the port with the mentioned ID is connected, link it up
                whenConnected(portReference.id, browserPort => link(browserPort, intermediateMessagePort))
            }

            // Forward message, with MessagePorts
            adapterPort.postMessage(message, transfer)

            // Handle release messages (sent before the other end is closed)
            // by cleaning up all Ports we control
            if (message.type === RELEASE_MESSAGE_TYPE) {
                subscription.unsubscribe()
            }
        }
        browserPort.onMessage.addListener(browserPortListener)
        subscription.add(() => browserPort.onMessage.removeListener(browserPortListener))

        const disconnectListener = subscription.unsubscribe.bind(subscription)
        browserPort.onDisconnect.addListener(disconnectListener)
        subscription.add(() => browserPort.onDisconnect.removeListener(disconnectListener))

        adapterPort.start()
    }

    const { port1: returnedMessagePort, port2: adapterPort } = new MessageChannel()
    link(browserPort, adapterPort)

    return {
        messagePort: returnedMessagePort,
        subscription: rootSubscription,
    }
}

interface PortReference {
    /** Path at which the MessagePort appeared. */
    path: Path

    /** UUID of the browser Port that was created for it. */
    id: string
}

/**
 * Message format used to communicate over `brower.runtime.Port`s.
 */
interface BrowserPortMessage {
    /**
     * The original comlink message, but with `MessagePort`s replaced with `null`.
     */
    message: Message

    /**
     * Where in the message `MessagePort`s were referenced and the ID of the `browser.runtime.Port` created for each.
     */
    portRefs: PortReference[]
}

type Key = string | number | symbol
type Path = Key[]

interface PropertyIteratorEntry<T = unknown> {
    /**
     * The current value.
     */
    value: T

    /**
     * The key of the current value in the parent object. Equivalent to `path[path.length - 1]`.
     * `null` if the root value.
     */
    key: Key | null

    /**
     * The path of the current value in the root object.
     */
    path: Path

    /**
     * The parent object of the current value. `null` if the root object.
     */
    parent: any | null
}

/**
 * Replace a value in an object structure at a given path with another value.
 *
 * @returns The old value at the path.
 */
function replaceValueAtPath(value: any, path: Path, newValue: unknown): unknown {
    const lastProperty = path[path.length - 1]
    for (const property of path.slice(0, -1)) {
        value = value[property]
    }
    const oldValue = value[lastProperty]
    value[lastProperty] = newValue
    return oldValue
}

/**
 * Iterate all properties of a given value or object recursively, starting with the value itself.
 */
export function* iteratePropertiesDeep<T>(
    value: T,
    key: Key | null = null,
    parent: any = null,
    path: Path = [],
    visited = new WeakSet()
): Iterable<PropertyIteratorEntry> {
    yield { value, path, parent, key }
    if (!isObject(value) || visited.has(value)) {
        return
    }
    visited.add(value)

    const keys = Object.keys(value) as (keyof typeof value)[]
    for (const key of keys) {
        yield* iteratePropertiesDeep(value[key], key, value, [...path, key], visited)
    }
}

/**
 * Yields all `MessagePort`s found deeply in a given object.
 */
export function* findMessagePorts(message: unknown): Iterable<Transferable> {
    for (const { value } of iteratePropertiesDeep(message)) {
        if (value instanceof MessagePort) {
            yield value
        }
    }
}
