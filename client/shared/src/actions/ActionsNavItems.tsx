import * as React from 'react'
import { Subject, Subscription, combineLatest } from 'rxjs'
import { switchMap } from 'rxjs/operators'
import { ContributionScope, Context } from '../api/client/context/context'
import { getContributedActionItems } from '../contributions/contributions'
import { TelemetryProps } from '../telemetry/telemetryService'
import { ActionItem, ActionItemProps } from './ActionItem'
import { ActionsState } from './actions'
import { ActionsProps } from './ActionsContainer'
import classNames from 'classnames'

export interface ActionNavItemsClassProps {
    /**
     * CSS class name for one action item (`<button>` or `<a>`)
     */
    actionItemClass?: string

    /**
     * Additional CSS class name when the action item is a toogle in its enabled state.
     */
    actionItemPressedClass?: string

    actionItemIconClass?: string

    /**
     * CSS class name for each `<li>` element wrapping the action item.
     */
    listItemClass?: string
}

export interface ActionsNavItemsProps
    extends ActionsProps,
        ActionNavItemsClassProps,
        TelemetryProps,
        Pick<ActionItemProps, 'showLoadingSpinnerDuringExecution'> {
    /**
     * If true, it renders a `<ul className="nav">...</ul>` around the items. If there are no items, it renders `null`.
     *
     * If falsey (the default behavior), it emits a fragment of just the `<li>`s.
     */
    wrapInList?: boolean
    /**
     * Only applied if `wrapInList` is `true`
     */

    listClass?: string
}

/**
 * Renders the actions as a fragment of <li class="nav-item"> elements, for use in a Bootstrap <ul
 * class="nav"> or <ul class="navbar-nav">.
 */
export class ActionsNavItems extends React.PureComponent<ActionsNavItemsProps, ActionsState> {
    public state: ActionsState = {}

    private scopeChanges = new Subject<ContributionScope | undefined>()
    private extraContextChanges = new Subject<Context | undefined>()
    private subscriptions = new Subscription()

    public componentDidMount(): void {
        this.subscriptions.add(
            combineLatest([this.scopeChanges, this.extraContextChanges])
                .pipe(
                    switchMap(([scope, extraContext]) =>
                        this.props.extensionsController.services.contribution.getContributions(scope, extraContext)
                    )
                )
                .subscribe(contributions => this.setState({ contributions }))
        )
        this.scopeChanges.next(this.props.scope)
        this.extraContextChanges.next(this.props.extraContext)
    }

    public componentDidUpdate(previousProps: ActionsProps): void {
        if (previousProps.scope !== this.props.scope) {
            this.scopeChanges.next(this.props.scope)
        }
        if (previousProps.extraContext !== this.props.extraContext) {
            this.extraContextChanges.next(this.props.extraContext)
        }
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | React.ReactFragment | null {
        if (!this.state.contributions) {
            return null // loading
        }

        const actionItems = getContributedActionItems(this.state.contributions, this.props.menu).map(item => (
            <React.Fragment key={item.action.id}>
                {' '}
                <li className={this.props.listItemClass}>
                    <ActionItem
                        key={item.action.id}
                        {...item}
                        {...this.props}
                        variant="actionItem"
                        iconClassName={this.props.actionItemIconClass}
                        className={classNames('actions-nav-items__action-item', this.props.actionItemClass)}
                        pressedClassName={this.props.actionItemPressedClass}
                    />
                </li>
            </React.Fragment>
        ))
        if (this.props.wrapInList) {
            return actionItems.length > 0 ? <ul className={this.props.listClass}>{actionItems}</ul> : null
        }
        return <>{actionItems}</>
    }
}
