import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import * as React from 'react'
import { RouteComponentProps } from 'react-router'
import { concat, of, Subject, Subscription } from 'rxjs'
import { catchError, delay, mergeMap, startWith, switchMap, tap, map, distinctUntilKeyChanged } from 'rxjs/operators'
import { ORG_DISPLAY_NAME_MAX_LENGTH } from '../..'
import { Form } from '../../../../../branded/src/components/Form'
import { PageTitle } from '../../../components/PageTitle'
import { eventLogger } from '../../../tracking/eventLogger'
import { OrgAreaPageProps } from '../../area/OrgArea'
import { updateOrganization } from '../../backend'
import { ErrorAlert } from '../../../components/alerts'
import * as H from 'history'

interface Props extends OrgAreaPageProps, RouteComponentProps<{}> {
    history: H.History
}

interface State {
    displayName: string
    loading: boolean
    updated: boolean
    error?: string
}

/**
 * The organization profile settings page.
 */
export class OrgSettingsProfilePage extends React.PureComponent<Props, State> {
    private componentUpdates = new Subject<Props>()
    private submits = new Subject<void>()
    private subscriptions = new Subscription()

    constructor(props: Props) {
        super(props)

        this.state = {
            displayName: props.org.displayName || '',
            loading: false,
            updated: false,
        }
    }

    public componentDidMount(): void {
        this.subscriptions.add(
            this.componentUpdates
                .pipe(
                    map(props => props.org),
                    distinctUntilKeyChanged('id')
                )
                .subscribe(() => {
                    eventLogger.logViewEvent('OrgSettingsProfile')
                })
        )

        this.subscriptions.add(
            this.submits
                .pipe(
                    switchMap(() =>
                        updateOrganization(this.props.org.id, this.state.displayName).pipe(
                            tap(() => this.props.onOrganizationUpdate()),
                            mergeMap(() =>
                                concat(
                                    // Reset email, reenable submit button, flash "updated" text
                                    of<Partial<State>>({ loading: false, updated: true }),
                                    // Hide "updated" text again after 1s
                                    of<Partial<State>>({ updated: false }).pipe(delay(1000))
                                )
                            ),
                            catchError((error: Error) => [{ error: error.message, loading: false }]),
                            // Disable button while loading
                            startWith<Partial<State>>({ loading: true, error: undefined })
                        )
                    )
                )
                .subscribe(state => this.setState(state as State))
        )
        // TODO(sqs): handle errors

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
            <div className="org-settings-profile-page">
                <PageTitle title={this.props.org.name} />
                <h2>Organization profile</h2>
                <Form className="org-settings-profile-page" onSubmit={this.onSubmit}>
                    <div className="form-group">
                        <label>Display name</label>
                        <input
                            type="text"
                            className="form-control org-settings-profile-page__display-name"
                            placeholder="Organization name"
                            onChange={this.onDisplayNameFieldChange}
                            value={this.state.displayName}
                            spellCheck={false}
                            maxLength={ORG_DISPLAY_NAME_MAX_LENGTH}
                        />
                    </div>
                    <button
                        type="submit"
                        disabled={this.state.loading}
                        className="btn btn-primary org-settings-profile-page__submit-button"
                    >
                        Update
                    </button>
                    {this.state.loading && <LoadingSpinner className="icon-inline" />}
                    <div
                        className={
                            'org-settings-profile-page__updated-text' +
                            (this.state.updated ? ' org-settings-profile-page__updated-text--visible' : '')
                        }
                    >
                        <small>Updated!</small>
                    </div>
                    {this.state.error && <ErrorAlert error={this.state.error} history={this.props.history} />}
                </Form>
            </div>
        )
    }

    private onDisplayNameFieldChange = (event: React.ChangeEvent<HTMLInputElement>): void => {
        this.setState({ displayName: event.target.value })
    }

    private onSubmit = (event: React.FormEvent<HTMLFormElement>): void => {
        event.preventDefault()
        this.submits.next()
    }
}
