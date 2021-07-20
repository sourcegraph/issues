package searchcontexts

import (
	"context"
	"regexp"
	"strings"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/lazyregexp"
	"github.com/sourcegraph/sourcegraph/internal/search"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/schema"
)

const (
	GlobalSearchContextName           = "global"
	searchContextSpecPrefix           = "@"
	maxSearchContextNameLength        = 32
	maxSearchContextDescriptionLength = 1024
	maxRevisionLength                 = 255
)

var (
	validateSearchContextNameRegexp   = lazyregexp.New(`^[a-zA-Z0-9_\-\/\.]+$`)
	namespacedSearchContextSpecRegexp = lazyregexp.New(searchContextSpecPrefix + `(.*?)\/(.*)`)
)

type ParsedSearchContextSpec struct {
	NamespaceName     string
	SearchContextName string
}

func ParseSearchContextSpec(searchContextSpec string) ParsedSearchContextSpec {
	if submatches := namespacedSearchContextSpecRegexp.FindStringSubmatch(searchContextSpec); submatches != nil {
		// We expect 3 submatches, because FindStringSubmatch returns entire string as first submatch, and 2 captured groups
		// as additional submatches
		namespaceName, searchContextName := submatches[1], submatches[2]
		return ParsedSearchContextSpec{NamespaceName: namespaceName, SearchContextName: searchContextName}
	} else if strings.HasPrefix(searchContextSpec, searchContextSpecPrefix) {
		return ParsedSearchContextSpec{NamespaceName: searchContextSpec[1:]}
	}
	return ParsedSearchContextSpec{SearchContextName: searchContextSpec}
}

func ResolveSearchContextSpec(ctx context.Context, db dbutil.DB, searchContextSpec string) (*types.SearchContext, error) {
	parsedSearchContextSpec := ParseSearchContextSpec(searchContextSpec)
	hasNamespaceName := parsedSearchContextSpec.NamespaceName != ""
	hasSearchContextName := parsedSearchContextSpec.SearchContextName != ""

	if IsGlobalSearchContextSpec(searchContextSpec) {
		return GetGlobalSearchContext(), nil
	} else if hasNamespaceName && hasSearchContextName {
		namespace, err := database.Namespaces(db).GetByName(ctx, parsedSearchContextSpec.NamespaceName)
		if err != nil {
			return nil, err
		}
		return database.SearchContexts(db).GetSearchContext(ctx, database.GetSearchContextOptions{
			Name:            parsedSearchContextSpec.SearchContextName,
			NamespaceUserID: namespace.User,
			NamespaceOrgID:  namespace.Organization,
		})
	} else if hasNamespaceName && !hasSearchContextName {
		namespace, err := database.Namespaces(db).GetByName(ctx, parsedSearchContextSpec.NamespaceName)
		if err != nil {
			return nil, err
		}
		if namespace.User == 0 {
			return nil, errors.Errorf("search context %q not found", searchContextSpec)
		}
		return GetUserSearchContext(parsedSearchContextSpec.NamespaceName, namespace.User), nil
	}
	// Check if instance-level context
	return database.SearchContexts(db).GetSearchContext(ctx, database.GetSearchContextOptions{Name: parsedSearchContextSpec.SearchContextName})
}

func ValidateSearchContextWriteAccessForCurrentUser(ctx context.Context, db dbutil.DB, namespaceUserID, namespaceOrgID int32, public bool) error {
	if namespaceUserID != 0 && namespaceOrgID != 0 {
		return errors.New("namespaceUserID and namespaceOrgID are mutually exclusive")
	}

	user, err := backend.CurrentUser(ctx, db)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("current user not found")
	}

	// Site-admins have write access to all public search contexts
	if user.SiteAdmin && public {
		return nil
	}

	if namespaceUserID == 0 && namespaceOrgID == 0 && !user.SiteAdmin {
		// Only site-admins have write access to instance-level search contexts
		return errors.New("current user must be site-admin")
	} else if namespaceUserID != 0 && namespaceUserID != user.ID {
		// Only the creator of the search context has write access to its search contexts
		return errors.New("search context user does not match current user")
	} else if namespaceOrgID != 0 {
		// Only members of the org have write access to org search contexts
		membership, err := database.OrgMembers(db).GetByOrgIDAndUserID(ctx, namespaceOrgID, user.ID)
		if err != nil {
			return err
		}
		if membership == nil {
			return errors.New("current user is not an org member")
		}
	}

	return nil
}

func validateSearchContextName(name string) error {
	if len(name) > maxSearchContextNameLength {
		return errors.Errorf("search context name %q exceeds maximum allowed length (%d)", name, maxSearchContextNameLength)
	}

	if !validateSearchContextNameRegexp.MatchString(name) {
		return errors.Errorf("%q is not a valid search context name", name)
	}

	return nil
}

