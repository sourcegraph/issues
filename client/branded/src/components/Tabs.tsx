import * as H from 'history'
import * as React from 'react'
import { parseHash } from '../../../shared/src/util/url'
import { Link } from '../../../shared/src/components/Link'
import classNames from 'classnames'

/**
 * Describes a tab.
 *
 * @template ID The type that includes all possible tab IDs (typically a union of string constants).
 */
export interface Tab<ID extends string> {
    id: ID
    label: React.ReactFragment

    /**
     * Whether the tab is hidden in the tab bar.
     */
    hidden?: boolean
}

/**
 * Properties for the tab bar.
 *
 * @template ID The type that includes all possible tab IDs (typically a union of string constants).
 * @template T The type that describes a tab.
 */
interface TabBarProps<ID extends string, T extends Tab<ID>> {
    /** All tabs. */
    tabs: T[]

    /** The currently active tab. */
    activeTab: ID | undefined

    /** A fragment to render at the end of the tab bar. */
    endFragment?: React.ReactFragment

    /** The component used to render the tab (in the tab bar, not the active tab's content area). */
    tabComponent: React.ComponentType<{ tab: T; className: string }>

    tabClassName?: string
}

/**
 * A horizontal bar that displays tab titles, which the user can click to switch to the tab.
 *
 * @template ID The type that includes all possible tab IDs (typically a union of string constants).
 * @template T The type that describes a tab.
 */
class TabBar<ID extends string, T extends Tab<ID>> extends React.PureComponent<TabBarProps<ID, T>> {
    public render(): JSX.Element | null {
        return (
            <div className={`tab-bar ${this.props.tabs.length === 0 ? 'tab-bar--empty' : ''}`}>
                {this.props.tabs
                    .filter(({ hidden }) => !hidden)
                    .map(tab => (
                        <this.props.tabComponent
                            key={tab.id}
                            tab={tab}
                            className={classNames(
                                'btn',
                                'btn-link',
                                'btn-sm',
                                'tab-bar__tab',
                                !this.props.endFragment && 'tab-bar__tab--flex-grow',
                                'tab-bar__tab--' +
                                    (this.props.activeTab !== undefined && this.props.activeTab === tab.id
                                        ? 'active'
                                        : 'inactive'),
                                this.props.tabClassName
                            )}
                        />
                    ))}
                {this.props.endFragment}
            </div>
        )
    }
}

/**
 * An element to pass to Tab's tabBarEndFragment prop to fill all width between the tabs (on the left) and the
 * other tabBarEndFragment elements (on the right).
 */
export const Spacer: () => JSX.Element = () => <span className="tab-bar__spacer" />

/**
 * Properties for the Tabs components and its wrappers.
 *
 * @template ID The type that includes all possible tab IDs (typically a union of string constants).
 * @template T The type that describes a tab.
 */
interface TabsProps<ID extends string, T extends Tab<ID>> {
    /** All tabs. */
    tabs: T[]

    /**
     * A fragment to display at the end of the tab bar. If specified, the tabs will not flex grow to fill the
     * width.
     */
    tabBarEndFragment?: React.ReactFragment

    /**
     * A fragment to display underneath the tab bar.
     */
    toolbarFragment?: React.ReactFragment

    children: React.ReactFragment

    id?: string
    className?: string
    tabClassName?: string

    /** Optional handler when a tab is selected */
    onSelectTab?: (tab: ID) => void
}

/**
 * The class name to use for other elements injected via tabBarEndFragment that should have a bottom border.
 */
export const TabBorderClassName = 'tab-bar__end-fragment-other-element'

/**
 * A tabbed UI component, with a tab bar for switching between tabs and a content view that renders the active
 * tab's contents.
 *
 * Callers should use one of the TabsWithXyzViewStatePersistence components to handle view state persistence.
 */
class Tabs<ID extends string, T extends Tab<ID>> extends React.PureComponent<
    TabsProps<ID, T> & {
        /** The currently active tab. */
        activeTab: ID | undefined

        /** The component used to render the tab (in the tab bar, not the active tab's content area). */
        tabComponent: React.ComponentType<{ tab: T; className: string }>
    }
> {
    public render(): JSX.Element | null {
        let children: React.ReactElement<{ key: ID }>[] | undefined
        if (Array.isArray(this.props.children)) {
            children = this.props.children as React.ReactElement<{ key: ID }>[]
        } else if (this.props.children) {
            children = [this.props.children as React.ReactElement<{ key: ID }>]
        }

        return (
            <div id={this.props.id} className={`tabs ${this.props.className || ''}`}>
                <TabBar
                    tabs={this.props.tabs}
                    activeTab={this.props.activeTab}
                    endFragment={this.props.tabBarEndFragment}
                    tabClassName={this.props.tabClassName}
                    tabComponent={this.props.tabComponent}
                />
                {this.props.toolbarFragment && <div className="tabs__toolbar small">{this.props.toolbarFragment}</div>}
                {children?.find(child => child && child.key === this.props.activeTab)}
            </div>
        )
    }
}

/**
 * A wrapper for Tabs that persists view state (the currently active tab) in localStorage.
 */
