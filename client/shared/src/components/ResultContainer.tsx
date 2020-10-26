import ChevronDownIcon from 'mdi-react/ChevronDownIcon'
import ChevronRightIcon from 'mdi-react/ChevronRightIcon'
import ChevronUpIcon from 'mdi-react/ChevronUpIcon'
import * as React from 'react'

export interface Props {
    /**
     * Whether the result container's children are visible by default.
     * The header is always visible even when the component is not expanded.
     */
    defaultExpanded?: boolean

    /**
     * Whether the result container can be collapsed. If false, its children
     * are always displayed, and no expand/collapse actions are shown.
     */
    collapsible?: boolean

    /**
     * The icon to show left to the title.
     */
    icon: React.ComponentType<{ className?: string }>

    /**
     * The title component.
     */
    title: React.ReactFragment

    /**
     * CSS class name to apply to the title element.
     */
    titleClassName?: string

    /**
     * The content of the result displayed underneath the result container's
     * header when collapsed.
     */
    collapsedChildren?: React.ReactFragment

    /**
     * The content of the result displayed underneath the result container's
     * header when expanded.
     */
    expandedChildren?: React.ReactFragment

    /**
     * The label to display next to the collapse button
     */
    collapseLabel?: string

    /**
     * The label to display next to the expand button
     */
    expandLabel?: string

    /**
     * This component does not accept children.
     */
    children?: never

    /** Expand all results */
    allExpanded?: boolean
    stringIcon?: string
}

interface State {
    /**
     * Whether this result container is currently expanded.
     */
    expanded?: boolean
}

/**
 * The container component for a result in the SearchResults component.
 */
export class ResultContainer extends React.PureComponent<Props, State> {
    constructor(props: Props) {
        super(props)
        this.state = { expanded: this.props.allExpanded || this.props.defaultExpanded }
    }

    public componentDidUpdate(previousProps: Props): void {
        if (previousProps.allExpanded !== this.props.allExpanded) {
            if (this.state.expanded === previousProps.allExpanded) {
                // eslint-disable-next-line react/no-did-update-set-state
                this.setState({ expanded: this.props.allExpanded })
            } else {
                // eslint-disable-next-line react/no-did-update-set-state
                this.setState({ expanded: this.props.allExpanded })
            }
        }
    }

    public render(): JSX.Element | null {
        const Icon = this.props.icon
        const stringIcon = this.props.stringIcon ? this.props.stringIcon : undefined
        return (
            <div className="test-search-result result-container" data-testid="result-container">
                <div
                    className={
                        'result-container__header' +
                        (this.props.collapsible ? ' result-container__header--collapsible' : '')
                    }
                    onClick={this.toggle}
                >
                    {stringIcon ? (
                        <img src={stringIcon} className="icon-inline icon-inline__filtered" />
                    ) : (
                        <Icon className="icon-inline" />
                    )}
                    <div
                        className={`result-container__header-title ${this.props.titleClassName || ''}`}
                        data-testid="result-container-header"
                    >
                        {this.props.collapsible ? (
                            <span onClick={blockExpandAndCollapse}>{this.props.title}</span>
                        ) : (
                            this.props.title
                        )}
                    </div>
                    {this.props.collapsible &&
                        (this.state.expanded ? (
                            <small className="result-container__toggle-matches-container">
                                {this.props.collapseLabel}
                                {this.props.collapseLabel && <ChevronUpIcon className="icon-inline" />}
                                {!this.props.collapseLabel && <ChevronDownIcon className="icon-inline" />}
                            </small>
                        ) : (
                            <small className="result-container__toggle-matches-container">
                                {this.props.expandLabel}
                                {this.props.expandLabel && <ChevronDownIcon className="icon-inline" />}
                                {!this.props.expandLabel && <ChevronRightIcon className="icon-inline" />}
                            </small>
                        ))}
                </div>
                {!this.state.expanded && this.props.collapsedChildren}
                {this.state.expanded && this.props.expandedChildren}
            </div>
        )
    }

    private toggle = (): void => {
        this.setState(state => ({ expanded: !state.expanded }))
    }
}

function blockExpandAndCollapse(event: React.MouseEvent<HTMLElement>): void {
    event.stopPropagation()
}
