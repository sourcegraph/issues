import React from 'react'

export const CampaignCloseChangesetsListEmptyElement: React.FunctionComponent<{}> = () => (
    <div className="col-md-8 offset-md-2 col-sm-12 card mt-5">
        <div className="card-body campaign-close-changesets-list-empty-element__body p-5">
            <div className="d-flex mb-4 justify-content-center">
                <img
                    className="campaign-close-changesets-list-empty-element__logo"
                    src="/.assets/img/sourcegraph-mark.svg"
                />
            </div>
            <h2 className="text-center">Congratulations</h2>
            <h2 className="text-center font-weight-normal mb-4">All your changesets have been closed!</h2>
        </div>
    </div>
)
