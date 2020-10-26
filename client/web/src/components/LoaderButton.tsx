import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import React from 'react'

interface Props extends React.DetailedHTMLProps<React.ButtonHTMLAttributes<HTMLButtonElement>, HTMLButtonElement> {
    loading: boolean
    label: string
}

export const LoaderButton: React.FunctionComponent<Props> = ({ loading, label, ...props }) => (
    // eslint-disable-next-line react/button-has-type
    <button {...props} type={props.type ?? 'button'}>
        {loading ? <LoadingSpinner className="icon-inline" /> : label}
    </button>
)
