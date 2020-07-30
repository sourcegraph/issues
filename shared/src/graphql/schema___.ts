export type ID = string
export type GitObjectID = string
export type DateTime = string
export type JSONCString = string

export interface IGraphQLResponseRoot {
    data?: Query | Mutation
    errors?: IGraphQLResponseError[]
}

export interface IGraphQLResponseError {
    /** Required for all errors */
    message: string
    locations?: IGraphQLResponseErrorLocation[]
    /** 7.2.2 says 'GraphQL servers may provide additional entries to error' */
    [propName: string]: any
}

export interface IGraphQLResponseErrorLocation {
    line: number
    column: number
}

export interface Query {
    __typename: 'Query'

    /**
     * @deprecated "this will be removed."
     */
    root: Query
    node: Node | null
    campaigns: ICampaignConnection
    repository: IRepository | null
    repositoryRedirect: RepositoryRedirect | null
    externalServices: IExternalServiceConnection
    repositories: IRepositoryConnection
    phabricatorRepo: IPhabricatorRepo | null
    currentUser: IUser | null
    user: IUser | null
    users: IUserConnection
    organization: IOrg | null
    organizations: IOrgConnection
    renderMarkdown: string
    highlightCode: string
    settingsSubject: SettingsSubject | null
    viewerSettings: ISettingsCascade

    /**
     * @deprecated "use viewerSettings instead"
     */
    viewerConfiguration: IConfigurationCascade
    clientConfiguration: IClientConfigurationDetails
    searchFilterSuggestions: ISearchFilterSuggestions
    search: ISearch | null
    savedSearches: ISavedSearch[]
    repoGroups: IRepoGroup[]
    versionContexts: IVersionContext[]
    parseSearchQuery: any | null
    site: ISite
    surveyResponses: ISurveyResponseConnection
    extensionRegistry: IExtensionRegistry
    dotcom: IDotcomQuery
    statusMessages: StatusMessage[]
    namespace: Namespace | null
    authorizedUserRepositories: IRepositoryConnection
    usersWithPendingPermissions: string[]
    lsifUploads: ILSIFUploadConnection
    lsifIndexes: ILSIFIndexConnection
}

export interface INodeOnQueryArguments {
    id: ID
}

export interface ICampaignsOnQueryArguments {
    first?: number | null
    state?: CampaignState | null
    hasPatchSet?: boolean | null
    viewerCanAdminister?: boolean | null
}

export interface IRepositoryOnQueryArguments {
    name?: string | null
    cloneURL?: string | null
    uri?: string | null
}

export interface IRepositoryRedirectOnQueryArguments {
    name?: string | null
    cloneURL?: string | null
}

export interface IExternalServicesOnQueryArguments {
    first?: number | null
}

export interface IRepositoriesOnQueryArguments {
    first?: number | null
    query?: string | null
    names?: string[] | null

    /**
     * @default true
     */
    cloned?: boolean | null

    /**
     * @default true
     */
    cloneInProgress?: boolean | null

    /**
     * @default true
     */
    notCloned?: boolean | null

    /**
     * @default true
     */
    indexed?: boolean | null

    /**
     * @default true
     */
    notIndexed?: boolean | null

    /**
     * @default "REPOSITORY_NAME"
     */
    orderBy?: RepositoryOrderBy | null

    /**
     * @default false
     */
    descending?: boolean | null
}

export interface IPhabricatorRepoOnQueryArguments {
    name?: string | null
    uri?: string | null
}

export interface IUserOnQueryArguments {
    username?: string | null
    email?: string | null
}

export interface IUsersOnQueryArguments {
    first?: number | null
    query?: string | null
    tag?: string | null
    activePeriod?: UserActivePeriod | null
}

export interface IOrganizationOnQueryArguments {
    name: string
}

export interface IOrganizationsOnQueryArguments {
    first?: number | null
    query?: string | null
}

export interface IRenderMarkdownOnQueryArguments {
    markdown: string
    options?: IMarkdownOptions | null
}

export interface IHighlightCodeOnQueryArguments {
    code: string
    fuzzyLanguage: string
    disableTimeout: boolean
    isLightTheme: boolean
}

export interface ISettingsSubjectOnQueryArguments {
    id: ID
}

export interface ISearchOnQueryArguments {
    /**
     * @default "V1"
     */
    version?: SearchVersion | null
    patternType?: SearchPatternType | null

    /**
     * @default ""
     */
    query?: string | null
    versionContext?: string | null
    after?: string | null
    first?: number | null
}

export interface IParseSearchQueryOnQueryArguments {
    /**
     * @default ""
     */
    query?: string | null

    /**
     * @default "literal"
     */
    patternType?: SearchPatternType | null
}

export interface ISurveyResponsesOnQueryArguments {
    first?: number | null
}

export interface INamespaceOnQueryArguments {
    id: ID
}

export interface IAuthorizedUserRepositoriesOnQueryArguments {
    username?: string | null
    email?: string | null

    /**
     * @default "READ"
     */
    perm?: RepositoryPermission | null
    first: number
    after?: string | null
}

export interface ILsifUploadsOnQueryArguments {
    query?: string | null
    state?: LSIFUploadState | null
    isLatestForRepo?: boolean | null
    first?: number | null
    after?: string | null
}

export interface ILsifIndexesOnQueryArguments {
    query?: string | null
    state?: LSIFIndexState | null
    first?: number | null
    after?: string | null
}

export type Node =
    | ICampaign
    | IPatchSet
    | IUser
    | IOrg
    | IOrganizationInvitation
    | IAccessToken
    | IExternalAccount
    | IRepository
    | IGitCommit
    | IExternalService
    | IGitRef
    | ILSIFUpload
    | ILSIFIndex
    | ISavedSearch
    | IVersionContext
    | IRegistryExtension
    | IProductSubscription
    | IProductLicense
    | ExternalChangeset
    | IChangesetEvent
    | IPatch
    | IHiddenPatch
    | IHiddenExternalChangeset

export interface INode {
    __typename: 'Node'
    id: ID
}

export enum CampaignState {
    OPEN = 'OPEN',
    CLOSED = 'CLOSED',
}

export interface ICampaignConnection {
    __typename: 'CampaignConnection'
    nodes: ICampaign[]
    totalCount: number
    pageInfo: IPageInfo
}

export interface ICampaign {
    __typename: 'Campaign'
    id: ID
    patchSet: IPatchSet | null
    status: IBackgroundProcessStatus
    namespace: Namespace
    name: string
    description: string | null
    branch: string | null
    author: IUser
    viewerCanAdminister: boolean
    url: string
    createdAt: DateTime
    updatedAt: DateTime
    repositoryDiffs: IRepositoryComparisonConnection
    changesets: IChangesetConnection
    openChangesets: IChangesetConnection
    changesetCountsOverTime: IChangesetCounts[]
    closedAt: DateTime | null
    patches: IPatchConnection
    hasUnpublishedPatches: boolean
    diffStat: IDiffStat
}

export interface IRepositoryDiffsOnCampaignArguments {
    first?: number | null
}

export interface IChangesetsOnCampaignArguments {
    first?: number | null
    state?: ChangesetState | null
    reviewState?: ChangesetReviewState | null
    checkState?: ChangesetCheckState | null
}

export interface IChangesetCountsOverTimeOnCampaignArguments {
    from?: DateTime | null
    to?: DateTime | null
}

export interface IPatchesOnCampaignArguments {
    first?: number | null
}

export interface IPatchSet {
    __typename: 'PatchSet'
    id: ID
    patches: IPatchConnection
    previewURL: string
    diffStat: IDiffStat
}

export interface IPatchesOnPatchSetArguments {
    first?: number | null
}

export interface IPatchConnection {
    __typename: 'PatchConnection'
    nodes: PatchInterface[]
    totalCount: number
    pageInfo: IPageInfo
}

export type PatchInterface = IPatch | IHiddenPatch

export interface IPatchInterface {
    __typename: 'PatchInterface'
    id: ID
}

export interface IPageInfo {
    __typename: 'PageInfo'
    endCursor: string | null
    hasNextPage: boolean
}

export interface IDiffStat {
    __typename: 'DiffStat'
    added: number
    changed: number
    deleted: number
}

export interface IBackgroundProcessStatus {
    __typename: 'BackgroundProcessStatus'
    completedCount: number
    pendingCount: number
    state: BackgroundProcessState
    errors: string[]
}

export enum BackgroundProcessState {
    PROCESSING = 'PROCESSING',
    ERRORED = 'ERRORED',
    COMPLETED = 'COMPLETED',
    CANCELED = 'CANCELED',
}

export type Namespace = IUser | IOrg

export interface INamespace {
    __typename: 'Namespace'
    id: ID
    namespaceName: string
    url: string
}

export interface IUser {
    __typename: 'User'
    id: ID
    username: string

    /**
     * @deprecated "use emails instead"
     */
    email: string
    displayName: string | null
    avatarURL: string | null
    url: string
    settingsURL: string | null
    createdAt: DateTime
    updatedAt: DateTime | null
    siteAdmin: boolean
    builtinAuth: boolean
    latestSettings: ISettings | null
    settingsCascade: ISettingsCascade

