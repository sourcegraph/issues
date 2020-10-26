import CheckmarkCircleIcon from 'mdi-react/CheckCircleIcon'
import * as React from 'react'
import * as GQL from '../../../../shared/src/graphql/schema'
import { AccessTokenScopes } from '../../auth/accessToken'
import { CopyableText } from '../../components/CopyableText'

interface AccessTokenCreatedAlertProps {
    className: string
    tokenSecret: string
    token: GQL.IAccessToken
}

/**
 * Displays a message informing the user to copy and save the newly created access token.
 */
export class AccessTokenCreatedAlert extends React.PureComponent<AccessTokenCreatedAlertProps> {
    public render(): JSX.Element | null {
        const isSudoToken = this.props.token.scopes.includes(AccessTokenScopes.SiteAdminSudo)
        return (
            <div className={`access-token-created-alert ${this.props.className}`}>
                <p>
                    <CheckmarkCircleIcon className="icon-inline" /> Copy the new access token now. You won't be able to
                    see it again.
                </p>
                <CopyableText className="test-access-token" text={this.props.tokenSecret} size={48} />
                <h5 className="mt-4">
                    <strong>Example usage</strong>
                </h5>
                <pre className="my-1">
                    <code>{curlExampleCommand(this.props.tokenSecret, isSudoToken)}</code>
                </pre>
            </div>
        )
    }
}

function curlExampleCommand(tokenSecret: string, isSudoToken: boolean): string {
    const credentials = isSudoToken
        ? `token-sudo user="SUDO-TO-USERNAME",token="${tokenSecret}"`
        : `token ${tokenSecret}`

    return `curl \\
  -H 'Authorization: ${credentials}' \\
  -d '{"query":"query { currentUser { username } }"}' \\
  ${window.context.externalURL}/.api/graphql`
}
