import React from 'react'
import { mount } from 'enzyme'
import { NOOP_TELEMETRY_SERVICE } from '../../../../shared/src/telemetry/telemetryService'
import { of } from 'rxjs'
import { RecentFilesPanel } from './RecentFilesPanel'

describe('RecentFilesPanel', () => {
    test('duplicate files are only shown once', () => {
        const recentFiles = {
            nodes: [
                {
                    argument: '{"filePath": "go.mod", "repoName": "ghe.sgdev.org/sourcegraph/gorilla-mux"}',
                    timestamp: '2020-09-10T22:55:30Z',
                    url: 'https://sourcegraph.test:3443/ghe.sgdev.org/sourcegraph/gorilla-mux/-/blob/go.mod',
                },
                {
                    argument: '{"filePath": ".eslintrc.js", "repoName": "github.com/sourcegraph/sourcegraph"}',
                    timestamp: '2020-09-10T22:55:18Z',
                    url: 'https://sourcegraph.test:3443/github.com/sourcegraph/sourcegraph/-/blob/.eslintrc.js',
                },
                {
                    argument: '{"filePath": "go.mod", "repoName": "ghe.sgdev.org/sourcegraph/gorilla-mux"}',
                    timestamp: '2020-09-10T22:55:06Z',
                    url: 'https://sourcegraph.test:3443/ghe.sgdev.org/sourcegraph/gorilla-mux/-/blob/go.mod',
                },
            ],
            pageInfo: {
                endCursor: null,
                hasNextPage: false,
            },
            totalCount: 3,
        }

        const props = {
            authenticatedUser: null,
            fetchRecentFileViews: () => of(recentFiles),
            telemetryService: NOOP_TELEMETRY_SERVICE,
        }

        const component = mount(<RecentFilesPanel {...props} />)
        const listItems = component.find('.test-recent-files-item')
        expect(listItems.length).toStrictEqual(2)
        expect(listItems.at(0).text()).toStrictEqual('ghe.sgdev.org/sourcegraph/gorilla-mux › go.mod')
        expect(listItems.at(1).text()).toStrictEqual('github.com/sourcegraph/sourcegraph › .eslintrc.js')
    })

    test('files with missing data can extract it from the URL if available', () => {
        const recentFiles = {
            nodes: [
                {
                    argument: '{"filePath": ".eslintrc.js", "repoName": "github.com/sourcegraph/sourcegraph"}',
                    timestamp: '2020-09-10T22:55:18Z',
                    url: 'https://sourcegraph.test:3443/github.com/sourcegraph/sourcegraph/-/blob/.eslintrc.js',
                },
                {
                    argument: '{}',
                    timestamp: '2020-09-10T22:55:06Z',
                    url: 'https://sourcegraph.test:3443/ghe.sgdev.org/sourcegraph/gorilla-mux/-/blob/go.mod',
                },
                {
                    argument: '{}',
                    timestamp: '2020-09-10T22:55:06Z',
                    url: 'https://sourcegraph.test:3443/bitbucket.sgdev.org/SOURCEGRAPH/jsonrpc2',
                },
            ],
            pageInfo: {
                endCursor: null,
                hasNextPage: false,
            },
            totalCount: 2,
        }

        const props = {
            authenticatedUser: null,
            fetchRecentFileViews: () => of(recentFiles),
            telemetryService: NOOP_TELEMETRY_SERVICE,
        }

        const component = mount(<RecentFilesPanel {...props} />)
        const listItems = component.find('.test-recent-files-item')
        expect(listItems.length).toStrictEqual(2)
        expect(listItems.at(0).text()).toStrictEqual('github.com/sourcegraph/sourcegraph › .eslintrc.js')
        expect(listItems.at(1).text()).toStrictEqual('ghe.sgdev.org/sourcegraph/gorilla-mux › go.mod')
    })

    test('Show More button shown when more items can be loaded', () => {
        const recentFiles = {
            nodes: [
                {
                    argument: '{"filePath": ".eslintrc.js", "repoName": "github.com/sourcegraph/sourcegraph"}',
                    timestamp: '2020-09-10T22:55:18Z',
                    url: 'https://sourcegraph.test:3443/github.com/sourcegraph/sourcegraph/-/blob/.eslintrc.js',
                },
                {
                    argument: '{}',
                    timestamp: '2020-09-10T22:55:06Z',
                    url: 'https://sourcegraph.test:3443/ghe.sgdev.org/sourcegraph/gorilla-mux/-/blob/go.mod',
                },
            ],
            pageInfo: {
                endCursor: null,
                hasNextPage: true,
            },
            totalCount: 2,
        }

        const props = {
            authenticatedUser: null,
            fetchRecentFileViews: () => of(recentFiles),
            telemetryService: NOOP_TELEMETRY_SERVICE,
        }

        const component = mount(<RecentFilesPanel {...props} />)
        const showMoreButton = component.getDOMNode().querySelectorAll('.test-recent-files-panel-show-more')
        expect(showMoreButton.length).toStrictEqual(1)
    })

    test('Show More button not shown when more items cannot be loaded', () => {
        const recentFiles = {
            nodes: [
                {
                    argument: '{"filePath": ".eslintrc.js", "repoName": "github.com/sourcegraph/sourcegraph"}',
                    timestamp: '2020-09-10T22:55:18Z',
                    url: 'https://sourcegraph.test:3443/github.com/sourcegraph/sourcegraph/-/blob/.eslintrc.js',
                },
                {
                    argument: '{}',
                    timestamp: '2020-09-10T22:55:06Z',
                    url: 'https://sourcegraph.test:3443/ghe.sgdev.org/sourcegraph/gorilla-mux/-/blob/go.mod',
                },
            ],
            pageInfo: {
                endCursor: null,
                hasNextPage: false,
            },
            totalCount: 2,
        }

        const props = {
            authenticatedUser: null,
            fetchRecentFileViews: () => of(recentFiles),
            telemetryService: NOOP_TELEMETRY_SERVICE,
        }

        const component = mount(<RecentFilesPanel {...props} />)
        const showMoreButton = component.find('.test-recent-files-panel-show-more')
        expect(showMoreButton.length).toStrictEqual(0)
    })
})
