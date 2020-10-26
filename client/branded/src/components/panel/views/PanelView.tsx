import * as H from 'history'
import React from 'react'
import { Observable } from 'rxjs'
import {
    PanelViewWithComponent,
    PanelViewProviderRegistrationOptions,
} from '../../../../../shared/src/api/client/services/panelViews'
import { FetchFileParameters } from '../../../../../shared/src/components/CodeExcerpt'
import { Markdown } from '../../../../../shared/src/components/Markdown'
import { ExtensionsControllerProps } from '../../../../../shared/src/extensions/controller'
import { SettingsCascadeProps } from '../../../../../shared/src/settings/settings'
import { renderMarkdown } from '../../../../../shared/src/util/markdown'
import { EmptyPanelView } from './EmptyPanelView'
import { HierarchicalLocationsView } from './HierarchicalLocationsView'
import { VersionContextProps } from '../../../../../shared/src/search/util'

interface Props extends ExtensionsControllerProps, SettingsCascadeProps, VersionContextProps {
    panelView: PanelViewWithComponent & Pick<PanelViewProviderRegistrationOptions, 'id'>
    repoName?: string
    history: H.History
    location: H.Location
    isLightTheme: boolean
    fetchHighlightedFileLines: (parameters: FetchFileParameters, force?: boolean) => Observable<string[]>
}

interface State {}

/**
 * A panel view contributed by an extension using {@link sourcegraph.app.createPanelView}.
 */
export class PanelView extends React.PureComponent<Props, State> {
    public render(): JSX.Element | null {
        return (
            <div className="panel__tabs-content panel__tabs-content--scroll">
                {this.props.panelView.content && (
                    <div className="px-2 pt-2">
                        <Markdown
                            dangerousInnerHTML={renderMarkdown(this.props.panelView.content)}
                            history={this.props.history}
                        />
                    </div>
                )}
                {this.props.panelView.reactElement}
                {this.props.panelView.locationProvider && this.props.repoName && (
                    <HierarchicalLocationsView
                        location={this.props.location}
                        locations={this.props.panelView.locationProvider}
                        defaultGroup={this.props.repoName}
                        isLightTheme={this.props.isLightTheme}
                        fetchHighlightedFileLines={this.props.fetchHighlightedFileLines}
                        extensionsController={this.props.extensionsController}
                        settingsCascade={this.props.settingsCascade}
                        versionContext={this.props.versionContext}
                    />
                )}
                {!this.props.panelView.content &&
                    !this.props.panelView.reactElement &&
                    !this.props.panelView.locationProvider && <EmptyPanelView className="mt-3" />}
            </div>
        )
    }
}
