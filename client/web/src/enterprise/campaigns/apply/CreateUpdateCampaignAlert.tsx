import * as H from 'history'
import React, { useCallback, useState } from 'react'
import { CampaignSpecFields } from '../../../graphql-operations'
import { createCampaign, applyCampaign } from './backend'
import { Link } from '../../../../../shared/src/components/Link'
import classNames from 'classnames'
import { isErrorLike } from '../../../../../shared/src/util/errors'
import { ErrorAlert } from '../../../components/alerts'

export interface CreateUpdateCampaignAlertProps {
    specID: string
    campaign: CampaignSpecFields['appliesToCampaign']
    viewerCanAdminister: boolean
    history: H.History
}

export const CreateUpdateCampaignAlert: React.FunctionComponent<CreateUpdateCampaignAlertProps> = ({
    specID,
    campaign,
    viewerCanAdminister,
    history,
}) => {
    const campaignID = campaign?.id
    const [isLoading, setIsLoading] = useState<boolean | Error>(false)
    const onApply = useCallback(async () => {
        if (!confirm(`Are you sure you want to ${campaignID ? 'update' : 'create'} this campaign?`)) {
            return
        }
        setIsLoading(true)
        try {
            const campaign = campaignID
                ? await applyCampaign({ campaignSpec: specID, campaign: campaignID })
                : await createCampaign({ campaignSpec: specID })
            history.push(campaign.url)
        } catch (error) {
            setIsLoading(error)
        }
    }, [specID, setIsLoading, history, campaignID])
    return (
        <>
            <div className="alert alert-info p-3 mb-3 d-block d-md-flex align-items-center body-lead">
                <h2 className="m-0 mr-3 create-update-campaign-alert__badge">
                    <span className="badge badge-info text-uppercase mb-0">Preview</span>
                </h2>
                {!campaign && (
                    <p className="mb-0 flex-grow-1 mr-3 create-update-campaign-alert__copy">
                        Review the proposed changesets below. Click 'Apply spec' or run 'src campaigns apply' against
                        your campaign spec to create the campaign and perform the indicated action on each changeset.
                    </p>
                )}
                {campaign && (
                    <p className="mb-0 flex-grow-1 mr-3 create-update-campaign-alert__copy">
                        This operation will update the existing campaign <Link to={campaign.url}>{campaign.name}</Link>.
                        Click 'Apply spec' or run 'src campaigns apply' against your campaign spec to update the
                        campaign and perform the indicated action on each changeset.
                    </p>
                )}
                <div className="create-update-campaign-alert__btn">
                    <button
                        type="button"
                        className={classNames(
                            'btn btn-primary test-campaigns-confirm-apply-btn text-nowrap',
                            isLoading === true || (!viewerCanAdminister && 'disabled')
                        )}
                        onClick={onApply}
                        disabled={isLoading === true || !viewerCanAdminister}
                        data-tooltip={
                            !viewerCanAdminister ? 'You have no permission to apply this campaign.' : undefined
                        }
                    >
                        Apply spec
                    </button>
                </div>
            </div>
            {isErrorLike(isLoading) && <ErrorAlert error={isLoading} history={history} />}
        </>
    )
}
