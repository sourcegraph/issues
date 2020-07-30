import format from 'date-fns/format'
import React from 'react'
import { Subscription } from 'rxjs'
import { catchError } from 'rxjs/operators'
import * as GQL from '../../../../shared/src/graphql/schema'
import { asError, ErrorLike, isErrorLike } from '../../../../shared/src/util/errors'
import { BarChart } from '../../components/d3/BarChart'
import { fetchSiteUsageStatistics } from '../../site-admin/backend'
import { ErrorAlert } from '../../components/alerts'
import * as H from 'history'

interface Props {
    isLightTheme: boolean
    history: H.History
}

const LOADING = 'loading' as const

interface State {
    /** The site usage statistics, loading, or an error. */
    siteUsageStatisticsOrError: typeof LOADING | GQL.SiteUsageStatistics | ErrorLike
}

/**
 * An explore section that shows site usage statistics.
 */
export class SiteUsageExploreSection extends React.PureComponent<Props, State> {
    public state: State = { siteUsageStatisticsOrError: LOADING }

    private subscriptions = new Subscription()

    public componentDidMount(): void {
        this.subscriptions.add(
            fetchSiteUsageStatistics()
                .pipe(catchError(error => [asError(error)]))
                .subscribe(siteUsageStatisticsOrError => this.setState({ siteUsageStatisticsOrError }))
        )
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        return (
            <div className="site-usage-explore-section">
                <h2>Site usage</h2>
                {isErrorLike(this.state.siteUsageStatisticsOrError) ? (
                    <ErrorAlert error={this.state.siteUsageStatisticsOrError} history={this.props.history} />
                ) : this.state.siteUsageStatisticsOrError === LOADING ? (
                    <p>Loading...</p>
                ) : (
                    <div className="col-md-10 col-lg-8 mt-4">
                        <BarChart
                            showLabels={true}
                            showLegend={true}
                            width={500}
                            height={200}
                            isLightTheme={this.props.isLightTheme}
                            data={this.state.siteUsageStatisticsOrError.waus.slice(0, 4).map(usagePeriod => ({
                                xLabel: format(Date.parse(usagePeriod.startTime) + 1000 * 60 * 60 * 24, 'E, MMM d'),
                                yValues: {
                                    'Weekly users': usagePeriod.registeredUserCount + usagePeriod.anonymousUserCount,
                                },
                            }))}
                        />
                    </div>
                )}
            </div>
        )
    }
}