    /**
     * @deprecated "Use settingsCascade instead. This field is a deprecated alias for it and will be removed in a future release."
     */
    configurationCascade: IConfigurationCascade
    organizations: IOrgConnection
    organizationMemberships: IOrganizationMembershipConnection
    tags: string[]
    usageStatistics: IUserUsageStatistics
    eventLogs: IEventLogsConnection
    emails: IUserEmail[]
    accessTokens: IAccessTokenConnection
    externalAccounts: IExternalAccountConnection
    session: ISession
    viewerCanAdminister: boolean
    viewerCanChangeUsername: boolean
    surveyResponses: ISurveyResponse[]
    urlForSiteAdminBilling: string | null
    databaseID: number
    namespaceName: string
    permissionsInfo: IPermissionsInfo | null
}

export interface IEventLogsOnUserArguments {
    first?: number | null
}

export interface IAccessTokensOnUserArguments {
    first?: number | null
}

export interface IExternalAccountsOnUserArguments {
    first?: number | null
}

export type SettingsSubject = IUser | IOrg | ISite | IDefaultSettings

export interface ISettingsSubject {
    __typename: 'SettingsSubject'
    id: ID
    latestSettings: ISettings | null
    settingsURL: string | null
    viewerCanAdminister: boolean
    settingsCascade: ISettingsCascade

    /**
     * @deprecated "Use settingsCascade instead. This field is a deprecated alias for it and will be removed in a future release."
     */
    configurationCascade: IConfigurationCascade
}

export interface ISettings {
    __typename: 'Settings'
    id: number
    subject: SettingsSubject
    author: IUser | null
    createdAt: DateTime
    contents: JSONCString

    /**
     * @deprecated "use the contents field instead"
     */
    configuration: IConfiguration
}

export interface IConfiguration {
    __typename: 'Configuration'

    /**
     * @deprecated "use the contents field on the parent type instead"
     */
    contents: JSONCString

    /**
     * @deprecated "use client-side JSON Schema validation instead"
     */
    messages: string[]
}

export interface ISettingsCascade {
    __typename: 'SettingsCascade'
    subjects: SettingsSubject[]
    final: string

    /**
     * @deprecated "use final instead"
     */
    merged: IConfiguration
}

export interface IConfigurationCascade {
    __typename: 'ConfigurationCascade'

    /**
     * @deprecated "use SettingsCascade.subjects instead"
     */
    subjects: SettingsSubject[]

    /**
     * @deprecated "use SettingsCascade.final instead"
     */
    merged: IConfiguration
}

export interface IOrgConnection {
    __typename: 'OrgConnection'
    nodes: IOrg[]
    totalCount: number
}

export interface IOrg {
    __typename: 'Org'
    id: ID
    name: string
    displayName: string | null
    createdAt: DateTime
    members: IUserConnection
    latestSettings: ISettings | null
    settingsCascade: ISettingsCascade

    /**
     * @deprecated "Use settingsCascade instead. This field is a deprecated alias for it and will be removed in a future release."
     */
    configurationCascade: IConfigurationCascade
    viewerPendingInvitation: IOrganizationInvitation | null
    viewerCanAdminister: boolean
    viewerIsMember: boolean
    url: string
    settingsURL: string | null
    namespaceName: string
}

export interface IUserConnection {
    __typename: 'UserConnection'
    nodes: IUser[]
    totalCount: number
    pageInfo: IPageInfo
}

export interface IOrganizationInvitation {
    __typename: 'OrganizationInvitation'
    id: ID
    organization: IOrg
    sender: IUser
    recipient: IUser
    createdAt: DateTime
    notifiedAt: DateTime | null
    respondedAt: DateTime | null
    responseType: OrganizationInvitationResponseType | null
    respondURL: string | null
    revokedAt: DateTime | null
}

export enum OrganizationInvitationResponseType {
    ACCEPT = 'ACCEPT',
    REJECT = 'REJECT',
}

export interface IOrganizationMembershipConnection {
    __typename: 'OrganizationMembershipConnection'
    nodes: IOrganizationMembership[]
    totalCount: number
}

export interface IOrganizationMembership {
    __typename: 'OrganizationMembership'
    organization: IOrg
    user: IUser
    createdAt: DateTime
    updatedAt: DateTime
}

export interface IUserUsageStatistics {
    __typename: 'UserUsageStatistics'
    searchQueries: number
    pageViews: number
    codeIntelligenceActions: number
    findReferencesActions: number
    lastActiveTime: string | null
    lastActiveCodeHostIntegrationTime: string | null
}

export interface IEventLogsConnection {
    __typename: 'EventLogsConnection'
    nodes: IEventLog[]
    totalCount: number
    pageInfo: IPageInfo
}

export interface IEventLog {
    __typename: 'EventLog'
    name: string
    user: IUser | null
    anonymousUserID: string
    url: string
    source: EventSource
    argument: string | null
    version: string
    timestamp: DateTime
}

export enum EventSource {
    WEB = 'WEB',
    CODEHOSTINTEGRATION = 'CODEHOSTINTEGRATION',
    BACKEND = 'BACKEND',
}

export interface IUserEmail {
    __typename: 'UserEmail'
    email: string
    isPrimary: boolean
    verified: boolean
    verificationPending: boolean
    user: IUser
    viewerCanManuallyVerify: boolean
}

export interface IAccessTokenConnection {
    __typename: 'AccessTokenConnection'
    nodes: IAccessToken[]
    totalCount: number
    pageInfo: IPageInfo
}

export interface IAccessToken {
    __typename: 'AccessToken'
    id: ID
    subject: IUser
    scopes: string[]
    note: string
    creator: IUser
    createdAt: DateTime
    lastUsedAt: DateTime | null
}

export interface IExternalAccountConnection {
    __typename: 'ExternalAccountConnection'
    nodes: IExternalAccount[]
    totalCount: number
    pageInfo: IPageInfo
}

export interface IExternalAccount {
    __typename: 'ExternalAccount'
    id: ID
    user: IUser
    serviceType: string
    serviceID: string
    clientID: string
    accountID: string
    createdAt: DateTime
    updatedAt: DateTime
    refreshURL: string | null
    accountData: any | null
}

export interface ISession {
    __typename: 'Session'
    canSignOut: boolean
}

export interface ISurveyResponse {
    __typename: 'SurveyResponse'
    id: ID
    user: IUser | null
    email: string | null
    score: number
    reason: string | null
    better: string | null
    createdAt: DateTime
}

export interface IPermissionsInfo {
    __typename: 'PermissionsInfo'
    permissions: RepositoryPermission[]
    syncedAt: DateTime | null
    updatedAt: DateTime
}

export enum RepositoryPermission {
    READ = 'READ',
}

export interface IRepositoryComparisonConnection {
    __typename: 'RepositoryComparisonConnection'
    nodes: RepositoryComparison[]
    totalCount: number
    pageInfo: IPageInfo
}

export interface RepositoryComparison {
    __typename: 'RepositoryComparison'
    baseRepository: IRepository
    headRepository: IRepository
    range: IGitRevisionRange
    commits: IGitCommitConnection
    fileDiffs: IFileDiffConnection
}

export interface ICommitsOnRepositoryComparisonArguments {
    first?: number | null
}

export interface IFileDiffsOnRepositoryComparisonArguments {
    first?: number | null
    after?: string | null
}

export interface IRepository {
    __typename: 'Repository'
    id: ID
    name: string

    /**
     * @deprecated "Use name."
     */
    uri: string
    description: string
    language: string
    createdAt: DateTime
    updatedAt: DateTime | null
    commit: IGitCommit | null
    mirrorInfo: IMirrorRepositoryInfo
    externalRepository: IExternalRepository
    isFork: boolean
    isArchived: boolean
    isPrivate: boolean
    externalServices: IExternalServiceConnection

    /**
     * @deprecated "use Repository.mirrorInfo.cloneInProgress instead"
     */
    cloneInProgress: boolean
    textSearchIndex: IRepositoryTextSearchIndex | null
    url: string
    externalURLs: IExternalLink[]
    defaultBranch: IGitRef | null
    gitRefs: IGitRefConnection
    branches: IGitRefConnection
    tags: IGitRefConnection
    comparison: RepositoryComparison
    contributors: IRepositoryContributorConnection
    viewerCanAdminister: boolean
    icon: string
    label: IMarkdown
    detail: IMarkdown
    matches: ISearchResultMatch[]
    lsifUploads: ILSIFUploadConnection
    lsifIndexes: ILSIFIndexConnection
    authorizedUsers: IUserConnection
    permissionsInfo: IPermissionsInfo | null
}

export interface ICommitOnRepositoryArguments {
    rev: string
    inputRevspec?: string | null
}

export interface IExternalServicesOnRepositoryArguments {
    first?: number | null
}

export interface IGitRefsOnRepositoryArguments {
    first?: number | null
    query?: string | null
    type?: GitRefType | null
    orderBy?: GitRefOrder | null

    /**
     * @default true
     */
    interactive?: boolean | null
}

export interface IBranchesOnRepositoryArguments {
    first?: number | null
    query?: string | null
    orderBy?: GitRefOrder | null

    /**
     * @default true
     */
    interactive?: boolean | null
}

export interface ITagsOnRepositoryArguments {
    first?: number | null
    query?: string | null
}

export interface IComparisonOnRepositoryArguments {
    base?: string | null
    head?: string | null

    /**
     * @default true
     */
    fetchMissing?: boolean | null
}

export interface IContributorsOnRepositoryArguments {
    revisionRange?: string | null
    after?: string | null
    path?: string | null
    first?: number | null
}

export interface ILsifUploadsOnRepositoryArguments {
    query?: string | null
    state?: LSIFUploadState | null
    isLatestForRepo?: boolean | null
    first?: number | null
    after?: string | null
}

