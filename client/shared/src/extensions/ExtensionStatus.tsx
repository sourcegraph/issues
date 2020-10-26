import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import MenuUpIcon from 'mdi-react/MenuUpIcon'
import * as React from 'react'
import { UncontrolledPopover } from 'reactstrap'
import { Subject, Subscription } from 'rxjs'
import { catchError, distinctUntilChanged, map, switchMap } from 'rxjs/operators'
import { ExecutableExtension } from '../api/client/services/extensionsService'
import { Link } from '../components/Link'
import { ExtensionsControllerProps } from './controller'
import { PlatformContextProps } from '../platform/context'
import { asError, ErrorLike, isErrorLike } from '../util/errors'

interface Props extends ExtensionsControllerProps, PlatformContextProps<'sideloadedExtensionURL'> {
    link: React.ComponentType<{ id: string }>
}

interface State {
    /** The extension IDs of extensions that are active, an error, or undefined while loading. */
    extensionsOrError?: Pick<ExecutableExtension, 'id'>[] | ErrorLike

    sideloadedExtensionURL?: string | null
}

class ExtensionStatus extends React.PureComponent<Props, State> {
    public state: State = {}

    private componentUpdates = new Subject<Props>()
    private subscriptions = new Subscription()

    public componentDidMount(): void {
        const extensionsController = this.componentUpdates.pipe(
            map(({ extensionsController }) => extensionsController),
            distinctUntilChanged()
        )
        this.subscriptions.add(
            extensionsController
                .pipe(
                    switchMap(extensionsController => extensionsController.services.extensions.activeExtensions),
                    catchError(error => [asError(error)]),
                    map(extensionsOrError => ({ extensionsOrError }))
                )
                .subscribe(
                    stateUpdate => this.setState(stateUpdate),
                    error => console.error(error)
                )
        )

        const platformContext = this.componentUpdates.pipe(
            map(({ platformContext }) => platformContext),
            distinctUntilChanged()
        )

        this.subscriptions.add(
            platformContext
                .pipe(switchMap(({ sideloadedExtensionURL }) => sideloadedExtensionURL))
                .subscribe(sideloadedExtensionURL => this.setState({ sideloadedExtensionURL }))
        )

        this.componentUpdates.next(this.props)
    }

    public componentDidUpdate(): void {
        this.componentUpdates.next(this.props)
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        return (
            <div className="extension-status card border-0">
                <div className="card-header">Active extensions (DEBUG)</div>
                {this.state.extensionsOrError ? (
                    isErrorLike(this.state.extensionsOrError) ? (
                        <div className="alert alert-danger mb-0 rounded-0">{this.state.extensionsOrError.message}</div>
                    ) : this.state.extensionsOrError.length > 0 ? (
                        <div className="list-group list-group-flush">
                            {this.state.extensionsOrError.map(({ id }, index) => (
                                <div
                                    key={index}
                                    className="list-group-item py-2 d-flex align-items-center justify-content-between"
                                >
                                    <this.props.link id={id} />
                                </div>
                            ))}
                        </div>
                    ) : (
                        <span className="card-body">No active extensions.</span>
                    )
                ) : (
                    <span className="card-body">
                        <LoadingSpinner className="icon-inline" /> Loading extensions...
                    </span>
                )}
                <div className="card-body border-top">
                    <h6>Sideload extension</h6>
                    {this.state.sideloadedExtensionURL ? (
                        <div>
                            <p>
                                <span>Load from: </span>
                                <Link to={this.state.sideloadedExtensionURL}>{this.state.sideloadedExtensionURL}</Link>
                            </p>
                            <div>
                                <button
                                    type="button"
                                    className="btn btn-sm btn-primary mr-1"
                                    onClick={this.setSideloadedExtensionURL}
                                >
                                    Change
                                </button>
                                <button
                                    type="button"
                                    className="btn btn-sm btn-danger"
                                    onClick={this.clearSideloadedExtensionURL}
                                >
                                    Clear
                                </button>
                            </div>
                        </div>
                    ) : (
                        <div>
                            <p>
                                <span>No sideloaded extension</span>
                            </p>
                            <div>
                                <button
                                    type="button"
                                    className="btn btn-sm btn-primary"
                                    onClick={this.setSideloadedExtensionURL}
                                >
                                    Load extension
                                </button>
                            </div>
                        </div>
                    )}
                </div>
            </div>
        )
    }

    private setSideloadedExtensionURL = (): void => {
        const url = window.prompt(
            'Parcel dev server URL:',
            this.state.sideloadedExtensionURL || 'http://localhost:1234'
        )
        this.props.platformContext.sideloadedExtensionURL.next(url)
    }

    private clearSideloadedExtensionURL = (): void => {
        this.props.platformContext.sideloadedExtensionURL.next(null)
    }
}

/** A button that toggles the visibility of the ExtensionStatus element in a popover. */
export class ExtensionStatusPopover extends React.PureComponent<Props> {
    public render(): JSX.Element | null {
        return (
            <>
                <button type="button" id="extension-status-popover" className="btn btn-link text-decoration-none px-2">
                    <span className="text-muted">Ext</span> <MenuUpIcon className="icon-inline" />
                </button>
                <UncontrolledPopover
                    placement="auto-end"
                    target="extension-status-popover"
                    hideArrow={true}
                    popperClassName="border-0"
                >
                    <ExtensionStatus {...this.props} />
                </UncontrolledPopover>
            </>
        )
    }
}
