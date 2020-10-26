import * as React from 'react'
import ReactDOM from 'react-dom'
import isAbsoluteUrl from 'is-absolute-url'
import { DecorationAttachmentRenderOptions } from 'sourcegraph'
import { decorationAttachmentStyleForTheme } from '../../../../shared/src/api/client/services/decoration'
import { LinkOrSpan } from '../../../../shared/src/components/LinkOrSpan'
import { AbsoluteRepoFile } from '../../../../shared/src/util/url'
import { ThemeProps } from '../../../../shared/src/theme'

interface LineDecorationAttachmentProps extends AbsoluteRepoFile, ThemeProps {
    line: number
    portalID: string
    attachment: DecorationAttachmentRenderOptions
}

/** Displays text after a line in Blob. */
// @lguychard 2019-08-16 using UNSAFE_componentWillMount and UNSAFE_componentWillReceiveProps because this.portal
// needs to be updated *before* render, which componentDidMount and componentDidUpdate don't allow us to do.
// Using componentDidMount and componentDidUpdate led to decorations only being displayed when clicking in the viewport
// (thereby triggering a re-render after the initial componentDidMount/DidUpdate had been called).
//
// See https://github.com/sourcegraph/sourcegraph/issues/5236
//
// eslint-disable-next-line react/no-unsafe
export class LineDecorationAttachment extends React.PureComponent<LineDecorationAttachmentProps> {
    private portal: Element | null = null

    public UNSAFE_componentWillMount(): void {
        this.portal = document.querySelector(`#${this.props.portalID}`)
    }

    public UNSAFE_componentWillReceiveProps(nextProps: Readonly<LineDecorationAttachmentProps>): void {
        if (
            this.props.repoName !== nextProps.repoName ||
            this.props.revision !== nextProps.revision ||
            this.props.filePath !== nextProps.filePath ||
            this.props.line !== nextProps.line ||
            this.props.portalID !== nextProps.portalID ||
            this.props.attachment !== nextProps.attachment
        ) {
            this.portal = document.querySelector(`#${nextProps.portalID}`)
        }
    }

    public render(): React.ReactPortal | null {
        if (!this.portal) {
            return null
        }

        const style = decorationAttachmentStyleForTheme(this.props.attachment, this.props.isLightTheme)

        return ReactDOM.createPortal(
            <LinkOrSpan
                className="line-decoration-attachment"
                to={this.props.attachment.linkURL}
                data-tooltip={this.props.attachment.hoverMessage}
                // Use target to open external URLs
                target={
                    this.props.attachment.linkURL && isAbsoluteUrl(this.props.attachment.linkURL) ? '_blank' : undefined
                }
                // Avoid leaking referrer URLs (which contain repository and path names, etc.) to external sites.
                rel="noreferrer noopener"
            >
                <span
                    className="line-decoration-attachment__contents"
                    // eslint-disable-next-line react/forbid-dom-props
                    style={{
                        color: style.color,
                        backgroundColor: style.backgroundColor,
                    }}
                    data-contents={this.props.attachment.contentText || ''}
                />
            </LinkOrSpan>,
            this.portal
        )
    }
}
