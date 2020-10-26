import { Primitive } from 'utility-types'
import { Expression, TemplateExpression } from '../client/context/expr/evaluator'
import { KeyPath } from '../client/services/settings'

// NOTE: You must manually keep this file in sync with extension.schema.json#/properties/contributes (and possibly
// extension_schema.go, if your changes are relevant to the subset of this schema used by our Go code).
//
// The available tools for automatically generating the JSON Schema from this file add more complexity than it's
// worth.

/**
 * The given contribution type as it's specified in a package.json (expressions as strings)
 */
export type Raw<TContribution> = { [K in keyof TContribution]: RawValue<TContribution[K]> }
// need a type alias so union types (including undefined for optional properties) distribute
type RawValue<V> = V extends Expression<any> ? string : V extends Primitive | null | undefined ? V : Raw<V>

/**
 * The given contribution type after being evaluated (all expressions replaced with their value)
 */
export type Evaluated<TContribution> = { [K in keyof TContribution]: EvaluatedValue<TContribution[K]> }
// need a type alias so union types (including undefined for optional properties) distribute
type EvaluatedValue<V> = V extends Expression<infer T> ? T : V extends Primitive | null | undefined ? V : Evaluated<V>

/**
 * Describes the functionality provided by an extension.
 */
export interface Contributions {
    /** Actions contributed by the extension. */
    actions?: ActionContribution[]

    /** Menu items contributed by the extension. */
    menus?: MenuContributions

    /** Views contributed by the extension. */
    views?: ViewContribution[]

    /** Search filters contributed by the extension */
    searchFilters?: SearchFilters[]
}

/**
 * An action contribution describes a command that can be invoked, along with a title, description, icon, etc.
 */
export interface ActionContribution {
    /**
     * The identifier for this action, which must be unique among all contributed actions.
     *
     * Extensions: By convention, this is a dotted string of the form `myExtensionName.myActionName`. It is common
     * to use the same values for `id` and `command` (for the common case where the command has only one action
     * that mentions it).
     */
    id: string

    /**
     * The command that this action invokes. It can refer to a command registered by the same
     * extension or any other extension, or to a builtin command. If it is undefined, the action is
     * a noop.
     *
     * See "[Builtin commands](../../../../doc/extensions/authoring/builtin_commands.md)" (online at
     * https://docs.sourcegraph.com/extensions/authoring/builtin_commands) for documentation on
     * builtin client commands.
     *
     * Extensions: The command must be registered (unless it is a builtin command). Extensions can
     * register commands using {@link sourcegraph.commands.registerCommand}.
     *
     * Clients: All clients must implement the builtin commands as specified in the documentation
     * above. If the command is undefined (which means the action is a noop), the action should be
     * displayed in such a way that avoids suggesting it can be clicked/activated (e.g., there
     * should not be any hover or active state on the button).
     *
     * @see ActionContributionClientCommandOpen
     * @see ActionContributionClientCommandUpdateConfiguration
     */
    command?: string

    /**
     * Optional arguments to pass to the extension when the action is invoked.
     * Strings are interpreted as template strings, any other value is taken as-is.
     */
    commandArguments?: (TemplateExpression | number | boolean | null | object | any[])[]

    /** The title that succinctly describes what this action does. */
    title?: TemplateExpression

    /**
     * The category that describes the group of related actions of which this action is a member.
     *
     * Clients: When displaying this action's title alongside titles of actions from other groups, the client
     * should display each action as "${category}: ${title}" if the prefix is set.
     */
    category?: TemplateExpression

    /**
     * A longer description of the action taken by this action.
     *
     * Extensions: The description should not be unnecessarily repetitive with the title.
     *
     * Clients: If the description is shown, the title must be shown nearby.
     */
    description?: TemplateExpression

