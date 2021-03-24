import { Polly, Request as PollyRequest } from '@pollyjs/core'
import Puppeteer from 'puppeteer'
import Protocol from 'devtools-protocol'
import PollyAdapter from '@pollyjs/adapter'
import { isErrorLike } from '../../../util/errors'
import { Observable, Subject } from 'rxjs'
import { noop } from 'lodash'

function toBase64(input: string): string {
    return Buffer.from(input).toString('base64')
}

function fromBase64(input: string): string {
    return Buffer.from(input, 'base64').toString()
}

export interface CdpAdapterOptions {
    browser: Puppeteer.Browser
}

interface PollyResponse {
    statusCode: number
    headers: Record<string, string>
    body: string
}

/**
 * "Request arguments" are the custom data that Polly allows us to attach to
 * PollyRequests, which we use to store the CDP's requestId to be able to refer
 * to requests.
 */
interface PollyRequestArguments {
    requestArguments: {
        requestId: string

        cdpSession: Puppeteer.CDPSession
    }
}

interface PollyPromise extends Promise<PollyResponse> {
    resolve(response: PollyResponse): void
    reject(error: any): void
}

/**
 * A Puppeteer adapter for Polly that uses "Fetch domain" of the Chrome Devtools
 * protocol to intercept and fulfill requests.
 *
 */
export class CdpAdapter extends PollyAdapter {
    /**
     * The adapter's ID, used to reference it in the Polly constructor.
     */
    public static get id(): string {
        return 'cdp'
    }

    /**
     * `adapterOptions` passed to Polly.
     */
    public options!: CdpAdapterOptions

    private readonly _errors = new Subject<unknown>()

    /**
     * Event that can be subscribed to handle errors that occurred in request handlers.
     */
    public readonly errors: Observable<unknown> = this._errors.asObservable()

    /**
     * The puppeteer Browser this adapter is attached to, obtained from
     * options passed to the Polly constructor.
     */
    private browser: Puppeteer.Browser

    /**
     * A map of all intercepted requests to their respond function, which will be called by the
     * 'response' event listener, causing Polly to record the response content.
     */
    private pendingRequests = new Map<string, PollyPromise>()

    /**
     * A map of CDPSessions keyed by their associated Targets.
     * Each CDP session is used to control request interception for a Target.
     */
    private cdpSessions = new Map<Puppeteer.Target, Puppeteer.CDPSession>()

    /**
     * A map of all intercepted requests to their passthrough callbacks function, which will be called by the
     * onResponseReceived event listener, causing Polly to record the response content.
     */
    private passthroughPromises = new Map<
        string,
        {
            resolve: (response: PollyResponse) => void
            reject: (error: any) => void
        }
    >()

    constructor(polly: Polly) {
        // Rationale for the following ts-ignore:
        // The type declaration provided for Polly's Adapter is missing the
        // constructor argument.
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-ignore
        super(polly)
        this.browser = this.options.browser
    }

    /**
     * Called when the adapter is connected to. Sets up CDP request
     * interception for all browser pages using the CDP "Fetch domain".
     */
    public async onConnect(): Promise<void> {
        // Create CDP sessions for all existing targets
        const targets = await this.browser.targets()

        await Promise.all(
            targets.map(async target => {
                await this.setupCDPSessionForTarget(target)
            })
        )

        // Listen for future pages, create CDP session on creation
        this.browser.on('targetcreated', target => {
            this.setupCDPSessionForTarget(target).catch(noop)
        })

        // Delete associated CDP session when target is destroyed
        this.browser.on('targetdestroyed', target => {
            const cdpSession = this.cdpSessions.get(target)
            this.cdpSessions.delete(target)
            // is this necessary?
            cdpSession?.detach().catch(noop)
        })
    }

    /**
     * Creates a CDP session for the given target and listens
     * for paused requests
     */
    private async setupCDPSessionForTarget(target: Puppeteer.Target): Promise<void> {
        const cdpSession = await target.createCDPSession()
        this.cdpSessions.set(target, cdpSession)

        // TODO: This is where we narrow down the interception with patterns.
        // Request and respond stages are independent, so we can set a different
        // set of patterns for each.
        const fetchEnableRequest: Protocol.Fetch.EnableRequest = {
            patterns: [{ requestStage: 'Request' }, { requestStage: 'Response' }],
        }
        await cdpSession.send('Fetch.enable', fetchEnableRequest)

        cdpSession.on('Fetch.requestPaused', (event: Protocol.Fetch.RequestPausedEvent): void => {
            const isInResponseStage = eventIsInResponseStage(event)
            if (isInResponseStage) {
                this.handlePausedRequestInResponseStage(event, cdpSession)
            } else {
                this.handlePausedRequestInRequestStage(event, cdpSession)
            }
        })
    }

