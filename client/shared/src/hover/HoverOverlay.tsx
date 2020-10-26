import { HoverOverlayProps as GenericHoverOverlayProps } from '@sourcegraph/codeintellify'
import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import classNames from 'classnames'
import { isEqual, upperFirst } from 'lodash'
import CloseIcon from 'mdi-react/CloseIcon'
import * as React from 'react'
import { ActionItem, ActionItemAction, ActionItemComponentProps } from '../actions/ActionItem'
import { HoverMerged } from '../api/client/types/hover'
import { TelemetryProps } from '../telemetry/telemetryService'
import { isErrorLike, asError } from '../util/errors'
import { renderMarkdown } from '../util/markdown'
import { sanitizeClass } from '../util/strings'
import { FileSpec, RepoSpec, ResolvedRevisionSpec, RevisionSpec } from '../util/url'
import { toNativeEvent } from './helpers'
import { BadgeAttachment } from '../components/BadgeAttachment'
import { ThemeProps } from '../theme'
import { PlatformContextProps } from '../platform/context'
import { Subscription } from 'rxjs'

const LOADING = 'loading' as const

const transformMouseEvent = (handler: (event: MouseEvent) => void) => (event: React.MouseEvent<HTMLElement>) =>
    handler(toNativeEvent(event))

export type HoverContext = RepoSpec & RevisionSpec & FileSpec & ResolvedRevisionSpec

export interface HoverOverlayClassProps {
    /** An optional class name to apply to the outermost element of the HoverOverlay */
    className?: string
    iconButtonClassName?: string

    iconClassName?: string

    actionItemClassName?: string
    actionItemPressedClassName?: string

    infoAlertClassName?: string
    errorAlertClassName?: string
}

export interface HoverOverlayProps
    extends GenericHoverOverlayProps<HoverContext, HoverMerged, ActionItemAction>,
        ActionItemComponentProps,
        HoverOverlayClassProps,
        TelemetryProps,
        ThemeProps,
        PlatformContextProps<'forceUpdateTooltip' | 'settings'> {
    /** A ref callback to get the root overlay element. Use this to calculate the position. */
    hoverRef?: React.Ref<HTMLDivElement>

    /** Called when the close button is clicked */
    onCloseButtonClick?: (event: MouseEvent) => void
    /** Called when an alert is dismissed, with the type of the dismissed alert. */
    onAlertDismissed?: (alertType: string) => void
}

interface HoverOverlayState {
    showBadges: boolean
}

const isEmptyHover = ({
    hoveredToken,
    hoverOrError,
    actionsOrError,
}: Pick<HoverOverlayProps, 'hoveredToken' | 'hoverOrError' | 'actionsOrError'>): boolean =>
    !hoveredToken ||
    ((!hoverOrError || hoverOrError === LOADING || isErrorLike(hoverOrError)) &&
        (!actionsOrError || actionsOrError === LOADING || isErrorLike(actionsOrError)))

export class HoverOverlay extends React.PureComponent<HoverOverlayProps, HoverOverlayState> {
    private subscription = new Subscription()

    constructor(props: HoverOverlayProps) {
        super(props)
        this.state = {
            showBadges: false,
        }
    }

    public componentDidMount(): void {
        this.logTelemetryEvent()

        this.subscription.add(
            this.props.platformContext.settings.subscribe(settingsCascadeOrError => {
                if (settingsCascadeOrError.final && !isErrorLike(settingsCascadeOrError.final)) {
                    // Default to true if experimentalFeatures or showBadgeAttachments are not set
                    this.setState({
                        showBadges:
                            !settingsCascadeOrError.final.experimentalFeatures ||
                            settingsCascadeOrError.final.experimentalFeatures.showBadgeAttachments !== false,
                    })
                } else {
                    this.setState({ showBadges: false })
                }
            })
        )
    }

    public componentDidUpdate(previousProps: HoverOverlayProps): void {
        // Log a telemetry event for this hover being displayed, but only do it once per position and when it is
        // non-empty.
        if (
            !isEmptyHover(this.props) &&
            (!isEqual(this.props.hoveredToken, previousProps.hoveredToken) || isEmptyHover(previousProps))
        ) {
            this.logTelemetryEvent()
        }
    }

    public componentWillUnmount(): void {
        this.subscription.unsubscribe()
    }

