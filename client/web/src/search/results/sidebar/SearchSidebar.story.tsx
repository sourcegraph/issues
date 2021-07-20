import { storiesOf } from '@storybook/react'
import React from 'react'

import { Filter } from '@sourcegraph/shared/src/search/stream'
import { EMPTY_SETTINGS_CASCADE } from '@sourcegraph/shared/src/settings/settings'
import { NOOP_TELEMETRY_SERVICE } from '@sourcegraph/shared/src/telemetry/telemetryService'

import { AuthenticatedUser } from '../../../auth'
import { WebStory } from '../../../components/WebStory'
import { EMPTY_FEATURE_FLAGS } from '../../../featureFlags/featureFlags'
import { SearchPatternType } from '../../../graphql-operations'
import { QuickLink, SearchScope } from '../../../schema/settings.schema'

import { SearchSidebar, SearchSidebarProps } from './SearchSidebar'

const { add } = storiesOf('web/search/results/sidebar/SearchSidebar', module).addParameters({
    design: {
        type: 'figma',
        url: 'https://www.figma.com/file/NIsN34NH7lPu04olBzddTw/?node-id=1018%3A13883',
    },
    chromatic: { viewports: [544, 577, 993] },
})

const authenticatedUser: AuthenticatedUser = {
    __typename: 'User',
    id: '0',
    email: 'alice@sourcegraph.com',
    username: 'alice',
    avatarURL: null,
    session: { canSignOut: true },
    displayName: null,
    url: '',
    settingsURL: '#',
    siteAdmin: true,
    organizations: {
        nodes: [],
    },
    tags: [],
    viewerCanAdminister: true,
    databaseID: 0,
}

const defaultProps: SearchSidebarProps = {
    authenticatedUser,
    caseSensitive: false,
    patternType: SearchPatternType.literal,
    versionContext: undefined,
    selectedSearchContextSpec: 'global',
    query: '',
    settingsCascade: EMPTY_SETTINGS_CASCADE,
    telemetryService: NOOP_TELEMETRY_SERVICE,
    featureFlags: EMPTY_FEATURE_FLAGS,
    navbarSearchQueryState: { query: '' },
    onNavbarQueryChange: () => {},
    isSourcegraphDotCom: false,
}

const quicklinks: QuickLink[] = [
    { name: 'Home', url: '/' },
    {
        name: 'This is a quicklink with a very long name lorem ipsum dolor sit amet',
        url: 'http://example.com',
        description: 'Example QuickLink',
    },
]

const scopes: SearchScope[] = [
    { name: 'This is a search scope with a very long name lorem ipsum dolor sit amet', value: 'repo:sourcegraph' },
    { name: 'All results', value: 'count:all' },
]

const filters: Filter[] = [
    {
        label: 'github.com/test/this-is-a-very-long-repo-name',
        value: 'repo:^github\\.com/test/this-is-a-very-long-repo-name$',
        count: 5,
        limitHit: false,
        kind: 'repo',
    },
    {
        label: 'gitlab.com/sourcegraph/sourcegraph',
        value: 'repo:^gitlab\\.com/sourcegraph/sourcegraph$',
        count: 201,
        limitHit: true,
        kind: 'repo',
    },
    {
        label: 'github.com/microsoft/vscode',
        value: 'repo:^github\\.com/microsoft/vscode$',
        count: 10,
        limitHit: true,
        kind: 'repo',
    },
    {
        label: 'bitbucket.com/test/test',
        value: 'repo:^bitbucket\\.com/test/test$',
        count: 1,
        limitHit: true,
        kind: 'repo',
    },
    {
        label: 'gitlab.sgdev.org/example/test',
        value: 'repo:^gitlab\\.sgdev\\.org/example/test$',
        count: 10,
        limitHit: true,
        kind: 'repo',
    },

    {
        label: 'lang:go',
        value: 'lang:go',
        count: 500,
        limitHit: true,
        kind: 'lang',
    },

    {
        label: 'lang:verylonglanguagenameloremipsumdolor',
        value: 'lang:verylonglanguagenameloremipsumdolor',
        count: 241,
        limitHit: false,
        kind: 'lang',
    },

    {
        label: '-file:_test\\.go$',
        value: '-file:_test\\.go$',
        count: 1230,
        limitHit: false,
        kind: 'file',
    },

    ...['typescript', 'javascript', 'c++', 'c', 'c#', 'python', 'ruby', 'haskell', 'java'].map(lang => ({
        label: `lang:${lang}`,
        value: `lang:${lang}`,
        count: 10,
        limitHit: true,
        kind: 'lang',
    })),
]

add('empty sidebar', () => <WebStory>{() => <SearchSidebar {...defaultProps} />}</WebStory>)

add('with everything', () => (
    <WebStory>
        {() => (
            <SearchSidebar
                {...defaultProps}
                settingsCascade={{ subjects: [], final: { quicklinks, 'search.scopes': scopes } }}
                filters={filters}
            />
        )}
    </WebStory>
))
