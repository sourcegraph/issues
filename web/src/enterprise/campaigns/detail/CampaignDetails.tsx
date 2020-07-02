import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import AlertCircleIcon from 'mdi-react/AlertCircleIcon'
import React, { useState, useEffect, useRef, useMemo, useCallback } from 'react'
import * as GQL from '../../../../../shared/src/graphql/schema'
import { HeroPage } from '../../../components/HeroPage'
import { PageTitle } from '../../../components/PageTitle'
import { UserAvatar } from '../../../user/UserAvatar'
import { Timestamp } from '../../../components/time/Timestamp'
import { noop, isEqual } from 'lodash'
import { fetchCampaignById, deleteCampaign, closeCampaign } from './backend'
import { useError } from '../../../../../shared/src/util/useObservable'
import { asError } from '../../../../../shared/src/util/errors'
import * as H from 'history'
import { CampaignBurndownChart } from './BurndownChart'
import { Subject, of, merge, Observable } from 'rxjs'
import { renderMarkdown } from '../../../../../shared/src/util/markdown'
import { ErrorAlert } from '../../../components/alerts'
import { Markdown } from '../../../../../shared/src/components/Markdown'
import { switchMap, distinctUntilChanged } from 'rxjs/operators'
import { ThemeProps } from '../../../../../shared/src/theme'
import { CampaignStatus } from './CampaignStatus'
import { CampaignActionsBar } from './CampaignActionsBar'
import { CampaignChangesets } from './changesets/CampaignChangesets'
import { CampaignDiffStat } from './CampaignDiffStat'
import { pluralize } from '../../../../../shared/src/util/strings'
import { ExtensionsControllerProps } from '../../../../../shared/src/extensions/controller'
import { PlatformContextProps } from '../../../../../shared/src/platform/context'
import { TelemetryProps } from '../../../../../shared/src/telemetry/telemetryService'
import { repeatUntil } from '../../../../../shared/src/util/rxjs/repeatUntil'

export type CampaignUIMode = 'viewing' | 'editing' | 'saving' | 'deleting' | 'closing'

interface Campaign
    extends Pick<
        GQL.ICampaign,
        | '__typename'
        | 'id'
        | 'name'
        | 'description'
        | 'author'
        | 'changesetCountsOverTime'
        | 'createdAt'
        | 'updatedAt'
        | 'closedAt'
        | 'viewerCanAdminister'
        | 'branch'
    > {
    changesets: Pick<GQL.ICampaign['changesets'], 'totalCount'>
    status: Pick<GQL.ICampaign['status'], 'completedCount' | 'pendingCount' | 'errors' | 'state'>
    diffStat: Pick<GQL.ICampaign['diffStat'], 'added' | 'deleted' | 'changed'>
}

interface Props extends ThemeProps, ExtensionsControllerProps, PlatformContextProps, TelemetryProps {
    /**
     * The campaign ID.
     * If not given, will display a creation form.
     */
    campaignID?: GQL.ID
    authenticatedUser: Pick<GQL.IUser, 'id' | 'username' | 'avatarURL'>
    history: H.History
    location: H.Location

    /** For testing only. */
    _fetchCampaignById?: typeof fetchCampaignById | ((campaign: GQL.ID) => Observable<Campaign | null>)
}

/**
 * The area for a single campaign.
 */
