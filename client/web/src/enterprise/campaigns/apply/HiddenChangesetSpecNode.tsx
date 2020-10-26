import React from 'react'
import { HiddenChangesetSpecFields, ChangesetSpecType } from '../../../graphql-operations'
import InfoCircleOutlineIcon from 'mdi-react/InfoCircleOutlineIcon'
import { ChangesetSpecAction } from './ChangesetSpecAction'

export interface HiddenChangesetSpecNodeProps {
    node: HiddenChangesetSpecFields
}

export const HiddenChangesetSpecNode: React.FunctionComponent<HiddenChangesetSpecNodeProps> = ({ node }) => (
    <>
        <span className="d-none d-sm-block" />
        <ChangesetSpecAction spec={node} className="hidden-changeset-spec-node__action" />
        <div className="d-flex flex-column hidden-changeset-spec-node__information">
            <h3 className="text-muted">
                {node.type === ChangesetSpecType.EXISTING && <>Import changeset from a private repository</>}
                {node.type === ChangesetSpecType.BRANCH && <>Create changeset in a private repository</>}
            </h3>
            <span className="text-danger">
                No action will be taken on apply.{' '}
                <InfoCircleOutlineIcon
                    className="icon-inline"
                    data-tooltip="You have no permissions to access this repository."
                />
            </span>
        </div>
        <span />
    </>
)
