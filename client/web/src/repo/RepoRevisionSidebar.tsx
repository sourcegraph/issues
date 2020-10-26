import * as H from 'history'
import CloseIcon from 'mdi-react/CloseIcon'
import * as React from 'react'
import { fromEvent, Subscription } from 'rxjs'
import { filter } from 'rxjs/operators'
import { FormatListBulletedIcon } from '../../../shared/src/components/icons'
import { Resizable } from '../../../shared/src/components/Resizable'
import {
    Spacer,
    Tab,
    TabBorderClassName,
    TabsWithLocalStorageViewStatePersistence,
} from '../../../branded/src/components/Tabs'
import { ExtensionsControllerProps } from '../../../shared/src/extensions/controller'
import * as GQL from '../../../shared/src/graphql/schema'
import { AbsoluteRepoFile } from '../../../shared/src/util/url'
import { eventLogger } from '../tracking/eventLogger'
import { Tree } from '../tree/Tree'
import { RepoRevisionSidebarSymbols } from './RepoRevisionSidebarSymbols'

type SidebarTabID = 'files' | 'symbols' | 'history'

interface Props extends AbsoluteRepoFile, ExtensionsControllerProps {
    repoID: GQL.ID
    isDir: boolean
    defaultBranch: string
    className: string
    history: H.History
    location: H.Location
}

interface State {
    showSidebar: boolean
}

/**
 * The sidebar for a specific repo revision that shows the list of files and directories.
 */
export class RepoRevisionSidebar extends React.PureComponent<Props, State> {
    private static LAST_TAB_STORAGE_KEY = 'repo-revision-sidebar-last-tab'
    private static HIDDEN_STORAGE_KEY = 'repo-revision-sidebar-hidden'

    private static TABS: Tab<SidebarTabID>[] = [
        { id: 'files', label: 'Files' },
        { id: 'symbols', label: 'Symbols' },
    ]

    public state: State = {
        showSidebar: localStorage.getItem(RepoRevisionSidebar.HIDDEN_STORAGE_KEY) === null,
    }

    private subscriptions = new Subscription()

    public componentDidMount(): void {
        // Toggle sidebar visibility when the user presses 'alt+s'.
        this.subscriptions.add(
            fromEvent<KeyboardEvent>(window, 'keydown')
                .pipe(filter(event => event.altKey && event.key === 's'))
                .subscribe(event => {
                    event.preventDefault()
                    this.setState(previousState => ({ showSidebar: !previousState.showSidebar }))
                })
        )
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        if (!this.state.showSidebar) {
            return (
                <button
                    type="button"
                    className={`btn btn-icon repo-revision-sidebar-toggle ${this.props.className}-toggle`}
                    onClick={this.onSidebarToggle}
                    data-tooltip="Show sidebar (Alt+S/Opt+S)"
                >
                    <FormatListBulletedIcon className="icon-inline" />
                </button>
            )
        }

        const STORAGE_KEY = 'repo-revision-sidebar'

        return (
            <Resizable
                className="repo-revision-container__sidebar-resizable"
                handlePosition="right"
                storageKey={STORAGE_KEY}
                defaultSize={256 /* px */}
                element={
                    <TabsWithLocalStorageViewStatePersistence
                        tabs={RepoRevisionSidebar.TABS}
                        storageKey={RepoRevisionSidebar.LAST_TAB_STORAGE_KEY}
                        tabBarEndFragment={
                            <>
                                <Spacer />
                                <button
                                    type="button"
                                    onClick={this.onSidebarToggle}
                                    className={`btn btn-icon tab_bar__close-button ${TabBorderClassName}`}
                                    title="Close sidebar (Alt+S/Opt+S)"
                                >
                                    <CloseIcon className="icon-inline" />
                                </button>
                            </>
                        }
                        id="explorer"
                        className={`repo-revision-sidebar ${this.props.className} ${
                            this.state.showSidebar ? `repo-revision-sidebar--open ${this.props.className}--open` : ''
                        } test-repo-revision-sidebar`}
                        tabClassName="tab-bar__tab--h5like"
                        onSelectTab={this.onSelectTab}
                    >
                        <Tree
                            key="files"
                            repoName={this.props.repoName}
                            revision={this.props.revision}
                            commitID={this.props.commitID}
                            history={this.props.history}
                            location={this.props.location}
                            scrollRootSelector="#explorer"
                            activePath={this.props.filePath}
                            activePathIsDir={this.props.isDir}
                            sizeKey={`Resizable:${STORAGE_KEY}`}
                        />
                        <RepoRevisionSidebarSymbols
                            key="symbols"
                            repoID={this.props.repoID}
                            revision={this.props.revision}
                            activePath={this.props.filePath}
                            history={this.props.history}
                            location={this.props.location}
                        />
                    </TabsWithLocalStorageViewStatePersistence>
                }
            />
        )
    }

    private onSidebarToggle = (): void => {
        if (this.state.showSidebar) {
            localStorage.setItem(RepoRevisionSidebar.HIDDEN_STORAGE_KEY, 'true')
        } else {
            localStorage.removeItem(RepoRevisionSidebar.HIDDEN_STORAGE_KEY)
        }
        this.setState(state => ({ showSidebar: !state.showSidebar }))
    }

    private onSelectTab = (tab: string): void => {
        if (tab === 'symbols') {
            eventLogger.log('SidebarSymbolsTabSelected')
        } else if (tab === 'files') {
            eventLogger.log('SidebarFilesTabSelected')
        } else if (tab === 'history') {
            eventLogger.log('SidebarHistoryTabSelected')
        }
    }
}
