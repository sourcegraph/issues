import { noop } from 'lodash'
import { Observable, ReplaySubject, Subscription } from 'rxjs'
import { take } from 'rxjs/operators'
import * as uuid from 'uuid'
import { PlatformContext } from '../../../../shared/src/platform/context'
import { TelemetryService } from '../../../../shared/src/telemetry/telemetryService'
import { storage } from '../../browser-extension/web-extension-api/storage'
import { isInPage } from '../context'
import { logUserEvent, logEvent } from '../backend/userEvents'
import { observeSourcegraphURL, getPlatformName } from '../util/context'
import { UserEvent } from '../../graphql-operations'

const uidKey = 'sourcegraphAnonymousUid'

/**
 * Telemetry Service which only logs when the enable flag is set. Accepts an
 * observable that emits the enabled value.
 *
 * This was implemented as a wrapper around TelemetryService in order to avoid
 * modifying EventLogger, but the enabled flag could be rolled into EventLogger.
 *
 * TODO: Potential to be improved by buffering log events until the first emit
 * of the enabled value.
 */
export class ConditionalTelemetryService implements TelemetryService {
    /** Log events are passed on to the inner TelemetryService */
    private innerTelemetryService: TelemetryService
    private subscription = new Subscription()

    /**
     * The enabled state set by an observable, provided upon instantiation
     */
    private isEnabled = false

    constructor(innerTelemetryService: TelemetryService, isEnabled: Observable<boolean>) {
        this.subscription.add(
            isEnabled.subscribe(value => {
                this.isEnabled = value
            })
        )
        this.innerTelemetryService = innerTelemetryService
    }

    public log(eventName: string, eventProperties?: any): void {
        if (this.isEnabled) {
            this.innerTelemetryService.log(eventName, eventProperties)
        }
    }
    public logViewEvent(eventName: string, eventProperties?: any): void {
        if (this.isEnabled) {
            this.innerTelemetryService.logViewEvent(eventName, eventProperties)
        }
    }

    public unsubscribe(): void {
        return this.subscription.unsubscribe()
    }
}

export class EventLogger implements TelemetryService {
    private uid: string | null = null

    private platform = getPlatformName()

    /**
     * Buffered Observable for the latest Sourcegraph URL
     */
    private sourcegraphURLs: Observable<string>

    constructor(isExtension: boolean, private requestGraphQL: PlatformContext['requestGraphQL']) {
        const replaySubject = new ReplaySubject<string>(1)
        this.sourcegraphURLs = replaySubject.asObservable()
        // TODO pass this Observable as a parameter
        // eslint-disable-next-line rxjs/no-ignored-subscription
        observeSourcegraphURL(isExtension).subscribe(replaySubject)
        // Fetch user ID on initial load.
        this.getAnonUserID().catch(noop)
    }

    /**
     * Generate a new anonymous user ID if one has not yet been set and stored.
     */
    private generateAnonUserID = (): string => uuid.v4()

    /**
     * Get the anonymous identifier for this user (allows site admins on a private Sourcegraph
     * instance to see a count of unique users on a daily, weekly, and monthly basis).
     *
     * Not used at all for public/Sourcegraph.com usage.
     */
    private async getAnonUserID(): Promise<string> {
        if (this.uid) {
            return this.uid
        }

        if (isInPage) {
            let id = localStorage.getItem(uidKey)
            if (id === null) {
                id = this.generateAnonUserID()
                localStorage.setItem(uidKey, id)
            }
            this.uid = id
            return this.uid
        }

        let { sourcegraphAnonymousUid } = await storage.sync.get()
        if (!sourcegraphAnonymousUid) {
            sourcegraphAnonymousUid = this.generateAnonUserID()
            await storage.sync.set({ sourcegraphAnonymousUid })
        }
        this.uid = sourcegraphAnonymousUid
        return sourcegraphAnonymousUid
    }

    /**
     * Log a user action on the associated self-hosted Sourcegraph instance (allows site admins on a private
     * Sourcegraph instance to see a count of unique users on a daily, weekly, and monthly basis).
     */
    public async logCodeIntelligenceEvent(event: string, userEvent: UserEvent, eventProperties?: any): Promise<void> {
        const anonUserId = await this.getAnonUserID()
        const sourcegraphURL = await this.sourcegraphURLs.pipe(take(1)).toPromise()
        logUserEvent(userEvent, anonUserId, sourcegraphURL, this.requestGraphQL)
        logEvent(
            {
                name: event,
                userCookieID: anonUserId,
                url: sourcegraphURL,
                argument: { platform: this.platform, ...eventProperties },
            },
            this.requestGraphQL
        )
    }

    /**
     * Implements {@link TelemetryService}.
     *
     * @todo Handle arbitrary action IDs.
     *
     * @param eventName The ID of the action executed.
     */
    public async log(eventName: string, eventProperties?: any): Promise<void> {
        switch (eventName) {
            case 'goToDefinition':
            case 'goToDefinition.preloaded':
            case 'hover':
                await this.logCodeIntelligenceEvent(eventName, UserEvent.CODEINTELINTEGRATION, eventProperties)
                break
            case 'findReferences':
                await this.logCodeIntelligenceEvent(eventName, UserEvent.CODEINTELINTEGRATIONREFS, eventProperties)
                break
        }
    }

    /**
     * Implements {@link TelemetryService}.
     *
     * @param pageTitle The title of the page being viewed.
     */
    public async logViewEvent(pageTitle: string, eventProperties?: any): Promise<void> {
        const anonUserId = await this.getAnonUserID()
        const sourcegraphURL = await this.sourcegraphURLs.pipe(take(1)).toPromise()
        logEvent(
            {
                name: `View${pageTitle}`,
                userCookieID: anonUserId,
                url: sourcegraphURL,
                argument: { ...eventProperties, platform: this.platform },
            },
            this.requestGraphQL
        )
    }
}
