import React from 'react'
import renderer from 'react-test-renderer'
import { RepoFileLink } from './RepoFileLink'

describe('RepoFileLink', () => {
    test('renders', () => {
        const component = renderer.create(
            <RepoFileLink
                repoName="example.com/my/repo"
                repoURL="https://example.com"
                filePath="my/file"
                fileURL="https://example.com/file"
            />
        )
        expect(component.toJSON()).toMatchSnapshot()
    })
})
