import * as H from 'history'
import HistoryIcon from 'mdi-react/HistoryIcon'
import * as React from 'react'
import { fromEvent, Subject, Subscription } from 'rxjs'
import { filter } from 'rxjs/operators'
import { ButtonLink } from '../../../../../shared/src/components/LinkOrButton'
import {
    lprToRange,
    parseHash,
    toPositionOrRangeHash,
    toViewStateHashComponent,
} from '../../../../../shared/src/util/url'
import { Tooltip } from '../../../../../branded/src/components/tooltip/Tooltip'
import { eventLogger } from '../../../tracking/eventLogger'
import { BlobPanelTabID } from '../panel/BlobPanel'

/**
 * A repository header action that toggles the visibility of the history panel.
 */
export class ToggleHistoryPanel extends React.PureComponent<{
    location: H.Location
    history: H.History
}> {
    private toggles = new Subject<boolean>()
    private subscriptions = new Subscription()

    /**
     * Reports the current visibility (derived from the location).
     */
    public static isVisible(location: H.Location): boolean {
        return parseHash<BlobPanelTabID>(location.hash).viewState === 'history'
    }

    /**
     * Returns the location object (that can be passed to H.History's push/replace methods) that sets visibility to
     * the given value.
     */
    private static locationWithVisibility(location: H.Location, visible: boolean): H.LocationDescriptorObject {
        const hash = parseHash<BlobPanelTabID>(location.hash)
        if (visible) {
            hash.viewState = 'history' // defaults to last-viewed tab, or first tab
        } else {
            delete hash.viewState
        }
        return { hash: toPositionOrRangeHash({ range: lprToRange(hash) }) + toViewStateHashComponent(hash.viewState) }
    }

    public componentDidMount(): void {
        this.subscriptions.add(
            this.toggles.subscribe(() => {
                const visible = ToggleHistoryPanel.isVisible(this.props.location)
                eventLogger.log(visible ? 'HideHistoryPanel' : 'ShowHistoryPanel')
                this.props.history.push(ToggleHistoryPanel.locationWithVisibility(this.props.location, !visible))
                Tooltip.forceUpdate()
            })
        )

        // Toggle when the user presses 'alt+h' or 'opt+h'.
        this.subscriptions.add(
            fromEvent<KeyboardEvent>(window, 'keydown')
                .pipe(filter(event => event.altKey && event.key === 'h'))
                .subscribe(event => {
                    event.preventDefault()
                    this.toggles.next()
                })
        )
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        const visible = ToggleHistoryPanel.isVisible(this.props.location)
        return (
            <ButtonLink onSelect={this.onClick} data-tooltip={`${visible ? 'Hide' : 'Show'} history (Alt+H/Opt+H)`}>
                <HistoryIcon className="icon-inline" />
            </ButtonLink>
        )
    }

    private onClick = (): void => this.toggles.next()
}
