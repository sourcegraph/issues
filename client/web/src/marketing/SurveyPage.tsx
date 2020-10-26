import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import * as H from 'history'
import * as React from 'react'
import { RouteComponentProps } from 'react-router'
import { catchError } from 'rxjs/operators'
import { FeedbackText } from '../components/FeedbackText'
import { Form } from '../../../branded/src/components/Form'
import { HeroPage } from '../components/HeroPage'
import { PageTitle } from '../components/PageTitle'
import { eventLogger } from '../tracking/eventLogger'
import { submitSurvey } from './backend'
import { SurveyCTA } from './SurveyToast'
import { Subscription } from 'rxjs'
import { ThemeProps } from '../../../shared/src/theme'
import TwitterIcon from 'mdi-react/TwitterIcon'
import { AuthenticatedUser } from '../auth'

interface SurveyFormProps {
    location: H.Location
    history: H.History
    authenticatedUser: AuthenticatedUser | null
    score?: number
    onScoreChange?: (score: number) => void
    onSubmit?: () => void
}

interface SurveyFormState {
    error?: Error
    reason: string
    betterProduct: string
    email: string
    loading: boolean
}

export interface SurveyResponse {
    score: number
    email?: string
    reason?: string
    better?: string
}

class SurveyForm extends React.Component<SurveyFormProps, SurveyFormState> {
    private subscriptions = new Subscription()

    constructor(props: SurveyFormProps) {
        super(props)
        this.state = {
            reason: '',
            betterProduct: '',
            email: '',
            loading: false,
        }
    }

    public render(): JSX.Element | null {
        return (
            <Form className="survey-form" onSubmit={this.handleSubmit}>
                {this.state.error && <p className="survey-form__error">{this.state.error.message}</p>}
                <label className="survey-form__label">
                    How likely is it that you would recommend Sourcegraph to a friend?
                </label>
                <SurveyCTA className="survey-form__scores" onClick={this.onScoreChange} score={this.props.score} />
                {!this.props.authenticatedUser && (
                    <div className="form-group">
                        <input
                            className="form-control survey-form__input"
                            type="text"
                            placeholder="Email"
                            onChange={this.onEmailFieldChange}
                            value={this.state.email}
                            disabled={this.state.loading}
                        />
                    </div>
                )}
                <div className="form-group">
                    <label className="survey-form__label">
                        What is the most important reason for the score you gave Sourcegraph?
                    </label>
                    <textarea
                        className="form-control survey-form__input"
                        onChange={this.onReasonFieldChange}
                        value={this.state.reason}
                        disabled={this.state.loading}
                        autoFocus={true}
                    />
                </div>
                <div className="form-group">
                    <label className="survey-form__label">What could Sourcegraph do to provide a better product?</label>
                    <textarea
                        className="form-control survey-form__input"
                        onChange={this.onBetterProductFieldChange}
                        value={this.state.betterProduct}
                        disabled={this.state.loading}
                    />
                </div>
                <div className="form-group">
                    <button className="btn btn-primary btn-block" type="submit" disabled={this.state.loading}>
                        Submit
                    </button>
                </div>
                {this.state.loading && (
                    <div className="survey-form__loader">
                        <LoadingSpinner className="icon-inline" />
                    </div>
                )}
                <div>
                    <small>
                        Your response to this survey will be sent to Sourcegraph, and will be visible to your
                        Sourcegraph site admins.
                    </small>
                </div>
            </Form>
        )
    }

    private onScoreChange = (score: number): void => {
        if (this.props.onScoreChange) {
            this.props.onScoreChange(score)
        }
        this.setState({ error: undefined })
    }

    private onEmailFieldChange = (event: React.ChangeEvent<HTMLInputElement>): void => {
        this.setState({ email: event.target.value })
    }

    private onReasonFieldChange = (event: React.ChangeEvent<HTMLTextAreaElement>): void => {
        this.setState({ reason: event.target.value })
    }

    private onBetterProductFieldChange = (event: React.ChangeEvent<HTMLTextAreaElement>): void => {
        this.setState({ betterProduct: event.target.value })
    }

    private handleSubmit = (event: React.FormEvent<HTMLFormElement>): void => {
        event.preventDefault()
        if (this.state.loading) {
            return
        }

        if (this.props.score === undefined) {
            this.setState({ error: new Error('Please select a score') })
            return
        }

        eventLogger.log('SurveySubmitted')
        this.setState({ loading: true })

        this.subscriptions.add(
            submitSurvey({
                score: this.props.score,
                email: this.state.email,
                reason: this.state.reason,
                better: this.state.betterProduct,
            })
                .pipe(
                    catchError(error => {
                        this.setState({ error })
                        return []
                    })
                )
                .subscribe(() => {
                    if (this.props.onSubmit) {
                        this.props.onSubmit()
                    }
                    this.props.history.push({
                        pathname: '/survey/thanks',
                        state: {
                            score: this.props.score,
                            feedback: this.state.reason,
                        },
                    })
                })
        )
    }
}

interface SurveyPageProps extends RouteComponentProps<{ score?: string }>, ThemeProps {
    authenticatedUser: AuthenticatedUser | null
}

export interface TweetFeedbackProps {
    score: number
    feedback: string
}

const SCORE_TO_TWEET = 9
const TweetFeedback: React.FunctionComponent<TweetFeedbackProps> = ({ feedback, score }) => {
    if (score >= SCORE_TO_TWEET) {
        const url = new URL('https://twitter.com/intent/tweet')
        url.searchParams.set('text', `After using @srcgraph: ${feedback}`)
        return (
            <>
                <p className="mt-2">
                    One more favor, could you share your feedback on Twitter? We'd really appreciate it!
                </p>
                <a
                    className="d-inline-block mt-2 btn btn-primary"
                    href={url.href}
                    target="_blank"
                    rel="noreferrer noopener"
                >
                    <TwitterIcon className="icon-inline mr-2" />
                    Tweet feedback
                </a>
            </>
        )
    }
    return null
}

export class SurveyPage extends React.Component<SurveyPageProps> {
    public componentDidMount(): void {
        eventLogger.logViewEvent('Survey')
    }

    public render(): JSX.Element | null {
        if (this.props.match.params.score === 'thanks') {
            return (
                <div className="survey-page">
                    <PageTitle title="Thanks" />
                    <HeroPage
                        title="Thanks for the feedback!"
                        body={
                            <TweetFeedback
                                score={this.props.location.state.score}
                                feedback={this.props.location.state.feedback}
                            />
                        }
                        cta={<FeedbackText headerText="Anything else?" />}
                    />
                </div>
            )
        }
        return (
            <div className="survey-page">
                <PageTitle title="Almost there..." />
                <HeroPage
                    title="Almost there..."
                    cta={<SurveyForm score={this.intScore(this.props.match.params.score)} {...this.props} />}
                />
            </div>
        )
    }

    private intScore = (score?: string): number | undefined =>
        score ? Math.max(0, Math.min(10, Math.round(+score))) : undefined
}