    /**
     * A URL to an icon for this action (data: URIs are OK).
     *
     * Clients: The client should show this icon before the title, proportionally scaling the dimensions as
     * necessary to avoid unduly enlarging the item beyond the dimensions necessary to render the text. The client
     * should assume the icon is square (or roughly square). The client must not display a border around the icon.
     * The client may choose not to display this icon.
     */
    iconURL?: TemplateExpression

    /**
     * A specification of how to display this action as a button on a toolbar. The client is responsible for
     * displaying contributions and defining which parts of its interface are considered to be toolbars. Generally,
     * items on a toolbar are always visible and, compared to items in a dropdown menu or list, are expected to be
     * smaller and to convey information (in addition to performing an action).
     *
     * For example, a "Toggle code coverage" action may prefer to display a summarized status (such as "Coverage:
     * 77%") on a toolbar instead of the full title.
     *
     * Clients: If the label is empty and only an iconURL is set, and the client decides not to display the icon
     * (e.g., because the client is not graphical), then the client may hide the item from the toolbar.
     */
    actionItem?: ActionItem
}

/**
 * Narrows the type of {@link ActionContribution} for actions that invoke the `open` client command.
 */
export interface ActionContributionClientCommandOpen extends ActionContribution {
    command: 'open'

    /**
     * The arguments for the `open` client command. The first array element is a URL, which is opened by the client
     * using the default URL handler.
     */
    commandArguments: [TemplateExpression]
}

/**
 * Narrows the type of {@link ActionContribution} for actions that invoke the `updateConfiguration` client command.
 */
export interface ActionContributionClientCommandUpdateConfiguration extends ActionContribution {
    command: 'updateConfiguration'

    /**
     * The arguments for the `updateConfiguration` client command:
     *
     * 1. The key path of the value (in the configuration settings) to update
     * 2. The value to insert at the key path
     * 3. Optional: reserved for future use (must always be `null`)
     * 4. Optional: 'json' if the client should parse the 2nd argument using JSON.parse before inserting the value
     */
    commandArguments:
        | [KeyPath, TemplateExpression | number | boolean | object | any[] | null]
        | [KeyPath, TemplateExpression, null, TemplateExpression<'json'>]
}

/**
 * A description of how to display an {@link ActionContribution} on a toolbar,
 * with all template properties interpolated, and all expression properties replaced
 * by their evaluated value.
 *
 * This value is set on the {@link ActionContribution#actionItem} interface field
 * (it always is directly associated with an {@link ActionContribution}).
 *
 * It is necessary because an action's ({@link ActionContribution}'s) fields are intended for display in a command
 * palette or list, not in a button. The {@link ActionContribution} fields usually have long, descriptive text
 * values, and it does not make sense for them to show an icon. When the action is displayed as a button, however,
 * it needs to have a much shorter text label and it often does make sense to show an icon. Therefore, the action's
 * representation as a command in a list ({@link ActionContribution}) is separate from its representation as a
 * button (in this type, {@link ActionItem}).
 *
 * Example: Consider a code coverage extension that adds an action to toggle showing coverage overlays on a file.
 * The command title ({@link ActionContribution#title}) might be "Show/hide code coverage overlays", and the button
 * label ({@link ActionItem#label}) would be "Coverage: 78%" (where the "78%" is the live coverage ratio).
 *
 * It is convenient to be able to specify both the command and button display representations of an action
 * together, which is why {@link ActionItem} is in the field {@link ActionContribution#actionItem} instead of being
 * listed as a separate contribution. They share most behavior and many other fields, so it reduces the amount of
 * duplicate code to combine them. Also, some clients may not be able to display buttons and need to display the
 * more verbose command form of an action, which is possible when they are combined.
 */
export interface ActionItem {
    /** The text label for this item. */
    label?: Expression<string>

    /**
     * A description associated with this action item.
     *
     * Clients: The description should be shown in a tooltip when the user focuses or hovers this toolbar item.
     */
    description?: Expression<string>