export interface ILsifIndexesOnRepositoryArguments {
    query?: string | null
    state?: LSIFIndexState | null
    first?: number | null
    after?: string | null
}

export interface IAuthorizedUsersOnRepositoryArguments {
    /**
     * @default "READ"
     */
    permission?: RepositoryPermission | null
    first: number
    after?: string | null
}

export type GenericSearchResultInterface = IRepository | ICommitSearchResult | ICodemodResult

export interface IGenericSearchResultInterface {
    __typename: 'GenericSearchResultInterface'
    icon: string
    label: IMarkdown
    url: string
    detail: IMarkdown
    matches: ISearchResultMatch[]
}

export interface IMarkdown {
    __typename: 'Markdown'
    text: string
    html: string
}

export interface ISearchResultMatch {
    __typename: 'SearchResultMatch'
    url: string
    body: IMarkdown
    highlights: IHighlight[]
}

export interface IHighlight {
    __typename: 'Highlight'
    line: number
    character: number
    length: number
}

export interface IGitCommit {
    __typename: 'GitCommit'
    id: ID
    repository: IRepository
    oid: GitObjectID
    abbreviatedOID: string
    author: ISignature
    committer: ISignature | null
    message: string
    subject: string
    body: string | null
    parents: IGitCommit[]
    url: string
    canonicalURL: string
    externalURLs: IExternalLink[]
    tree: IGitTree | null
    blob: IGitBlob | null
    file: File2 | null
    languages: string[]
    languageStatistics: ILanguageStatistics[]
    ancestors: IGitCommitConnection
    behindAhead: IBehindAheadCounts
    symbols: ISymbolConnection
}

export interface ITreeOnGitCommitArguments {
    /**
     * @default ""
     */
    path?: string | null

    /**
     * @default false
     */
    recursive?: boolean | null
}

export interface IBlobOnGitCommitArguments {
    path: string
}

export interface IFileOnGitCommitArguments {
    path: string
}

export interface IAncestorsOnGitCommitArguments {
    first?: number | null
    query?: string | null
    path?: string | null
    after?: string | null
}

export interface IBehindAheadOnGitCommitArguments {
    revspec: string
}

export interface ISymbolsOnGitCommitArguments {
    first?: number | null
    query?: string | null
    includePatterns?: string[] | null
}

export interface ISignature {
    __typename: 'Signature'
    person: IPerson
    date: string
}

export interface IPerson {
    __typename: 'Person'
    name: string
    email: string
    displayName: string
    avatarURL: string
    user: IUser | null
}

export interface IExternalLink {
    __typename: 'ExternalLink'
    url: string
    serviceType: string | null
}

export interface IGitTree {
    __typename: 'GitTree'
    path: string
    isRoot: boolean
    name: string
    isDirectory: boolean
    commit: IGitCommit
    repository: IRepository
    url: string
    canonicalURL: string
    externalURLs: IExternalLink[]
    rawZipArchiveURL: string
    submodule: ISubmodule | null
    directories: IGitTree[]
    files: IFile[]
    entries: TreeEntry[]
    symbols: ISymbolConnection
    isSingleChild: boolean
    lsif: TreeEntryLSIFData | null
}

export interface IDirectoriesOnGitTreeArguments {
    first?: number | null

    /**
     * @default false
     */
    recursive?: boolean | null
}

export interface IFilesOnGitTreeArguments {
    first?: number | null

    /**
     * @default false
     */
    recursive?: boolean | null
}

export interface IEntriesOnGitTreeArguments {
    first?: number | null

    /**
     * @default false
     */
    recursive?: boolean | null

    /**
     * @default false
     */
    recursiveSingleChild?: boolean | null
}

export interface ISymbolsOnGitTreeArguments {
    first?: number | null
    query?: string | null
}

export interface IIsSingleChildOnGitTreeArguments {
    first?: number | null

    /**
     * @default false
     */
    recursive?: boolean | null

    /**
     * @default false
     */
    recursiveSingleChild?: boolean | null
}

export interface ILsifOnGitTreeArguments {
    toolName?: string | null
}

export type TreeEntry = IGitTree | IGitBlob

export interface ITreeEntry {
    __typename: 'TreeEntry'
    path: string
    name: string
    isDirectory: boolean
    url: string
    canonicalURL: string
    externalURLs: IExternalLink[]
    symbols: ISymbolConnection
    submodule: ISubmodule | null
    isSingleChild: boolean
    lsif: TreeEntryLSIFData | null
}

export interface ISymbolsOnTreeEntryArguments {
    first?: number | null
    query?: string | null
}

export interface IIsSingleChildOnTreeEntryArguments {
    first?: number | null

    /**
     * @default false
     */
    recursive?: boolean | null

    /**
     * @default false
     */
    recursiveSingleChild?: boolean | null
}

export interface ILsifOnTreeEntryArguments {
    toolName?: string | null
}

export interface ISymbolConnection {
    __typename: 'SymbolConnection'
    nodes: ISymbol[]
    pageInfo: IPageInfo
}

export interface ISymbol {
    __typename: 'Symbol'
    name: string
    containerName: string | null
    kind: SymbolKind
    language: string
    location: ILocation
    url: string
    canonicalURL: string
    fileLocal: boolean
}

export enum SymbolKind {
    UNKNOWN = 'UNKNOWN',
    FILE = 'FILE',
    MODULE = 'MODULE',
    NAMESPACE = 'NAMESPACE',
    PACKAGE = 'PACKAGE',
    CLASS = 'CLASS',
    METHOD = 'METHOD',
    PROPERTY = 'PROPERTY',
    FIELD = 'FIELD',
    CONSTRUCTOR = 'CONSTRUCTOR',
    ENUM = 'ENUM',
    INTERFACE = 'INTERFACE',
    FUNCTION = 'FUNCTION',
    VARIABLE = 'VARIABLE',
    CONSTANT = 'CONSTANT',
    STRING = 'STRING',
    NUMBER = 'NUMBER',
    BOOLEAN = 'BOOLEAN',
    ARRAY = 'ARRAY',
    OBJECT = 'OBJECT',
    KEY = 'KEY',
    NULL = 'NULL',
    ENUMMEMBER = 'ENUMMEMBER',
    STRUCT = 'STRUCT',
    EVENT = 'EVENT',
    OPERATOR = 'OPERATOR',
    TYPEPARAMETER = 'TYPEPARAMETER',
}

export interface ILocation {
    __typename: 'Location'
    resource: IGitBlob
    range: IRange | null
    url: string
    canonicalURL: string
}

export interface IGitBlob {
    __typename: 'GitBlob'
    path: string
    name: string
    isDirectory: boolean
    content: string
    byteSize: number
    binary: boolean
    richHTML: string
    commit: IGitCommit
    repository: IRepository
    url: string
    canonicalURL: string
    externalURLs: IExternalLink[]
    blame: IHunk[]
    highlight: IHighlightedFile
    submodule: ISubmodule | null
    symbols: ISymbolConnection
    isSingleChild: boolean
    lsif: IGitBlobLSIFData | null
}

export interface IBlameOnGitBlobArguments {
    startLine: number
    endLine: number
}

export interface IHighlightOnGitBlobArguments {
    disableTimeout: boolean
    isLightTheme: boolean

    /**
     * @default false
     */
    highlightLongLines?: boolean | null
}

export interface ISymbolsOnGitBlobArguments {
    first?: number | null
    query?: string | null
}

export interface IIsSingleChildOnGitBlobArguments {
    first?: number | null

    /**
     * @default false
     */
    recursive?: boolean | null

    /**
     * @default false
     */
    recursiveSingleChild?: boolean | null
}

export interface ILsifOnGitBlobArguments {
    toolName?: string | null
}

export type File2 = IGitBlob | IVirtualFile

export interface IFile2 {
    __typename: 'File2'
    path: string
    name: string
    isDirectory: boolean
    content: string
    byteSize: number
    binary: boolean
    richHTML: string
    url: string
    canonicalURL: string
    externalURLs: IExternalLink[]
    highlight: IHighlightedFile
}

export interface IHighlightOnFile2Arguments {
    disableTimeout: boolean
    isLightTheme: boolean

    /**
     * @default false
     */
    highlightLongLines?: boolean | null
}

export interface IHighlightedFile {
    __typename: 'HighlightedFile'
    aborted: boolean
    html: string
}

export interface IHunk {
    __typename: 'Hunk'
    startLine: number
    endLine: number
    startByte: number
    endByte: number
    rev: string
    author: ISignature
    message: string
    commit: IGitCommit
}

export interface ISubmodule {
    __typename: 'Submodule'
    url: string
    commit: string
    path: string
}

export interface IGitBlobLSIFData {
    __typename: 'GitBlobLSIFData'
    ranges: ICodeIntelligenceRangeConnection | null
    definitions: ILocationConnection
    references: ILocationConnection
    hover: IHover | null
    diagnostics: IDiagnosticConnection
}

export interface IRangesOnGitBlobLSIFDataArguments {
    startLine: number
    endLine: number
}

export interface IDefinitionsOnGitBlobLSIFDataArguments {
    line: number
    character: number
}

export interface IReferencesOnGitBlobLSIFDataArguments {
    line: number
    character: number
    after?: string | null
    first?: number | null
}

export interface IHoverOnGitBlobLSIFDataArguments {
    line: number
    character: number
}

