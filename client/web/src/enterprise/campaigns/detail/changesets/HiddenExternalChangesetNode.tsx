import React from 'react'
import { HiddenExternalChangesetFields } from '../../../../graphql-operations'
import { ChangesetStatusCell } from './ChangesetStatusCell'
import { HiddenExternalChangesetInfoCell } from './HiddenExternalChangesetInfoCell'

export interface HiddenExternalChangesetNodeProps {
    node: Pick<
        HiddenExternalChangesetFields,
        'id' | 'nextSyncAt' | 'updatedAt' | 'externalState' | 'publicationState' | 'reconcilerState'
    >
}

export const HiddenExternalChangesetNode: React.FunctionComponent<HiddenExternalChangesetNodeProps> = ({ node }) => (
    <>
        <span className="d-none d-sm-block" />
        <ChangesetStatusCell changeset={node} className="hidden-external-changeset-node__status d-block d-sm-flex" />
        <HiddenExternalChangesetInfoCell node={node} className="hidden-external-changeset-node__information" />
        <span className="d-none d-sm-block" />
        <span className="d-none d-sm-block" />
        <span className="d-none d-sm-block" />
    </>
)
