import EmoticonIcon from 'mdi-react/EmoticonIcon'
import * as React from 'react'
import { Link } from 'react-router-dom'
import { eventLogger } from '../tracking/eventLogger'
import { Toast } from './Toast'
import { daysActiveCount } from './util'
import { range } from 'lodash'
import { AuthenticatedUser } from '../auth'

const HAS_DISMISSED_TOAST_KEY = 'has-dismissed-survey-toast'

interface SurveyCTAProps {
    className?: string
    score?: number
    onClick?: (score: number) => void
    openSurveyInNewTab?: boolean
}

export class SurveyCTA extends React.PureComponent<SurveyCTAProps> {
    public render(): JSX.Element | null {
        return (
            <div className={this.props.className}>
                {range(0, 11).map(score => {
                    const pressed = score === this.props.score
                    return (
                        /* eslint-disable react/jsx-no-bind */
                        <Link
                            key={score}
                            className={`btn btn-primary toast__rating-btn ${pressed ? 'active' : ''}`}
                            aria-pressed={pressed || undefined}
                            onClick={() => this.onClick(score)}
                            to={`/survey/${score}`}
                            target={this.props.openSurveyInNewTab ? '_blank' : undefined}
                        >
                            {score}
                        </Link>
                        /* eslint-enable react/jsx-no-bind */
                    )
                })}
            </div>
        )
    }

    private onClick = (score: number): void => {
        eventLogger.log('SurveyButtonClicked', { score })
        if (this.props.onClick) {
            this.props.onClick(score)
        }
    }
}

interface Props {
    authenticatedUser: AuthenticatedUser | null
}

interface State {
    visible: boolean
}

export class SurveyToast extends React.Component<Props, State> {
    constructor(props: Props) {
        super(props)
        this.state = {
            visible: localStorage.getItem(HAS_DISMISSED_TOAST_KEY) !== 'true' && daysActiveCount % 30 === 3,
        }
        if (this.state.visible) {
            eventLogger.log('SurveyReminderViewed')
        } else if (daysActiveCount % 30 === 0) {
            // Reset toast dismissal 3 days before it will be shown
            localStorage.setItem(HAS_DISMISSED_TOAST_KEY, 'false')
        }
    }

    public render(): JSX.Element | null {
        if (!this.state.visible) {
            return null
        }

        return (
            <Toast
                icon={<EmoticonIcon className="icon-inline" />}
                title="Tell us what you think"
                subtitle="How likely is it that you would recommend Sourcegraph to a friend?"
                cta={<SurveyCTA onClick={this.onClickScore} openSurveyInNewTab={true} />}
                onDismiss={this.onDismiss}
            />
        )
    }

    private onClickScore = (): void => this.onDismiss()

    private onDismiss = (): void => {
        localStorage.setItem(HAS_DISMISSED_TOAST_KEY, 'true')
        this.setState({ visible: false })
    }
}
