import { isEqual } from 'lodash'
import * as React from 'react'
import { RepoHeaderContribution, RepoHeaderContributionsLifecycleProps } from './RepoHeader'

interface Props extends RepoHeaderContribution, RepoHeaderContributionsLifecycleProps {}

/**
 * Contributes an item to the RepoHeader. The contribution is added and remains present for this component's entire
 * lifecycle.
 *
 * A React component that needs to contribute an item to the RepoHeader should include a
 * <RepoHeaderContributionPortal> element that specifies the item.
 *
 * It is called a "portal" because it is similar to the React concept of a "portal": it effectively renders an
 * element in a different DOM hierarchy (i.e., in RepoHeader).
 */
export class RepoHeaderContributionPortal extends React.Component<Props> {
    public componentDidMount(): void {
        this.addOrUpdateContribution()
    }

    public componentDidUpdate(): void {
        this.addOrUpdateContribution()
    }

    public shouldComponentUpdate(nextProps: Props): boolean {
        // This "smart" comparison lets us skip ~75% of the updates that extending React.PureComponent (and not
        // implementing shouldComponentUpdate) or always returning true here would yield.
        return (
            this.props.repoHeaderContributionsLifecycleProps !== nextProps.repoHeaderContributionsLifecycleProps ||
            this.props.position !== nextProps.position ||
            this.props.priority !== nextProps.priority ||
            !isEqual(this.props.element.props, nextProps.element.props)
        )
    }

    public componentWillUnmount(): void {
        const key = this.props.element.key as string // enforced in RepoHeaderContributionStore
        if (this.props.repoHeaderContributionsLifecycleProps) {
            // Don't need to worry about being unable to remove this because once
            // this.props.repoHeaderContributionsLifecycleProps is set (from RepoHeader), it is never unset.
            this.props.repoHeaderContributionsLifecycleProps.onRepoHeaderContributionRemove(key)
        }
    }

    public render(): JSX.Element | null {
        return null
    }

    private addOrUpdateContribution(): void {
        if (this.props.repoHeaderContributionsLifecycleProps) {
            this.props.repoHeaderContributionsLifecycleProps.onRepoHeaderContributionAdd(this.props)
        }
    }
}
