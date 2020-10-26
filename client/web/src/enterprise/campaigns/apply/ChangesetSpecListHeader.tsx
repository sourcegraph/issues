import React from 'react'

export interface ChangesetSpecListHeaderProps {
    // Nothing for now.
}

export const ChangesetSpecListHeader: React.FunctionComponent<ChangesetSpecListHeaderProps> = () => (
    <>
        <span className="d-none d-sm-block" />
        <h5 className="d-none d-sm-block text-uppercase text-center text-nowrap">Action</h5>
        <h5 className="d-none d-sm-block text-uppercase text-nowrap">Changeset information</h5>
        <h5 className="d-none d-sm-block text-uppercase text-center text-nowrap">Changes</h5>
    </>
)
