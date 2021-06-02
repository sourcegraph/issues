import classNames from 'classnames'
import * as H from 'history'
import * as React from 'react'

import { asError } from '@sourcegraph/shared/src/util/errors'

import { ErrorAlert } from '../../../components/alerts'

export const BaseActionContainer: React.FunctionComponent<{
    title: React.ReactFragment
    description: React.ReactFragment
    action: React.ReactFragment
    details?: React.ReactFragment
    className?: string
}> = ({ title, description, action, details, className }) => (
    <div className={classNames('action-container', className)}>
        <div className="action-container__row">
            <div className="action-container__description">
                <h4 className="action-container__title">{title}</h4>
                {description}
            </div>
            <div className="action-container__btn-container">{action}</div>
        </div>
        {details && <div className="action-container__row">{details}</div>}
    </div>
)

interface Props {
    title: React.ReactFragment
    description: React.ReactFragment
    buttonClassName?: string
    buttonLabel: React.ReactFragment
    buttonSubtitle?: string
    buttonDisabled?: boolean
    info?: React.ReactNode
    className?: string

    /** The message to briefly display below the button when the action is successful. */
    flashText?: string

    run: () => Promise<void>
    history: H.History
}

interface State {
    loading: boolean
    flash: boolean
    error?: string
}

/**
 * Displays an action button in a container with a title and description.
 */
export class ActionContainer extends React.PureComponent<Props, State> {
    public state: State = {
        loading: false,
        flash: false,
    }

    private timeoutHandle?: number

    public componentWillUnmount(): void {
        if (this.timeoutHandle) {
            window.clearTimeout(this.timeoutHandle)
        }
    }

    public render(): JSX.Element | null {
        return (
            <BaseActionContainer
                title={this.props.title}
                description={this.props.description}
                className={this.props.className}
                action={
                    <>
                        <button
                            type="button"
                            className={`btn ${this.props.buttonClassName || 'btn-primary'} action-container__btn`}
                            onClick={this.onClick}
                            data-tooltip={this.props.buttonSubtitle}
                            disabled={this.props.buttonDisabled || this.state.loading}
                        >
                            {this.props.buttonLabel}
                        </button>
                        {this.props.buttonSubtitle && (
                            <div className="action-container__btn-subtitle">
                                <small>{this.props.buttonSubtitle}</small>
                            </div>
                        )}
                        {!this.props.buttonSubtitle && this.props.flashText && (
                            <div
                                className={
                                    'action-container__flash' +
                                    (this.state.flash ? ' action-container__flash--visible' : '')
                                }
                            >
                                <small>{this.props.flashText}</small>
                            </div>
                        )}
                    </>
                }
                details={
                    <>
                        {this.state.error ? (
                            <ErrorAlert className="mb-0 mt-3" error={this.state.error} />
                        ) : (
                            this.props.info
                        )}
                    </>
                }
            />
        )
    }

    private onClick = (): void => {
        this.setState({
            error: undefined,
            loading: true,
        })

        this.props.run().then(
            () => {
                this.setState({ loading: false, flash: true })
                if (typeof this.timeoutHandle === 'number') {
                    window.clearTimeout(this.timeoutHandle)
                }
                this.timeoutHandle = window.setTimeout(() => this.setState({ flash: false }), 1000)
            },
            error => this.setState({ loading: false, error: asError(error).message })
        )
    }
}
