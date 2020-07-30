import * as H from 'history'
import * as React from 'react'
import { ExtensionsControllerProps } from '../../../shared/src/extensions/controller'
import * as GQL from '../../../shared/src/graphql/schema'
import { SettingsCascadeOrError } from '../../../shared/src/settings/settings'
import { eventLogger } from '../tracking/eventLogger'
import { ComponentDescriptor } from '../util/contributions'
import { PatternTypeProps } from '../search'
import { ThemeProps } from '../../../shared/src/theme'
import { OptionalAuthProps } from '../auth'

/**
 * Properties passed to all section components in the explore area.
 */
export interface ExploreAreaSectionContext
    extends ExtensionsControllerProps,
        ThemeProps,
        OptionalAuthProps,
        Omit<PatternTypeProps, 'setPatternType'> {
    /** The subject whose extensions and settings to display. */
    viewerSubject: Pick<GQL.SettingsSubject, 'id' | 'viewerCanAdminister'>

    /** The viewer's settings. */
    settingsCascade: SettingsCascadeOrError

    location: H.Location
    history: H.History
}

/** A section shown in the explore area. */
export interface ExploreSectionDescriptor extends ComponentDescriptor<ExploreAreaSectionContext> {}

interface ExploreAreaProps extends ExploreAreaSectionContext {
    exploreSections: readonly ExploreSectionDescriptor[]
}

interface ExploreAreaState {}

/**
 * The explore area, which shows cards containing summaries and actions from product features. The purpose of it is
 * to expose information at a glance and make it easy to navigate to features (without requiring them to add a link
 * on the space-constrained global nav).
 */
export class ExploreArea extends React.Component<ExploreAreaProps, ExploreAreaState> {
    public state: ExploreAreaState = {}

    public componentDidMount(): void {
        eventLogger.logViewEvent('Explore')
    }

    public render(): JSX.Element | null {
        const context: ExploreAreaSectionContext = {
            extensionsController: this.props.extensionsController,
            authenticatedUser: this.props.authenticatedUser,
            viewerSubject: this.props.viewerSubject,
            settingsCascade: this.props.settingsCascade,
            isLightTheme: this.props.isLightTheme,
            location: this.props.location,
            history: this.props.history,
            patternType: this.props.patternType,
        }

        return (
            <div className="explore-area container my-3">
                <h1>Explore</h1>
                {this.props.exploreSections.map(
                    ({ condition = () => true, render }, index) =>
                        condition(context) && (
                            <div className="mb-5" key={index}>
                                {render(context)}
                            </div>
                        )
                )}
            </div>
        )
    }
}
