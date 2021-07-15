import { GraphQLError } from 'graphql'
import React from 'react'

import { ErrorMessage } from '../../alerts'

interface ConnectionErrorProps {
    errors: string[] | readonly GraphQLError[]
}

export const ConnectionError: React.FunctionComponent<ConnectionErrorProps> = ({ errors }) => (
    <div className="alert alert-danger filtered-connection__error">
        {errors.map((error, index) => (
            <React.Fragment key={index}>
                <ErrorMessage error={error} />
            </React.Fragment>
        ))}
    </div>
)
