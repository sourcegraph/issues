import uuid from 'uuid'
import * as GQL from '../../../../shared/src/graphql/schema'
import { TelemetryService } from '../../../../shared/src/telemetry/telemetryService'
import { storage } from '../../browser/storage'
import { isInPage } from '../../context'
import { logUserEvent } from '../backend/userEvents'

const uidKey = 'sourcegraphAnonymousUid'

export class EventLogger implements TelemetryService {
    private uid: string | null = null

    constructor() {
        // Fetch user ID on initial load.
        this.getAnonUserID().then(
            () => {
                /* noop */
            },
            () => {
                /* noop */
            }
        )
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
        if (!sourcegraphAnonymousUid || sourcegraphAnonymousUid === '') {
            sourcegraphAnonymousUid = this.generateAnonUserID()
            storage.sync.set({ sourcegraphAnonymousUid })
        }
        this.uid = sourcegraphAnonymousUid
        return sourcegraphAnonymousUid
    }

    /**
     * Log a user action on the associated self-hosted Sourcegraph instance (allows site admins on a private
     * Sourcegraph instance to see a count of unique users on a daily, weekly, and monthly basis).
     *
     * This is never sent to Sourcegraph.com (i.e., when using the integration with open source code).
     */
    public logCodeIntelligenceEvent(event: GQL.UserEvent): void {
        this.getAnonUserID().then(
            anonUserId => logUserEvent(event, anonUserId),
            () => {
                /* noop */
            }
        )
    }

    /**
     * Implements {@link TelemetryService}.
     *
     * @todo Handle arbitrary action IDs.
     *
     * @param _eventName The ID of the action executed.
     */
    public log(_eventName: string): void {
        if (_eventName === 'goToDefinition' || _eventName === 'goToDefinition.preloaded' || _eventName === 'hover') {
            this.logCodeIntelligenceEvent(GQL.UserEvent.CODEINTELINTEGRATION)
        } else if (_eventName === 'findReferences') {
            this.logCodeIntelligenceEvent(GQL.UserEvent.CODEINTELINTEGRATIONREFS)
        }
    }
}
