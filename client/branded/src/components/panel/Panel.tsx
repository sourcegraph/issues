import * as H from 'history'
import CloseIcon from 'mdi-react/CloseIcon'
import * as React from 'react'
import { Observable, Subscription } from 'rxjs'
import { map } from 'rxjs/operators'
import {
    PanelViewWithComponent,
    PanelViewProviderRegistrationOptions,
} from '../../../../shared/src/api/client/services/panelViews'
import { ContributableMenu, ContributableViewContainer } from '../../../../shared/src/api/protocol/contribution'
import { ExtensionsControllerProps } from '../../../../shared/src/extensions/controller'
import { ActionsNavItems } from '../../../../shared/src/actions/ActionsNavItems'
import { ActivationProps } from '../../../../shared/src/components/activation/Activation'
import { FetchFileParameters } from '../../../../shared/src/components/CodeExcerpt'
import { Resizable } from '../../../../shared/src/components/Resizable'
import { Spacer, Tab, TabsWithURLViewStatePersistence } from '../Tabs'
import { PlatformContextProps } from '../../../../shared/src/platform/context'
import { SettingsCascadeProps } from '../../../../shared/src/settings/settings'
import { TelemetryProps } from '../../../../shared/src/telemetry/telemetryService'
import { EmptyPanelView } from './views/EmptyPanelView'
import { PanelView } from './views/PanelView'
import { ThemeProps } from '../../../../shared/src/theme'
import { VersionContextProps } from '../../../../shared/src/search/util'

interface Props
    extends ExtensionsControllerProps,
        PlatformContextProps,
        SettingsCascadeProps,
        ActivationProps,
        TelemetryProps,
        ThemeProps,
        VersionContextProps {
    location: H.Location
    history: H.History
    repoName?: string
    fetchHighlightedFileLines: (parameters: FetchFileParameters, force?: boolean) => Observable<string[]>
}

interface State {
    /** Panel views contributed by extensions. */
    panelViews?: (PanelViewWithComponent & Pick<PanelViewProviderRegistrationOptions, 'id'>)[] | null
}

/**
 * A tab and corresponding content to display in the panel.
 */
interface PanelItem extends Tab<string> {
    /**
     * Controls the relative order of panel items. The items are laid out from highest priority (at the beginning)
     * to lowest priority (at the end). The default is 0.
     */
    priority: number

    /** The content element to display when the tab is active. */
    element: JSX.Element

    /**
     * Whether this panel contains a list of locations (from a location provider). This value is
     * exposed to contributions as `panel.activeView.hasLocations`. It is true if there is a
     * location provider (even if the result set is empty).
     */
    hasLocations?: boolean
}

/**
 * The panel, which is a tabbed component with contextual information. Components rendering the panel should
 * generally use ResizablePanel, not Panel.
 *
 * Other components can contribute panel items to the panel.
 */
class Panel extends React.PureComponent<Props, State> {
    public state: State = {}

    private subscriptions = new Subscription()

    public componentDidMount(): void {
        this.subscriptions.add(
            this.props.extensionsController.services.panelViews
                .getPanelViews(ContributableViewContainer.Panel)
                .pipe(map(panelViews => ({ panelViews })))
                .subscribe(stateUpdate => this.setState(stateUpdate))
        )
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        const items = this.state.panelViews
            ? this.state.panelViews
                  .map(
                      (panelView): PanelItem => ({
                          label: panelView.title,
                          id: panelView.id,
                          priority: panelView.priority,
                          element: <PanelView {...this.props} panelView={panelView} />,
                          hasLocations: !!panelView.locationProvider,
                      })
                  )
                  .sort(byPriority)
            : []

        const hasTabs = items.length > 0
        const activePanelViewID = TabsWithURLViewStatePersistence.readFromURL(this.props.location, items)
        const activePanelView = items.find(item => item.id === activePanelViewID)

        return (
            <div className="panel">
                {hasTabs ? (
                    <TabsWithURLViewStatePersistence
                        tabs={items}
                        tabBarEndFragment={
                            <>
                                <Spacer />
                                <button
                                    type="button"
                                    onClick={this.onDismiss}
                                    className="btn btn-icon tab-bar__end-fragment-other-element"
                                    data-tooltip="Close"
                                >
                                    <CloseIcon className="icon-inline" />
                                </button>
                            </>
                        }
                        toolbarFragment={
                            <ActionsNavItems
                                {...this.props}
                                // TODO remove references to Bootstrap from shared, get class name from prop
                                // This is okay for now because the Panel is currently only used in the webapp
                                listClass="nav w-100 justify-content-end"
                                actionItemClass="nav-link"
                                actionItemIconClass="icon-inline"
                                menu={ContributableMenu.PanelToolbar}
                                scope={
                                    activePanelViewID !== undefined
                                        ? {
                                              type: 'panelView',
                                              id: activePanelViewID,
                                              hasLocations: Boolean(activePanelView?.hasLocations),
                                          }
                                        : undefined
                                }
                                wrapInList={true}
                            />
                        }
                        className="panel__tabs"
                        tabClassName="tab-bar__tab--h5like"
                        location={this.props.location}
                    >
                        {items?.map(({ id, element }) => React.cloneElement(element, { key: id }))}
                    </TabsWithURLViewStatePersistence>
                ) : (
                    <EmptyPanelView />
                )}
            </div>
        )
    }

    private onDismiss = (): void =>
        this.props.history.push(TabsWithURLViewStatePersistence.urlForTabID(this.props.location, null))
}

function byPriority(a: { priority: number }, b: { priority: number }): number {
    return b.priority - a.priority
}

/** A wrapper around Panel that makes it resizable. */
export const ResizablePanel: React.FunctionComponent<Props> = props => (
    <Resizable
        className="resizable-panel"
        handlePosition="top"
        defaultSize={350}
        storageKey="panel-size"
        element={<Panel {...props} />}
    />
)
