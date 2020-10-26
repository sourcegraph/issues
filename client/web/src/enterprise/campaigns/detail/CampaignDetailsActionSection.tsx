import * as H from 'history'
import React, { useCallback, useState } from 'react'
import { Scalars } from '../../../graphql-operations'
import DeleteIcon from 'mdi-react/DeleteIcon'
import { Link } from '../../../../../shared/src/components/Link'
import { deleteCampaign as _deleteCampaign } from './backend'
import { isErrorLike, asError } from '../../../../../shared/src/util/errors'
import InformationIcon from 'mdi-react/InformationIcon'

export interface CampaignDetailsActionSectionProps {
    campaignID: Scalars['ID']
    campaignClosed: boolean
    campaignNamespaceURL: string
    history: H.History

    /** For testing only. */
    deleteCampaign?: typeof _deleteCampaign
}

export const CampaignDetailsActionSection: React.FunctionComponent<CampaignDetailsActionSectionProps> = ({
    campaignID,
    campaignClosed,
    campaignNamespaceURL,
    history,
    deleteCampaign = _deleteCampaign,
}) => {
    const [isDeleting, setIsDeleting] = useState<boolean | Error>(false)
    const onDeleteCampaign = useCallback(async () => {
        if (!confirm('Do you really want to delete this campaign?')) {
            return
        }
        setIsDeleting(true)
        try {
            await deleteCampaign(campaignID)
            history.push(campaignNamespaceURL + '/campaigns')
        } catch (error) {
            setIsDeleting(asError(error))
        }
    }, [campaignID, deleteCampaign, history, campaignNamespaceURL])
    if (campaignClosed) {
        return (
            <button
                type="button"
                className="btn btn-outline-danger test-campaigns-delete-btn"
                onClick={onDeleteCampaign}
                data-tooltip="Deleting this campaign is a final action."
                disabled={isDeleting === true}
            >
                {isErrorLike(isDeleting) && <InformationIcon className="icon-inline" data-tooltip={isDeleting} />}
                <DeleteIcon className="icon-inline" /> Delete
            </button>
        )
    }
    return (
        <Link
            to={`${location.pathname}/close`}
            className="btn btn-outline-danger test-campaigns-close-btn"
            data-tooltip="View a preview of all changes that will happen when you close this campaign."
        >
            <DeleteIcon className="icon-inline" /> Close
        </Link>
    )
}
