import React from 'react'
import renderer from 'react-test-renderer'
import { RepoLink } from './RepoLink'

describe('RepoLink', () => {
    test('renders a link when "to" is set', () => {
        const component = renderer.create(<RepoLink repoName="my/repo" to="http://example.com" />)
        expect(component.toJSON()).toMatchSnapshot()
    })

    test('renders a fragment when "to" is null', () => {
        const component = renderer.create(<RepoLink repoName="my/repo" to={null} />)
        expect(component.toJSON()).toMatchSnapshot()
    })
})
