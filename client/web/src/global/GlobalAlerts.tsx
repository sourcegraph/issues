import { parseISO } from 'date-fns'
import differenceInDays from 'date-fns/differenceInDays'
import * as React from 'react'
import { Subscription } from 'rxjs'
import { Markdown } from '../../../shared/src/components/Markdown'
import { isSettingsValid, SettingsCascadeProps } from '../../../shared/src/settings/settings'
import { renderMarkdown } from '../../../shared/src/util/markdown'
import { DismissibleAlert } from '../components/DismissibleAlert'
import { Settings } from '../schema/settings.schema'
import { SiteFlags } from '../site'
import { siteFlags } from '../site/backend'
import { DockerForMacAlert } from '../site/DockerForMacAlert'
import { FreeUsersExceededAlert } from '../site/FreeUsersExceededAlert'
import { LicenseExpirationAlert } from '../site/LicenseExpirationAlert'
import { NeedsRepositoryConfigurationAlert } from '../site/NeedsRepositoryConfigurationAlert'
import { GlobalAlert } from './GlobalAlert'
import { Notices } from './Notices'
import * as H from 'history'

interface Props extends SettingsCascadeProps {
    history: H.History
    isSiteAdmin: boolean
}

interface State {
    siteFlags?: SiteFlags
}

/**
 * Fetches and displays relevant global alerts at the top of the page
 */
export class GlobalAlerts extends React.PureComponent<Props, State> {
    public state: State = {}

    private subscriptions = new Subscription()

    public componentDidMount(): void {
        this.subscriptions.add(siteFlags.subscribe(siteFlags => this.setState({ siteFlags })))
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        return (
            <div className="global-alerts test-global-alert">
                {this.state.siteFlags && (
                    <>
                        {this.state.siteFlags.needsRepositoryConfiguration && (
                            <NeedsRepositoryConfigurationAlert className="global-alerts__alert" />
                        )}
                        {this.state.siteFlags.freeUsersExceeded && (
                            <FreeUsersExceededAlert
                                noLicenseWarningUserCount={
                                    this.state.siteFlags.productSubscription.noLicenseWarningUserCount
                                }
                                className="global-alerts__alert"
                            />
                        )}
                        {/* Only show if the user has already added repositories; if not yet, the user wouldn't experience any Docker for Mac perf issues anyway. */}
                        {window.context.likelyDockerOnMac && <DockerForMacAlert className="global-alerts__alert" />}
                        {this.state.siteFlags.alerts.map((alert, index) => (
                            <GlobalAlert
                                key={index}
                                alert={alert}
                                className="global-alerts__alert"
                                history={this.props.history}
                            />
                        ))}
                        {this.state.siteFlags.productSubscription.license &&
                            (() => {
                                const expiresAt = parseISO(this.state.siteFlags.productSubscription.license.expiresAt)
                                return (
                                    differenceInDays(expiresAt, Date.now()) <= 7 && (
                                        <LicenseExpirationAlert
                                            expiresAt={expiresAt}
                                            daysLeft={Math.floor(differenceInDays(expiresAt, Date.now()))}
                                            className="global-alerts__alert"
                                        />
                                    )
                                )
                            })()}
                    </>
                )}
                {isSettingsValid<Settings>(this.props.settingsCascade) &&
                    this.props.settingsCascade.final.motd &&
                    Array.isArray(this.props.settingsCascade.final.motd) &&
                    this.props.settingsCascade.final.motd.map(motd => (
                        <DismissibleAlert
                            key={motd}
                            partialStorageKey={`motd.${motd}`}
                            className="alert alert-info global-alerts__alert"
                        >
                            <Markdown dangerousInnerHTML={renderMarkdown(motd)} history={this.props.history} />
                        </DismissibleAlert>
                    ))}
                <Notices
                    alertClassName="global-alerts__alert"
                    location="top"
                    settingsCascade={this.props.settingsCascade}
                    history={this.props.history}
                />
            </div>
        )
    }
}
