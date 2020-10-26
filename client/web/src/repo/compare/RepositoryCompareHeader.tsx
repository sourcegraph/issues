import DotsHorizontalIcon from 'mdi-react/DotsHorizontalIcon'
import * as React from 'react'
import { Form } from '../../../../branded/src/components/Form'
import { eventLogger } from '../../tracking/eventLogger'
import { RepositoryCompareAreaPageProps } from './RepositoryCompareArea'

interface Props extends RepositoryCompareAreaPageProps {
    className: string

    /** Called when the user updates the comparison spec and submits the form. */
    onUpdateComparisonSpec: (newBaseSpec: string, newHeadSpec: string) => void
}

interface State {
    /** The (possibly unsubmitted) value of the input field containing the comparison base spec. */
    comparisonBaseSpec: string

    /** The (possibly unsubmitted) value of the input field containing the comparison head spec. */
    comparisonHeadSpec: string
}

/**
 * Header for the repository compare area.
 */
export class RepositoryCompareHeader extends React.PureComponent<Props, State> {
    private static BASE_INPUT_ID = 'repository-compare-header__base-spec'
    private static HEAD_INPUT_ID = 'repository-compare-header__head-spec'

    public state: State = {
        comparisonBaseSpec: this.props.base.revision || '',
        comparisonHeadSpec: this.props.head.revision || '',
    }

    public render(): JSX.Element | null {
        // Whether the user has entered new base/head values that differ from what's in the props and has not yet
        // submitted the form.
        const stateDiffers =
            this.state.comparisonBaseSpec !== (this.props.base.revision || '') ||
            this.state.comparisonHeadSpec !== (this.props.head.revision || '')

        const specIsEmpty = this.props.base === null && this.props.head === null

        return (
            <div className={`repository-compare-header ${this.props.className}`}>
                <div className={`${this.props.className}-inner`}>
                    <Form className="form-inline mb-2" onSubmit={this.onSubmit}>
                        <label htmlFor={RepositoryCompareHeader.BASE_INPUT_ID} className="sr-only">
                            Base Git revspec for comparison
                        </label>
                        <input
                            type="text"
                            id={RepositoryCompareHeader.BASE_INPUT_ID}
                            className="form-control mr-2 mb-2"
                            value={this.state.comparisonBaseSpec}
                            onChange={this.onChange}
                            placeholder="HEAD"
                            size={12}
                            autoCapitalize="off"
                            spellCheck={false}
                            autoCorrect="off"
                            autoComplete="off"
                        />
                        <DotsHorizontalIcon className="icon-inline mr-2 mb-2" />
                        <label htmlFor={RepositoryCompareHeader.HEAD_INPUT_ID} className="sr-only">
                            Head Git revspec for comparison
                        </label>
                        <input
                            type="text"
                            id={RepositoryCompareHeader.HEAD_INPUT_ID}
                            className="form-control mr-2 mb-2"
                            value={this.state.comparisonHeadSpec}
                            onChange={this.onChange}
                            placeholder="HEAD"
                            size={12}
                            autoCapitalize="off"
                            spellCheck={false}
                            autoCorrect="off"
                            autoComplete="off"
                        />
                        {(stateDiffers || specIsEmpty) && (
                            <button type="submit" className="btn btn-primary mr-2 mb-2">
                                {stateDiffers ? 'Update comparison' : 'Compare'}
                            </button>
                        )}
                        {stateDiffers && !specIsEmpty && (
                            <button type="reset" className="btn btn-secondary mb-2" onClick={this.onCancel}>
                                Cancel
                            </button>
                        )}
                    </Form>
                </div>
            </div>
        )
    }

    private onChange: React.ChangeEventHandler<HTMLInputElement> = event => {
        this.setState(
            (event.currentTarget.id === RepositoryCompareHeader.BASE_INPUT_ID
                ? { comparisonBaseSpec: event.currentTarget.value }
                : { comparisonHeadSpec: event.currentTarget.value }) as Pick<
                State,
                'comparisonBaseSpec' & 'comparisonHeadSpec'
            >
        )
    }

    private onSubmit: React.FormEventHandler<HTMLFormElement> = event => {
        event.preventDefault()
        eventLogger.log('RepositoryComparisonSubmitted')
        this.props.onUpdateComparisonSpec(this.state.comparisonBaseSpec, this.state.comparisonHeadSpec)
    }

    private onCancel: React.MouseEventHandler<HTMLButtonElement> = event => {
        event.preventDefault()
        eventLogger.log('RepositoryComparisonCanceled')
        this.setState({
            comparisonBaseSpec: this.props.base.revision || '',
            comparisonHeadSpec: this.props.head.revision || '',
        })
    }
}
