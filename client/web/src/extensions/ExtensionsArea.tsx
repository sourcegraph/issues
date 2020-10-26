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
import { AuthenticatedUser } from '../auth'
import { useBreadcrumbs, Breadcrumbs, BreadcrumbSetters } from '../components/Breadcrumbs'

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
        BreadcrumbSetters {
    /** The currently authenticated user. */
    authenticatedUser: AuthenticatedUser | null

    /** The subject whose extensions and configuration to display. */
    subject: Pick<GQL.ISettingsSubject, 'id' | 'viewerCanAdminister'>
    extensionAreaRoutes: readonly ExtensionAreaRoute[]
    extensionAreaHeaderNavItems: readonly ExtensionAreaHeaderNavItem[]
}

interface ExtensionsAreaProps
    extends RouteComponentProps<{ extensionID: string }>,
        SettingsCascadeProps,
        PlatformContextProps,
        ThemeProps,
        TelemetryProps {
    routes: readonly ExtensionsAreaRoute[]

    /**
     * The currently authenticated user.
     */
    authenticatedUser: AuthenticatedUser | null

    viewerSubject: Pick<GQL.ISettingsSubject, 'id' | 'viewerCanAdminister'>
    extensionAreaRoutes: readonly ExtensionAreaRoute[]
    extensionsAreaHeaderActionButtons: readonly ExtensionsAreaHeaderActionButton[]
    extensionAreaHeaderNavItems: readonly ExtensionAreaHeaderNavItem[]
}

/**
 * The extensions area.
 */
export const ExtensionsArea: React.FunctionComponent<ExtensionsAreaProps> = props => {
    const { breadcrumbs, ...rootBreadcrumbSetters } = useBreadcrumbs()

    const childBreadcrumbSetters = rootBreadcrumbSetters.useBreadcrumb(
        React.useMemo(() => ({ link: { to: '/extensions', label: 'Extensions' }, key: 'Extensions' }), [])
    )

    const context: ExtensionsAreaRouteContext = {
        authenticatedUser: props.authenticatedUser,
        settingsCascade: props.settingsCascade,
        platformContext: props.platformContext,
        subject: props.viewerSubject,
        extensionAreaRoutes: props.extensionAreaRoutes,
        extensionAreaHeaderNavItems: props.extensionAreaHeaderNavItems,
        isLightTheme: props.isLightTheme,
        telemetryService: props.telemetryService,
        ...childBreadcrumbSetters,
    }

    return (
        <div className="extensions-area">
            <Breadcrumbs breadcrumbs={breadcrumbs} location={props.location} />
            <div className="web-content">
                <ExtensionsAreaHeader
                    {...props}
                    {...context}
                    actionButtons={props.extensionsAreaHeaderActionButtons}
                    isPrimaryHeader={props.location.pathname === props.match.path}
                />
                <Switch>
                    {props.routes.map(
                        /* eslint-disable react/jsx-no-bind */
                        ({ path, exact, condition = () => true, render }) =>
                            condition(context) && (
                                <Route
                                    key="hardcoded-key"
                                    path={props.match.url + path}
                                    exact={exact}
                                    render={routeComponentProps => render({ ...context, ...routeComponentProps })}
                                />
                            )
                        /* eslint-enable react/jsx-no-bind */
                    )}
                    <Route key="hardcoded-key" component={NotFoundPage} />
                </Switch>
            </div>
        </div>
    )
}