    /**
     * Called when the adapter is disconnected from
     */
    public async onDisconnect(): Promise<void> {
        await Promise.allSettled(
            [...this.cdpSessions].map(async ([, cdpSession]) => this.trySendCdpRequest('Fetch.disable', cdpSession))
        )
    }

    /**
     * Given a request that should be allowed to pass through (not be intercepted),
     * return a Promise of the Response for that request, which will be passed to
     * request.respond().
     */
    public passthroughRequest(pollyRequest: PollyRequest & PollyRequestArguments): Promise<PollyResponse> {
        const {
            requestArguments: { requestId, cdpSession },
        } = pollyRequest

        return new Promise<PollyResponse>((resolve, reject) => {
            this.passthroughPromises.set(requestId, { resolve, reject })
            this.continuePausedRequest({ requestId }, cdpSession).catch(console.error)
        })
    }

    /**
     * Responds to an intercepted request with the given response.
     *
     * If an error happened when retrieving the response, abort the request.
     */
    public async respondToRequest(
        pollyRequest: PollyRequest & { response: PollyResponse } & PollyRequestArguments,
        error?: unknown
    ): Promise<void> {
        const { response: pollyResponse, requestArguments } = pollyRequest
        const { headers, body = '' } = pollyResponse
        const { requestId, cdpSession } = requestArguments
        if (error) {
            // This function receives a value in the `error` argument if we're
            // intercepting a request with the Polly server route which throws
            // an error.
            this._errors.next(error)
            await this.abortPausedRequest({ requestId, errorReason: 'Failed' }, cdpSession)
        } else {
            // Fulfill by converting the Polly response to a CDP response
            const cdpRequestToFulfill = {
                requestId,
                responseCode: pollyResponse.statusCode,
                responseHeaders: headerObjectToHeaderEntries(headers),
                body: toBase64(body),
            }
            await this.fulfillPausedRequest(cdpRequestToFulfill, cdpSession)
        }
    }

    /**
     * Called when a request is intercepted, for all requests (passthrough or stubbed).
     *
     * Adds an entry to pendingRequests, that will call the provided promise.resolve function
     * when a response for this request is received.
     */
    public onRequest(pollyRequest: PollyRequest & PollyRequestArguments & { promise: PollyPromise }): void {
        const { requestArguments, promise } = pollyRequest

        const { requestId } = requestArguments
        this.pendingRequests.set(requestId, promise)
    }

    private async fulfillPausedRequest(
        request: Protocol.Fetch.FulfillRequestRequest,
        cdpSession: Puppeteer.CDPSession
    ): Promise<void> {
        await this.trySendCdpRequest('Fetch.fulfillRequest', cdpSession, request)
    }

    private async continuePausedRequest(
        request: Protocol.Fetch.ContinueRequestRequest,
        cdpSession: Puppeteer.CDPSession
    ): Promise<void> {
        await this.trySendCdpRequest('Fetch.continueRequest', cdpSession, request)
    }

    /**
     * Perform a CDP request call that doesn't return a result, while ignoring
     * errors due to the page being closed already.
     */
    private async trySendCdpRequest(
        cdpRequestName: string,
        cdpSession: Puppeteer.CDPSession,
        request?: object
    ): Promise<void> {
        try {
            await cdpSession?.send(cdpRequestName, request)
        } catch (error) {
            // TODO: also ignore "target closed" error
            if (
                isErrorLike(error) &&
                (error.message.endsWith('Session closed. Most likely the page has been closed.') ||
                    error.message.endsWith('Target closed.') ||
                    // Invalid interceptionId probably means the request has been aborted.
                    error.message.includes('Invalid InterceptionId'))
            ) {
                return
            }
            throw error
        }
    }

    private async abortPausedRequest(
        request: Protocol.Fetch.FailRequestRequest,
        cdpSession: Puppeteer.CDPSession
    ): Promise<void> {
        await cdpSession?.send('Fetch.failRequest', request)
    }