    /**
     * The icon URL for this action (data: URIs are OK).
     *
     * Clients: The client should this icon before the label (if any), proportionally scaling the dimensions as
     * necessary to avoid unduly enlarging the toolbar item beyond the dimensions necessary to show the label.
     * In space-constrained situations, the client should show only the icon and omit the label. The client
     * must not display a border around the icon. The client may choose not to display this icon.
     */
    iconURL?: Expression<string>

    /**
     * A description of the information represented by the icon.
     *
     * Clients: The client should not display this text directly. Instead, the client should use the
     * accessibility facilities of the client's platform (such as the <img alt> attribute) to make it available
     * to users needing the textual description.
     */
    iconDescription?: Expression<string>

    /**
     * Whether the action item should be rendered as a pressed button.
     */
    pressed?: Expression<boolean>
}

export enum ContributableMenu {
    /** The global command palette. */
    CommandPalette = 'commandPalette',

    /** The global navigation bar in the application. */
    GlobalNav = 'global/nav',

    /** The title bar for the current document. */
    EditorTitle = 'editor/title',

    /** A directory page (including for the root directory of a repository). */
    DirectoryPage = 'directory/page',

    /** The hover tooltip. */
    Hover = 'hover',

    /** The panel toolbar. */
    PanelToolbar = 'panel/toolbar',

    /** The search results toolbar. */
    SearchResultsToolbar = 'search/results/toolbar',

    /** The help menu in the application. */
    Help = 'help',
}

/**
 * MenuContributions describes the menu items contributed by an extension.
 */
export interface MenuContributions extends Partial<Record<ContributableMenu, MenuItemContribution[]>> {}

/**
 * MenuItemContribution is a menu item contributed by an extension.
 */
export interface MenuItemContribution {
    /**
     * The action to invoke when the item is selected. The value refers to a {@link ActionContribution#id} value.
     */
    action: string

    /**
     * An alternative action to invoke when the item is selected while pressing the Option/Alt/Meta/Ctrl/Cmd keys
     * or using the middle mouse button. The value refers to a {@link ActionContribution#id} value.
     */
    alt?: string

    /**
     * An expression that, if given, must evaluate to true (or a truthy value) for this contribution to be
     * displayed. The expression may use values from the context in which the contribution would be displayed.
     */
    when?: Expression<boolean>

    /**
     * The group in which this item is displayed. This defines the sort order of menu items. The group value is an
     * opaque string (it is just compared relative to other items' group values); there is no specification set of
     * expected or supported values.
     *
     * Clients: On a toolbar, the client should sort toolbar items by (group, action), with toolbar items lacking a
     * group sorting last. The client must not display the group value.
     */
    group?: string
}

/**
 * A search filters interface with `name` and `value` to display on a filter chip
 * in the search results filters bar.
 */
export interface SearchFilters {
    /**
     * The name to be displayed on the search filter chip.
     */
    name: string

    /**
     * The value of the search filter chip (i.e. the literal search query string).
     */
    value: string
}

/** The containers to which an extension can contribute views. */
export const ContributableViewContainer = {
    /**
     * A view that is displayed in the panel for a window.
     *
     * Clients: The client should render this as a resizable panel in a window, with multiple tabs to switch
     * between different panel views.
     */
    Panel: 'window/panel',

    /**
     * A global page view, displayed as a standalone page at `/views/ID`.
     */
    GlobalPage: 'global/page',

    /**
     * A view contributed to directory pages.
     */
    Directory: 'directory',

    /**
     * A view contributed to the area on the homepage below the search box.
     */
    Homepage: 'homepage',

    /**
     * A view contributed to the dashboard on the insights page.
     */
    InsightsPage: 'insightsPage',
} as const
export type ContributableViewContainer = typeof ContributableViewContainer[keyof typeof ContributableViewContainer]

/**
 * A view contributed by an extension.
 */
export interface ViewContribution {
    /**
     * The identifier for this view, which must be unique among all contributed views.
     */
    id: string

    /**
     * The container where this view will be displayed.
     */
    where: ContributableViewContainer
}