export interface IDiagnosticsOnGitBlobLSIFDataArguments {
    first?: number | null
}

export type TreeEntryLSIFData = IGitBlobLSIFData

export interface ITreeEntryLSIFData {
    __typename: 'TreeEntryLSIFData'
    diagnostics: IDiagnosticConnection
}

export interface IDiagnosticsOnTreeEntryLSIFDataArguments {
    first?: number | null
}

export interface IDiagnosticConnection {
    __typename: 'DiagnosticConnection'
    nodes: IDiagnostic[]
    totalCount: number
    pageInfo: IPageInfo
}

export interface IDiagnostic {
    __typename: 'Diagnostic'
    location: ILocation
    severity: DiagnosticSeverity | null
    code: string | null
    source: string | null
    message: string | null
}

export enum DiagnosticSeverity {
    ERROR = 'ERROR',
    WARNING = 'WARNING',
    INFORMATION = 'INFORMATION',
    HINT = 'HINT',
}

export interface ICodeIntelligenceRangeConnection {
    __typename: 'CodeIntelligenceRangeConnection'
    nodes: ICodeIntelligenceRange[]
}

export interface ICodeIntelligenceRange {
    __typename: 'CodeIntelligenceRange'
    range: IRange
    definitions: ILocationConnection
    references: ILocationConnection
    hover: IHover | null
}

export interface IRange {
    __typename: 'Range'
    start: IPosition
    end: IPosition
}

export interface IPosition {
    __typename: 'Position'
    line: number
    character: number
}

export interface ILocationConnection {
    __typename: 'LocationConnection'
    nodes: ILocation[]
    pageInfo: IPageInfo
}

export interface IHover {
    __typename: 'Hover'
    markdown: IMarkdown
    range: IRange
}

export interface IFile {
    __typename: 'File'
    path: string
    name: string
    isDirectory: boolean
    url: string
    repository: IRepository
}

export interface ILanguageStatistics {
    __typename: 'LanguageStatistics'
    name: string
    totalBytes: number
    totalLines: number
}

export interface IGitCommitConnection {
    __typename: 'GitCommitConnection'
    nodes: IGitCommit[]
    totalCount: number | null
    pageInfo: IPageInfo
}

export interface IBehindAheadCounts {
    __typename: 'BehindAheadCounts'
    behind: number
    ahead: number
}

export interface IMirrorRepositoryInfo {
    __typename: 'MirrorRepositoryInfo'
    remoteURL: string
    cloneInProgress: boolean
    cloneProgress: string | null
    cloned: boolean
    updatedAt: DateTime | null
    updateSchedule: IUpdateSchedule | null
    updateQueue: IUpdateQueue | null
}

export interface IUpdateSchedule {
    __typename: 'UpdateSchedule'
    intervalSeconds: number
    due: DateTime
    index: number
    total: number
}

export interface IUpdateQueue {
    __typename: 'UpdateQueue'
    index: number
    updating: boolean
    total: number
}

export interface IExternalRepository {
    __typename: 'ExternalRepository'
    id: string
    serviceType: string
    serviceID: string
}

export interface IExternalServiceConnection {
    __typename: 'ExternalServiceConnection'
    nodes: IExternalService[]
    totalCount: number
    pageInfo: IPageInfo
}

export interface IExternalService {
    __typename: 'ExternalService'
    id: ID
    kind: ExternalServiceKind
    displayName: string
    config: JSONCString
    createdAt: DateTime
    updatedAt: DateTime
    webhookURL: string | null
    warning: string | null
}

export enum ExternalServiceKind {
    AWSCODECOMMIT = 'AWSCODECOMMIT',
    BITBUCKETCLOUD = 'BITBUCKETCLOUD',
    BITBUCKETSERVER = 'BITBUCKETSERVER',
    GITHUB = 'GITHUB',
    GITLAB = 'GITLAB',
    GITOLITE = 'GITOLITE',
    PHABRICATOR = 'PHABRICATOR',
    OTHER = 'OTHER',
}

export interface IRepositoryTextSearchIndex {
    __typename: 'RepositoryTextSearchIndex'
    repository: IRepository
    status: IRepositoryTextSearchIndexStatus | null
    refs: IRepositoryTextSearchIndexedRef[]
}

export interface IRepositoryTextSearchIndexStatus {
    __typename: 'RepositoryTextSearchIndexStatus'
    updatedAt: DateTime
    contentByteSize: number
    contentFilesCount: number
    indexByteSize: number
    indexShardsCount: number
}

export interface IRepositoryTextSearchIndexedRef {
    __typename: 'RepositoryTextSearchIndexedRef'
    ref: IGitRef
    indexed: boolean
    current: boolean
    indexedCommit: IGitObject | null
}

export interface IGitRef {
    __typename: 'GitRef'
    id: ID
    name: string
    abbrevName: string
    displayName: string
    prefix: string
    type: GitRefType
    target: IGitObject
    repository: IRepository
    url: string
}

export enum GitRefType {
    GIT_BRANCH = 'GIT_BRANCH',
    GIT_TAG = 'GIT_TAG',
    GIT_REF_OTHER = 'GIT_REF_OTHER',
}

export interface IGitObject {
    __typename: 'GitObject'
    oid: GitObjectID
    abbreviatedOID: string
    commit: IGitCommit | null
    type: GitObjectType
}

export enum GitObjectType {
    GIT_COMMIT = 'GIT_COMMIT',
    GIT_TAG = 'GIT_TAG',
    GIT_TREE = 'GIT_TREE',
    GIT_BLOB = 'GIT_BLOB',
    GIT_UNKNOWN = 'GIT_UNKNOWN',
}

export enum GitRefOrder {
    AUTHORED_OR_COMMITTED_AT = 'AUTHORED_OR_COMMITTED_AT',
}

export interface IGitRefConnection {
    __typename: 'GitRefConnection'
    nodes: IGitRef[]
    totalCount: number
    pageInfo: IPageInfo
}

export interface IRepositoryContributorConnection {
    __typename: 'RepositoryContributorConnection'
    nodes: IRepositoryContributor[]
    totalCount: number
    pageInfo: IPageInfo
}

export interface IRepositoryContributor {
    __typename: 'RepositoryContributor'
    person: IPerson
    count: number
    repository: IRepository
    commits: IGitCommitConnection
}

export interface ICommitsOnRepositoryContributorArguments {
    first?: number | null
}

export enum LSIFUploadState {
    PROCESSING = 'PROCESSING',
    ERRORED = 'ERRORED',
    COMPLETED = 'COMPLETED',
    QUEUED = 'QUEUED',
    UPLOADING = 'UPLOADING',
}

export interface ILSIFUploadConnection {
    __typename: 'LSIFUploadConnection'
    nodes: ILSIFUpload[]
    totalCount: number | null
    pageInfo: IPageInfo
}

export interface ILSIFUpload {
    __typename: 'LSIFUpload'
    id: ID
    projectRoot: IGitTree | null
    inputCommit: string
    inputRoot: string
    inputIndexer: string
    state: LSIFUploadState
    uploadedAt: DateTime
    startedAt: DateTime | null
    finishedAt: DateTime | null
    failure: string | null
    isLatestForRepo: boolean
    placeInQueue: number | null
}

export enum LSIFIndexState {
    PROCESSING = 'PROCESSING',
    ERRORED = 'ERRORED',
    COMPLETED = 'COMPLETED',
    QUEUED = 'QUEUED',
}

export interface ILSIFIndexConnection {
    __typename: 'LSIFIndexConnection'
    nodes: ILSIFIndex[]
    totalCount: number | null
    pageInfo: IPageInfo
}

export interface ILSIFIndex {
    __typename: 'LSIFIndex'
    id: ID
    projectRoot: IGitTree | null
    inputCommit: string
    state: LSIFIndexState
    queuedAt: DateTime
    startedAt: DateTime | null
    finishedAt: DateTime | null
    failure: string | null
    placeInQueue: number | null
}

export interface IGitRevisionRange {
    __typename: 'GitRevisionRange'
    expr: string
    base: GitRevSpec
    baseRevSpec: IGitRevSpecExpr
    head: GitRevSpec
    headRevSpec: IGitRevSpecExpr
    mergeBase: IGitObject | null
}

export type GitRevSpec = IGitRef | IGitRevSpecExpr | IGitObject

export interface IGitRevSpecExpr {
    __typename: 'GitRevSpecExpr'
    expr: string
    object: IGitObject | null
}

export interface IFileDiffConnection {
    __typename: 'FileDiffConnection'
    nodes: IFileDiff[]
    totalCount: number | null
    pageInfo: IPageInfo
    diffStat: IDiffStat
    rawDiff: string
}

export interface IFileDiff {
    __typename: 'FileDiff'
    oldPath: string | null
    oldFile: File2 | null
    newPath: string | null
    newFile: File2 | null
    mostRelevantFile: File2
    hunks: IFileDiffHunk[]
    stat: IDiffStat
    internalID: string
}

export interface IFileDiffHunk {
    __typename: 'FileDiffHunk'
    oldRange: IFileDiffHunkRange
    oldNoNewlineAt: boolean
    newRange: IFileDiffHunkRange
    section: string | null
    body: string
    highlight: IHighlightedDiffHunkBody
}

export interface IHighlightOnFileDiffHunkArguments {
    disableTimeout: boolean
    isLightTheme: boolean

    /**
     * @default false
     */
    highlightLongLines?: boolean | null
}

