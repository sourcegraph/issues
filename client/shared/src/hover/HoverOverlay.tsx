import classNames from 'classnames'
import CloseIcon from 'mdi-react/CloseIcon'
import React, { CSSProperties } from 'react'

import { ActionItem, ActionItemComponentProps } from '../actions/ActionItem'
import { NotificationType } from '../api/extension/extensionHostApi'
import { PlatformContextProps } from '../platform/context'
import { TelemetryProps } from '../telemetry/telemetryService'
import { ThemeProps } from '../theme'
import { isErrorLike } from '../util/errors'
import { sanitizeClass } from '../util/strings'

import { toNativeEvent } from './helpers'
import type { HoverContext, HoverOverlayBaseProps, GetAlertClassName } from './HoverOverlay.types'
import { HoverOverlayAlerts, HoverOverlayAlertsProps } from './HoverOverlayAlerts'
import { HoverOverlayContents } from './HoverOverlayContents'
import { useLogTelemetryEvent } from './useLogTelemetryEvent'

const LOADING = 'loading' as const

const transformMouseEvent = (handler: (event: MouseEvent) => void) => (event: React.MouseEvent<HTMLElement>) =>
    handler(toNativeEvent(event))

export type { HoverContext }

export interface HoverOverlayClassProps {
    /** An optional class name to apply to the outermost element of the HoverOverlay */
    className?: string
    closeButtonClassName?: string

    iconClassName?: string
    badgeClassName?: string

    actionItemClassName?: string
    actionItemPressedClassName?: string

    getAlertClassName?: GetAlertClassName
}

export interface HoverOverlayProps
    extends HoverOverlayBaseProps,
        ActionItemComponentProps,
        HoverOverlayClassProps,
        TelemetryProps,
        ThemeProps,
        Pick<HoverOverlayAlertsProps, 'onAlertDismissed'>,
        PlatformContextProps<'forceUpdateTooltip' | 'settings'> {
    /** A ref callback to get the root overlay element. Use this to calculate the position. */
    hoverRef?: React.Ref<HTMLDivElement>
    /** Called when the close button is clicked */
    onCloseButtonClick?: (event: MouseEvent) => void
}

const getOverlayStyle = (overlayPosition: HoverOverlayProps['overlayPosition']): CSSProperties =>
    overlayPosition
        ? {
              opacity: 1,
              visibility: 'visible',
              left: `${overlayPosition.left}px`,
              top: `${overlayPosition.top}px`,
          }
        : {
              opacity: 0,
              visibility: 'hidden',
          }

export const HoverOverlay: React.FunctionComponent<HoverOverlayProps> = props => {
    const {
        hoverOrError,
        hoverRef,
        overlayPosition,
        actionsOrError,
        platformContext,
        telemetryService,
        extensionsController,
        showCloseButton,
        location,

        className,
        closeButtonClassName,
        iconClassName,
        badgeClassName,
        actionItemClassName,
        actionItemPressedClassName,

        getAlertClassName,
        onAlertDismissed,
        onCloseButtonClick,
    } = props

    useLogTelemetryEvent(props)

    if (!hoverOrError && (!actionsOrError || isErrorLike(actionsOrError))) {
        return null
    }

    return (
        <div
            // needed for dynamic styling
            // eslint-disable-next-line react/forbid-dom-props
            style={getOverlayStyle(overlayPosition)}
            className={classNames('hover-overlay', className)}
            ref={hoverRef}
        >
            <div
                className={classNames(
                    'hover-overlay__contents',
                    hoverOrError === LOADING && 'hover-overlay__contents--loading',
                    showCloseButton && 'hover-overlay__contents--with-close-button'
                )}
            >
                {showCloseButton && (
                    <button
                        type="button"
                        onClick={onCloseButtonClick ? transformMouseEvent(onCloseButtonClick) : undefined}
                        className={classNames(
                            'hover-overlay__close-button',
                            closeButtonClassName,
                            hoverOrError === LOADING && 'hover-overlay__close-button--loading'
                        )}
                    >
                        <CloseIcon className={iconClassName} />
                    </button>
                )}
                <HoverOverlayContents
                    hoverOrError={hoverOrError}
                    iconClassName={iconClassName}
                    badgeClassName={badgeClassName}
                    errorAlertClassName={getAlertClassName?.(NotificationType.Error)}
                />
            </div>
            {hoverOrError &&
                hoverOrError !== LOADING &&
                !isErrorLike(hoverOrError) &&
                hoverOrError.alerts &&
                hoverOrError.alerts.length > 0 && (
                    <HoverOverlayAlerts
                        hoverAlerts={hoverOrError.alerts}
                        iconClassName={iconClassName}
                        getAlertClassName={getAlertClassName}
                        onAlertDismissed={onAlertDismissed}
                    />
                )}
            {actionsOrError !== undefined &&
                actionsOrError !== null &&
                actionsOrError !== LOADING &&
                !isErrorLike(actionsOrError) &&
                actionsOrError.length > 0 && (
                    <div className="hover-overlay__actions">
                        {actionsOrError.map((action, index) => (
                            <ActionItem
                                key={index}
                                {...action}
                                className={classNames(
                                    'hover-overlay__action',
                                    actionItemClassName,
                                    `test-tooltip-${sanitizeClass(action.action.title || 'untitled')}`
                                )}
                                iconClassName={iconClassName}
                                pressedClassName={actionItemPressedClassName}
                                variant="actionItem"
                                disabledDuringExecution={true}
                                showLoadingSpinnerDuringExecution={true}
                                showInlineError={true}
                                platformContext={platformContext}
                                telemetryService={telemetryService}
                                extensionsController={extensionsController}
                                location={location}
                            />
                        ))}
                    </div>
                )}
        </div>
    )
}
