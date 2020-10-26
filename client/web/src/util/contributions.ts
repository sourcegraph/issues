import { RouteComponentProps } from 'react-router'

interface Conditional<C extends object> {
    /** Optional condition under which this item should be used */
    readonly condition?: (context: C) => boolean
}

interface WithIcon {
    readonly icon?: React.ComponentType<{ className?: string }>
}

/**
 * Configuration for a component.
 *
 * @template C Context information that is passed to `render` and `condition`
 */
export interface ComponentDescriptor<C extends object = {}> extends Conditional<C> {
    readonly render: (props: C) => React.ReactNode
}

/**
 * Configuration for a route.
 *
 * @template C Context information that is passed to `render` and `condition`
 */
export interface RouteDescriptor<C extends object = {}, P = any> extends Conditional<C> {
    /** Path of this route (appended to the current match) */
    readonly path: string
    readonly exact?: boolean
    readonly render: (props: C & RouteComponentProps<P>) => React.ReactNode
}

export interface NavGroupDescriptor<C extends object = {}> extends Conditional<C> {
    readonly header?: {
        readonly label: string
        readonly icon?: React.ComponentType<{ className?: string }>
    }
    readonly items: readonly NavItemDescriptor<C>[]
}

/**
 * Used to customize sidebar items.
 * The difference between this and an action button is that nav items get highlighted if their `to` route matches.
 *
 * @template C Context information that is made available to determine whether the item should be shown (different for each sidebar)
 */
export interface NavItemDescriptor<C extends object = {}> extends Conditional<C> {
    /** The text of the item */
    readonly label: string

    /** The link destination (appended to the current match) */
    readonly to: string

    /** Whether highlighting the item should only be done if `to` matches exactly */
    readonly exact?: boolean
}

export interface NavItemWithIconDescriptor<C extends object = {}> extends NavItemDescriptor<C>, WithIcon {}

/**
 * A descriptor for an action button that should appear somewhere in the UI.
 *
 * @template C Context information that is made available to determine whether the item should be shown and the link destination
 */
export interface ActionButtonDescriptor<C extends object = {}> extends Conditional<C>, WithIcon {
    /** Label for for the button  */
    readonly label: string

    /** Optional tooltip for the button (if set, should include more information than the label) */
    readonly tooltip?: string

    /** Function to return the destination link for the button */
    readonly to: (context: C) => string
}
