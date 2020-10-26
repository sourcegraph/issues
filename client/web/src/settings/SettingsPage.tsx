import * as React from 'react'
import { RouteComponentProps } from 'react-router'
import { overwriteSettings } from '../../../shared/src/settings/edit'
import { ThemeProps } from '../../../shared/src/theme'
import { SettingsAreaPageProps } from './SettingsArea'
import { SettingsFile } from './SettingsFile'
import { TelemetryProps } from '../../../shared/src/telemetry/telemetryService'

interface Props
    extends SettingsAreaPageProps,
        Pick<RouteComponentProps<{}>, 'history' | 'location'>,
        ThemeProps,
        TelemetryProps {
    /** Optional description to render above the editor. */
    description?: JSX.Element
}

interface State {
    commitError?: Error
}

/**
 * Displays a page where the settings for a subject can be edited.
 */
export class SettingsPage extends React.PureComponent<Props, State> {
    public state: State = {}

    public render(): JSX.Element | null {
        return (
            <SettingsFile
                settings={this.props.data.subjects[this.props.data.subjects.length - 1].latestSettings}
                jsonSchema={this.props.data.settingsJSONSchema}
                commitError={this.state.commitError}
                onDidCommit={this.onDidCommit}
                onDidDiscard={this.onDidDiscard}
                history={this.props.history}
                isLightTheme={this.props.isLightTheme}
                telemetryService={this.props.telemetryService}
            />
        )
    }

    private onDidCommit = async (lastID: number | null, contents: string): Promise<void> => {
        this.setState({ commitError: undefined })

        // When updating settings for a settings subject that is in the viewer's settings cascade (i.e., if the
        // update will affect the viewer's settings), perform the update by calling through the shared
        // {@link PlatformContext#updateSettings} so that the update is seen by all settings observers.
        //
        // If the settings update is for some other subject that is unrelated to the viewer, then this is not
        // necessary.
        const isSubjectInViewerSettingsCascade = this.props.settingsCascade.subjects?.some(
            ({ subject }) => subject.id === this.props.subject.id
        )

        try {
            if (isSubjectInViewerSettingsCascade) {
                await this.props.platformContext.updateSettings(this.props.subject.id, contents)
            } else {
                await overwriteSettings(this.props.platformContext, this.props.subject.id, lastID, contents)
            }
            this.setState({ commitError: undefined })
            this.props.onUpdate()
        } catch (commitError) {
            this.setState({ commitError })
            console.error(commitError)
        }
    }

    private onDidDiscard = (): void => {
        this.setState({ commitError: undefined })
    }
}
