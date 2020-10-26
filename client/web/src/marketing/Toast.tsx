import CloseIcon from 'mdi-react/CloseIcon'
import * as React from 'react'

interface Props {
    icon: JSX.Element
    title: string
    subtitle?: string
    cta?: JSX.Element
    onDismiss: () => void
}

export const Toast: React.FunctionComponent<Props> = props => (
    <div className="toast card">
        <div className="card-body px-3 pb-3 pt-2">
            <header className="card-title d-flex align-items-center">
                <span className="toast__logo icon-inline">{props.icon}</span>
                <h2 className="mb-0">{props.title}</h2>
                <div className="flex-1" />
                <button
                    type="button"
                    onClick={props.onDismiss}
                    className="toast__close-button btn btn-icon test-close-toast"
                    aria-label="Close"
                >
                    <CloseIcon className="icon-inline" />
                </button>
            </header>
            {props.subtitle}
            {props.cta && <div className="toast__contents-cta">{props.cta}</div>}
        </div>
    </div>
)
