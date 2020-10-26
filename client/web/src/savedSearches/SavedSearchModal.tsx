import * as H from 'history'
import * as React from 'react'
import { Form } from '../../../branded/src/components/Form'
import { PatternTypeProps } from '../search'
import { AuthenticatedUser } from '../auth'
import Dialog from '@reach/dialog'

interface Props extends Omit<PatternTypeProps, 'setPatternType'> {
    location: H.Location
    history: H.History
    authenticatedUser: AuthenticatedUser | null
    query?: string
    onDidCancel: () => void
}

enum UserOrOrg {
    User = 'User',
    Org = 'Org',
}

interface State {
    saveLocation: UserOrOrg
    organization?: string
}

export class SavedSearchModal extends React.Component<Props, State> {
    constructor(props: Props) {
        super(props)
        this.state = {
            saveLocation: UserOrOrg.User,
        }
    }

    private onLocationChange = (event: React.ChangeEvent<HTMLSelectElement>): void => {
        const locationType = event.target.value
        this.setState({ saveLocation: locationType as UserOrOrg })
    }

    private onOrganizationChange = (event: React.ChangeEvent<HTMLSelectElement>): void => {
        const orgName = event.target.value
        this.setState({ organization: orgName })
    }

    public render(): JSX.Element | null {
        return (
            this.props.authenticatedUser && (
                <Dialog className="saved-search-modal-form " onDismiss={this.props.onDidCancel}>
                    <Form onSubmit={this.onSubmit} className="test-saved-search-modal">
                        <h3>Save search query to: </h3>
                        <div className="form-group">
                            <select
                                onChange={this.onLocationChange}
                                className="form-control saved-search-modal-form__select"
                            >
                                <option value={UserOrOrg.User}>User</option>
                                {this.props.authenticatedUser.organizations &&
                                    this.props.authenticatedUser.organizations.nodes.length > 0 && (
                                        <option value={UserOrOrg.Org}>Organization</option>
                                    )}
                            </select>
                            {this.props.authenticatedUser.organizations &&
                                this.props.authenticatedUser.organizations.nodes.length > 0 &&
                                this.state.saveLocation === UserOrOrg.Org && (
                                    <select
                                        onChange={this.onOrganizationChange}
                                        placeholder="Select an organization"
                                        className="form-control saved-search-modal-form__select"
                                    >
                                        <option value="" disabled={true} selected={true}>
                                            Select an organization
                                        </option>
                                        {this.props.authenticatedUser.organizations.nodes.map(org => (
                                            <option value={org.name} key={org.name}>
                                                {org.displayName ? org.displayName : org.name}
                                            </option>
                                        ))}
                                    </select>
                                )}
                        </div>
                        <button
                            type="submit"
                            disabled={this.state.saveLocation === UserOrOrg.Org && !this.state.organization}
                            className="btn btn-primary saved-search-modal-form__button test-saved-search-modal-save-button"
                        >
                            Save query
                        </button>
                    </Form>
                </Dialog>
            )
        )
    }

    private onSubmit = (): void => {
        if (this.props.query && this.props.authenticatedUser) {
            const encodedQuery = encodeURIComponent(this.props.query)
            this.props.history.push(
                this.state.saveLocation.toLowerCase() === 'user'
                    ? `/users/${this.props.authenticatedUser.username}/searches/add?query=${encodedQuery}&patternType=${this.props.patternType}`
                    : `/organizations/${this.state.organization!}/searches/add?query=${encodedQuery}&patternType=${
                          this.props.patternType
                      }`
            )
        }
    }
}
