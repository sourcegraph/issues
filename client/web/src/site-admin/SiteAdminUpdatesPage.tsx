import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { parseISO } from 'date-fns'
import formatDistance from 'date-fns/formatDistance'
import { upperFirst } from 'lodash'
import CheckIcon from 'mdi-react/CheckIcon'
import CloudDownloadIcon from 'mdi-react/CloudDownloadIcon'
import * as React from 'react'
import { RouteComponentProps } from 'react-router'
import { Link } from 'react-router-dom'
import { Subscription } from 'rxjs'
import { withLatestFrom } from 'rxjs/operators'
import * as GQL from '../../../shared/src/graphql/schema'
import { PageTitle } from '../components/PageTitle'
import { eventLogger } from '../tracking/eventLogger'
import { fetchSite, fetchSiteUpdateCheck } from './backend'
import { ErrorAlert } from '../components/alerts'
import { asError } from '../../../shared/src/util/errors'

interface Props extends RouteComponentProps<{}> {}

interface State {
    buildVersion?: string
    productVersion?: string
    updateCheck?: GQL.IUpdateCheck
    error?: string
}

/**
 * A page displaying information about available updates for the server.
 */
export class SiteAdminUpdatesPage extends React.Component<Props, State> {
    public state: State = {}

    private subscriptions = new Subscription()

    public componentDidMount(): void {
        eventLogger.logViewEvent('SiteAdminUpdates')

        this.subscriptions.add(
            fetchSite()
                .pipe(withLatestFrom(fetchSiteUpdateCheck()))
                .subscribe(
                    ([, { buildVersion, productVersion, updateCheck }]) =>
                        this.setState({
                            buildVersion,
                            productVersion,
                            updateCheck,
                            error: undefined,
                        }),
                    error => this.setState({ error: asError(error).message })
                )
        )
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        const autoUpdateCheckingEnabled = window.context.site['update.channel'] === 'release'
        return (
            <div className="site-admin-updates-page">
                <PageTitle title="Updates - Admin" />
                <h2>Updates</h2>
                {this.state.error && (
                    <p className="site-admin-updates-page__error">Error: {upperFirst(this.state.error)}</p>
                )}
                {this.state.updateCheck && (this.state.updateCheck.pending || this.state.updateCheck.checkedAt) && (
                    <div>
                        {this.state.updateCheck.pending && (
                            <div className="site-admin-updates-page__alert alert alert-primary">
                                <LoadingSpinner className="icon-inline" /> Checking for updates... (reload in a few
                                seconds)
                            </div>
                        )}
                        {!this.state.updateCheck.errorMessage &&
                            (this.state.updateCheck.updateVersionAvailable ? (
                                <div className="site-admin-updates-page__alert alert alert-success">
                                    <CloudDownloadIcon className="icon-inline" /> Update available:{' '}
                                    <a href="https://about.sourcegraph.com">
                                        {this.state.updateCheck.updateVersionAvailable}
                                    </a>
                                </div>
                            ) : (
                                <div className="site-admin-updates-page__alert alert alert-success">
                                    <CheckIcon className="icon-inline" /> Up to date.
                                </div>
                            ))}
                        {this.state.updateCheck.errorMessage && (
                            <ErrorAlert
                                className="site-admin-updates-page__alert"
                                prefix="Error checking for updates"
                                error={this.state.updateCheck.errorMessage}
                                history={this.props.history}
                            />
                        )}
                    </div>
                )}
                {!autoUpdateCheckingEnabled && (
                    <div className="site-admin-updates-page__alert alert alert-warning">
                        Automatic update checking is disabled.
                    </div>
                )}

                <p className="site-admin-updates_page__info">
                    <small>
                        <strong>Current product version:</strong> {this.state.productVersion} ({this.state.buildVersion}
                        )
                    </small>
                    <br />
                    <small>
                        <strong>Last update check:</strong>{' '}
                        {this.state.updateCheck?.checkedAt
                            ? formatDistance(parseISO(this.state.updateCheck.checkedAt), new Date(), {
                                  addSuffix: true,
                              })
                            : 'never'}
                        .
                    </small>
                    <br />
                    <small>
                        <strong>Automatic update checking:</strong> {autoUpdateCheckingEnabled ? 'on' : 'off'}.{' '}
                        <Link to="/site-admin/configuration">Configure</Link> <code>update.channel</code> to{' '}
                        {autoUpdateCheckingEnabled ? 'disable' : 'enable'}.
                    </small>
                </p>
                <p>
                    {/* eslint-disable-next-line react/jsx-no-target-blank */}
                    <a href="https://about.sourcegraph.com/changelog" target="_blank" rel="noopener">
                        Sourcegraph changelog
                    </a>
                </p>
            </div>
        )
    }
}
