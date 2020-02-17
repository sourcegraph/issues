/* eslint-disable react/jsx-no-target-blank */
import * as React from 'react'
import * as GQL from '../../../shared/src/graphql/schema'
import { HeroPage } from '../components/HeroPage'
import { PageTitle } from '../components/PageTitle'
import { eventLogger } from '../tracking/eventLogger'
import { ThemeProps } from '../../../shared/src/theme'
import HelpCircleIcon from 'mdi-react/HelpCircleIcon'
import { Link } from '../../../shared/src/components/Link'
import { showDotComMarketing } from '../util/features'

interface Props extends ThemeProps {
    authenticatedUser: GQL.IUser | null
}

export class HelpPage extends React.Component<Props> {
    public componentDidMount(): void {
        eventLogger.logViewEvent('Help')
    }

    public render(): JSX.Element | null {
        const emailQueryString = this.props.authenticatedUser ? `?email=${this.props.authenticatedUser.email}` : ''

        return (
            <div className="help-page">
                <PageTitle title="Help" />
                <HeroPage
                    icon={HelpCircleIcon}
                    title="Looking for answers?"
                    body={
                        <div className="help-content">
                            <div>
                                We have many different resources available to help answer any questions you may have:
                            </div>
                            {!showDotComMarketing && (
                                <div>
                                    <h3>Email support</h3>
                                    Have a question about your custom setup? Need help scaling your instance? File a
                                    support ticket by emailing us at{' '}
                                    <a href="mailto:support@sourcegraph.com">support@sourcegraph.com</a>
                                </div>
                            )}
                            <div>
                                <h3>File a GitHub issue</h3>
                                We are an{' '}
                                <a
                                    href="https://about.sourcegraph.com/company/open_source_open_company"
                                    target="_blank"
                                >
                                    open company
                                </a>
                                , so our{' '}
                                <a href="https://github.com/sourcegraph/sourcegraph/issues" target="_blank">
                                    issue tracker on GitHub
                                </a>{' '}
                                is available to the public. Look to see if there is already an issue filed, or feel free
                                to{' '}
                                <a href="https://github.com/sourcegraph/sourcegraph/issues/new/choose" target="_blank">
                                    file an issue
                                </a>{' '}
                                with feature requests, bug reports, or product enhancements!
                            </div>
                            <div>
                                <h3>Search our documentation</h3>
                                Our <Link to="/documentation">documentation</Link> has information that can help you
                                with any questions you have about using the product, setup, and configuration. If you
                                notice that something is missing in our docs please let us know or even submit a pull
                                request with an update!
                            </div>
                            <div>
                                <h3>Submit product feedback</h3>
                                <p>
                                    We love hearing from our users! Send us your product feedback, ideas, and feature
                                    requests. This form is especially useful for anyone wishing to stay anonymous to the
                                    public with any requests.
                                </p>
                                <a
                                    href={`https://share.hsforms.com/1kZm78wx9QiGpjRRrHEcapA1n7ku?site_id=${window.context.siteID}${emailQueryString}`}
                                    className="btn btn-secondary"
                                    target="_blank"
                                >
                                    Share feedback
                                </a>
                            </div>
                        </div>
                    }
                />
            </div>
        )
    }
}