func validateSearchContextDescription(description string) error {
	if len(description) > maxSearchContextDescriptionLength {
		return errors.Errorf("search context description exceeds maximum allowed length (%d)", maxSearchContextDescriptionLength)
	}
	return nil
}

func validateSearchContextRepositoryRevisions(repositoryRevisions []*types.SearchContextRepositoryRevisions) error {
	for _, repository := range repositoryRevisions {
		for _, revision := range repository.Revisions {
			if len(revision) > maxRevisionLength {
				return errors.Errorf("revision %q exceeds maximum allowed length (%d)", revision, maxRevisionLength)
			}
		}
	}
	return nil
}

func validateSearchContextDoesNotExist(ctx context.Context, db dbutil.DB, searchContext *types.SearchContext) error {
	_, err := database.SearchContexts(db).GetSearchContext(ctx, database.GetSearchContextOptions{
		Name:            searchContext.Name,
		NamespaceUserID: searchContext.NamespaceUserID,
		NamespaceOrgID:  searchContext.NamespaceOrgID,
	})
	if err == nil {
		return errors.New("search context already exists")
	}
	if err == database.ErrSearchContextNotFound {
		return nil
	}
	// Unknown error
	return err
}

func CreateSearchContextWithRepositoryRevisions(ctx context.Context, db dbutil.DB, searchContext *types.SearchContext, repositoryRevisions []*types.SearchContextRepositoryRevisions) (*types.SearchContext, error) {
	if IsGlobalSearchContext(searchContext) {
		return nil, errors.New("cannot override global search context")
	}

	err := ValidateSearchContextWriteAccessForCurrentUser(ctx, db, searchContext.NamespaceUserID, searchContext.NamespaceOrgID, searchContext.Public)
	if err != nil {
		return nil, err
	}

	err = validateSearchContextName(searchContext.Name)
	if err != nil {
		return nil, err
	}

	err = validateSearchContextDescription(searchContext.Description)
	if err != nil {
		return nil, err
	}

	err = validateSearchContextRepositoryRevisions(repositoryRevisions)
	if err != nil {
		return nil, err
	}

	err = validateSearchContextDoesNotExist(ctx, db, searchContext)
	if err != nil {
		return nil, err
	}

	searchContext, err = database.SearchContexts(db).CreateSearchContextWithRepositoryRevisions(ctx, searchContext, repositoryRevisions)
	if err != nil {
		return nil, err
	}
	return searchContext, nil
}

func UpdateSearchContextWithRepositoryRevisions(ctx context.Context, db dbutil.DB, searchContext *types.SearchContext, repositoryRevisions []*types.SearchContextRepositoryRevisions) (*types.SearchContext, error) {
	if IsGlobalSearchContext(searchContext) {
		return nil, errors.New("cannot update global search context")
	}

	err := ValidateSearchContextWriteAccessForCurrentUser(ctx, db, searchContext.NamespaceUserID, searchContext.NamespaceOrgID, searchContext.Public)
	if err != nil {
		return nil, err
	}

	err = validateSearchContextName(searchContext.Name)
	if err != nil {
		return nil, err
	}

	err = validateSearchContextDescription(searchContext.Description)
	if err != nil {
		return nil, err
	}

	err = validateSearchContextRepositoryRevisions(repositoryRevisions)
	if err != nil {
		return nil, err
	}

	searchContext, err = database.SearchContexts(db).UpdateSearchContextWithRepositoryRevisions(ctx, searchContext, repositoryRevisions)
	if err != nil {
		return nil, err
	}
	return searchContext, nil
}

func DeleteSearchContext(ctx context.Context, db dbutil.DB, searchContext *types.SearchContext) error {
	if IsAutoDefinedSearchContext(searchContext) {
		return errors.New("cannot delete auto-defined search context")
	}

	err := ValidateSearchContextWriteAccessForCurrentUser(ctx, db, searchContext.NamespaceUserID, searchContext.NamespaceOrgID, searchContext.Public)
	if err != nil {
		return err
	}

	return database.SearchContexts(db).DeleteSearchContext(ctx, searchContext.ID)
}

func GetAutoDefinedSearchContexts(ctx context.Context, db dbutil.DB) ([]*types.SearchContext, error) {
	searchContexts := []*types.SearchContext{GetGlobalSearchContext()}
	a := actor.FromContext(ctx)
	if a.IsAuthenticated() && envvar.SourcegraphDotComMode() {
		user, err := database.Users(db).GetByID(ctx, a.UID)
		if err != nil {
			return nil, err
		}
		searchContexts = append(searchContexts, GetUserSearchContext(user.Username, a.UID))
	}
	return searchContexts, nil
}

