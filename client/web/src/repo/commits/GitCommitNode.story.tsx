import { storiesOf } from '@storybook/react'
import { boolean } from '@storybook/addon-knobs'
import React from 'react'
import { GitCommitNode } from './GitCommitNode'
import { subDays } from 'date-fns'
import { GitCommitFields } from '../../graphql-operations'
import { WebStory } from '../../components/WebStory'

const { add } = storiesOf('web/GitCommitNode', module).addDecorator(story => (
    <div className="p-3 container web-content">{story()}</div>
))

const gitCommitNode: GitCommitFields = {
    id: 'commit123',
    abbreviatedOID: 'abcdefg',
    oid: 'abcdefghijklmnopqrstuvwxyz12345678904321',
    author: {
        date: subDays(new Date(), 5).toISOString(),
        person: {
            avatarURL: 'http://test.test/useravatar',
            displayName: 'alice',
            email: 'alice@sourcegraph.com',
            name: 'Alice',
            user: {
                id: 'alice123',
                url: '/users/alice',
                displayName: 'Alice',
                username: 'alice',
            },
        },
    },
    committer: {
        date: subDays(new Date(), 5).toISOString(),
        person: {
            avatarURL: 'http://test.test/useravatar',
            displayName: 'alice',
            email: 'alice@sourcegraph.com',
            name: 'Alice',
            user: {
                id: 'alice123',
                url: '/users/alice',
                displayName: 'Alice',
                username: 'alice',
            },
        },
    },
    body:
        'Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua.',
    parents: [
        {
            abbreviatedOID: '987654',
            oid: '98765432101234abcdefghijklmnopqrstuvwxyz',
            url: '/commits/987654',
        },
    ],
    subject: 'Super awesome commit',
    url: '/commits/abcdefg',
    tree: null,
    canonicalURL: 'asd',
    externalURLs: [],
    message:
        'Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore.',
}

add('Full customizable', () => (
    <WebStory>
        {() => (
            <div className="card">
                <GitCommitNode
                    node={gitCommitNode}
                    compact={boolean('compact', false)}
                    expandCommitMessageBody={boolean('expandCommitMessageBody', false)}
                    showSHAAndParentsRow={boolean('showSHAAndParentsRow', false)}
                    hideExpandCommitMessageBody={boolean('hideExpandCommitMessageBody', false)}
                />
            </div>
        )}
    </WebStory>
))
add('Compact', () => (
    <WebStory>
        {() => (
            <div className="card">
                <GitCommitNode
                    node={gitCommitNode}
                    compact={true}
                    expandCommitMessageBody={false}
                    showSHAAndParentsRow={false}
                    hideExpandCommitMessageBody={false}
                />
            </div>
        )}
    </WebStory>
))
add('Commit message expanded', () => (
    <WebStory>
        {() => (
            <div className="card">
                <GitCommitNode
                    node={gitCommitNode}
                    compact={false}
                    expandCommitMessageBody={true}
                    showSHAAndParentsRow={false}
                    hideExpandCommitMessageBody={false}
                />
            </div>
        )}
    </WebStory>
))
add('SHA and parent shown', () => (
    <WebStory>
        {() => (
            <div className="card">
                <GitCommitNode
                    node={gitCommitNode}
                    compact={false}
                    expandCommitMessageBody={false}
                    showSHAAndParentsRow={true}
                    hideExpandCommitMessageBody={false}
                />
            </div>
        )}
    </WebStory>
))
add('Expand commit message btn hidden', () => (
    <WebStory>
        {() => (
            <div className="card">
                <GitCommitNode
                    node={gitCommitNode}
                    compact={false}
                    expandCommitMessageBody={false}
                    showSHAAndParentsRow={false}
                    hideExpandCommitMessageBody={true}
                />
            </div>
        )}
    </WebStory>
))
