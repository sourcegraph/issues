import { MockedResponse } from '@apollo/client/testing'
import { storiesOf } from '@storybook/react'
import { noop } from 'lodash'
import React from 'react'

import { ExternalServiceKind } from '@sourcegraph/shared/src/graphql-operations'
import { getDocumentNode } from '@sourcegraph/shared/src/graphql/utils'

import { EnterpriseWebStory } from '../../components/EnterpriseWebStory'

import { DELETE_BATCH_CHANGES_CREDENTIAL } from './backend'
import { RemoveCredentialModal } from './RemoveCredentialModal'

const { add } = storiesOf('web/batches/settings/RemoveCredentialModal', module)
    .addDecorator(story => <div className="p-3 container">{story()}</div>)
    .addParameters({
        chromatic: {
            // Delay screenshot taking, so the modal has opened by the time the screenshot is taken.
            delay: 2000,
        },
    })

const credential = {
    id: '123',
    isSiteCredential: false,
    sshPublicKey:
        'ssh-rsa randorandorandorandorandorandorandorandorandorandorandorandorandorandorandorandorandorandorandorandorandorandorandorandorandorandorandorandorandorando',
}

const mocks: MockedResponse[] = [
    {
        request: {
            query: getDocumentNode(DELETE_BATCH_CHANGES_CREDENTIAL),
            variables: {},
        },
        result: {
            data: {
                alwaysNill: null,
            },
        },
    },
]

add('No ssh', () => (
    <EnterpriseWebStory mocks={mocks}>
        {props => (
            <RemoveCredentialModal
                {...props}
                codeHost={{
                    credential,
                    requiresSSH: false,
                    externalServiceKind: ExternalServiceKind.GITHUB,
                    externalServiceURL: 'https://github.com/',
                }}
                credential={credential}
                afterDelete={noop}
                onCancel={noop}
            />
        )}
    </EnterpriseWebStory>
))

add('Requires ssh', () => (
    <EnterpriseWebStory mocks={mocks}>
        {props => (
            <RemoveCredentialModal
                {...props}
                codeHost={{
                    credential,
                    requiresSSH: true,
                    externalServiceKind: ExternalServiceKind.GITHUB,
                    externalServiceURL: 'https://github.com/',
                }}
                credential={credential}
                afterDelete={noop}
                onCancel={noop}
            />
        )}
    </EnterpriseWebStory>
))
