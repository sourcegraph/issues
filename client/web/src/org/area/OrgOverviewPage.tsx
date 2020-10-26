import * as React from 'react'
import { RouteComponentProps } from 'react-router'
import { PageTitle } from '../../components/PageTitle'
import { Timestamp } from '../../components/time/Timestamp'
import { eventLogger } from '../../tracking/eventLogger'
import { OrgAreaPageProps } from './OrgArea'

interface Props extends OrgAreaPageProps, RouteComponentProps<{}> {}

interface State {}

/**
 * The organization overview page.
 */
export class OrgOverviewPage extends React.PureComponent<Props, State> {
    public componentDidMount(): void {
        eventLogger.logViewEvent('OrgOverview')
    }

    public render(): JSX.Element | null {
        return (
            <div className="org-page org-overview-page">
                <PageTitle title={this.props.org.name} />
                <p>
                    {this.props.org.displayName ? (
                        <>
                            {this.props.org.displayName} ({this.props.org.name})
                        </>
                    ) : (
                        this.props.org.name
                    )}{' '}
                    was created <Timestamp date={this.props.org.createdAt} />.
                </p>
            </div>
        )
    }
}
