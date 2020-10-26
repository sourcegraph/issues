import * as H from 'history'
import * as React from 'react'
import { Subscription } from 'rxjs'
import { ContributableMenu } from '../../../shared/src/api/protocol'
import { ActivationProps } from '../../../shared/src/components/activation/Activation'
import { ActivationDropdown } from '../../../shared/src/components/activation/ActivationDropdown'
import { Link } from '../../../shared/src/components/Link'
import { ExtensionsControllerProps } from '../../../shared/src/extensions/controller'
import { PlatformContextProps } from '../../../shared/src/platform/context'
import { SettingsCascadeProps } from '../../../shared/src/settings/settings'
import { WebActionsNavItems, WebCommandListPopoverButton } from '../components/shared'
import { ThemeProps } from '../../../shared/src/theme'
import { StatusMessagesNavItem } from './StatusMessagesNavItem'
import { ExtensionAlertAnimationProps, UserNavItem } from './UserNavItem'
import { CampaignsNavItem } from '../enterprise/campaigns/global/nav/CampaignsNavItem'
import { ThemePreferenceProps } from '../theme'
import {
    KeyboardShortcutsProps,
    KEYBOARD_SHORTCUT_SHOW_COMMAND_PALETTE,
    KEYBOARD_SHORTCUT_SWITCH_THEME,
} from '../keyboardShortcuts/keyboardShortcuts'
import { isErrorLike } from '../../../shared/src/util/errors'
import { Settings } from '../schema/settings.schema'
import { InsightsNavItem } from '../insights/InsightsNavLink'
import { AuthenticatedUser } from '../auth'
import { TelemetryProps } from '../../../shared/src/telemetry/telemetryService'
import { ExtensionsNavItem } from '../extensions/ExtensionsNavItem'

interface Props
    extends SettingsCascadeProps<Settings>,
        KeyboardShortcutsProps,
        ExtensionsControllerProps<'executeCommand' | 'services'>,
        PlatformContextProps<'forceUpdateTooltip' | 'settings' | 'sourcegraphURL'>,
        ThemeProps,
        ThemePreferenceProps,
        ExtensionAlertAnimationProps,
        TelemetryProps,
        ActivationProps {
    location: H.Location
    history: H.History
    authenticatedUser: AuthenticatedUser | null
    showDotComMarketing: boolean
    showCampaigns: boolean
    isSourcegraphDotCom: boolean
    minimalNavLinks?: boolean
}

export class NavLinks extends React.PureComponent<Props> {
    private subscriptions = new Subscription()

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        return (
            <ul className="nav-links nav align-items-center pl-2 pr-1">
                {/* Show "Search" link on small screens when GlobalNavbar hides the SearchNavbarItem. */}
                {this.props.location.pathname !== '/search' && (
                    <li className="nav-item d-sm-none">
                        <Link className="nav-link" to="/search">
                            Search
                        </Link>
                    </li>
                )}
                <WebActionsNavItems {...this.props} menu={ContributableMenu.GlobalNav} />
                {this.props.activation && (
                    <li className="nav-item">
                        <ActivationDropdown activation={this.props.activation} history={this.props.history} />
                    </li>
                )}
                {!isErrorLike(this.props.settingsCascade.final) &&
                    this.props.settingsCascade.final?.experimentalFeatures?.codeInsights &&
                    !this.props.minimalNavLinks && (
                        <li className="nav-item">
                            <InsightsNavItem />
                        </li>
                    )}
                {!this.props.minimalNavLinks && (
                    <li className="nav-item">
                        <ExtensionsNavItem />
                    </li>
                )}
                {!this.props.minimalNavLinks && this.props.showCampaigns && (
                    <li className="nav-item">
                        <CampaignsNavItem />
                    </li>
                )}
                {!this.props.authenticatedUser && (
                    <>
                        {this.props.location.pathname !== '/sign-in' && (
                            <li className="nav-item mx-1">
                                <Link className="nav-link btn btn-primary" to="/sign-in">
                                    Sign in
                                </Link>
                            </li>
                        )}
                        <li className="nav-item">
                            <Link to="/help" className="nav-link" target="_blank" rel="noopener">
                                Docs
                            </Link>
                        </li>
                        {this.props.showDotComMarketing && (
                            <li className="nav-item">
                                <a
                                    href="https://about.sourcegraph.com"
                                    className="nav-link"
                                    target="_blank"
                                    rel="noopener"
                                >
                                    About
                                </a>
                            </li>
                        )}
                    </>
                )}
                {!this.props.isSourcegraphDotCom && this.props.authenticatedUser?.siteAdmin && (
                    <li className="nav-item">
                        <StatusMessagesNavItem
                            isSiteAdmin={this.props.authenticatedUser.siteAdmin}
                            history={this.props.history}
                        />
                    </li>
                )}
                {!this.props.minimalNavLinks && (
                    <li className="nav-item">
                        <WebCommandListPopoverButton
                            {...this.props}
                            buttonClassName="nav-link btn btn-link"
                            menu={ContributableMenu.CommandPalette}
                            keyboardShortcutForShow={KEYBOARD_SHORTCUT_SHOW_COMMAND_PALETTE}
                        />
                    </li>
                )}
                {this.props.authenticatedUser && (
                    <li className="nav-item">
                        <UserNavItem
                            {...this.props}
                            authenticatedUser={this.props.authenticatedUser}
                            showDotComMarketing={this.props.showDotComMarketing}
                            codeHostIntegrationMessaging={
                                (!isErrorLike(this.props.settingsCascade.final) &&
                                    this.props.settingsCascade.final?.['alerts.codeHostIntegrationMessaging']) ||
                                'browser-extension'
                            }
                            keyboardShortcutForSwitchTheme={KEYBOARD_SHORTCUT_SWITCH_THEME}
                        />
                    </li>
                )}
            </ul>
        )
    }
}
