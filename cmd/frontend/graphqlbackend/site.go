package graphqlbackend

import (
	"context"
	"os"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/siteid"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/internal/version"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/globals"
)

const singletonSiteGQLID = "site"

func (r *schemaResolver) siteByGQLID(ctx context.Context, id graphql.ID) (Node, error) {
	siteGQLID, err := unmarshalSiteGQLID(id)
	if err != nil {
		return nil, err
	}
	if siteGQLID != singletonSiteGQLID {
		return nil, errors.Errorf("site not found: %q", siteGQLID)
	}
	return &siteResolver{db: r.db, gqlID: siteGQLID}, nil
}

func marshalSiteGQLID(siteID string) graphql.ID { return relay.MarshalID("Site", siteID) }

// SiteGQLID is the GraphQL ID of the Sourcegraph site. It is a constant across all Sourcegraph
// instances.
func SiteGQLID() graphql.ID { return (&siteResolver{gqlID: singletonSiteGQLID}).ID() }

func unmarshalSiteGQLID(id graphql.ID) (siteID string, err error) {
	err = relay.UnmarshalSpec(id, &siteID)
	return
}

func (r *schemaResolver) Site() *siteResolver {
	return &siteResolver{db: r.db, gqlID: singletonSiteGQLID}
}

type siteResolver struct {
	db    dbutil.DB
	gqlID string // == singletonSiteGQLID, not the site ID
}

func (r *siteResolver) ID() graphql.ID { return marshalSiteGQLID(r.gqlID) }

func (r *siteResolver) SiteID() string { return siteid.Get() }

func (r *siteResolver) Configuration(ctx context.Context) (*siteConfigurationResolver, error) {
	// 🚨 SECURITY: The site configuration contains secret tokens and credentials,
	// so only admins may view it.
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx, r.db); err != nil {
		return nil, err
	}
	return &siteConfigurationResolver{db: r.db}, nil
}

func (r *siteResolver) ViewerCanAdminister(ctx context.Context) (bool, error) {
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx, r.db); err == backend.ErrMustBeSiteAdmin || err == backend.ErrNotAuthenticated {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func (r *siteResolver) settingsSubject() api.SettingsSubject {
	return api.SettingsSubject{Site: true}
}

func (r *siteResolver) LatestSettings(ctx context.Context) (*settingsResolver, error) {
	settings, err := database.Settings(r.db).GetLatest(ctx, r.settingsSubject())
	if err != nil {
		return nil, err
	}
	if settings == nil {
		return nil, nil
	}
	return &settingsResolver{r.db, &settingsSubject{site: r}, settings, nil}, nil
}

func (r *siteResolver) SettingsCascade() *settingsCascade {
	return &settingsCascade{db: r.db, subject: &settingsSubject{site: r}}
}

func (r *siteResolver) ConfigurationCascade() *settingsCascade { return r.SettingsCascade() }

func (r *siteResolver) SettingsURL() *string { return strptr("/site-admin/global-settings") }

func (r *siteResolver) CanReloadSite(ctx context.Context) bool {
	err := backend.CheckCurrentUserIsSiteAdmin(ctx, r.db)
	return canReloadSite && err == nil
}

func (r *siteResolver) BuildVersion() string { return version.Version() }

func (r *siteResolver) ProductVersion() string { return version.Version() }

func (r *siteResolver) HasCodeIntelligence() bool {
	// BACKCOMPAT: Always return true.
	return true
}

func (r *siteResolver) ProductSubscription() *productSubscriptionStatus {
	return &productSubscriptionStatus{}
}

func (r *siteResolver) AllowSiteSettingsEdits() bool {
	return canUpdateSiteConfiguration()
}

type siteConfigurationResolver struct {
	db dbutil.DB
}

func (r *siteConfigurationResolver) ID(ctx context.Context) (int32, error) {
	// 🚨 SECURITY: The site configuration contains secret tokens and credentials,
	// so only admins may view it.
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx, r.db); err != nil {
		return 0, err
	}
	return 0, nil // TODO(slimsag): future: return the real ID here to prevent races
}

func (r *siteConfigurationResolver) EffectiveContents(ctx context.Context) (JSONCString, error) {
	// 🚨 SECURITY: The site configuration contains secret tokens and credentials,
	// so only admins may view it.
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx, r.db); err != nil {
		return "", err
	}
	siteConfig := globals.ConfigurationServerFrontendOnly.Raw().Site
	return JSONCString(siteConfig), nil
}

func (r *siteConfigurationResolver) ValidationMessages(ctx context.Context) ([]string, error) {
	contents, err := r.EffectiveContents(ctx)
	if err != nil {
		return nil, err
	}
	return conf.ValidateSite(string(contents))
}

func (r *schemaResolver) UpdateSiteConfiguration(ctx context.Context, args *struct {
	LastID int32
	Input  string
}) (bool, error) {
	// 🚨 SECURITY: The site configuration contains secret tokens and credentials,
	// so only admins may view it.
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx, r.db); err != nil {
		return false, err
	}
	if !canUpdateSiteConfiguration() {
		return false, errors.New("updating site configuration not allowed when using SITE_CONFIG_FILE")
	}
	if strings.TrimSpace(args.Input) == "" {
		return false, errors.Errorf("blank site configuration is invalid (you can clear the site configuration by entering an empty JSON object: {})")
	}

	if problems, err := conf.ValidateSite(args.Input); err != nil {
		return false, errors.Errorf("failed to validate site configuration: %w", err)
	} else if len(problems) > 0 {
		return false, errors.Errorf("site configuration is invalid: %s", strings.Join(problems, ","))
	}

	prev := globals.ConfigurationServerFrontendOnly.Raw()
	prev.Site = args.Input
	// TODO(slimsag): future: actually pass lastID through to prevent race conditions
	if err := globals.ConfigurationServerFrontendOnly.Write(ctx, prev); err != nil {
		return false, err
	}
	return globals.ConfigurationServerFrontendOnly.NeedServerRestart(), nil
}

var siteConfigAllowEdits, _ = strconv.ParseBool(env.Get("SITE_CONFIG_ALLOW_EDITS", "false", "When SITE_CONFIG_FILE is in use, allow edits in the application to be made which will be overwritten on next process restart"))

func canUpdateSiteConfiguration() bool {
	return os.Getenv("SITE_CONFIG_FILE") == "" || siteConfigAllowEdits
}
