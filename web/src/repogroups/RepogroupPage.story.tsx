import * as H from 'history'
import { storiesOf } from '@storybook/react'
import { RepogroupPage, RepogroupPageProps } from './RepogroupPage'
import React from 'react'
import { python2To3Metadata } from './Python2To3'
import * as GQL from '../../../shared/src/graphql/schema'
import { NEVER } from 'rxjs'
import { NOOP_SETTINGS_CASCADE } from '../../../shared/src/util/searchTestHelpers'
import sinon from 'sinon'
import { ThemePreference } from '../theme'
import { NOOP_TELEMETRY_SERVICE } from '../../../shared/src/telemetry/telemetryService'
import { ActionItemComponentProps } from '../../../shared/src/actions/ActionItem'
import { EventLogger } from '../tracking/eventLogger'
import { Services } from '../../../shared/src/api/client/services'
import { MemoryRouter } from 'react-router'
import webStyles from '../SourcegraphWebApp.scss'

const { add } = storiesOf('web/RepogroupPage', module)
    .addParameters({
        percy: { widths: [993] },
        design: {
            type: 'figma',
            url: 'https://www.figma.com/file/Xc4M24VTQq8itU0Lgb1Wwm/RFC-159-Visual-Design?node-id=66%3A611',
        },
        chromatic: { viewports: [769, 993, 1200] },
    })
    .addDecorator(story => (
        <>
            <style>{webStyles}</style>
            <div className="theme-light">{story()}</div>
        </>
    ))

const history = H.createMemoryHistory()

const EXTENSIONS_CONTROLLER: ActionItemComponentProps['extensionsController'] = {
    executeCommand: () => new Promise(resolve => setTimeout(resolve, 750)),
}

const PLATFORM_CONTEXT: ActionItemComponentProps['platformContext'] = {
    forceUpdateTooltip: () => undefined,
    settings: NEVER,
}

const NOOP_CONFIG: GQL.Configuration = {
    __typename: 'Configuration',
    contents: '{}',
    messages: [],
}

const NOOP_CONFIG_CASCADE: GQL.ConfigurationCascade = {
    __typename: 'ConfigurationCascade',
    subjects: [],
    merged: NOOP_CONFIG,
}

const authUser = {
    __typename: 'User',
    id: '0',
    email: 'alice@sourcegraph.com',
    username: 'alice',
    avatarURL: null,
    session: { __typename: 'Session', canSignOut: true },
    displayName: null,
    url: '',
    settingsURL: '#',
    createdAt: '',
    updatedAt: null,
    siteAdmin: true,
    builtinAuth: true,
    latestSettings: null,
    settingsCascade: {
        __typename: 'SettingsCascade',
        ...NOOP_SETTINGS_CASCADE,
        subjects: [],
        final: '{search.repositoryGroups: "python": ["github.com/python/test"]}',
        merged: NOOP_CONFIG,
    },
    configurationCascade: NOOP_CONFIG_CASCADE,
    organizationMemberships: {
        __typename: 'OrganizationMembershipConnection',
        nodes: [],
        totalCount: 0,
    },
    organizations: {
        __typename: 'OrgConnection',
        totalCount: 3,
        nodes: [
            { id: '0', settingsURL: '#', displayName: 'Acme Corp' },
            { id: '1', settingsURL: '#', displayName: 'Beta Inc' },
        ] as GQL.Org[],
    },
    tags: [],
    usageStatistics: {
        __typename: 'UserUsageStatistics',
        searchQueries: 0,
        pageViews: 0,
        codeIntelligenceActions: 0,
        findReferencesActions: 0,
        lastActiveTime: null,
        lastActiveCodeHostIntegrationTime: null,
    },
    eventLogs: {
        __typename: 'EventLogsConnection',
        nodes: [],
        totalCount: 0,
        pageInfo: { __typename: 'PageInfo', endCursor: null, hasNextPage: false },
    } as GQL.EventLogsConnection,
    emails: [],
    accessTokens: {} as GQL.AccessTokenConnection,
    externalAccounts: {} as GQL.ExternalAccountConnection,
    viewerCanAdminister: true,
    viewerCanChangeUsername: true,
    surveyResponses: [],
    urlForSiteAdminBilling: null,
    databaseID: 0,
    namespaceName: '',
    permissionsInfo: null,
} as GQL.User

const commonProps: RepogroupPageProps = {
    settingsCascade: {
        ...NOOP_SETTINGS_CASCADE,
        subjects: [],
        final: {
            'search.repositoryGroups': {
                python: [
                    'github.com/python/test',
                    'github.com/python/test2',
                    'github.com/python/test3',
                    'github.com/python/test4',
                ],
            },
        },
    },
    isLightTheme: true,
    themePreference: ThemePreference.Light,
    onThemePreferenceChange: sinon.spy(() => {}),
    patternType: GQL.SearchPatternType.literal,
    setPatternType: sinon.spy(() => {}),
    caseSensitive: false,
    copyQueryButton: false,
    extensionsController: { ...EXTENSIONS_CONTROLLER, services: {} as Services },
    platformContext: PLATFORM_CONTEXT,
    filtersInQuery: {},
    history,
    interactiveSearchMode: false,
    keyboardShortcuts: [],
    onFiltersInQueryChange: sinon.spy(() => {}),
    setCaseSensitivity: sinon.spy(() => {}),
    smartSearchField: true,
    splitSearchModes: true,
    telemetryService: ({
        ...NOOP_TELEMETRY_SERVICE,
        logViewEvent: () => {},
        logToConsole: () => {},
        getAnonUserID: sinon.spy(() => 'test'),
        hasStrippedQueryParameters: false,
    } as unknown) as EventLogger,
    toggleSearchMode: sinon.spy(() => {}),
    versionContext: undefined,
    activation: undefined,
    location: history.location,
    isSourcegraphDotCom: true,
    setVersionContext: sinon.spy(() => {}),
    availableVersionContexts: [],
    authRequired: false,
    showCampaigns: false,
    authenticatedUser: authUser,
    repogroupMetadata: python2To3Metadata,
    autoFocus: false,
    globbing: false,
}

add('Repogroup page with smart search field', () => (
    <MemoryRouter>
        <RepogroupPage {...commonProps} />
    </MemoryRouter>
))

add('Repogroup page without smart search field', () => (
    <MemoryRouter>
        <RepogroupPage {...commonProps} smartSearchField={false} />
    </MemoryRouter>
))