    public render(): JSX.Element | null {
        const {
            hoverOrError,
            hoverRef,
            onCloseButtonClick,
            overlayPosition,
            showCloseButton,
            actionsOrError,
            className = '',
            actionItemClassName,
            actionItemPressedClassName,
        } = this.props

        if (!hoverOrError && (!actionsOrError || isErrorLike(actionsOrError))) {
            return null
        }
        return (
            <div
                // needed for dynamic styling
                // eslint-disable-next-line react/forbid-dom-props
                style={
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
                }
                className={classNames('hover-overlay', className)}
                ref={hoverRef}
            >
                <div className={classNames('hover-overlay__contents')}>
                    {showCloseButton && (
                        <button
                            type="button"
                            className={classNames(
                                'hover-overlay__close-button',
                                this.props.iconButtonClassName,
                                hoverOrError === LOADING && 'hover-overlay__close-button--loading'
                            )}
                            onClick={onCloseButtonClick ? transformMouseEvent(onCloseButtonClick) : undefined}
                        >
                            <CloseIcon className={this.props.iconClassName} />
                        </button>
                    )}
                    {hoverOrError === LOADING ? (
                        <div className="hover-overlay__loader-row">
                            <LoadingSpinner className={this.props.iconClassName} />
                        </div>
                    ) : isErrorLike(hoverOrError) ? (
                        <div className={classNames('hover-overlay__hover-error', this.props.errorAlertClassName)}>
                            {upperFirst(hoverOrError.message)}
                        </div>
                    ) : hoverOrError === null || (!hoverOrError?.contents.length && hoverOrError?.alerts?.length) ? (
                        // Show some content to give the close button space
                        // and communicate to the user we couldn't find a hover.
                        <em>No hover information available.</em>
                    ) : (
                        hoverOrError?.contents.map((content, index) => {
                            if (content.kind === 'markdown') {
                                try {
                                    return (
                                        <React.Fragment key={index}>
                                            {index !== 0 && <hr />}

                                            {content.badge && this.state.showBadges && (
                                                <BadgeAttachment
                                                    className="hover-overlay__badge test-hover-badge"
                                                    iconClassName={this.props.iconClassName}
                                                    iconButtonClassName={this.props.iconButtonClassName}
                                                    attachment={content.badge}
                                                    isLightTheme={this.props.isLightTheme}
                                                />
                                            )}

                                            <span
                                                className="hover-overlay__content test-tooltip-content"
                                                dangerouslySetInnerHTML={{
                                                    __html: renderMarkdown(content.value),
                                                }}
                                            />
                                        </React.Fragment>
                                    )
                                } catch (error) {
                                    return (
                                        <div className={classNames(this.props.errorAlertClassName)} key={index}>
                                            {upperFirst(asError(error).message)}
                                        </div>
                                    )
                                }
                            }
                            return (
                                <span className="hover-overlay__content" key={index}>
                                    {content.value}
                                </span>
                            )
                        })
                    )}
                </div>
                {hoverOrError && hoverOrError !== LOADING && !isErrorLike(hoverOrError) && hoverOrError.alerts && (
                    <div className="hover-overlay__alerts">
                        {hoverOrError.alerts.map(({ summary, badge, type }, index) => (
                            <div
                                className={classNames('hover-overlay__alert', this.props.infoAlertClassName)}
                                key={index}
                            >
                                {/* Show badge in the top-right if there is no "Dismiss" action */}
                                {badge && !type && (
                                    <BadgeAttachment
                                        className="hover-overlay__badge test-hover-badge"
                                        iconClassName={this.props.iconClassName}
                                        iconButtonClassName={this.props.iconButtonClassName}
                                        attachment={badge}
                                        isLightTheme={this.props.isLightTheme}
                                    />
                                )}

                                {summary.kind === 'plaintext' ? (
                                    <span className="hover-overlay__content">{summary.value}</span>
                                ) : (
                                    <span
                                        className="hover-overlay__content"
                                        dangerouslySetInnerHTML={{ __html: renderMarkdown(summary.value) }}
                                    />
                                )}

                                {/* Show badge and "Dismiss" in the bottom-right if there is a dismiss button */}
                                {type && (
                                    <div className="hover-overlay__alert-actions">
                                        {badge && (
                                            <BadgeAttachment
                                                className="hover-overlay__badge test-hover-badge"
                                                iconClassName={this.props.iconClassName}
                                                iconButtonClassName={this.props.iconButtonClassName}
                                                attachment={badge}
                                                isLightTheme={this.props.isLightTheme}
                                            />
                                        )}
                                        <a href="" onClick={this.onAlertDismissedCallback(type)}>
                                            <small>Dismiss</small>
                                        </a>
                                    </div>
                                )}
                            </div>
                        ))}
                    </div>
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
                                    iconClassName={this.props.iconClassName}
                                    pressedClassName={actionItemPressedClassName}
                                    variant="actionItem"
                                    disabledDuringExecution={true}
                                    showLoadingSpinnerDuringExecution={true}
                                    showInlineError={true}
                                    platformContext={this.props.platformContext}
                                    telemetryService={this.props.telemetryService}
                                    extensionsController={this.props.extensionsController}
                                    location={this.props.location}
                                />
                            ))}
                        </div>
                    )}
            </div>
        )
    }

    private onAlertDismissedCallback(alertType: string): (event: React.MouseEvent<HTMLAnchorElement>) => void {
        return event => {
            event.preventDefault()
            if (this.props.onAlertDismissed) {
                this.props.onAlertDismissed(alertType)
            }
        }
    }

    private logTelemetryEvent(): void {
        this.props.telemetryService.log('hover')
    }
}
