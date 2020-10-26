import { upperFirst } from 'lodash'
import DeleteIcon from 'mdi-react/DeleteIcon'
import WarningIcon from 'mdi-react/WarningIcon'
import * as React from 'react'
import { Subject, Subscription } from 'rxjs'
import { catchError, map, mapTo, startWith, switchMap, tap } from 'rxjs/operators'
import * as GQL from '../../../../../shared/src/graphql/schema'
import { asError, ErrorLike, isErrorLike } from '../../../../../shared/src/util/errors'
import { deleteRegistryExtensionWithConfirmation } from '../registry/backend'

interface RegistryExtensionDeleteButtonProps {
    extension: Pick<GQL.IRegistryExtension, 'id'>

    compact?: boolean

    className?: string
    disabled?: boolean

    /** Called when the extension is deleted. */
    onDidUpdate: () => void
}

interface RegistryExtensionDeleteButtonState {
    /** Undefined means in progress, null means done or not started. */
    deletionOrError?: null | ErrorLike
}

/** A button that deletes an extension from the registry. */
export class RegistryExtensionDeleteButton extends React.PureComponent<
    RegistryExtensionDeleteButtonProps,
    RegistryExtensionDeleteButtonState
> {
    public state: RegistryExtensionDeleteButtonState = {
        deletionOrError: null,
    }

    private deletes = new Subject<void>()
    private subscriptions = new Subscription()

    public componentDidMount(): void {
        this.subscriptions.add(
            this.deletes
                .pipe(
                    switchMap(() =>
                        deleteRegistryExtensionWithConfirmation(this.props.extension.id).pipe(
                            tap(deleted => {
                                if (deleted && this.props.onDidUpdate) {
                                    this.props.onDidUpdate()
                                }
                            }),
                            mapTo(null),
                            catchError(error => [asError(error)]),
                            map(deletionOrError => ({ deletionOrError })),
                            startWith<Pick<RegistryExtensionDeleteButtonState, 'deletionOrError'>>({
                                deletionOrError: undefined,
                            })
                        )
                    )
                )
                .subscribe(
                    stateUpdate => this.setState(stateUpdate),
                    error => console.error(error)
                )
        )
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        return (
            <div className="btn-group" role="group">
                <button
                    type="button"
                    className="btn btn-danger"
                    onClick={this.deleteExtension}
                    disabled={this.props.disabled || this.state.deletionOrError === undefined}
                    title={this.props.compact ? 'Delete extension' : ''}
                >
                    <DeleteIcon className="icon-inline" /> {!this.props.compact && 'Delete extension'}
                </button>
                {isErrorLike(this.state.deletionOrError) && (
                    <button
                        type="button"
                        disabled={true}
                        className="btn btn-danger"
                        title={upperFirst(this.state.deletionOrError.message)}
                    >
                        <WarningIcon className="icon-inline" />
                    </button>
                )}
            </div>
        )
    }

    private deleteExtension = (): void => this.deletes.next()
}
