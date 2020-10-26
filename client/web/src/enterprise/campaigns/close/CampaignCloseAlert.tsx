import React, { useCallback, useState } from 'react'
import * as H from 'history'
import { closeCampaign as _closeCampaign } from './backend'
import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { isErrorLike, asError } from '../../../../../shared/src/util/errors'
import { ErrorAlert } from '../../../components/alerts'
import { Scalars } from '../../../graphql-operations'

export interface CampaignCloseAlertProps {
    campaignID: Scalars['ID']
    campaignURL: string
    closeChangesets: boolean
    viewerCanAdminister: boolean
    totalCount: number
    setCloseChangesets: (newValue: boolean) => void
    history: H.History

    /** For testing only. */
    closeCampaign?: typeof _closeCampaign
}

export const CampaignCloseAlert: React.FunctionComponent<CampaignCloseAlertProps> = ({
    campaignID,
    campaignURL,
    closeChangesets,
    totalCount,
    setCloseChangesets,
    viewerCanAdminister,
    history,
    closeCampaign = _closeCampaign,
}) => {
    const onChangeCloseChangesets = useCallback<React.ChangeEventHandler<HTMLInputElement>>(
        event => {
            setCloseChangesets(event.target.checked)
        },
        [setCloseChangesets]
    )
    const onCancel = useCallback<React.MouseEventHandler>(() => {
        history.push(campaignURL)
    }, [history, campaignURL])
    const [isClosing, setIsClosing] = useState<boolean | Error>(false)
    const onClose = useCallback<React.MouseEventHandler>(async () => {
        setIsClosing(true)
        try {
            await closeCampaign({ campaign: campaignID, closeChangesets })
            history.push(campaignURL)
        } catch (error) {
            setIsClosing(asError(error))
        }
    }, [history, closeChangesets, closeCampaign, campaignID, campaignURL])
    return (
        <>
            <div className="card mb-3">
                <div className="card-body p-3">
                    <p>
                        <strong>
                            After closing this campaign, it will be read-only and no new campaign specs can be applied.
                        </strong>
                    </p>
                    {totalCount > 0 && (
                        <>
                            <p>By default, all changesets remain untouched.</p>
                            <div className="form-check mb-3">
                                <input
                                    id="closeChangesets"
                                    type="checkbox"
                                    checked={closeChangesets}
                                    onChange={onChangeCloseChangesets}
                                    className="test-campaigns-close-changesets-checkbox form-check-input"
                                    disabled={isClosing === true || !viewerCanAdminister}
                                />
                                <label className="form-check-label" htmlFor="closeChangesets">
                                    Also close all {totalCount} open changesets on code hosts.
                                </label>
                            </div>
                            {!viewerCanAdminister && (
                                <p className="text-warning">
                                    You don't have permission to close this campaign. See{' '}
                                    <a href="https://docs.sourcegraph.com/campaigns/explanations/permissions_in_campaigns">
                                        Permissions in campaigns
                                    </a>{' '}
                                    for more information about the campaigns permission model.
                                </p>
                            )}
                        </>
                    )}
                    <div className="d-flex justify-content-end">
                        <button
                            type="button"
                            className="btn btn-secondary mr-3 test-campaigns-close-abort-btn"
                            onClick={onCancel}
                            disabled={isClosing === true || !viewerCanAdminister}
                        >
                            Cancel
                        </button>
                        <button
                            type="button"
                            className="btn btn-danger test-campaigns-confirm-close-btn"
                            onClick={onClose}
                            disabled={isClosing === true || !viewerCanAdminister}
                        >
                            {isClosing === true && <LoadingSpinner className="icon-inline" />} Close campaign
                        </button>
                    </div>
                </div>
            </div>
            {isErrorLike(isClosing) && <ErrorAlert error={isClosing} history={history} />}
        </>
    )
}