export interface IFileDiffHunkRange {
    __typename: 'FileDiffHunkRange'
    startLine: number
    lines: number
}

export interface IHighlightedDiffHunkBody {
    __typename: 'HighlightedDiffHunkBody'
    aborted: boolean
    lines: IHighlightedDiffHunkLine[]
}

export interface IHighlightedDiffHunkLine {
    __typename: 'HighlightedDiffHunkLine'
    html: string
    kind: DiffHunkLineType
}

export enum DiffHunkLineType {
    ADDED = 'ADDED',
    UNCHANGED = 'UNCHANGED',
    DELETED = 'DELETED',
}

export enum ChangesetState {
    OPEN = 'OPEN',
    CLOSED = 'CLOSED',
    MERGED = 'MERGED',
    DELETED = 'DELETED',
}

export enum ChangesetReviewState {
    APPROVED = 'APPROVED',
    CHANGES_REQUESTED = 'CHANGES_REQUESTED',
    PENDING = 'PENDING',
    COMMENTED = 'COMMENTED',
    DISMISSED = 'DISMISSED',
}

export enum ChangesetCheckState {
    PENDING = 'PENDING',
    PASSED = 'PASSED',
    FAILED = 'FAILED',
}

export interface IChangesetConnection {
    __typename: 'ChangesetConnection'
    nodes: Changeset[]
    totalCount: number
    pageInfo: IPageInfo
}

export type Changeset = ExternalChangeset | IHiddenExternalChangeset

export interface IChangeset {
    __typename: 'Changeset'
    id: ID
    campaigns: ICampaignConnection
    state: ChangesetState
    createdAt: DateTime
    updatedAt: DateTime
    nextSyncAt: DateTime | null
}

export interface ICampaignsOnChangesetArguments {
    first?: number | null
    state?: CampaignState | null
    hasPatchSet?: boolean | null
}

export interface IChangesetCounts {
    __typename: 'ChangesetCounts'
    date: DateTime
    total: number
    merged: number
    closed: number
    open: number
    openApproved: number
    openChangesRequested: number
    openPending: number
}

export type RepositoryRedirect = IRepository | IRedirect

export interface IRedirect {
    __typename: 'Redirect'
    url: string
}

export enum RepositoryOrderBy {
    REPOSITORY_NAME = 'REPOSITORY_NAME',
    REPO_CREATED_AT = 'REPO_CREATED_AT',
    REPOSITORY_CREATED_AT = 'REPOSITORY_CREATED_AT',
}

export interface IRepositoryConnection {
    __typename: 'RepositoryConnection'
    nodes: IRepository[]
    totalCount: number | null
    pageInfo: IPageInfo
}

export interface ITotalCountOnRepositoryConnectionArguments {
    /**
     * @default false
     */
    precise?: boolean | null
}

export interface IPhabricatorRepo {
    __typename: 'PhabricatorRepo'
    name: string

    /**
     * @deprecated "use name instead"
     */
    uri: string
    callsign: string
    url: string
}

export enum UserActivePeriod {
    TODAY = 'TODAY',
    THIS_WEEK = 'THIS_WEEK',
    THIS_MONTH = 'THIS_MONTH',
    ALL_TIME = 'ALL_TIME',
}

export interface IMarkdownOptions {
    alwaysNil?: string | null
}

export interface IClientConfigurationDetails {
    __typename: 'ClientConfigurationDetails'
    contentScriptUrls: string[]
    parentSourcegraph: IParentSourcegraphDetails
}

export interface IParentSourcegraphDetails {
    __typename: 'ParentSourcegraphDetails'
    url: string
}

export interface ISearchFilterSuggestions {
    __typename: 'SearchFilterSuggestions'
    repogroup: string[]
    repo: string[]
}

export enum SearchVersion {
    V1 = 'V1',
    V2 = 'V2',
}

export enum SearchPatternType {
    literal = 'literal',
    regexp = 'regexp',
    structural = 'structural',
}

export interface ISearch {
    __typename: 'Search'
    results: ISearchResults
    suggestions: SearchSuggestion[]
    stats: ISearchResultsStats
}

export interface ISuggestionsOnSearchArguments {
    first?: number | null
}

export interface ISearchResults {
    __typename: 'SearchResults'
    results: SearchResult[]
    matchCount: number

    /**
     * @deprecated "renamed to matchCount for less ambiguity"
     */
    resultCount: number
    approximateResultCount: string
    limitHit: boolean
    sparkline: number[]
    repositories: IRepository[]
    repositoriesCount: number
    repositoriesSearched: IRepository[]
    indexedRepositoriesSearched: IRepository[]
    cloning: IRepository[]
    missing: IRepository[]
    timedout: IRepository[]
    indexUnavailable: boolean
    alert: ISearchAlert | null
    elapsedMilliseconds: number
    dynamicFilters: ISearchFilter[]
    pageInfo: IPageInfo
}

export type SearchResult = IFileMatch | ICommitSearchResult | IRepository | ICodemodResult

export interface IFileMatch {
    __typename: 'FileMatch'
    file: IGitBlob
    repository: IRepository
    revSpec: GitRevSpec | null

    /**
     * @deprecated "use the file field instead"
     */
    resource: string
    symbols: ISymbol[]
    lineMatches: ILineMatch[]
    limitHit: boolean
}

export interface ILineMatch {
    __typename: 'LineMatch'
    preview: string
    lineNumber: number
    offsetAndLengths: number[][]
    limitHit: boolean
}

export interface ICommitSearchResult {
    __typename: 'CommitSearchResult'
    icon: string
    label: IMarkdown
    url: string
    detail: IMarkdown
    matches: ISearchResultMatch[]
    commit: IGitCommit
    refs: IGitRef[]
    sourceRefs: IGitRef[]
    messagePreview: IHighlightedString | null
    diffPreview: IHighlightedString | null
}

export interface IHighlightedString {
    __typename: 'HighlightedString'
    value: string
    highlights: IHighlight[]
}

export interface ICodemodResult {
    __typename: 'CodemodResult'
    icon: string
    label: IMarkdown
    url: string
    detail: IMarkdown
    matches: ISearchResultMatch[]
    commit: IGitCommit
    rawDiff: string
}

export interface ISearchAlert {
    __typename: 'SearchAlert'
    title: string
    description: string | null
    proposedQueries: ISearchQueryDescription[] | null
}

export interface ISearchQueryDescription {
    __typename: 'SearchQueryDescription'
    description: string | null
    query: string
}

export interface ISearchFilter {
    __typename: 'SearchFilter'
    value: string
    label: string
    count: number
    limitHit: boolean
    kind: string
}

export type SearchSuggestion = IRepository | IFile | ISymbol | ILanguage

export interface ILanguage {
    __typename: 'Language'
    name: string
}

export interface ISearchResultsStats {
    __typename: 'SearchResultsStats'
    approximateResultCount: string
    sparkline: number[]
    languages: ILanguageStatistics[]
}

export interface ISavedSearch {
    __typename: 'SavedSearch'
    id: ID
    description: string
    query: string
    notify: boolean
    notifySlack: boolean
    namespace: Namespace
    slackWebhookURL: string | null
}

export interface IRepoGroup {
    __typename: 'RepoGroup'
    name: string
    repositories: string[]
}

export interface IVersionContext {
    __typename: 'VersionContext'
    id: ID
    name: string
    description: string
}

export interface ISite {
    __typename: 'Site'
    id: ID
    siteID: string
    configuration: ISiteConfiguration
    latestSettings: ISettings | null
    settingsCascade: ISettingsCascade

    /**
     * @deprecated "Use settingsCascade instead. This field is a deprecated alias for it and will be removed in a future release."
     */
    configurationCascade: IConfigurationCascade
    settingsURL: string | null
    canReloadSite: boolean
    viewerCanAdminister: boolean
    accessTokens: IAccessTokenConnection
    authProviders: IAuthProviderConnection
    externalAccounts: IExternalAccountConnection
    buildVersion: string
    productVersion: string
    updateCheck: IUpdateCheck
    needsRepositoryConfiguration: boolean
    freeUsersExceeded: boolean

    /**
     * @deprecated "All repositories are enabled by default now. This field is always false."
     */
    noRepositoriesEnabled: boolean
    alerts: IAlert[]
    hasCodeIntelligence: boolean
    disableBuiltInSearches: boolean
    sendsEmailVerificationEmails: boolean
    productSubscription: IProductSubscriptionStatus
    usageStatistics: ISiteUsageStatistics
    monitoringStatistics: IMonitoringStatistics
}

export interface IAccessTokensOnSiteArguments {
    first?: number | null
}

export interface IExternalAccountsOnSiteArguments {
    first?: number | null
    user?: ID | null
    serviceType?: string | null
    serviceID?: string | null
    clientID?: string | null
}

export interface IUsageStatisticsOnSiteArguments {
    days?: number | null
    weeks?: number | null
    months?: number | null
}

export interface IMonitoringStatisticsOnSiteArguments {
    days?: number | null
}

export interface ISiteConfiguration {
    __typename: 'SiteConfiguration'
    id: number
    effectiveContents: JSONCString
    validationMessages: string[]
}

export interface IAuthProviderConnection {
    __typename: 'AuthProviderConnection'
    nodes: IAuthProvider[]
    totalCount: number
    pageInfo: IPageInfo
}

export interface IAuthProvider {
    __typename: 'AuthProvider'
    serviceType: string
    serviceID: string
    clientID: string
    displayName: string
    isBuiltin: boolean
    authenticationURL: string | null
}

