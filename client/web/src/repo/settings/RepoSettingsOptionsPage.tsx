import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import * as React from 'react'
import { RouteComponentProps } from 'react-router'
import { Subject, Subscription } from 'rxjs'
import { switchMap } from 'rxjs/operators'
import * as GQL from '../../../../shared/src/graphql/schema'
import { ExternalServiceCard } from '../../components/externalServices/ExternalServiceCard'
import { Form } from '../../../../branded/src/components/Form'
import { PageTitle } from '../../components/PageTitle'
import { eventLogger } from '../../tracking/eventLogger'
import { fetchRepository } from './backend'
import { ErrorAlert } from '../../components/alerts'
import { defaultExternalServices } from '../../components/externalServices/externalServices'
import { asError } from '../../../../shared/src/util/errors'
import * as H from 'history'

interface Props extends RouteComponentProps<{}> {
    repo: GQL.IRepository
    onDidUpdateRepository: (update: Partial<GQL.IRepository>) => void
    history: H.History
}

interface State {
    /**
     * The repository object, refreshed after we make changes that modify it.
     */
    repo: GQL.IRepository

    loading: boolean
    error?: string
}

/**
 * The repository settings options page.
 */
export class RepoSettingsOptionsPage extends React.PureComponent<Props, State> {
    private repoUpdates = new Subject<void>()
    private subscriptions = new Subscription()

    constructor(props: Props) {
        super(props)

        this.state = {
            loading: false,
            repo: props.repo,
        }
    }

    public componentDidMount(): void {
        eventLogger.logViewEvent('RepoSettings')

        this.subscriptions.add(
            this.repoUpdates.pipe(switchMap(() => fetchRepository(this.props.repo.name))).subscribe(
                repo => this.setState({ repo }),
                error => this.setState({ error: asError(error).message })
            )
        )
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        const services = this.state.repo.externalServices.nodes
        return (
            <div className="repo-settings-options-page">
                <PageTitle title="Repository settings" />
                <h2>Settings</h2>
                {this.state.loading && <LoadingSpinner className="icon-inline" />}
                {this.state.error && <ErrorAlert error={this.state.error} history={this.props.history} />}
                {services.length > 0 && (
                    <div className="mb-4">
                        {services.map(service => (
                            <div className="mb-3" key={service.id}>
                                <ExternalServiceCard
                                    {...defaultExternalServices[service.kind]}
                                    kind={service.kind}
                                    title={service.displayName}
                                    shortDescription="Update this external service configuration to manage repository mirroring."
                                    to={`/site-admin/external-services/${service.id}`}
                                />
                            </div>
                        ))}
                        {services.length > 1 && (
                            <p>
                                This repository is mirrored by multiple external services. To change access, disable, or
                                remove this repository, the configuration must be updated on all external services.
                            </p>
                        )}
                    </div>
                )}
                <Form>
                    <div className="form-group">
                        <label htmlFor="repo-settings-options-page__name">Repository name</label>
                        <input
                            id="repo-settings-options-page__name"
                            type="text"
                            className="form-control"
                            readOnly={true}
                            disabled={true}
                            value={this.state.repo.name}
                            required={true}
                            spellCheck={false}
                            autoCapitalize="off"
                            autoCorrect="off"
                            aria-describedby="repo-settings-options-page__name-help"
                        />
                    </div>
                </Form>
            </div>
        )
    }
}
