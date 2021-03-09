import classNames from 'classnames'
import React, { FunctionComponent, useMemo } from 'react'
import { LoaderInput } from '../../../../../branded/src/components/LoaderInput'
import { deriveInputClassName, useInputValidation } from '../../../../../shared/src/util/useInputValidation'
import { ErrorAlert } from '../../../components/alerts'
import { LoaderButton } from '../../../components/LoaderButton'
import { useAddUserEmail } from './useUserEmail'

interface Props {
    user: string
    className?: string
}

export const AddUserEmailForm: FunctionComponent<Props> = ({ user, className }) => {
    const { mutate, isLoading, error } = useAddUserEmail()
    const [emailState, nextEmailFieldChange, emailInputReference] = useInputValidation(
        useMemo(
            () => ({
                synchronousValidators: [],
                asynchronousValidators: [],
            }),
            []
        )
    )

    const onSubmit: React.FormEventHandler<HTMLFormElement> = event => {
        event.preventDefault()
        mutate({ user, email: emailState.value })
    }

    return (
        <div className={`add-user-email-form ${className || ''}`}>
            <label
                htmlFor="AddUserEmailForm-email"
                className={classNames('align-self-start', {
                    'text-danger font-weight-bold': emailState.kind === 'INVALID',
                })}
            >
                Email address
            </label>
            {/* eslint-disable-next-line react/forbid-elements */}
            <form className="form-inline" onSubmit={onSubmit} noValidate={true}>
                <LoaderInput className={classNames(deriveInputClassName(emailState), 'mr-sm-2')} loading={isLoading}>
                    <input
                        id="AddUserEmailForm-email"
                        type="email"
                        name="email"
                        className={classNames(
                            'form-control test-user-email-add-input',
                            deriveInputClassName(emailState)
                        )}
                        onChange={nextEmailFieldChange}
                        size={32}
                        value={emailState.value}
                        ref={emailInputReference}
                        required={true}
                        autoComplete="email"
                        autoCorrect="off"
                        autoCapitalize="off"
                        spellCheck={false}
                        readOnly={false}
                    />
                </LoaderInput>
                <LoaderButton
                    loading={isLoading}
                    label="Add"
                    type="submit"
                    disabled={isLoading || emailState.kind !== 'VALID'}
                    className="btn btn-primary"
                />
                {emailState.kind === 'INVALID' && (
                    <small className="invalid-feedback" role="alert">
                        {emailState.reason}
                    </small>
                )}
            </form>
            {error && <ErrorAlert className="mt-2" error={error} />}
        </div>
    )
}
