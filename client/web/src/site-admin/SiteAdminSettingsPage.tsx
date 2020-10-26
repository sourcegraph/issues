import * as React from 'react'
import { RouteComponentProps } from 'react-router'
import * as GQL from '../../../shared/src/graphql/schema'
import { PlatformContextProps } from '../../../shared/src/platform/context'
import { SettingsCascadeProps } from '../../../shared/src/settings/settings'
import { PageTitle } from '../components/PageTitle'
import { SettingsArea } from '../settings/SettingsArea'
import { ThemeProps } from '../../../shared/src/theme'
import { TelemetryProps } from '../../../shared/src/telemetry/telemetryService'
import { AuthenticatedUser } from '../auth'

interface Props
    extends RouteComponentProps<{}>,
        PlatformContextProps,
        SettingsCascadeProps,
        ThemeProps,
        TelemetryProps {
    authenticatedUser: AuthenticatedUser
    site: Pick<GQL.ISite, '__typename' | 'id'>
}

export const SiteAdminSettingsPage: React.FunctionComponent<Props> = props => (
    <>
        <PageTitle title="Global settings" />
        <SettingsArea
            {...props}
            subject={props.site}
            authenticatedUser={props.authenticatedUser}
            extraHeader={
                <p>
                    Global settings apply to all organizations and users. Settings for a user or organization override
                    global settings.
                </p>
            }
        />
    </>
)
