import InformationOutlineIcon from 'mdi-react/InformationOutlineIcon'
import * as React from 'react'
import { Link } from 'react-router-dom'

export const DirectImportRepoAlert: React.FunctionComponent<{ className?: string }> = ({ className = '' }) => (
    <>
        {['dev', 'docker-container'].includes(window.context.deployType) && (
            <div className={`alert alert-info ${className}`}>
                <InformationOutlineIcon className="icon-inline" /> Very large repository? See{' '}
                <Link to="/help/admin/repo/pre_load_from_local_disk#add-repositories-already-cloned-to-disk">
                    how to reuse an existing local clone
                </Link>{' '}
                to speed this up.
            </div>
        )}
    </>
)
