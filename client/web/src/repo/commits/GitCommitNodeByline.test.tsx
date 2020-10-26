import React from 'react'
import renderer from 'react-test-renderer'
import { GitCommitNodeByline } from './GitCommitNodeByline'
import { SignatureFields } from '../../graphql-operations'

const FIXTURE_SIGNATURE_1: SignatureFields = {
    date: '1990-01-01',
    person: {
        name: 'Alice Zhao',
        displayName: 'Alice Zhao',
        email: 'alice@example.com',
        avatarURL: 'http://example.com/alice.png',
        user: null,
    },
}

const FIXTURE_SIGNATURE_2: SignatureFields = {
    date: '1991-01-01',
    person: {
        name: 'Bob Yang',
        displayName: 'Bob Yang',
        email: 'bob@example.com',
        avatarURL: 'http://example.com/bob.png',
        user: {
            username: 'bYang',
            id: 'user123',
            displayName: 'Bob Yang',
            url: 'https://example.com/bobyang',
        },
    },
}

describe('GitCommitNodeByline', () => {
    test('author', () =>
        expect(
            renderer.create(<GitCommitNodeByline author={FIXTURE_SIGNATURE_1} committer={null} />).toJSON()
        ).toMatchSnapshot())

    test('different author and committer', () =>
        expect(
            renderer
                .create(<GitCommitNodeByline author={FIXTURE_SIGNATURE_1} committer={FIXTURE_SIGNATURE_2} />)
                .toJSON()
        ).toMatchSnapshot())

    test('author (compact)', () =>
        expect(
            renderer.create(<GitCommitNodeByline author={FIXTURE_SIGNATURE_1} committer={null} />).toJSON()
        ).toMatchSnapshot())

    test('different author and committer (compact)', () =>
        expect(
            renderer
                .create(<GitCommitNodeByline author={FIXTURE_SIGNATURE_1} committer={FIXTURE_SIGNATURE_2} />)
                .toJSON()
        ).toMatchSnapshot())

    test('same author and committer', () =>
        expect(
            renderer
                .create(<GitCommitNodeByline author={FIXTURE_SIGNATURE_1} committer={FIXTURE_SIGNATURE_1} />)
                .toJSON()
        ).toMatchSnapshot())

    test('omit GitHub committer', () =>
        expect(
            renderer
                .create(
                    <GitCommitNodeByline
                        author={FIXTURE_SIGNATURE_1}
                        committer={{
                            date: '1992-01-01',
                            person: {
                                name: 'GitHub',
                                email: 'noreply@github.com',
                                displayName: 'GitHub',
                                avatarURL: 'http://example.com/github.png',
                                user: {
                                    username: 'gitUserName',
                                    id: 'user123',
                                    displayName: 'Alice',
                                    url: 'https://example.com',
                                },
                            },
                        }}
                    />
                )
                .toJSON()
        ).toMatchSnapshot())
})
