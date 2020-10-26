import * as React from 'react'
import { isExternalLink } from '../util/url'
import InformationIcon from 'mdi-react/InfoCircleOutlineIcon'
import WarningIcon from 'mdi-react/AlertCircleOutlineIcon'
import ErrorIcon from 'mdi-react/AlertDecagramOutlineIcon'
import { BadgeAttachmentRenderOptions } from 'sourcegraph'
import { badgeAttachmentStyleForTheme } from '../api/client/services/decoration'
import { LinkOrSpan } from './LinkOrSpan'
import { isEncodedImage } from '../util/icon'
import { MdiReactIconComponentType, MdiReactIconProps } from 'mdi-react'
import classNames from 'classnames'

const iconComponents: Record<BadgeAttachmentRenderOptions['kind'], MdiReactIconComponentType> = {
    info: InformationIcon,
    warning: WarningIcon,
    error: ErrorIcon,
}

export const BadgeAttachment: React.FunctionComponent<{
    attachment: BadgeAttachmentRenderOptions
    isLightTheme: boolean
    className?: string
    iconClassName?: string
    iconButtonClassName?: string
}> = ({ attachment, isLightTheme, className, iconButtonClassName, iconClassName }) => {
    const style = badgeAttachmentStyleForTheme(attachment, isLightTheme)
    const PredefinedIcon: React.ComponentType<MdiReactIconProps> | undefined =
        attachment.kind && iconComponents[attachment.kind]

    return (
        <LinkOrSpan
            className={classNames(className, attachment.linkURL && iconButtonClassName)}
            to={attachment.linkURL}
            data-tooltip={attachment.hoverMessage}
            data-placement="left"
            // Use target to open external URLs
            target={attachment.linkURL && isExternalLink(attachment.linkURL) ? '_blank' : undefined}
            // Avoid leaking referrer URLs (which contain repository and path names, etc.) to external sites.
            rel="noreferrer noopener"
        >
            {PredefinedIcon ? (
                <PredefinedIcon className={iconClassName} />
            ) : (
                style.icon &&
                isEncodedImage(style.icon) && (
                    <img
                        className={iconClassName}
                        // eslint-disable-next-line react/forbid-dom-props
                        style={{
                            color: style.color,
                            backgroundColor: style.backgroundColor,
                        }}
                        src={style.icon}
                    />
                )
            )}
        </LinkOrSpan>
    )
}
