import slugify from 'slugify'
import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import AlertCircleIcon from 'mdi-react/AlertCircleIcon'
import React, { useState, useEffect, useRef, useMemo, useCallback } from 'react'
import * as GQL from '../../../../../shared/src/graphql/schema'
import { HeroPage } from '../../../components/HeroPage'
import { PageTitle } from '../../../components/PageTitle'
import { UserAvatar } from '../../../user/UserAvatar'
import { Timestamp } from '../../../components/time/Timestamp'
import { noop, isEqual } from 'lodash'
import { Form } from '../../../components/Form'
import {
    fetchCampaignById,
    updateCampaign,
    deleteCampaign,
    createCampaign,
    closeCampaign,
    fetchPatchSetById,
} from './backend'
import { useError, useObservable } from '../../../../../shared/src/util/useObservable'
import { asError } from '../../../../../shared/src/util/errors'
import * as H from 'history'
import { CampaignBurndownChart } from './BurndownChart'
import { AddChangesetForm } from './AddChangesetForm'
import { Subject, of, merge, Observable, NEVER } from 'rxjs'
import { renderMarkdown, highlightCodeSafe } from '../../../../../shared/src/util/markdown'
import { ErrorAlert } from '../../../components/alerts'
import { Markdown } from '../../../../../shared/src/components/Markdown'
import { switchMap, distinctUntilChanged } from 'rxjs/operators'
import { ThemeProps } from '../../../../../shared/src/theme'
import { CampaignDescriptionField } from './form/CampaignDescriptionField'
import { CampaignStatus } from './CampaignStatus'
import { CampaignUpdateDiff } from './CampaignUpdateDiff'
import { CampaignActionsBar } from './CampaignActionsBar'
import { CampaignTitleField } from './form/CampaignTitleField'
import { CampaignChangesets } from './changesets/CampaignChangesets'
import { CampaignDiffStat } from './CampaignDiffStat'
import { pluralize } from '../../../../../shared/src/util/strings'
import { ExtensionsControllerProps } from '../../../../../shared/src/extensions/controller'
import { PlatformContextProps } from '../../../../../shared/src/platform/context'
import { TelemetryProps } from '../../../../../shared/src/telemetry/telemetryService'
import { CampaignPatches } from './patches/CampaignPatches'
import { PatchSetPatches } from './patches/PatchSetPatches'
import { CampaignBranchField } from './form/CampaignBranchField'
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
        | 'hasUnpublishedPatches'
    > {
    changesets: Pick<GQL.ICampaign['changesets'], 'totalCount'>
    patches: Pick<GQL.ICampaign['patches'], 'totalCount'>
    status: Pick<GQL.ICampaign['status'], 'completedCount' | 'pendingCount' | 'errors' | 'state'>
    diffStat: Pick<GQL.ICampaign['diffStat'], 'added' | 'deleted' | 'changed'>
}

interface PatchSet extends Pick<GQL.IPatchSet, '__typename' | 'id'> {
    diffStat: Pick<GQL.IPatchSet['diffStat'], 'added' | 'deleted' | 'changed'>
    patches: Pick<GQL.IPatchSet['patches'], 'totalCount'>
}

interface Props extends ThemeProps, ExtensionsControllerProps, PlatformContextProps, TelemetryProps {
    /**
     * The campaign ID.
     * If not given, will display a creation form.
     */
    campaignID?: GQL.ID
    authenticatedUser: Pick<GQL.IUser, 'id' | 'username' | 'avatarURL'> | null
    history: H.History
    location: H.Location

