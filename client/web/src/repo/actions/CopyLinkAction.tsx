import copy from 'copy-to-clipboard'
import * as H from 'history'
import ContentCopyIcon from 'mdi-react/ContentCopyIcon'
import * as React from 'react'
import { Tooltip } from '../../../../branded/src/components/tooltip/Tooltip'
import { eventLogger } from '../../tracking/eventLogger'

interface Props {
    location: H.Location
}

interface State {
    copied: boolean
}

/**
 * A repository header action that copies the current page's URL to the clipboard.
 */
export class CopyLinkAction extends React.PureComponent<Props, State> {
    public state = { copied: false }

    public componentDidUpdate(previousProps: Props, previousState: State): void {
        if (previousState.copied !== this.state.copied) {
            Tooltip.forceUpdate()
        }
    }

    public render(): JSX.Element | null {
        return (
            <button
                type="button"
                className="btn btn-icon"
                data-tooltip={this.state.copied ? 'Copied!' : 'Copy link to clipboard'}
                aria-label="Copy link"
                onClick={this.onClick}
            >
                <ContentCopyIcon className="icon-inline" />
            </button>
        )
    }

    private onClick: React.MouseEventHandler<HTMLElement> = event => {
        event.preventDefault()
        eventLogger.log('ShareButtonClicked')
        const location = this.props.location
        const shareLink = new URL(location.pathname + location.search + location.hash, window.location.href)
        shareLink.searchParams.set('utm_source', 'share')
        copy(shareLink.href)
        this.setState({ copied: true })

        setTimeout(() => {
            this.setState({ copied: false })
        }, 1000)
    }
}
