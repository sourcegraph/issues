import MapSearchIcon from 'mdi-react/MapSearchIcon'
import * as React from 'react'
import { Route, RouteComponentProps, Switch } from 'react-router'
import { ExtensionsControllerProps } from '../../../shared/src/extensions/controller'
import { PlatformContextProps } from '../../../shared/src/platform/context'
import { SettingsCascadeProps } from '../../../shared/src/settings/settings'
import { HeroPage } from '../components/HeroPage'
import { ThemeProps } from '../../../shared/src/theme'
import { OrgArea, OrgAreaRoute } from './area/OrgArea'
import { OrgAreaHeaderNavItem } from './area/OrgHeader'
import { NewOrganizationPage } from './new/NewOrganizationPage'
import { PatternTypeProps } from '../search'
import { TelemetryProps } from '../../../shared/src/telemetry/telemetryService'
import { AuthenticatedUser } from '../auth'
import { BreadcrumbsProps, BreadcrumbSetters } from '../components/Breadcrumbs'

const NotFoundPage: React.FunctionComponent = () => (
    <HeroPage
        icon={MapSearchIcon}
        title="404: Not Found"
        subtitle="Sorry, the requested organization page was not found."
    />
)

interface Props
    extends RouteComponentProps<{}>,
        ExtensionsControllerProps,
        PlatformContextProps,
        SettingsCascadeProps,
        ThemeProps,
        TelemetryProps,
        BreadcrumbsProps,
        BreadcrumbSetters,
        Omit<PatternTypeProps, 'setPatternType'> {
    orgAreaRoutes: readonly OrgAreaRoute[]
    orgAreaHeaderNavItems: readonly OrgAreaHeaderNavItem[]

    authenticatedUser: AuthenticatedUser | null
    isSourcegraphDotCom: boolean
}

/**
 * Renders a layout of a sidebar and a content area to display organization-related pages.
 */
export const OrgsArea: React.FunctionComponent<Props> = props => (
    /* eslint-disable react/jsx-no-bind */
    <Switch>
        <Route path={`${props.match.url}/new`} component={NewOrganizationPage} exact={true} />
        <Route
            path={`${props.match.url}/:name`}
            render={routeComponentProps => <OrgArea {...props} {...routeComponentProps} />}
        />
        <Route component={NotFoundPage} />
    </Switch>
    /* eslint-enable react/jsx-no-bind */
)
