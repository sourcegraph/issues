import React from 'react'
import { RouteComponentProps, Switch, Route } from 'react-router'
import { GlobalCampaignListPage } from './list/GlobalCampaignListPage'
import { CampaignDetails } from '../detail/CampaignDetails'
import { IUser } from '../../../../../shared/src/graphql/schema'
import { ThemeProps } from '../../../../../shared/src/theme'
import { CreateCampaign } from './create/CreateCampaign'
import { ExtensionsControllerProps } from '../../../../../shared/src/extensions/controller'
import { PlatformContextProps } from '../../../../../shared/src/platform/context'
import { TelemetryProps } from '../../../../../shared/src/telemetry/telemetryService'
import { CampaignUpdateSelection } from '../detail/CampaignUpdateSelection'
import { CampaignCliHelp } from './create/CampaignCliHelp'
import { CampaignsDotComPage } from './marketing/CampaignsDotComPage'
import { CampaignsSiteAdminMarketingPage } from './marketing/CampaignsSiteAdminMarketingPage'
import { CampaignsUserMarketingPage } from './marketing/CampaignsUserMarketingPage'
import { DismissibleAlert } from '../../../components/DismissibleAlert'

interface Props
    extends RouteComponentProps<{}>,
        ThemeProps,
        ExtensionsControllerProps,
        TelemetryProps,
        PlatformContextProps {
    authenticatedUser: IUser | null
    isSourcegraphDotCom: boolean
}

/**
 * The global campaigns area.
 */
export const GlobalCampaignsArea: React.FunctionComponent<Props> = ({ match, ...outerProps }) => {
    let content: React.ReactFragment
    if (outerProps.isSourcegraphDotCom) {
        content = <CampaignsDotComPage {...outerProps} />
    } else if (window.context.experimentalFeatures?.automation === 'enabled') {
        if (
            (!outerProps.authenticatedUser || !outerProps.authenticatedUser.siteAdmin) &&
            window.context.site['campaigns.readAccess.enabled'] !== true
        ) {
            content = <CampaignsUserMarketingPage {...outerProps} enableReadAccess={true} />
        } else {
            content = (
                <>
                    <DismissibleAlert partialStorageKey="campaigns-beta" className="alert-info">
                        <p className="mb-0">
                            Campaigns are currently in beta. During the beta period, campaigns are free to use. After
                            the beta period, campaigns will be available as a paid add-on. Get in touch on Twitter{' '}
                            <a href="https://twitter.com/srcgraph">@srcgraph</a>, file an issue in our{' '}
                            <a href="https://github.com/sourcegraph/sourcegraph/issues">public issue tracker</a>, or
                            email{' '}
                            <a href="mailto:feedback@sourcegraph.com?subject=Feedback on Campaigns">
                                feedback@sourcegraph.com
                            </a>
                            . We're looking forward to your feedback!
                        </p>
                    </DismissibleAlert>
                    {/* eslint-disable react/jsx-no-bind */}
                    <Switch>
                        <Route
                            render={props => <GlobalCampaignListPage {...outerProps} {...props} />}
                            path={match.url}
                            exact={true}
                        />
                        <Route
                            path={`${match.url}/create`}
                            render={props => <CreateCampaign {...outerProps} {...props} />}
                            exact={true}
                        />
                        <Route
                            path={`${match.url}/cli`}
                            render={props => <CampaignCliHelp {...outerProps} {...props} />}
                            exact={true}
                        />
                        <Route
                            path={`${match.url}/new`}
                            render={props => <CampaignDetails {...outerProps} {...props} />}
                            exact={true}
                        />
                        <Route
                            path={`${match.url}/update`}
                            render={props => <CampaignUpdateSelection {...outerProps} {...props} />}
                            exact={true}
                        />
                        <Route
                            path={`${match.url}/:campaignID`}
                            render={({ match, ...props }: RouteComponentProps<{ campaignID: string }>) => (
                                <CampaignDetails {...outerProps} {...props} campaignID={match.params.campaignID} />
                            )}
                        />
                    </Switch>
                    {/* eslint-enable react/jsx-no-bind */}
                </>
            )
        }
    } else if (outerProps.authenticatedUser?.siteAdmin) {
        content = <CampaignsSiteAdminMarketingPage {...outerProps} />
    } else {
        content = <CampaignsUserMarketingPage {...outerProps} enableReadAccess={false} />
    }
    return <div className="container mt-4">{content}</div>
}
