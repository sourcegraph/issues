package db

var (
	AccessTokens     = &accessTokens{}
	ExternalServices = &ExternalServicesStore{}
	DefaultRepos     = &defaultRepos{}
	Repos            = &repos{}
	Phabricator      = &phabricator{}
	QueryRunnerState = &queryRunnerState{}
	Namespaces       = &namespaces{}
	Orgs             = &orgs{}
	OrgMembers       = &orgMembers{}
	SavedSearches    = &savedSearches{}
	Settings         = &settings{}
	Users            = &users{}
	UserEmails       = &userEmails{}
	EventLogs        = &eventLogs{}

	SurveyResponses = &surveyResponses{}

	ExternalAccounts = &userExternalAccounts{}

	OrgInvitations = &orgInvitations{}

	Authz AuthzStore = &authzStore{}

	Secrets = &secrets{}
)
