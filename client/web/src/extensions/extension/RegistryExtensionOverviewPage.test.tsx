import { noop } from 'lodash'
import React from 'react'
import renderer from 'react-test-renderer'
import { RegistryExtensionOverviewPage } from './RegistryExtensionOverviewPage'
import { PageTitle } from '../../components/PageTitle'
import { createMemoryHistory } from 'history'
import { Router } from 'react-router'

jest.mock('mdi-react/GithubIcon', () => 'GithubIcon')

describe('RegistryExtensionOverviewPage', () => {
    afterEach(() => {
        PageTitle.titleSet = false
    })
    test('renders', () => {
        const history = createMemoryHistory()
        expect(
            renderer
                .create(
                    <Router history={history}>
                        <RegistryExtensionOverviewPage
                            eventLogger={{ logViewEvent: noop }}
                            extension={{
                                id: 'x',
                                rawManifest: '{}',
                                manifest: {
                                    url: 'https://example.com',
                                    activationEvents: ['*'],
                                    categories: ['Programming languages', 'Other'],
                                    tags: ['T1', 'T2'],
                                    readme: '**A**',
                                    repository: {
                                        url: 'https://github.com/foo/bar',
                                        type: 'git',
                                    },
                                },
                            }}
                            history={history}
                        />
                    </Router>
                )
                .toJSON()
        ).toMatchSnapshot()
    })

    describe('categories', () => {
        test('filters out unrecognized categories', () => {
            const history = createMemoryHistory()
            const output = renderer.create(
                <Router history={history}>
                    <RegistryExtensionOverviewPage
                        eventLogger={{ logViewEvent: noop }}
                        extension={{
                            id: 'x',
                            rawManifest: '',
                            manifest: {
                                url: 'https://example.com',
                                activationEvents: ['*'],
                                categories: ['Programming languages', 'invalid', 'Other'],
                            },
                        }}
                        history={createMemoryHistory()}
                    />
                </Router>
            ).root
            expect(
                toText(
                    output.findAll(({ props: { className } }) =>
                        className?.includes('test-registry-extension-categories')
                    )
                )
            ).toEqual(['Other', 'Programming languages' /* no 'invalid' */])
        })
    })
})

function toText(values: (string | renderer.ReactTestInstance)[]): string[] {
    const textNodes: string[] = []
    for (const value of values) {
        if (typeof value === 'string') {
            textNodes.push(value)
        } else {
            textNodes.push(...toText(value.children))
        }
    }
    return textNodes
}
