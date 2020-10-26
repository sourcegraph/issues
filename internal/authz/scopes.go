package authz

const (
	// Access token scopes.
	ScopeUserAll       = "user:all"        // Full control of all resources accessible to the user account.
	ScopeSiteAdminSudo = "site-admin:sudo" // Ability to perform any action as any other user.
)

// AllScopes is a list of all known access token scopes.
var AllScopes = []string{
	ScopeUserAll,
	ScopeSiteAdminSudo,
}
