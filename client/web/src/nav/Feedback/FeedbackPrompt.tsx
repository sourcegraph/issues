import MessageDrawIcon from 'mdi-react/MessageDrawIcon'
import TickIcon from 'mdi-react/TickIcon'
import React, { useCallback, useEffect, useState } from 'react'
import TextAreaAutosize from 'react-textarea-autosize'
import { ButtonDropdown, DropdownMenu, DropdownToggle } from 'reactstrap'
import CloseIcon from 'mdi-react/CloseIcon'
import * as H from 'history'
import { Link } from '../../../../shared/src/components/Link'
import { gql } from '../../../../shared/src/graphql/graphql'
import { LoaderButton } from '../../components/LoaderButton'
import { SubmitHappinessFeedbackResult, SubmitHappinessFeedbackVariables } from '../../graphql-operations'
import { useLocalStorage } from '../../util/useLocalStorage'
import { useMutation } from '../../hooks/useMutation'
import { IconRadioButtons } from '../IconRadioButtons'
import { Happy, Sad, VeryHappy, VerySad } from './FeedbackIcons'
import { Form } from '../../../../branded/src/components/Form'
import { ErrorAlert } from '../../components/alerts'

export const HAPPINESS_FEEDBACK_OPTIONS = [
    {
        name: 'Very sad',
        value: 1,
        icon: VerySad,
    },
    {
        name: 'Sad',
        value: 2,
        icon: Sad,
    },
    {
        name: 'Happy',
        value: 3,
        icon: Happy,
    },
    {
        name: 'Very Happy',
        value: 4,
        icon: VeryHappy,
    },
]

const SUBMIT_HAPPINESS_FEEDBACK_QUERY = gql`
    mutation SubmitHappinessFeedback($input: HappinessFeedbackSubmissionInput!) {
        submitHappinessFeedback(input: $input) {
            alwaysNil
        }
    }
`

interface ContentProps {
    closePrompt: () => void
    history: H.History
}

const LOCAL_STORAGE_KEY_RATING = 'feedbackPromptRating'
const LOCAL_STORAGE_KEY_TEXT = 'feedbackPromptText'

const FeedbackPromptContent: React.FunctionComponent<ContentProps> = ({ closePrompt, history }) => {
    const [rating, setRating] = useLocalStorage<number | undefined>(LOCAL_STORAGE_KEY_RATING, undefined)
    const [text, setText] = useLocalStorage<string>(LOCAL_STORAGE_KEY_TEXT, '')
    const handleRateChange = useCallback((value: number) => setRating(value), [setRating])
    const handleTextChange = useCallback(
        (event: React.ChangeEvent<HTMLTextAreaElement>) => setText(event.target.value),
        [setText]
    )
    const [submitFeedback, response] = useMutation<SubmitHappinessFeedbackResult, SubmitHappinessFeedbackVariables>(
        SUBMIT_HAPPINESS_FEEDBACK_QUERY
    )

    const handleSubmit = useCallback(
        (event: React.FormEvent<HTMLFormElement>): void => {
            event.preventDefault()
            if (rating) {
                return submitFeedback({
                    input: { score: rating, feedback: text, currentURL: window.location.href },
                })
            }
        },
        [rating, submitFeedback, text]
    )

    useEffect(() => {
        if ('data' in response && response.data.submitHappinessFeedback) {
            // Reset local storage when successfully submitted
            localStorage.removeItem(LOCAL_STORAGE_KEY_TEXT)
            localStorage.removeItem(LOCAL_STORAGE_KEY_RATING)
        }
    }, [response])

    return (
        <>
            {'data' in response ? (
                <div className="feedback-prompt__success">
                    <TickIcon className="feedback-prompt__success--tick" />
                    <h3>We‘ve received your feedback!</h3>
                    <p className="d-inline">
                        Thank you for your help.
                        {window.context.productResearchPageEnabled && (
                            <>
                                {' '}
                                Want to help keep making Sourcegraph better?{' '}
                                <Link to="/user/settings/product-research" onClick={closePrompt}>
                                    Join us for occasional user research
                                </Link>{' '}
                                and share your feedback on our latest features and ideas.
                            </>
                        )}
                    </p>
                </div>
            ) : (
                <Form onSubmit={handleSubmit}>
                    <header className="feedback-prompt__header">
                        <h3 className="mb-0">What’s on your mind?</h3>
                        <button type="button" className="feedback-prompt__header--close" onClick={closePrompt}>
                            <CloseIcon className="feedback-prompt__header-icon" />
                        </button>
                    </header>

                    <TextAreaAutosize
                        onChange={handleTextChange}
                        value={text}
                        minRows={3}
                        maxRows={6}
                        placeholder="What’s going well? What could be better?"
                        className="form-control feedback-prompt__textarea"
                        autoFocus={true}
                    />

                    <IconRadioButtons
                        name="emoji-selector"
                        icons={HAPPINESS_FEEDBACK_OPTIONS}
                        selected={rating}
                        onChange={handleRateChange}
                        disabled={response.loading}
                    />
                    {'error' in response && (
                        <ErrorAlert
                            error={response.error}
                            history={history}
                            icon={false}
                            className="mt-3"
                            prefix="Error submitting feedback"
                        />
                    )}
                    <LoaderButton
                        role="menuitem"
                        type="submit"
                        className="btn btn-block btn-secondary feedback-prompt__button"
                        loading={response.loading}
                        label="Send"
                        disabled={!rating || response.loading}
                    />
                </Form>
            )}
        </>
    )
}

interface Props {
    open?: boolean
    history: H.History
}

export const FeedbackPrompt: React.FunctionComponent<Props> = ({ open, history }) => {
    const [isOpen, setIsOpen] = useState(() => !!open)
    const handleToggle = useCallback(() => setIsOpen(open => !open), [])

    const forceClose = useCallback(() => setIsOpen(false), [])

    return (
        <ButtonDropdown a11y={false} isOpen={isOpen} toggle={handleToggle} className="feedback-prompt" group={false}>
            <DropdownToggle
                tag="button"
                caret={false}
                className="btn btn-link btn-sm text-decoration-none feedback-prompt__toggle"
                nav={true}
                aria-label="Feedback"
            >
                <MessageDrawIcon className="d-lg-none icon-inline" />
                <span className="d-none d-lg-block">Feedback</span>
            </DropdownToggle>
            <DropdownMenu right={true} className="web-content feedback-prompt__menu">
                <FeedbackPromptContent closePrompt={forceClose} history={history} />
            </DropdownMenu>
        </ButtonDropdown>
    )
}