export const CampaignDetails: React.FunctionComponent<Props> = ({
    campaignID,
    history,
    location,
    authenticatedUser,
    isLightTheme,
    extensionsController,
    platformContext,
    telemetryService,
    _fetchCampaignById = fetchCampaignById,
}) => {
    // For errors during fetching
    const triggerError = useError()

    /** Retrigger campaign fetching */
    const campaignUpdates = useMemo(() => new Subject<void>(), [])
    /** Retrigger changeset fetching */
    const changesetUpdates = useMemo(() => new Subject<void>(), [])

    const [campaign, setCampaign] = useState<Campaign | null>()
    useEffect(() => {
        if (!campaignID) {
            return
        }
        // on the very first fetch, a reload of the changesets is not required
        let isFirstCampaignFetch = true

        // Fetch campaign if ID was given
        const subscription = merge(of(undefined), campaignUpdates)
            .pipe(
                switchMap(() =>
                    _fetchCampaignById(campaignID).pipe(
                        // repeat fetching the campaign as long as the state is still processing
                        repeatUntil(campaign => campaign?.status?.state !== GQL.BackgroundProcessState.PROCESSING, {
                            delay: 2000,
                        })
                    )
                ),
                distinctUntilChanged((a, b) => isEqual(a, b))
            )
            .subscribe({
                next: fetchedCampaign => {
                    setCampaign(fetchedCampaign)
                    if (!isFirstCampaignFetch) {
                        changesetUpdates.next()
                    }
                    isFirstCampaignFetch = false
                },
                error: triggerError,
            })
        return () => subscription.unsubscribe()
    }, [campaignID, triggerError, changesetUpdates, campaignUpdates, _fetchCampaignById])

    const [mode, setMode] = useState<CampaignUIMode>(campaignID ? 'viewing' : 'editing')

    // To report errors from saving or deleting
    const [alertError, setAlertError] = useState<Error>()

    // To unblock the history after leaving edit mode
    const unblockHistoryReference = useRef<H.UnregisterCallback>(noop)
    useEffect(() => {
        if (!campaignID) {
            unblockHistoryReference.current()
            unblockHistoryReference.current = history.block('Do you want to discard this campaign?')
        }
        // Note: the current() method gets dynamically reassigned,
        // therefor we can't return it directly.
        return () => unblockHistoryReference.current()
    }, [campaignID, history])

    const onClose = useCallback(
        async (closeChangesets: boolean): Promise<void> => {
            if (!confirm('Are you sure you want to close the campaign?')) {
                return
            }
            setMode('closing')
            try {
                await closeCampaign(campaign!.id, closeChangesets)
                campaignUpdates.next()
            } catch (error) {
                setAlertError(asError(error))
            } finally {
                setMode('viewing')
            }
        },
        [campaign, campaignUpdates]
    )

    const onDelete = useCallback(async (): Promise<void> => {
        if (!confirm('Are you sure you want to delete the campaign?')) {
            return
        }
        setMode('deleting')
        try {
            await deleteCampaign(campaign!.id)
            history.push('/campaigns')
        } catch (error) {
            setAlertError(asError(error))
            setMode('viewing')
        }
    }, [campaign, history])

    // Is loading
    if (campaignID && campaign === undefined) {
        return (
            <div className="text-center">
                <LoadingSpinner className="icon-inline mx-auto my-4" />
            </div>
        )
    }
    // Campaign was not found
    if (campaign === null) {
        return <HeroPage icon={AlertCircleIcon} title="Campaign not found" />
    }

    const author = campaign ? campaign.author : authenticatedUser

    const totalChangesetCount = campaign?.changesets.totalCount ?? 0

    const campaignFormID = 'campaign-form'

    return (
        <>
            <PageTitle title={campaign?.name} />
            <CampaignActionsBar
                mode={mode}
                campaign={campaign}
                onClose={onClose}
                onDelete={onDelete}
                formID={campaignFormID}
            />
            {alertError && <ErrorAlert error={alertError} history={history} />}
            {campaign && !['saving', 'editing'].includes(mode) && (
                <CampaignStatus campaign={campaign} history={history} />
            )}
            {campaign && !['saving', 'editing'].includes(mode) && (
                <>
                    <div className="card mt-2">
                        <div className="card-header">
                            <strong>
                                <UserAvatar user={author} className="icon-inline" /> {author.username}
                            </strong>{' '}
                            started <Timestamp date={campaign.createdAt} />
                        </div>
                        <div className="card-body">
                            <Markdown
                                dangerousInnerHTML={renderMarkdown(campaign.description || '_No description_')}
                                history={history}
                            />
                        </div>
                    </div>
                    {totalChangesetCount > 0 && (
                        <>
                            <h3 className="mt-4 mb-2">Progress</h3>
                            <CampaignBurndownChart
                                changesetCountsOverTime={campaign.changesetCountsOverTime}
                                history={history}
                            />
                        </>
                    )}
                </>
            )}

            {totalChangesetCount && (
                <>
                    <h3 className="mt-4 d-flex align-items-end mb-0">
                        {(totalChangesetCount > 0 || !!campaign) && (
                            <>
                                {totalChangesetCount} {pluralize('Changeset', totalChangesetCount)}
                            </>
                        )}{' '}
                        {campaign && <CampaignDiffStat campaign={campaign} className="ml-2 mb-0" />}
                    </h3>
                    {totalChangesetCount > 0 && (
                        <CampaignChangesets
                            campaign={campaign!}
                            changesetUpdates={changesetUpdates}
                            campaignUpdates={campaignUpdates}
                            history={history}
                            location={location}
                            isLightTheme={isLightTheme}
                            extensionsController={extensionsController}
                            platformContext={platformContext}
                            telemetryService={telemetryService}
                        />
                    )}
                </>
            )}
        </>
    )
}