export interface IUpdateCheck {
    __typename: 'UpdateCheck'
    pending: boolean
    checkedAt: DateTime | null
    errorMessage: string | null
    updateVersionAvailable: string | null
}

export interface IAlert {
    __typename: 'Alert'
    type: AlertType
    message: string
    isDismissibleWithKey: string | null
}

export enum AlertType {
    INFO = 'INFO',
    WARNING = 'WARNING',
    ERROR = 'ERROR',
}

export interface IProductSubscriptionStatus {
    __typename: 'ProductSubscriptionStatus'
    productNameWithBrand: string
    actualUserCount: number
    actualUserCountDate: string
    maximumAllowedUserCount: number | null
    noLicenseWarningUserCount: number | null
    license: IProductLicenseInfo | null
}

export interface IProductLicenseInfo {
    __typename: 'ProductLicenseInfo'
    productNameWithBrand: string
    tags: string[]
    userCount: number
    expiresAt: DateTime
}

export interface ISiteUsageStatistics {
    __typename: 'SiteUsageStatistics'
    daus: ISiteUsagePeriod[]
    waus: ISiteUsagePeriod[]
    maus: ISiteUsagePeriod[]
}

export interface ISiteUsagePeriod {
    __typename: 'SiteUsagePeriod'
    startTime: string
    userCount: number
    registeredUserCount: number
    anonymousUserCount: number
    integrationUserCount: number
    stages: ISiteUsageStages | null
}

export interface ISiteUsageStages {
    __typename: 'SiteUsageStages'
    manage: number
    plan: number
    code: number
    review: number
    verify: number
    package: number
    deploy: number
    configure: number
    monitor: number
    secure: number
    automate: number
}

export interface IMonitoringStatistics {
    __typename: 'MonitoringStatistics'
    alerts: IMonitoringAlert[]
}

export interface IMonitoringAlert {
    __typename: 'MonitoringAlert'
    timestamp: DateTime
    name: string
    serviceName: string
    average: number
}

export interface ISurveyResponseConnection {
    __typename: 'SurveyResponseConnection'
    nodes: ISurveyResponse[]
    totalCount: number
    last30DaysCount: number
    averageScore: number
    netPromoterScore: number
}

export interface IExtensionRegistry {
    __typename: 'ExtensionRegistry'
    extension: IRegistryExtension | null
    extensions: IRegistryExtensionConnection
    publishers: IRegistryPublisherConnection
    viewerPublishers: RegistryPublisher[]
    localExtensionIDPrefix: string | null
}

export interface IExtensionOnExtensionRegistryArguments {
    extensionID: string
}

export interface IExtensionsOnExtensionRegistryArguments {
    first?: number | null
    publisher?: ID | null
    query?: string | null

    /**
     * @default true
     */
    local?: boolean | null

    /**
     * @default true
     */
    remote?: boolean | null
    prioritizeExtensionIDs?: string[] | null
}

export interface IPublishersOnExtensionRegistryArguments {
    first?: number | null
}

export interface IRegistryExtension {
    __typename: 'RegistryExtension'
    id: ID
    uuid: string
    publisher: RegistryPublisher | null
    extensionID: string
    extensionIDWithoutRegistry: string
    name: string
    manifest: IExtensionManifest | null
    createdAt: DateTime | null
    updatedAt: DateTime | null
    publishedAt: DateTime | null
    url: string
    remoteURL: string | null
    registryName: string
    isLocal: boolean
    isWorkInProgress: boolean
    viewerCanAdminister: boolean
}

export type RegistryPublisher = IUser | IOrg

export interface IExtensionManifest {
    __typename: 'ExtensionManifest'
    raw: string
    description: string | null
    bundleURL: string | null
}

export interface IRegistryExtensionConnection {
    __typename: 'RegistryExtensionConnection'
    nodes: IRegistryExtension[]
    totalCount: number
    pageInfo: IPageInfo
    url: string | null
    error: string | null
}

export interface IRegistryPublisherConnection {
    __typename: 'RegistryPublisherConnection'
    nodes: RegistryPublisher[]
    totalCount: number
    pageInfo: IPageInfo
}

export interface IDotcomQuery {
    __typename: 'DotcomQuery'
    productSubscription: IProductSubscription
    productSubscriptions: IProductSubscriptionConnection
    previewProductSubscriptionInvoice: IProductSubscriptionPreviewInvoice
    productLicenses: IProductLicenseConnection
    productPlans: IProductPlan[]
}

export interface IProductSubscriptionOnDotcomQueryArguments {
    uuid: string
}

export interface IProductSubscriptionsOnDotcomQueryArguments {
    first?: number | null
    account?: ID | null
    query?: string | null
}

export interface IPreviewProductSubscriptionInvoiceOnDotcomQueryArguments {
    account?: ID | null
    subscriptionToUpdate?: ID | null
    productSubscription: IProductSubscriptionInput
}

export interface IProductLicensesOnDotcomQueryArguments {
    first?: number | null
    licenseKeySubstring?: string | null
    productSubscriptionID?: ID | null
}

export interface IProductSubscription {
    __typename: 'ProductSubscription'
    id: ID
    uuid: string
    name: string
    account: IUser | null
    invoiceItem: IProductSubscriptionInvoiceItem | null
    events: IProductSubscriptionEvent[]
    activeLicense: IProductLicense | null
    productLicenses: IProductLicenseConnection
    createdAt: DateTime
    isArchived: boolean
    url: string
    urlForSiteAdmin: string | null
    urlForSiteAdminBilling: string | null
}

export interface IProductLicensesOnProductSubscriptionArguments {
    first?: number | null
}

export interface IProductSubscriptionInvoiceItem {
    __typename: 'ProductSubscriptionInvoiceItem'
    plan: IProductPlan
    userCount: number
    expiresAt: DateTime
}

export interface IProductPlan {
    __typename: 'ProductPlan'
    billingPlanID: string
    productPlanID: string
    name: string
    nameWithBrand: string
    pricePerUserPerYear: number
    minQuantity: number | null
    maxQuantity: number | null
    tiersMode: string
    planTiers: IPlanTier[]
}

export interface IPlanTier {
    __typename: 'PlanTier'
    unitAmount: number
    upTo: number
    flatAmount: number
}

export interface IProductSubscriptionEvent {
    __typename: 'ProductSubscriptionEvent'
    id: string
    date: string
    title: string
    description: string | null
    url: string | null
}

export interface IProductLicense {
    __typename: 'ProductLicense'
    id: ID
    subscription: IProductSubscription
    info: IProductLicenseInfo | null
    licenseKey: string
    createdAt: DateTime
}

export interface IProductLicenseConnection {
    __typename: 'ProductLicenseConnection'
    nodes: IProductLicense[]
    totalCount: number
    pageInfo: IPageInfo
}

export interface IProductSubscriptionConnection {
    __typename: 'ProductSubscriptionConnection'
    nodes: IProductSubscription[]
    totalCount: number
    pageInfo: IPageInfo
}

export interface IProductSubscriptionInput {
    billingPlanID: string
    userCount: number
}

export interface IProductSubscriptionPreviewInvoice {
    __typename: 'ProductSubscriptionPreviewInvoice'
    price: number
    prorationDate: string | null
    isDowngradeRequiringManualIntervention: boolean
    beforeInvoiceItem: IProductSubscriptionInvoiceItem | null
    afterInvoiceItem: IProductSubscriptionInvoiceItem
}

export type StatusMessage = ICloningProgress | IExternalServiceSyncError | ISyncError

export interface ICloningProgress {
    __typename: 'CloningProgress'
    message: string
}

export interface IExternalServiceSyncError {
    __typename: 'ExternalServiceSyncError'
    message: string
    externalService: IExternalService
}

export interface ISyncError {
    __typename: 'SyncError'
    message: string
}

export interface Mutation {
    __typename: 'Mutation'
    createChangesets: ExternalChangeset[]
    addChangesetsToCampaign: ICampaign
    createCampaign: ICampaign
    createPatchSetFromPatches: IPatchSet
    updateCampaign: ICampaign
    retryCampaignChangesets: ICampaign
    deleteCampaign: IEmptyResponse | null
    closeCampaign: ICampaign
    publishCampaignChangesets: ICampaign
    publishChangeset: IEmptyResponse
    syncChangeset: IEmptyResponse
    updateUser: IEmptyResponse
    createOrganization: IOrg
    updateOrganization: IOrg
    deleteOrganization: IEmptyResponse | null
    addExternalService: IExternalService
    updateExternalService: IExternalService
    deleteExternalService: IEmptyResponse

    /**
     * @deprecated "update external service exclude setting."
     */
    setRepositoryEnabled: IEmptyResponse | null
    checkMirrorRepositoryConnection: ICheckMirrorRepositoryConnectionResult
    updateMirrorRepository: IEmptyResponse

    /**
     * @deprecated "syncer ensures all repositories are up to date."
     */
    updateAllMirrorRepositories: IEmptyResponse
    createUser: ICreateUserResult
    randomizeUserPassword: IRandomizeUserPasswordResult
    addUserEmail: IEmptyResponse
    removeUserEmail: IEmptyResponse
    setUserEmailVerified: IEmptyResponse
    deleteUser: IEmptyResponse | null
    updatePassword: IEmptyResponse | null
    createAccessToken: ICreateAccessTokenResult
    deleteAccessToken: IEmptyResponse
    deleteExternalAccount: IEmptyResponse
    inviteUserToOrganization: IInviteUserToOrganizationResult
    respondToOrganizationInvitation: IEmptyResponse
    resendOrganizationInvitationNotification: IEmptyResponse
    revokeOrganizationInvitation: IEmptyResponse
    addUserToOrganization: IEmptyResponse
    removeUserFromOrganization: IEmptyResponse | null
    setTag: IEmptyResponse
    addPhabricatorRepo: IEmptyResponse | null
    resolvePhabricatorDiff: IGitCommit | null

