import * as React from 'react'
import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'

export const LoadingPanelView: React.FunctionComponent<{ text: string }> = ({ text }) => (
    <div className="d-flex justify-content-center align-items-center panel-container__empty-container panel-container__loading-container">
        <div className="icon-inline">
            <LoadingSpinner />
        </div>
        <span className="text-muted">{text}</span>
    </div>
)