func GetRepositoryRevisions(ctx context.Context, db dbutil.DB, searchContextID int64) ([]*search.RepositoryRevisions, error) {
	searchContextRepositoryRevisions, err := database.SearchContexts(db).GetSearchContextRepositoryRevisions(ctx, searchContextID)
	if err != nil {
		return nil, err
	}

	repositoryRevisions := make([]*search.RepositoryRevisions, 0, len(searchContextRepositoryRevisions))
	for _, searchContextRepositoryRevision := range searchContextRepositoryRevisions {
		revisionSpecs := make([]search.RevisionSpecifier, 0, len(searchContextRepositoryRevision.Revisions))
		for _, revision := range searchContextRepositoryRevision.Revisions {
			revisionSpecs = append(revisionSpecs, search.RevisionSpecifier{RevSpec: revision})
		}
		repositoryRevisions = append(repositoryRevisions, &search.RepositoryRevisions{Repo: searchContextRepositoryRevision.Repo, Revs: revisionSpecs})
	}
	return repositoryRevisions, nil
}

func IsAutoDefinedSearchContext(searchContext *types.SearchContext) bool {
	return searchContext.ID == 0
}

func IsInstanceLevelSearchContext(searchContext *types.SearchContext) bool {
	return searchContext.NamespaceUserID == 0 && searchContext.NamespaceOrgID == 0
}

func IsGlobalSearchContextSpec(searchContextSpec string) bool {
	// Empty search context spec resolves to global search context
	return searchContextSpec == "" || searchContextSpec == GlobalSearchContextName
}

func IsGlobalSearchContext(searchContext *types.SearchContext) bool {
	return searchContext != nil && searchContext.Name == GlobalSearchContextName
}

func GetUserSearchContext(name string, userID int32) *types.SearchContext {
	return &types.SearchContext{Name: name, Public: true, Description: "All repositories you've added to Sourcegraph", NamespaceUserID: userID}
}

func GetGlobalSearchContext() *types.SearchContext {
	return &types.SearchContext{Name: GlobalSearchContextName, Public: true, Description: "All repositories on Sourcegraph"}
}

func GetSearchContextSpec(searchContext *types.SearchContext) string {
	if IsInstanceLevelSearchContext(searchContext) {
		return searchContext.Name
	} else if IsAutoDefinedSearchContext(searchContext) {
		return searchContextSpecPrefix + searchContext.Name
	} else {
		var namespaceName string
		if searchContext.NamespaceUserName != "" {
			namespaceName = searchContext.NamespaceUserName
		} else {
			namespaceName = searchContext.NamespaceOrgName
		}
		return searchContextSpecPrefix + namespaceName + "/" + searchContext.Name
	}
}

func getVersionContextRepositoryRevisions(ctx context.Context, db dbutil.DB, versionContext *schema.VersionContext) ([]*types.SearchContextRepositoryRevisions, error) {
	repositoriesToRevisions := map[string][]string{}
	for _, revision := range versionContext.Revisions {
		repositoriesToRevisions[revision.Repo] = append(repositoriesToRevisions[revision.Repo], revision.Rev)
	}

	repositories := make([]string, 0, len(repositoriesToRevisions))
	for repo := range repositoriesToRevisions {
		repositories = append(repositories, repo)
	}

	repositoryNames, err := database.Repos(db).ListRepoNames(ctx, database.ReposListOptions{Names: repositories})
	if err != nil {
		return nil, err
	}

	repositoryRevisions := make([]*types.SearchContextRepositoryRevisions, len(repositoryNames))
	for idx, repositoryName := range repositoryNames {
		revisions := repositoriesToRevisions[string(repositoryName.Name)]
		repositoryRevisions[idx] = &types.SearchContextRepositoryRevisions{Repo: repositoryName, Revisions: revisions}
	}

	return repositoryRevisions, nil
}

func getSearchContextFromVersionContext(versionContext *schema.VersionContext) *types.SearchContext {
	searchContextName := regexp.MustCompile(`\s+`).ReplaceAllString(versionContext.Name, "_")
	return &types.SearchContext{Name: searchContextName, Description: versionContext.Description, Public: true}
}

func ConvertVersionContextToSearchContext(ctx context.Context, db dbutil.DB, versionContext *schema.VersionContext) (*types.SearchContext, error) {
	repositoryRevisions, err := getVersionContextRepositoryRevisions(ctx, db, versionContext)
	if err != nil {
		return nil, err
	}

	searchContext, err := CreateSearchContextWithRepositoryRevisions(
		ctx,
		db,
		getSearchContextFromVersionContext(versionContext),
		repositoryRevisions,
	)
	if err != nil {
		return nil, err
	}

	return searchContext, nil
}