    /**
     * @deprecated "use logEvent instead"
     */
    logUserEvent: IEmptyResponse | null
    logEvent: IEmptyResponse | null
    sendSavedSearchTestNotification: IEmptyResponse | null
    settingsMutation: ISettingsMutation | null

    /**
     * @deprecated "use settingsMutation instead"
     */
    configurationMutation: ISettingsMutation | null
    updateSiteConfiguration: boolean
    setUserIsSiteAdmin: IEmptyResponse | null
    reloadSite: IEmptyResponse | null
    submitSurvey: IEmptyResponse | null
    requestTrial: IEmptyResponse | null
    extensionRegistry: IExtensionRegistryMutation
    dotcom: IDotcomMutation
    createSavedSearch: ISavedSearch
    updateSavedSearch: ISavedSearch
    deleteSavedSearch: IEmptyResponse | null
    deleteLSIFUpload: IEmptyResponse | null
    deleteLSIFIndex: IEmptyResponse | null
    setRepositoryPermissionsForUsers: IEmptyResponse
    scheduleRepositoryPermissionsSync: IEmptyResponse
    scheduleUserPermissionsSync: IEmptyResponse
}

export interface ICreateChangesetsOnMutationArguments {
    input: ICreateChangesetInput[]
}

export interface IAddChangesetsToCampaignOnMutationArguments {
    campaign: ID
    changesets: ID[]
}

export interface ICreateCampaignOnMutationArguments {
    input: ICreateCampaignInput
}

export interface ICreatePatchSetFromPatchesOnMutationArguments {
    patches: IPatchInput[]
}

export interface IUpdateCampaignOnMutationArguments {
    input: IUpdateCampaignInput
}

export interface IRetryCampaignChangesetsOnMutationArguments {
    campaign: ID
}

export interface IDeleteCampaignOnMutationArguments {
    campaign: ID

    /**
     * @default false
     */
    closeChangesets?: boolean | null
}

export interface ICloseCampaignOnMutationArguments {
    campaign: ID

    /**
     * @default false
     */
    closeChangesets?: boolean | null
}

export interface IPublishCampaignChangesetsOnMutationArguments {
    campaign: ID
}

export interface IPublishChangesetOnMutationArguments {
    patch: ID
}

export interface ISyncChangesetOnMutationArguments {
    changeset: ID
}

export interface IUpdateUserOnMutationArguments {
    user: ID
    username?: string | null
    displayName?: string | null
    avatarURL?: string | null
}

export interface ICreateOrganizationOnMutationArguments {
    name: string
    displayName?: string | null
}

export interface IUpdateOrganizationOnMutationArguments {
    id: ID
    displayName?: string | null
}

export interface IDeleteOrganizationOnMutationArguments {
    organization: ID
}

export interface IAddExternalServiceOnMutationArguments {
    input: IAddExternalServiceInput
}

export interface IUpdateExternalServiceOnMutationArguments {
    input: IUpdateExternalServiceInput
}

export interface IDeleteExternalServiceOnMutationArguments {
    externalService: ID
}

export interface ISetRepositoryEnabledOnMutationArguments {
    repository: ID
    enabled: boolean
}

export interface ICheckMirrorRepositoryConnectionOnMutationArguments {
    repository?: ID | null
    name?: string | null
}

export interface IUpdateMirrorRepositoryOnMutationArguments {
    repository: ID
}

export interface ICreateUserOnMutationArguments {
    username: string
    email?: string | null
}

export interface IRandomizeUserPasswordOnMutationArguments {
    user: ID
}

export interface IAddUserEmailOnMutationArguments {
    user: ID
    email: string
}

export interface IRemoveUserEmailOnMutationArguments {
    user: ID
    email: string
}

export interface ISetUserEmailVerifiedOnMutationArguments {
    user: ID
    email: string
    verified: boolean
}

export interface IDeleteUserOnMutationArguments {
    user: ID
    hard?: boolean | null
}

export interface IUpdatePasswordOnMutationArguments {
    oldPassword: string
    newPassword: string
}

export interface ICreateAccessTokenOnMutationArguments {
    user: ID
    scopes: string[]
    note: string
}

export interface IDeleteAccessTokenOnMutationArguments {
    byID?: ID | null
    byToken?: string | null
}

export interface IDeleteExternalAccountOnMutationArguments {
    externalAccount: ID
}

export interface IInviteUserToOrganizationOnMutationArguments {
    organization: ID
    username: string
}

export interface IRespondToOrganizationInvitationOnMutationArguments {
    organizationInvitation: ID
    responseType: OrganizationInvitationResponseType
}

export interface IResendOrganizationInvitationNotificationOnMutationArguments {
    organizationInvitation: ID
}

export interface IRevokeOrganizationInvitationOnMutationArguments {
    organizationInvitation: ID
}

export interface IAddUserToOrganizationOnMutationArguments {
    organization: ID
    username: string
}

export interface IRemoveUserFromOrganizationOnMutationArguments {
    user: ID
    organization: ID
}

export interface ISetTagOnMutationArguments {
    node: ID
    tag: string
    present: boolean
}

export interface IAddPhabricatorRepoOnMutationArguments {
    callsign: string
    name?: string | null
    uri?: string | null
    url: string
}

export interface IResolvePhabricatorDiffOnMutationArguments {
    repoName: string
    diffID: ID
    baseRev: string
    patch?: string | null
    description?: string | null
    authorName?: string | null
    authorEmail?: string | null
    date?: string | null
}

export interface ILogUserEventOnMutationArguments {
    event: UserEvent
    userCookieID: string
}

export interface ILogEventOnMutationArguments {
    event: string
    userCookieID: string
    url: string
    source: EventSource
    argument?: string | null
}

export interface ISendSavedSearchTestNotificationOnMutationArguments {
    id: ID
}

export interface ISettingsMutationOnMutationArguments {
    input: ISettingsMutationGroupInput
}

export interface IConfigurationMutationOnMutationArguments {
    input: ISettingsMutationGroupInput
}

export interface IUpdateSiteConfigurationOnMutationArguments {
    lastID: number
    input: string
}

export interface ISetUserIsSiteAdminOnMutationArguments {
    userID: ID
    siteAdmin: boolean
}

export interface ISubmitSurveyOnMutationArguments {
    input: ISurveySubmissionInput
}

export interface IRequestTrialOnMutationArguments {
    email: string
}

export interface ICreateSavedSearchOnMutationArguments {
    description: string
    query: string
    notifyOwner: boolean
    notifySlack: boolean
    orgID?: ID | null
    userID?: ID | null
}

export interface IUpdateSavedSearchOnMutationArguments {
    id: ID
    description: string
    query: string
    notifyOwner: boolean
    notifySlack: boolean
    orgID?: ID | null
    userID?: ID | null
}

export interface IDeleteSavedSearchOnMutationArguments {
    id: ID
}

export interface IDeleteLSIFUploadOnMutationArguments {
    id: ID
}

export interface IDeleteLSIFIndexOnMutationArguments {
    id: ID
}

export interface ISetRepositoryPermissionsForUsersOnMutationArguments {
    repository: ID
    userPermissions: IUserPermission[]
}

export interface IScheduleRepositoryPermissionsSyncOnMutationArguments {
    repository: ID
}

export interface IScheduleUserPermissionsSyncOnMutationArguments {
    user: ID
}

export interface ICreateChangesetInput {
    repository: ID
    externalID: string
}

export interface ExternalChangeset {
    __typename: 'ExternalChangeset'
    id: ID
    externalID: string
    repository: IRepository
    campaigns: ICampaignConnection
    events: IChangesetEventConnection
    createdAt: DateTime
    updatedAt: DateTime
    nextSyncAt: DateTime | null
    title: string
    body: string
    state: ChangesetState
    labels: IChangesetLabel[]
    externalURL: IExternalLink
    reviewState: ChangesetReviewState
    base: IGitRef | null
    head: IGitRef | null
    diff: RepositoryComparison | null
    diffStat: IDiffStat | null
    checkState: ChangesetCheckState | null
}

export interface ICampaignsOnExternalChangesetArguments {
    first?: number | null
    state?: CampaignState | null
    hasPatchSet?: boolean | null
    viewerCanAdminister?: boolean | null
}

export interface IEventsOnExternalChangesetArguments {
    first?: number | null
}

export interface IChangesetEventConnection {
    __typename: 'ChangesetEventConnection'
    nodes: IChangesetEvent[]
    totalCount: number
    pageInfo: IPageInfo
}

export interface IChangesetEvent {
    __typename: 'ChangesetEvent'
    id: ID
    changeset: ExternalChangeset
    createdAt: DateTime
}

export interface IChangesetLabel {
    __typename: 'ChangesetLabel'
    text: string
    color: string
    description: string | null
}

export interface ICreateCampaignInput {
    namespace: ID
    name: string
    description?: string | null
    branch?: string | null
    patchSet?: ID | null
}