    private async getResponseBody(
        event: Protocol.Fetch.RequestPausedEvent,
        cdpSession: Puppeteer.CDPSession
    ): Promise<string> {
        if (getLocationHeader(event)) {
            return '' // CDP doesn't let us obtain the body of redirect requests, so we don't attempt it.
        }
        if (!cdpSession) {
            throw new Error('Fetch.getResponseBody called before CDP session created')
        }

        const body = (await cdpSession.send('Fetch.getResponseBody', {
            requestId: event.requestId,
        })) as Protocol.Fetch.GetResponseBodyResponse

        return getBodyStringFromCdpBody(body)
    }

    /**
     * Handle a "request paused" event, for requests called at the request stage.
     */
    private handlePausedRequestInRequestStage(
        event: Protocol.Fetch.RequestPausedEvent,
        cdpSession: Puppeteer.CDPSession
    ): void {
        // Rationale for ts-ignore:
        // The type declaration provided for Polly's Adapter is missing a
        // declaration for the handleRequest method.
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-ignore
        this.handleRequest({
            url: event.request.url,
            method: event.request.method,
            headers: event.request.headers,
            // postData appears to be the field that contains the actual entire
            // body of the request
            body: event.request.postData ?? '',
            requestArguments: { requestId: event.requestId, cdpSession },
        })
    }

    /**
     * Handle a "request paused" event, for requests paused at the response stage.
     */
    private handlePausedRequestInResponseStage(
        event: Protocol.Fetch.RequestPausedEvent,
        cdpSession: Puppeteer.CDPSession
    ): void {
        const { requestId } = event

        // First case: response was not received and encountered an error (for
        // example the connection was refused)
        if (event.responseErrorReason) {
            const error = new Error(event.responseErrorReason)

            // Reject passthrough
            this.passthroughPromises.get(requestId)?.reject(error)
            this.passthroughPromises.delete(requestId)

            /// Reject pending request
            this.pendingRequests.get(requestId)?.reject(error)
            this.pendingRequests.delete(requestId)
            return
        }

        // Second case: response was received and therefore the response is
        // expected to have a status code, and may or may not have a body.
        this.getResponseBody(event, cdpSession)
            .then(body => {
                const statusCode = event.responseStatusCode
                if (!statusCode) {
                    throw new Error('Response expected to have a status code')
                }

                // Convert the CDP response into a Polly response
                const pollyResponse: PollyResponse = {
                    statusCode,
                    headers: headerEntriesToHeaderObject(event.responseHeaders),
                    body,
                }

                // The requestId may or may not have an associated passthrough callback.
                // If it does, call it and delete it.
                this.passthroughPromises.get(requestId)?.resolve(pollyResponse)
                this.passthroughPromises.delete(requestId)

                // Each pending request has an associated promise. Because at this point
                // the request is done (given that a response has been received), we can
                // resolve the pending request promise.
                this.pendingRequests.get(requestId)?.resolve(pollyResponse)
                this.pendingRequests.delete(requestId)
            })
            .catch(console.error)
    }
}

/**
 * Determine if the request is paused in the response stage. If false, then the
 * request is paused in the request stage.
 */
function eventIsInResponseStage(event: Protocol.Fetch.RequestPausedEvent): boolean {
    return event.responseStatusCode !== undefined || event.responseErrorReason !== undefined
}

/**
 * Get the value of the "Location" response header.
 **/
function getLocationHeader(event: Protocol.Fetch.RequestPausedEvent): string | undefined {
    const { responseHeaders = [] } = event
    const foundLocationHeader = responseHeaders.find(header => header.name === 'location')
    if (foundLocationHeader) {
        return foundLocationHeader.value
    }
    return undefined
}

/**
 * Transform a header represented by an array of
 * {@link Protocol.Fetch.HeaderEntry} (the format used by CDP) into a headers
 * object (the format used by Polly)
 */
function headerEntriesToHeaderObject(responseHeaders: Protocol.Fetch.HeaderEntry[] = []): Record<string, string> {
    return Object.fromEntries(responseHeaders.map(({ name, value }) => [name, value]))
}

/** Transform a header object (the format used by Polly) into an array of header
 * entries (the format used by CDP) */
function headerObjectToHeaderEntries(headers: Record<string, string>): Protocol.Fetch.HeaderEntry[] {
    return Object.entries(headers).map(([name, value]) => ({ name, value }))
}

/**
 * Get the body data as a string, from the response of a `Fetch.getResponseBody` call.
 */
function getBodyStringFromCdpBody(body: Protocol.Fetch.GetResponseBodyResponse): string {
    if (!body) {
        return ''
    }
    if (body.base64Encoded) {
        return fromBase64(body.body)
    }
    return body.body
}