    /** For testing only. */
    _fetchCampaignById?: typeof fetchCampaignById | ((campaign: GQL.ID) => Observable<Campaign | null>)
    /** For testing only. */
    _fetchPatchSetById?: typeof fetchPatchSetById | ((patchSet: GQL.ID) => Observable<PatchSet | null>)
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
    _fetchPatchSetById = fetchPatchSetById,
}) => {
    // State for the form in editing mode
    const [name, setName] = useState<string>('')
    const [description, setDescription] = useState<string>('')
    const [branch, setBranch] = useState<string>('')
    const [branchModified, setBranchModified] = useState<boolean>(false)

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
                    if (fetchedCampaign) {
                        setName(fetchedCampaign.name)
                        setDescription(fetchedCampaign.description ?? '')
                        setBranch(fetchedCampaign.branch ?? '')
                        setBranchModified(false)
                    }
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

    const patchSetID: GQL.ID | null = new URLSearchParams(location.search).get('patchSet')
    useEffect(() => {
        if (patchSetID) {
            setMode('editing')
        }
    }, [patchSetID])

    // To unblock the history after leaving edit mode
    const unblockHistoryReference = useRef<H.UnregisterCallback>(noop)
    useEffect(() => {
        if (!campaignID && patchSetID === null) {
            unblockHistoryReference.current()
            unblockHistoryReference.current = history.block('Do you want to discard this campaign?')
        }
        // Note: the current() method gets dynamically reassigned,
        // therefor we can't return it directly.
        return () => unblockHistoryReference.current()
    }, [campaignID, history, patchSetID])

    const patchSet = useObservable(
        useMemo(() => (!patchSetID ? NEVER : _fetchPatchSetById(patchSetID)), [patchSetID, _fetchPatchSetById])
    )

    const onAddChangeset = useCallback((): void => {
        // we also check the campaign.changesets.totalCount, so an update to the campaign is required as well
        campaignUpdates.next()
        changesetUpdates.next()
    }, [campaignUpdates, changesetUpdates])

    const onNameChange = useCallback(
        (newName: string): void => {
            if (!branchModified) {
                setBranch(slugify(newName, { remove: /[!"'()*+.:@\\^~]/g, lower: true }))
            }
            setName(newName)
        },
        [branchModified]
    )

    const onBranchChange = useCallback((newValue: string): void => {
        setBranch(newValue)
        setBranchModified(true)
    }, [])

    const specifyingBranchAllowed =
        // on campaign creation
        (!campaign && patchSet) ||
        // or when no changesets have been published or are being published as well
        (campaign &&
            campaign.changesets.totalCount === 0 &&
            campaign.status.state !== GQL.BackgroundProcessState.PROCESSING)

    const onSubmit: React.FormEventHandler = useCallback(
        async event => {
            event.preventDefault()
            setMode('saving')
            try {
                if (!authenticatedUser) {
                    throw new Error('must sign in')
                }

                if (campaignID) {
                    const newCampaign = await updateCampaign({
                        id: campaignID,
                        name,
                        description,
                        patchSet: patchSetID ?? undefined,
                        branch: specifyingBranchAllowed ? branch : undefined,
                    })
                    setCampaign(newCampaign)
                    setName(newCampaign.name)
                    setDescription(newCampaign.description ?? '')
                    setBranch(newCampaign.branch ?? '')
                    setBranchModified(false)
                    unblockHistoryReference.current()
                    history.push(`/campaigns/${newCampaign.id}`)
                } else {
                    const createdCampaign = await createCampaign({
                        name,
                        description,
                        namespace: authenticatedUser.id,
                        patchSet: patchSet ? patchSet.id : undefined,
                        branch: specifyingBranchAllowed ? branch : undefined,
                    })
                    unblockHistoryReference.current()
                    history.push(`/campaigns/${createdCampaign.id}`)
                }
                setMode('viewing')
                setAlertError(undefined)
                campaignUpdates.next()
            } catch (error) {
                setMode('editing')
                setAlertError(asError(error))
            }
        },
        [
            authenticatedUser,
            branch,
            campaignID,
            campaignUpdates,
            description,
            history,
            name,
            patchSet,
            patchSetID,
            specifyingBranchAllowed,
        ]
    )

    const discardChangesMessage = 'Do you want to discard your changes?'

    const onEdit: React.MouseEventHandler = useCallback(
        event => {
            event.preventDefault()
            unblockHistoryReference.current = history.block(discardChangesMessage)
            setMode('editing')
            setAlertError(undefined)
        },
        [history]
    )

    const onCancel: React.FormEventHandler = useCallback(
        event => {
            event.preventDefault()
            if (!confirm(discardChangesMessage)) {
                return
            }
            unblockHistoryReference.current()
            // clear query params
            history.replace(location.pathname)
            setMode('viewing')
            setAlertError(undefined)
        },
        [history, location.pathname]
    )

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

    const onDelete = useCallback(
        async (closeChangesets: boolean): Promise<void> => {
            if (!confirm('Are you sure you want to delete the campaign?')) {
                return
            }
            setMode('deleting')
            try {
                await deleteCampaign(campaign!.id, closeChangesets)
                history.push('/campaigns')
            } catch (error) {
                setAlertError(asError(error))
                setMode('viewing')
            }
        },
        [campaign, history]
    )

    const afterCampaignModify = useCallback(
        (updatedCampaign: Campaign): void => {
            setCampaign(updatedCampaign)
            campaignUpdates.next()
        },
        [campaignUpdates]
    )

    // Is loading
    if ((campaignID && campaign === undefined) || (patchSetID && patchSet === undefined)) {
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
    // Patch set was not found
    if (patchSet === null) {
        return <HeroPage icon={AlertCircleIcon} title="Patch set not found" />
    }

    // On update, check if an update is possible
    if (!!campaign && !!patchSet && campaign.closedAt) {
        return <HeroPage icon={AlertCircleIcon} title="Cannot update a closed campaign" />
    }

    const totalChangesetCount = campaign?.changesets.totalCount ?? 0

    const totalPatchCount = (campaign?.patches.totalCount ?? 0) + (patchSet?.patches.totalCount ?? 0)

    const campaignFormID = 'campaign-form'

    return (
        <>
            <PageTitle title={campaign ? campaign.name : 'New campaign'} />
            <CampaignActionsBar
                previewingPatchSet={!!patchSet}
                mode={mode}
                campaign={campaign}
                onEdit={onEdit}
                onClose={onClose}
                onDelete={onDelete}
                formID={campaignFormID}
            />
            {alertError && <ErrorAlert error={alertError} history={history} />}
            {campaign && !patchSet && !['saving', 'editing'].includes(mode) && (
                <CampaignStatus
                    campaign={campaign}
                    afterRetry={afterCampaignModify}
                    afterPublish={afterCampaignModify}
                    history={history}
                />
            )}
            <Form id={campaignFormID} onSubmit={onSubmit} onReset={onCancel} className="e2e-campaign-form">
                {['saving', 'editing'].includes(mode) && (
                    <>
                        <h3>Details</h3>
                        <CampaignTitleField value={name} onChange={onNameChange} disabled={mode === 'saving'} />
                        <CampaignDescriptionField
                            value={description}
                            onChange={setDescription}
                            disabled={mode === 'saving'}
                        />
                        {specifyingBranchAllowed && (
                            <CampaignBranchField
                                value={branch}
                                onChange={onBranchChange}
                                disabled={mode === 'saving'}
                            />
                        )}
                        {/* Existing non-manual campaign, but not updating with a new set of patches */}
                        {campaign && !patchSet && (
                            <div className="card">
                                <div className="card-body">
                                    <h3 className="card-title">Want to update the patches?</h3>
                                    <p>
                                        Using the{' '}
                                        <a
                                            href="https://github.com/sourcegraph/src-cli"
                                            rel="noopener noreferrer"
                                            target="_blank"
                                        >
                                            src CLI
                                        </a>
                                        , you can also apply a new patch set to an existing campaign. Following the
                                        creation of a new patch set that contains new patches, with the
                                    </p>
                                    <div className="alert alert-secondary">
                                        <code
                                            dangerouslySetInnerHTML={{
                                                __html: highlightCodeSafe(
                                                    '$ src action exec -f action.json | src campaign patchset create-from-patches',
                                                    'bash'
                                                ),
                                            }}
                                        />
                                    </div>
                                    <p>
                                        command, a URL will be output that will guide you to the web UI to allow you to
                                        change an existing campaign’s patch set.
                                    </p>
                                    <p className="mb-0">
                                        Take a look at the{' '}
                                        <a
                                            href="https://docs.sourcegraph.com/user/campaigns/updating_campaigns"
                                            rel="noopener noreferrer"
                                            target="_blank"
                                        >
                                            documentation on updating campaigns
                                        </a>{' '}
                                        for more information.
                                    </p>
                                </div>
                            </div>
                        )}
                    </>
                )}
                {/* If we are in the update view */}
                {campaign && patchSet && (
                    <>
                        <CampaignUpdateDiff
                            campaign={campaign}
                            patchSet={patchSet}
                            history={history}
                            location={location}
                            isLightTheme={isLightTheme}
                            className="mt-4"
                        />
                        <div className="mb-0">
                            <button
                                type="reset"
                                form={campaignFormID}
                                className="btn btn-secondary mr-1"
                                onClick={onCancel}
                                disabled={mode !== 'editing'}
                            >
                                Cancel
                            </button>
                            <button
                                type="submit"
                                form={campaignFormID}
                                className="btn btn-primary"
                                disabled={mode !== 'editing' || patchSet?.patches.totalCount === 0}
                            >
                                Update
                            </button>
                        </div>
                    </>
                )}
                {/* If campaign doesn't yet exist.. */}
                {!campaign && (
                    <>
                        <div className="mt-2">
                            <button
                                type="submit"
                                form={campaignFormID}
                                className="btn btn-primary e2e-campaign-create-btn"
                                disabled={mode !== 'editing' || patchSet?.patches.totalCount === 0}
                            >
                                Create
                            </button>
                        </div>
                    </>
                )}
            </Form>
            {/* Iff either campaign XOR patchset are present */}
            {!(campaign && patchSet) && (campaign || patchSet) && (
                <>
                    {campaign && !['saving', 'editing'].includes(mode) && (
                        <>
                            <div className="card mt-2">
                                <div className="card-header">
                                    <strong>
                                        <UserAvatar user={campaign.author} className="icon-inline" />{' '}
                                        {campaign.author.username}
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

                            {campaign.viewerCanAdminister && !campaign.closedAt && (
                                <>
                                    {totalChangesetCount === 0 && (
                                        <div className="mt-4 mb-2 alert alert-info e2e-campaign-get-started">
                                            Add a changeset to get started.
                                        </div>
                                    )}
                                    <AddChangesetForm
                                        campaignID={campaign.id}
                                        onAdd={onAddChangeset}
                                        history={history}
                                    />
                                </>
                            )}
                        </>
                    )}

                    {totalChangesetCount + totalPatchCount > 0 && (
                        <>
                            <h3 className="mt-4 d-flex align-items-end mb-0">
                                {totalPatchCount > 0 && (
                                    <>
                                        {totalPatchCount} {pluralize('Patch', totalPatchCount, 'Patches')}
                                    </>
                                )}
                                {(totalChangesetCount > 0 || !!campaign) && totalPatchCount > 0 && (
                                    <span className="mx-1">/</span>
                                )}
                                {(totalChangesetCount > 0 || !!campaign) && (
                                    <>
                                        {totalChangesetCount} {pluralize('Changeset', totalChangesetCount)}
                                    </>
                                )}{' '}
                                {(patchSet || campaign) && (
                                    <CampaignDiffStat campaign={campaign} patchSet={patchSet} className="ml-2 mb-0" />
                                )}
                            </h3>
                            {totalPatchCount > 0 &&
                                (campaign ? (
                                    <CampaignPatches
                                        campaign={campaign}
                                        campaignUpdates={campaignUpdates}
                                        changesetUpdates={changesetUpdates}
                                        enablePublishing={!campaign.closedAt && campaign.viewerCanAdminister}
                                        history={history}
                                        location={location}
                                        isLightTheme={isLightTheme}
                                    />
                                ) : (
                                    <PatchSetPatches
                                        patchSet={patchSet!}
                                        campaignUpdates={campaignUpdates}
                                        changesetUpdates={changesetUpdates}
                                        history={history}
                                        location={location}
                                        isLightTheme={isLightTheme}
                                    />
                                ))}
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
            )}
        </>
    )
}
