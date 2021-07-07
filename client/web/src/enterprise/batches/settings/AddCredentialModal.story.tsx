import { MockedResponse } from '@apollo/client/testing'
import { select } from '@storybook/addon-knobs'
import { storiesOf } from '@storybook/react'
import { noop } from 'lodash'
import React from 'react'

import { getDocumentNode } from '@sourcegraph/shared/src/graphql/utils'

import { ExternalServiceKind } from '../../../graphql-operations'
import { EnterpriseWebStory } from '../../components/EnterpriseWebStory'

import { AddCredentialModal } from './AddCredentialModal'
import { CREATE_BATCH_CHANGES_CREDENTIAL } from './backend'

const { add } = storiesOf('web/batches/settings/AddCredentialModal', module)
    .addDecorator(story => <div className="p-3 container">{story()}</div>)
    .addParameters({
        chromatic: {
            // Delay screenshot taking, so the modal has opened by the time the screenshot is taken.
            delay: 2000,
        },
    })

const mockCreateBatchChangesCredential = (
    externalServiceKind: ExternalServiceKind,
    externalServiceURL: string
): MockedResponse => ({
    request: {
        query: getDocumentNode(CREATE_BATCH_CHANGES_CREDENTIAL),
        variables: {
            user: 'user-id-1',
            credential: '123',
            externalServiceKind,
            externalServiceURL,
        },
    },
    result: {
        data: {
            createBatchChangesCredential: {
                id: '123',
                isSiteCredential: false,
                sshPublicKey:
                    'ssh-rsa randorandorandorandorandorandorandorandorandorandorandorandorandorandorandorandorandorandorandorandorandorandorandorandorandorando',
            },
        },
    },
})

add('Requires SSH - step 1', () => (
    <EnterpriseWebStory mocks={[mockCreateBatchChangesCredential(ExternalServiceKind.GITHUB, 'https://github.com/')]}>
        {props => (
            <AddCredentialModal
                {...props}
                userID="user-id-1"
                externalServiceKind={select(
                    'External service kind',
                    Object.values(ExternalServiceKind),
                    ExternalServiceKind.GITHUB
                )}
                externalServiceURL="https://github.com/"
                requiresSSH={true}
                afterCreate={noop}
                onCancel={noop}
            />
        )}
    </EnterpriseWebStory>
))
add('Requires SSH - step 2', () => (
    <EnterpriseWebStory>
        {props => (
            <AddCredentialModal
                {...props}
                userID="user-id-1"
                externalServiceKind={select(
                    'External service kind',
                    Object.values(ExternalServiceKind),
                    ExternalServiceKind.GITHUB
                )}
                externalServiceURL="https://github.com/"
                requiresSSH={true}
                afterCreate={noop}
                onCancel={noop}
                initialStep="get-ssh-key"
            />
        )}
    </EnterpriseWebStory>
))

add('GitHub', () => (
    <EnterpriseWebStory mocks={[mockCreateBatchChangesCredential(ExternalServiceKind.GITHUB, 'https://github.com/')]}>
        {props => (
            <AddCredentialModal
                {...props}
                userID="user-id-1"
                externalServiceKind={ExternalServiceKind.GITHUB}
                externalServiceURL="https://github.com/"
                requiresSSH={false}
                afterCreate={noop}
                onCancel={noop}
            />
        )}
    </EnterpriseWebStory>
))

add('GitLab', () => (
    <EnterpriseWebStory mocks={[mockCreateBatchChangesCredential(ExternalServiceKind.GITLAB, 'https://gitlab.com/')]}>
        {props => (
            <AddCredentialModal
                {...props}
                userID="user-id-1"
                externalServiceKind={ExternalServiceKind.GITLAB}
                externalServiceURL="https://gitlab.com/"
                requiresSSH={false}
                afterCreate={noop}
                onCancel={noop}
            />
        )}
    </EnterpriseWebStory>
))

add('Bitbucket Server', () => (
    <EnterpriseWebStory
        mocks={[mockCreateBatchChangesCredential(ExternalServiceKind.BITBUCKETSERVER, 'https://bitbucket.sgdev.org/')]}
    >
        {props => (
            <AddCredentialModal
                {...props}
                userID="user-id-1"
                externalServiceKind={ExternalServiceKind.BITBUCKETSERVER}
                externalServiceURL="https://bitbucket.sgdev.org/"
                requiresSSH={false}
                afterCreate={noop}
                onCancel={noop}
            />
        )}
    </EnterpriseWebStory>
))
