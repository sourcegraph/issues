import MapSearchIcon from 'mdi-react/MapSearchIcon'
import * as React from 'react'
import { Route, RouteComponentProps, Switch } from 'react-router'
import * as GQL from '../../../shared/src/graphql/schema'
import { PlatformContextProps } from '../../../shared/src/platform/context'
import { SettingsCascadeProps } from '../../../shared/src/settings/settings'
import { HeroPage } from '../components/HeroPage'
import { RouteDescriptor } from '../util/contributions'
import { ExtensionAreaRoute } from './extension/ExtensionArea'
import { ExtensionAreaHeaderNavItem } from './extension/ExtensionAreaHeader'
import { ExtensionsAreaHeader, ExtensionsAreaHeaderActionButton } from './ExtensionsAreaHeader'
import { ThemeProps } from '../../../shared/src/theme'
import { TelemetryProps } from '../../../shared/src/telemetry/telemetryService'
import { OptionalAuthProps } from '../auth'

const NotFoundPage: React.FunctionComponent = () => <HeroPage icon={MapSearchIcon} title="404: Not Found" />

export interface ExtensionsAreaRoute extends RouteDescriptor<ExtensionsAreaRouteContext> {}

/**
 * Properties passed to all page components in the extensions area.
 */
export interface ExtensionsAreaRouteContext
    extends SettingsCascadeProps,
        PlatformContextProps,
        ThemeProps,
        TelemetryProps,
        OptionalAuthProps {
    /** The subject whose extensions and configuration to display. */
    subject: Pick<GQL.SettingsSubject, 'id' | 'viewerCanAdminister'>
    extensionAreaRoutes: readonly ExtensionAreaRoute[]
    extensionAreaHeaderNavItems: readonly ExtensionAreaHeaderNavItem[]
}

interface ExtensionsAreaProps
    extends RouteComponentProps<{ extensionID: string }>,
        SettingsCascadeProps,
        PlatformContextProps,
        ThemeProps,
        OptionalAuthProps,
        TelemetryProps {
    routes: readonly ExtensionsAreaRoute[]

    viewerSubject: Pick<GQL.SettingsSubject, 'id' | 'viewerCanAdminister'>
    extensionAreaRoutes: readonly ExtensionAreaRoute[]
    extensionsAreaHeaderActionButtons: readonly ExtensionsAreaHeaderActionButton[]
    extensionAreaHeaderNavItems: readonly ExtensionAreaHeaderNavItem[]
}

interface ExtensionsAreaState {}

/**
 * The extensions area.
 */
export class ExtensionsArea extends React.Component<ExtensionsAreaProps, ExtensionsAreaState> {
    public state: ExtensionsAreaState = {}

    public render(): JSX.Element | null {
        const context: ExtensionsAreaRouteContext = {
            authenticatedUser: this.props.authenticatedUser,
            settingsCascade: this.props.settingsCascade,
            platformContext: this.props.platformContext,
            subject: this.props.viewerSubject,
            extensionAreaRoutes: this.props.extensionAreaRoutes,
            extensionAreaHeaderNavItems: this.props.extensionAreaHeaderNavItems,
            isLightTheme: this.props.isLightTheme,
            telemetryService: this.props.telemetryService,
        }

        return (
            <div className="extensions-area">
                <ExtensionsAreaHeader
                    {...this.props}
                    {...context}
                    actionButtons={this.props.extensionsAreaHeaderActionButtons}
                    isPrimaryHeader={this.props.location.pathname === this.props.match.path}
                />
                <Switch>
                    {this.props.routes.map(
                        /* eslint-disable react/jsx-no-bind */
                        ({ path, exact, condition = () => true, render }) =>
                            condition(context) && (
                                <Route
                                    key="hardcoded-key"
                                    path={this.props.match.url + path}
                                    exact={exact}
                                    render={routeComponentProps => render({ ...context, ...routeComponentProps })}
                                />
                            )
                        /* eslint-enable react/jsx-no-bind */
                    )}
                    <Route key="hardcoded-key" component={NotFoundPage} />
                </Switch>
            </div>
        )
    }
}
