import React from 'react'
import { RouteComponentProps, Switch, Route } from 'react-router'
import { GlobalCampaignListPage } from './list/GlobalCampaignListPage'
import { CampaignDetails } from '../detail/CampaignDetails'
import { withAuthenticatedUser } from '../../../auth/withAuthenticatedUser'
import { ThemeProps } from '../../../../../shared/src/theme'
import { CreateCampaign } from './create/CreateCampaign'
import { ExtensionsControllerProps } from '../../../../../shared/src/extensions/controller'
import { PlatformContextProps } from '../../../../../shared/src/platform/context'
import { TelemetryProps } from '../../../../../shared/src/telemetry/telemetryService'
import { CampaignCliHelp } from './create/CampaignCliHelp'
import { CampaignsDotComPage } from './marketing/CampaignsDotComPage'
import { CampaignsSiteAdminMarketingPage } from './marketing/CampaignsSiteAdminMarketingPage'
import { CampaignsUserMarketingPage } from './marketing/CampaignsUserMarketingPage'
import { CampaignsBetaFeedbackAlert } from './CampaignsBetaFeedbackAlert'
import { AuthenticatedUser } from '../../../auth'

interface Props
    extends RouteComponentProps<{}>,
        ThemeProps,
        ExtensionsControllerProps,
        TelemetryProps,
        PlatformContextProps {
    authenticatedUser: AuthenticatedUser | null
    isSourcegraphDotCom: boolean
}

/**
 * The global campaigns area.
 */
export const GlobalCampaignsArea: React.FunctionComponent<Props> = props => {
    if (props.isSourcegraphDotCom) {
        return (
            <div className="container mt-4">
                <CampaignsDotComPage />
            </div>
        )
    }
    return <AuthenticatedCampaignsArea {...props} />
}

interface AuthenticatedProps extends Props {
    authenticatedUser: AuthenticatedUser
}

export const AuthenticatedCampaignsArea = withAuthenticatedUser<AuthenticatedProps>(({ match, ...outerProps }) => {
    let content: React.ReactFragment
    if (window.context.experimentalFeatures?.automation === 'enabled') {
        if (!outerProps.authenticatedUser.siteAdmin && window.context.site['campaigns.readAccess.enabled'] !== true) {
            content = <CampaignsUserMarketingPage enableReadAccess={true} />
        } else {
            content = (
                <>
                    <CampaignsBetaFeedbackAlert />
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
    } else if (outerProps.authenticatedUser.siteAdmin) {
        content = <CampaignsSiteAdminMarketingPage />
    } else {
        content = <CampaignsUserMarketingPage enableReadAccess={false} />
    }
    return <div className="container mt-4">{content}</div>
})
