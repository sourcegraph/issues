import * as React from 'react'

interface FormProps extends React.DetailedHTMLProps<React.FormHTMLAttributes<HTMLFormElement>, HTMLFormElement> {
    children: React.ReactNode
}

interface FormState {
    wasValidated: boolean
}

/**
 * Form component that handles validation.
 * If the user tries to submit the form and one of the inputs is invalid,
 * Bootstrap's `was-validated` class will be assigned so the invalid inputs get highlighted.
 */
export class Form extends React.PureComponent<FormProps, FormState> {
    constructor(props: FormProps) {
        super(props)
        this.state = { wasValidated: false }
    }

    public render(): React.ReactNode {
        return (
            // eslint-disable-next-line react/forbid-elements
            <form
                {...this.props}
                className={`${this.props.className || ''} ${this.state.wasValidated ? 'was-validated' : ''}`}
                onInvalid={this.onInvalid}
            >
                {this.props.children}
            </form>
        )
    }

    private onInvalid: React.EventHandler<React.InvalidEvent<HTMLFormElement>> = event => {
        this.setState({ wasValidated: true })
        if (this.props.onInvalid) {
            this.props.onInvalid(event)
        }
    }
}
