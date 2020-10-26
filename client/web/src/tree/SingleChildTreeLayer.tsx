import * as React from 'react'
import { ChildTreeLayer } from './ChildTreeLayer'
import { Directory } from './Directory'
import { TreeNode } from './Tree'
import { TreeLayerProps } from './TreeLayer'
import { maxEntries, SingleChildGitTree } from './util'
import classNames from 'classnames'

interface SingleChildTreeLayerProps extends TreeLayerProps {
    childrenEntries: SingleChildGitTree[]
}

/**
 * SingleChildTreeLayers are directories that are the only child of a parent directory.
 * These will automatically render and expand, so users don't need to
 * click each nested directory if there's no additional content to see.
 * There are no network requests made in single child layers. Rather, this layer's parent
 * will pass the entries this layer needs to load in `props.childrenEntries`.
 */
export class SingleChildTreeLayer extends React.Component<SingleChildTreeLayerProps> {
    public node: TreeNode

    constructor(props: SingleChildTreeLayerProps) {
        super(props)
        this.node = {
            index: this.props.index,
            parent: this.props.parent,
            childNodes: [],
            path: this.props.entryInfo ? this.props.entryInfo.path : '',
            url: this.props.entryInfo ? this.props.entryInfo.url : '',
        }
    }

    public componentDidMount(): void {
        this.props.setChildNodes(this.node, this.node.index)

        if (this.props.activePath === this.node.path) {
            this.props.setActiveNode(this.node)
        }

        // On mount, immediately set this to be expanded so it renders its child layers immediately.
        this.props.onToggleExpand(this.props.entryInfo.path, true, this.node)
    }

    public componentDidUpdate(previousProps: SingleChildTreeLayerProps): void {
        // Reset childNodes to none if the parent path changes, so we don't have children of past visited layers in the childNodes.
        if (previousProps.parentPath !== this.props.parentPath) {
            this.node.childNodes = []
        }
    }

    public shouldComponentUpdate(nextProps: SingleChildTreeLayerProps): boolean {
        if (nextProps.activeNode !== this.props.activeNode) {
            if (nextProps.activeNode === this.node) {
                return true
            }

            // Update if currently active node
            if (this.props.activeNode === this.node) {
                return true
            }

            // Update if parent of currently active node
            let currentParent = this.props.activeNode.parent
            while (currentParent) {
                if (currentParent === this.node) {
                    return true
                }
                currentParent = currentParent.parent
            }
        }

        if (nextProps.selectedNode !== this.props.selectedNode) {
            // Update if this row will be the selected node.
            if (nextProps.selectedNode === this.node) {
                return true
            }

            // Update if a parent of the next selected row.
            let parent = nextProps.selectedNode.parent
            while (parent) {
                if (parent === this.node) {
                    return true
                }
                parent = parent?.parent
            }

            // Update if currently selected node.
            if (this.props.selectedNode === this.node) {
                return true
            }

            // Update if parent of currently selected node.
            let currentParent = this.props.selectedNode.parent
            while (currentParent) {
                if (currentParent === this.node) {
                    return true
                }
                currentParent = currentParent?.parent
            }

            // If none of the above conditions are met, there's no need to update.
            return false
        }

        return true
    }

    public render(): JSX.Element | null {
        const className = classNames(
            'tree__row',
            this.props.isExpanded && 'tree__row--expanded',
            this.node === this.props.activeNode && 'tree__row--active',
            this.node === this.props.selectedNode && 'tree__row--selected'
        )

        return (
            <div>
                <table
                    className="tree-layer"
                    onMouseOver={this.props.entryInfo.isDirectory ? this.invokeOnHover : undefined}
                >
                    <tbody>
                        <Directory
                            {...this.props}
                            className={className}
                            maxEntries={maxEntries}
                            loading={false}
                            handleTreeClick={this.handleTreeClick}
                            noopRowClick={this.noopRowClick}
                            linkRowClick={this.linkRowClick}
                        />
                        {this.props.isExpanded && (
                            <tr>
                                <td>
                                    <ChildTreeLayer
                                        {...this.props}
                                        parent={this.node}
                                        entries={this.props.childrenEntries}
                                        singleChildTreeEntry={this.props.childrenEntries[0]}
                                        childrenEntries={this.props.childrenEntries[0].children}
                                        setChildNodes={this.setChildNode}
                                    />
                                </td>
                            </tr>
                        )}
                    </tbody>
                </table>
            </div>
        )
    }

    /**
     * Non-root tree layers call this to activate a prefetch request in the root tree layer
     */
    private invokeOnHover = (event: React.MouseEvent<HTMLElement>): void => {
        if (this.props.onHover) {
            event.stopPropagation()
            this.props.onHover(this.node.path)
        }
    }

    private handleTreeClick = (): void => {
        this.props.onSelect(this.node)
        const path = this.props.entryInfo ? this.props.entryInfo.path : ''
        this.props.onToggleExpand(path, !this.props.isExpanded, this.node)
    }

    /**
     * noopRowClick is the click handler for <a> rows of the tree element
     * that shouldn't update URL on click w/o modifier key (but should retain
     * anchor element properties, like right click "Copy link address").
     */
    private noopRowClick = (event: React.MouseEvent<HTMLAnchorElement>): void => {
        if (!event.altKey && !event.metaKey && !event.shiftKey && !event.ctrlKey) {
            event.preventDefault()
            event.stopPropagation()
        }
        this.handleTreeClick()
    }

    /**
     * linkRowClick is the click handler for <Link>
     */
    private linkRowClick: React.MouseEventHandler<HTMLAnchorElement> = () => {
        this.props.setActiveNode(this.node)
        this.props.onSelect(this.node)
    }

    private setChildNode = (node: TreeNode, index: number): void => {
        this.node.childNodes[index] = node
    }
}
