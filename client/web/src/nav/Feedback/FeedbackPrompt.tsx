import classNames from 'classnames'
import CloseIcon from 'mdi-react/CloseIcon'
import MessageDrawIcon from 'mdi-react/MessageDrawIcon'
import TickIcon from 'mdi-react/TickIcon'
import React, { useCallback, useEffect, useState } from 'react'
import TextAreaAutosize from 'react-textarea-autosize'
import { ButtonDropdown, DropdownMenu, DropdownToggle } from 'reactstrap'

import { Form } from '@sourcegraph/branded/src/components/Form'
import { Link } from '@sourcegraph/shared/src/components/Link'
import { gql, useMutation } from '@sourcegraph/shared/src/graphql/graphql'
import { useLocalStorage } from '@sourcegraph/shared/src/util/useLocalStorage'
import { useRedesignToggle } from '@sourcegraph/shared/src/util/useRedesignToggle'

import { ErrorAlert } from '../../components/alerts'
import { LoaderButton } from '../../components/LoaderButton'
import { SubmitHappinessFeedbackResult, SubmitHappinessFeedbackVariables } from '../../graphql-operations'
import { useRoutesMatch } from '../../hooks'
import { LayoutRouteProps } from '../../routes'
import { IconRadioButtons } from '../IconRadioButtons'

import { Happy, Sad, VeryHappy, VerySad } from './FeedbackIcons'

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

export const SUBMIT_HAPPINESS_FEEDBACK_QUERY = gql`
    mutation SubmitHappinessFeedback($input: HappinessFeedbackSubmissionInput!) {
        submitHappinessFeedback(input: $input) {
            alwaysNil
        }
    }
`

interface ContentProps {
    closePrompt: () => void
    routeMatch?: string
    /** Text to be prepended to user input on submission. */
    textPrefix?: string
}

const LOCAL_STORAGE_KEY_RATING = 'feedbackPromptRating'
const LOCAL_STORAGE_KEY_TEXT = 'feedbackPromptText'

export const FeedbackPromptContent: React.FunctionComponent<ContentProps> = ({
    closePrompt,
    routeMatch,
    textPrefix = '',
}) => {
    const [rating, setRating] = useLocalStorage<number | undefined>(LOCAL_STORAGE_KEY_RATING, undefined)
    const [text, setText] = useLocalStorage<string>(LOCAL_STORAGE_KEY_TEXT, '')
    const handleRateChange = useCallback((value: number) => setRating(value), [setRating])
    const handleTextChange = useCallback(
        (event: React.ChangeEvent<HTMLTextAreaElement>) => setText(event.target.value),
        [setText]
    )

    const [submitFeedback, { loading, data, error }] = useMutation<
        SubmitHappinessFeedbackResult,
        SubmitHappinessFeedbackVariables
    >(SUBMIT_HAPPINESS_FEEDBACK_QUERY)

    const handleSubmit = useCallback<React.FormEventHandler>(
        event => {
            event.preventDefault()

            if (!rating) {
                return
            }

            return submitFeedback({
                variables: {
                    input: { score: rating, feedback: `${textPrefix}${text}`, currentPath: routeMatch },
                },
            })
        },
        [rating, submitFeedback, text, routeMatch, textPrefix]
    )

    useEffect(() => {
        if (data?.submitHappinessFeedback) {
            // Reset local storage when successfully submitted
            localStorage.removeItem(LOCAL_STORAGE_KEY_TEXT)
            localStorage.removeItem(LOCAL_STORAGE_KEY_RATING)
        }
    }, [data?.submitHappinessFeedback])

    return (
        <>
            <button type="button" className="feedback-prompt__close" onClick={closePrompt}>
                <CloseIcon className="feedback-prompt__icon" />
            </button>
            {data?.submitHappinessFeedback ? (
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
                    <h3 className="mb-0">What’s on your mind?</h3>

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
                        disabled={loading}
                    />
                    {error && (
                        <ErrorAlert error={error} icon={false} className="mt-3" prefix="Error submitting feedback" />
                    )}
                    <LoaderButton
                        role="menuitem"
                        type="submit"
                        className="btn btn-block btn-secondary feedback-prompt__button"
                        loading={loading}
                        label="Send"
                        disabled={!rating || !text || loading}
                    />
                </Form>
            )}
        </>
    )
}

interface Props {
    open?: boolean
    routes: readonly LayoutRouteProps<{}>[]
}

export const FeedbackPrompt: React.FunctionComponent<Props> = ({ open, routes }) => {
    const [isOpen, setIsOpen] = useState(() => !!open)
    const handleToggle = useCallback(() => setIsOpen(open => !open), [])
    const forceClose = useCallback(() => setIsOpen(false), [])
    const match = useRoutesMatch(routes)
    const [isRedesignEnabled] = useRedesignToggle()

    return (
        <ButtonDropdown a11y={false} isOpen={isOpen} toggle={handleToggle} className="feedback-prompt" group={false}>
            <DropdownToggle
                tag="button"
                caret={false}
                className={classNames('btn btn-sm text-decoration-none feedback-prompt__toggle', {
                    'btn-outline-secondary': isRedesignEnabled,
                    'btn-link': !isRedesignEnabled,
                })}
                aria-label="Feedback"
            >
                {!isRedesignEnabled && <MessageDrawIcon className="d-lg-none icon-inline" />}
                <span className={classNames({ 'd-none d-lg-block': !isRedesignEnabled })}>Feedback</span>
            </DropdownToggle>
            <DropdownMenu right={true} className="feedback-prompt__menu">
                <FeedbackPromptContent closePrompt={forceClose} routeMatch={match} />
            </DropdownMenu>
        </ButtonDropdown>
    )
}