export interface IPatchInput {
    repository: ID
    baseRevision: string
    baseRef: string
    patch: string
}

export interface IUpdateCampaignInput {
    id: ID
    name?: string | null
    branch?: string | null
    description?: string | null
    patchSet?: ID | null
}

export interface IEmptyResponse {
    __typename: 'EmptyResponse'
    alwaysNil: string | null
}

export interface IAddExternalServiceInput {
    kind: ExternalServiceKind
    displayName: string
    config: string
}

export interface IUpdateExternalServiceInput {
    id: ID
    displayName?: string | null
    config?: string | null
}

export interface ICheckMirrorRepositoryConnectionResult {
    __typename: 'CheckMirrorRepositoryConnectionResult'
    error: string | null
}

export interface ICreateUserResult {
    __typename: 'CreateUserResult'
    user: IUser
    resetPasswordURL: string | null
}

export interface IRandomizeUserPasswordResult {
    __typename: 'RandomizeUserPasswordResult'
    resetPasswordURL: string | null
}

export interface ICreateAccessTokenResult {
    __typename: 'CreateAccessTokenResult'
    id: ID
    token: string
}

export interface IInviteUserToOrganizationResult {
    __typename: 'InviteUserToOrganizationResult'
    sentInvitationEmail: boolean
    invitationURL: string
}

export enum UserEvent {
    PAGEVIEW = 'PAGEVIEW',
    SEARCHQUERY = 'SEARCHQUERY',
    CODEINTEL = 'CODEINTEL',
    CODEINTELREFS = 'CODEINTELREFS',
    CODEINTELINTEGRATION = 'CODEINTELINTEGRATION',
    CODEINTELINTEGRATIONREFS = 'CODEINTELINTEGRATIONREFS',
    STAGEMANAGE = 'STAGEMANAGE',
    STAGEPLAN = 'STAGEPLAN',
    STAGECODE = 'STAGECODE',
    STAGEREVIEW = 'STAGEREVIEW',
    STAGEVERIFY = 'STAGEVERIFY',
    STAGEPACKAGE = 'STAGEPACKAGE',
    STAGEDEPLOY = 'STAGEDEPLOY',
    STAGECONFIGURE = 'STAGECONFIGURE',
    STAGEMONITOR = 'STAGEMONITOR',
    STAGESECURE = 'STAGESECURE',
    STAGEAUTOMATE = 'STAGEAUTOMATE',
}

export interface ISettingsMutationGroupInput {
    subject: ID
    lastID?: number | null
}

export interface ISettingsMutation {
    __typename: 'SettingsMutation'
    editSettings: IUpdateSettingsPayload | null

    /**
     * @deprecated "Use editSettings instead. This field is a deprecated alias for it and will be removed in a future release."
     */
    editConfiguration: IUpdateSettingsPayload | null
    overwriteSettings: IUpdateSettingsPayload | null
}

export interface IEditSettingsOnSettingsMutationArguments {
    edit: ISettingsEdit
}

export interface IEditConfigurationOnSettingsMutationArguments {
    edit: IConfigurationEdit
}

export interface IOverwriteSettingsOnSettingsMutationArguments {
    contents: string
}

export interface ISettingsEdit {
    keyPath: IKeyPathSegment[]
    value?: any | null

    /**
     * @default false
     */
    valueIsJSONCEncodedString?: boolean | null
}

export interface IKeyPathSegment {
    property?: string | null
    index?: number | null
}

export interface IUpdateSettingsPayload {
    __typename: 'UpdateSettingsPayload'
    empty: IEmptyResponse | null
}

export interface IConfigurationEdit {
    keyPath: IKeyPathSegment[]
    value?: any | null

    /**
     * @default false
     */
    valueIsJSONCEncodedString?: boolean | null
}

export interface ISurveySubmissionInput {
    email?: string | null
    score: number
    reason?: string | null
    better?: string | null
}

export interface IExtensionRegistryMutation {
    __typename: 'ExtensionRegistryMutation'
    createExtension: IExtensionRegistryCreateExtensionResult
    updateExtension: IExtensionRegistryUpdateExtensionResult
    deleteExtension: IEmptyResponse
    publishExtension: IExtensionRegistryCreateExtensionResult
}

export interface ICreateExtensionOnExtensionRegistryMutationArguments {
    publisher: ID
    name: string
}

export interface IUpdateExtensionOnExtensionRegistryMutationArguments {
    extension: ID
    name?: string | null
}

export interface IDeleteExtensionOnExtensionRegistryMutationArguments {
    extension: ID
}

export interface IPublishExtensionOnExtensionRegistryMutationArguments {
    extensionID: string
    manifest: string
    bundle?: string | null
    sourceMap?: string | null

    /**
     * @default false
     */
    force?: boolean | null
}

export interface IExtensionRegistryCreateExtensionResult {
    __typename: 'ExtensionRegistryCreateExtensionResult'
    extension: IRegistryExtension
}

export interface IExtensionRegistryUpdateExtensionResult {
    __typename: 'ExtensionRegistryUpdateExtensionResult'
    extension: IRegistryExtension
}

export interface IDotcomMutation {
    __typename: 'DotcomMutation'
    setUserBilling: IEmptyResponse
    createProductSubscription: IProductSubscription
    setProductSubscriptionBilling: IEmptyResponse
    generateProductLicenseForSubscription: IProductLicense
    createPaidProductSubscription: ICreatePaidProductSubscriptionResult
    updatePaidProductSubscription: IUpdatePaidProductSubscriptionResult
    archiveProductSubscription: IEmptyResponse
}

export interface ISetUserBillingOnDotcomMutationArguments {
    user: ID
    billingCustomerID?: string | null
}

export interface ICreateProductSubscriptionOnDotcomMutationArguments {
    accountID: ID
}

export interface ISetProductSubscriptionBillingOnDotcomMutationArguments {
    id: ID
    billingSubscriptionID?: string | null
}

export interface IGenerateProductLicenseForSubscriptionOnDotcomMutationArguments {
    productSubscriptionID: ID
    license: IProductLicenseInput
}

export interface ICreatePaidProductSubscriptionOnDotcomMutationArguments {
    accountID: ID
    productSubscription: IProductSubscriptionInput
    paymentToken?: string | null
}

export interface IUpdatePaidProductSubscriptionOnDotcomMutationArguments {
    subscriptionID: ID
    update: IProductSubscriptionInput
    paymentToken?: string | null
}

export interface IArchiveProductSubscriptionOnDotcomMutationArguments {
    id: ID
}

export interface IProductLicenseInput {
    tags: string[]
    userCount: number
    expiresAt: number
}

export interface ICreatePaidProductSubscriptionResult {
    __typename: 'CreatePaidProductSubscriptionResult'
    productSubscription: IProductSubscription
}

export interface IUpdatePaidProductSubscriptionResult {
    __typename: 'UpdatePaidProductSubscriptionResult'
    productSubscription: IProductSubscription
}

export interface IUserPermission {
    bindID: string

    /**
     * @default "READ"
     */
    permission?: RepositoryPermission | null
}

export interface IPatch {
    __typename: 'Patch'
    id: ID
    repository: IRepository
    diff: IPreviewRepositoryComparison
    publicationEnqueued: boolean
    publishable: boolean
}

export interface IPreviewRepositoryComparison {
    __typename: 'PreviewRepositoryComparison'
    baseRepository: IRepository
    fileDiffs: IFileDiffConnection
}

export interface IFileDiffsOnPreviewRepositoryComparisonArguments {
    first?: number | null
    after?: string | null
}

export interface IHiddenPatch {
    __typename: 'HiddenPatch'
    id: ID
}

export interface IHiddenExternalChangeset {
    __typename: 'HiddenExternalChangeset'
    id: ID
    campaigns: ICampaignConnection
    state: ChangesetState
    createdAt: DateTime
    updatedAt: DateTime
    nextSyncAt: DateTime | null
}

export interface ICampaignsOnHiddenExternalChangesetArguments {
    first?: number | null
    state?: CampaignState | null
    hasPatchSet?: boolean | null
}

export interface IDiff {
    __typename: 'Diff'
    repository: IRepository
    range: IGitRevisionRange
}

export interface IDiffSearchResult {
    __typename: 'DiffSearchResult'
    diff: IDiff
    preview: IHighlightedString
}

export interface IVirtualFile {
    __typename: 'VirtualFile'
    path: string
    name: string
    isDirectory: boolean
    content: string
    byteSize: number
    binary: boolean
    richHTML: string
    url: string
    canonicalURL: string
    externalURLs: IExternalLink[]
    highlight: IHighlightedFile
}

export interface IHighlightOnVirtualFileArguments {
    disableTimeout: boolean
    isLightTheme: boolean

    /**
     * @default false
     */
    highlightLongLines?: boolean | null
}

export interface IDefaultSettings {
    __typename: 'DefaultSettings'
    id: ID
    latestSettings: ISettings | null
    settingsURL: string | null
    viewerCanAdminister: boolean
    settingsCascade: ISettingsCascade

    /**
     * @deprecated "Use settingsCascade instead. This field is a deprecated alias for it and will be removed in a future release."
     */
    configurationCascade: IConfigurationCascade
}

export interface IDeploymentConfiguration {
    __typename: 'DeploymentConfiguration'
    email: string | null
    siteID: string | null
}

export interface IExtensionRegistryPublishExtensionResult {
    __typename: 'ExtensionRegistryPublishExtensionResult'
    extension: IRegistryExtension
}