export class TabsWithLocalStorageViewStatePersistence<ID extends string, T extends Tab<ID>> extends React.PureComponent<
    TabsProps<ID, T> & {
        /**
         * A key unique to this UI element that is used for persisting the view state.
         */
        storageKey: string
    },
    { activeTab: ID | undefined }
> {
    constructor(props: TabsProps<ID, T> & { storageKey: string }) {
        super(props)
        this.state = {
            activeTab: TabsWithLocalStorageViewStatePersistence.readFromLocalStorage(
                this.props.storageKey,
                this.props.tabs
            ),
        }
    }

    private static readFromLocalStorage<ID extends string, T extends Tab<ID>>(
        storageKey: string,
        tabs: T[]
    ): ID | undefined {
        const lastTabID = localStorage.getItem(storageKey)
        if (lastTabID !== null && tabs.find(tab => tab.id === lastTabID)) {
            return lastTabID as ID
        }
        if (tabs.length === 0) {
            return undefined
        }
        return tabs[0].id // default
    }

    private static saveToLocalStorage<ID extends string>(storageKey: string, lastTabID: ID): void {
        localStorage.setItem(storageKey, lastTabID)
    }

    public render(): JSX.Element | null {
        return (
            <Tabs
                {...this.props}
                onSelectTab={this.onSelectTab}
                activeTab={this.state.activeTab}
                tabComponent={this.renderTab}
            />
        )
    }

    private onSelectTab = (tab: ID): void => {
        if (this.props.onSelectTab) {
            this.props.onSelectTab(tab)
        }
        this.setState({ activeTab: tab }, () =>
            TabsWithLocalStorageViewStatePersistence.saveToLocalStorage(this.props.storageKey, tab)
        )
    }

    private renderTab = ({ tab, className }: { tab: T; className: string }): JSX.Element => (
        <button type="button" className={className} data-test-tab={tab.id} onClick={() => this.onSelectTab(tab.id)}>
            {tab.label}
        </button>
    )
}

interface TabsWithURLViewStatePersistenceProps<ID extends string, T extends Tab<ID>> extends TabsProps<ID, T> {
    location: H.Location
}

/**
 * A wrapper for Tabs that persists view state (the currently active tab) in the current page's URL.
 *
 * URL whose fragment (hash) ends with "$x" are considered to have active tab "x" (where "x" is the tab's ID).
 */
export class TabsWithURLViewStatePersistence<ID extends string, T extends Tab<ID>> extends React.PureComponent<
    TabsWithURLViewStatePersistenceProps<ID, T>,
    { activeTab: ID | undefined }
> {
    constructor(props: TabsWithURLViewStatePersistenceProps<ID, T>) {
        super(props)
        this.state = {
            activeTab: TabsWithURLViewStatePersistence.readFromURL(props.location, props.tabs),
        }
    }

    /**
     * Returns the URL hash (which can be used as a relative URL) that specifies the given tab. If the URL hash
     * already contains a tab ID, it replaces it; otherwise it appends it to the current URL fragment. If the tabID
     * argument is null, then the tab ID is removed from the URL.
     */
    public static urlForTabID(location: H.Location, tabID: string | null): H.LocationDescriptorObject {
        const hash = new URLSearchParams(location.hash.slice('#'.length))
        if (tabID) {
            hash.set('tab', tabID)
        } else {
            hash.delete('tab')

            // Remove other known keys that are only associated with a panel. This makes it so the URL
            // is nicer when the panel is closed (it is stripped of all irrelevant panel hash state).
            //
            // TODO: Un-hardcode these so that other panels don't need to remember to add their keys
            // here.
            hash.delete('threadID')
            hash.delete('commentID')
        }
        return {
            ...location,
            hash: hash.toString().replace(/%3A/g, ':').replace(/=$/, ''), // remove needless trailing `=` as in `#L12=`,
        }
    }

    public static readFromURL<ID extends string, T extends Tab<ID>>(location: H.Location, tabs: T[]): ID | undefined {
        const urlTabID = parseHash(location.hash).viewState
        if (urlTabID) {
            for (const tab of tabs) {
                if (tab.id === urlTabID) {
                    return tab.id
                }
            }
        }
        if (tabs.length === 0) {
            return undefined
        }
        return tabs[0].id // default
    }

    public componentDidUpdate(previousProps: TabsWithURLViewStatePersistenceProps<ID, T>): void {
        if (previousProps.location !== this.props.location || previousProps.tabs !== this.props.tabs) {
            // eslint-disable-next-line react/no-did-update-set-state
            this.setState({
                activeTab: TabsWithURLViewStatePersistence.readFromURL(this.props.location, this.props.tabs),
            })
        }
    }

    public render(): JSX.Element | null {
        return <Tabs {...this.props} activeTab={this.state.activeTab} tabComponent={this.renderTab} />
    }

    private renderTab = ({ tab, className }: { tab: T; className: string }): JSX.Element => (
        /* eslint-disable react/jsx-no-bind */
        <Link
            className={className}
            to={TabsWithURLViewStatePersistence.urlForTabID(this.props.location, tab.id)}
            onClick={() => {
                if (this.props.onSelectTab) {
                    this.props.onSelectTab(tab.id)
                }
            }}
        >
            {tab.label}
        </Link>
        /* eslint:enable react/jsx-no-bind */
    )
}
