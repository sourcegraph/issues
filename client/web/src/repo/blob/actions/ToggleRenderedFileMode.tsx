import * as H from 'history'
import EyeIcon from 'mdi-react/EyeIcon'
import * as React from 'react'

import { Tooltip } from '@sourcegraph/branded/src/components/tooltip/Tooltip'
import { ButtonLink } from '@sourcegraph/shared/src/components/LinkOrButton'
import { RenderMode } from '@sourcegraph/shared/src/util/url'

import { RepoHeaderContext } from '../../RepoHeader'

interface Props extends RepoHeaderContext {
    location: H.Location
    mode: RenderMode
}

/**
 * A repository header action that toggles between showing a rendered file and the file's original
 * source, for files that can be rendered (such as Markdown files).
 */
export class ToggleRenderedFileMode extends React.PureComponent<Props> {
    private static URL_QUERY_PARAM = 'view'

    /**
     * Reports whether the location's URL displays the blob as rendered or source.
     */
    public static getModeFromURL(location: H.Location): RenderMode {
        const searchParameters = new URLSearchParams(location.search)

        if (!searchParameters.has(ToggleRenderedFileMode.URL_QUERY_PARAM)) {
            return undefined
        }
        return searchParameters.get(ToggleRenderedFileMode.URL_QUERY_PARAM) === 'code' ? 'code' : 'rendered' // default to rendered
    }

    /**
     * Returns the URL that displays the blob using the specified mode.
     */
    private getURLForMode(location: H.Location, mode: RenderMode): H.Location {
        const searchParameters = new URLSearchParams(location.search)
        if (mode === 'code') {
            searchParameters.set(ToggleRenderedFileMode.URL_QUERY_PARAM, mode)
        } else {
            searchParameters.delete(ToggleRenderedFileMode.URL_QUERY_PARAM)
        }
        return { ...location, search: searchParameters.toString() }
    }

    public componentDidUpdate(previousProps: Props): void {
        if (previousProps.mode !== this.props.mode) {
            Tooltip.forceUpdate()
        }
    }

    public render(): JSX.Element | null {
        const otherMode: RenderMode = this.props.mode === 'code' ? 'rendered' : 'code'

        if (this.props.actionType === 'dropdown') {
            return (
                <ButtonLink
                    className="btn repo-header__file-action"
                    to={this.getURLForMode(this.props.location, otherMode)}
                >
                    <EyeIcon className="icon-inline" />
                    <span>{otherMode === 'code' ? 'Show raw code file' : 'Show formatted file'}</span>
                </ButtonLink>
            )
        }

        return (
            <ButtonLink
                to={this.getURLForMode(this.props.location, otherMode)}
                data-tooltip={otherMode === 'code' ? 'Show raw code file' : 'Show formatted file'}
                className="btn btn-icon repo-header__action"
            >
                <EyeIcon className="icon-inline" />{' '}
                <span className="d-none d-lg-inline ml-1">{otherMode === 'code' ? 'Raw' : 'Formatted'}</span>
            </ButtonLink>
        )
    }
}
