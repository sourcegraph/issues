import classNames from 'classnames'
import React from 'react'
import { Link } from '../../../../shared/src/components/Link'

export const SearchPageFooter: React.FunctionComponent<{ className?: string }> = ({ className }) => (
    <footer className={classNames(className, 'd-flex flex-column flex-lg-row align-items-center')}>
        <h4 className="mb-2 mb-lg-0">Explore and Extend</h4>
        <span className="d-flex flex-column flex-md-row align-items-center">
            <span className="d-flex flex-row mb-2 mb-md-0">
                <Link
                    className="px-3"
                    to="https://docs.sourcegraph.com/integration/browser_extension"
                    rel="noopener noreferrer"
                    target="_blank"
                >
                    Browser extensions
                </Link>
                <span aria-hidden="true" className="border-right d-none d-md-inline" />
                <Link className="px-3" to="/extensions" target="_blank">
                    Sourcegraph extensions
                </Link>
                <span aria-hidden="true" className="border-right d-none d-md-inline" />
            </span>
            <span className="d-flex flex-row">
                <Link
                    className="px-3"
                    to="https://docs.sourcegraph.com/integration/editor"
                    rel="noopener noreferrer"
                    target="_blank"
                >
                    Editor plugins
                </Link>
                <span aria-hidden="true" className="border-right d-none d-md-inline" />
                <Link
                    className="pl-3"
                    to="https://docs.sourcegraph.com/admin/external_service"
                    rel="noopener noreferrer"
                    target="_blank"
                >
                    Code host integrations
                </Link>
            </span>
        </span>
    </footer>
)
