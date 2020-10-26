import * as H from 'history'
import React from 'react'
import { Subscription } from 'rxjs'
import { ExtensionsControllerProps } from '../../shared/src/extensions/controller'
import { registerHighlightContributions } from '../../shared/src/highlight/contributions'
import { registerHoverContributions } from '../../shared/src/hover/actions'
import { PlatformContextProps } from '../../shared/src/platform/context'
import { registerSearchStatsContributions } from './enterprise/search/stats/contributions'

interface Props extends ExtensionsControllerProps, PlatformContextProps {
    history: H.History
}

/**
 * A component that registers global contributions. It is implemented as a React component so that its
 * registrations use the React lifecycle.
 */
export class GlobalContributions extends React.Component<Props> {
    private subscriptions = new Subscription()

    public componentDidMount(): void {
        registerHighlightContributions() // no way to unregister these
        this.subscriptions.add(
            registerHoverContributions({ ...this.props, locationAssign: location.assign.bind(location) })
        )
        this.subscriptions.add(registerSearchStatsContributions(this.props))
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        return null
    }
}
