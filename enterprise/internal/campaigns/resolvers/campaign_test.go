package resolvers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	ee "github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns/resolvers/apitest"
	ct "github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns/testing"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
)

func TestCampaignResolver(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := backend.WithAuthzBypass(context.Background())
	dbtesting.SetupGlobalTestDB(t)

	username := "campaign-resolver-username"
	userID := insertTestUser(t, dbconn.Global, username, true)
	org, err := db.Orgs.Create(ctx, "test-org", nil)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	clock := func() time.Time {
		return now.UTC().Truncate(time.Microsecond)
	}
	store := ee.NewStoreWithClock(dbconn.Global, clock)

	campaignSpec := &campaigns.CampaignSpec{
		RawSpec:        ct.TestRawCampaignSpec,
		UserID:         userID,
		NamespaceOrgID: org.ID,
	}
	if err := store.CreateCampaignSpec(ctx, campaignSpec); err != nil {
		t.Fatal(err)
	}

	campaign := &campaigns.Campaign{
		Name:             "my-unique-name",
		Description:      "The campaign description",
		NamespaceOrgID:   org.ID,
		InitialApplierID: userID,
		LastApplierID:    userID,
		LastAppliedAt:    now,
		CampaignSpecID:   campaignSpec.ID,
	}
	if err := store.CreateCampaign(ctx, campaign); err != nil {
		t.Fatal(err)
	}

	s, err := graphqlbackend.NewSchema(&Resolver{store: store}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	campaignAPIID := string(marshalCampaignID(campaign.ID))
	namespaceAPIID := string(graphqlbackend.MarshalOrgID(org.ID))
	apiUser := &apitest.User{DatabaseID: userID, SiteAdmin: true}
	wantCampaign := apitest.Campaign{
		ID:             campaignAPIID,
		Name:           campaign.Name,
		Description:    campaign.Description,
		Namespace:      apitest.UserOrg{ID: namespaceAPIID, Name: org.Name},
		InitialApplier: apiUser,
		LastApplier:    apiUser,
		SpecCreator:    apiUser,
		LastAppliedAt:  marshalDateTime(t, now),
		URL:            fmt.Sprintf("/organizations/%s/campaigns/%s", org.Name, campaign.Name),
		CreatedAt:      marshalDateTime(t, now),
		UpdatedAt:      marshalDateTime(t, now),
		// Not closed.
		ClosedAt: "",
	}

	input := map[string]interface{}{"campaign": campaignAPIID}
	{
		var response struct{ Node apitest.Campaign }
		apitest.MustExec(ctx, t, s, input, &response, queryCampaign)

		if diff := cmp.Diff(wantCampaign, response.Node); diff != "" {
			t.Fatalf("wrong campaign response (-want +got):\n%s", diff)
		}
	}
	// Test resolver by namespace and name
	byNameInput := map[string]interface{}{"name": campaign.Name, "namespace": namespaceAPIID}
	{
		var response struct{ Campaign apitest.Campaign }
		apitest.MustExec(ctx, t, s, byNameInput, &response, queryCampaignByName)

		if diff := cmp.Diff(wantCampaign, response.Campaign); diff != "" {
			t.Fatalf("wrong campaign response (-want +got):\n%s", diff)
		}
	}

	// Now soft-delete the user and check we can still access the campaign in the org namespace.
	err = db.Users.Delete(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}

	wantCampaign.InitialApplier = nil
	wantCampaign.LastApplier = nil
	wantCampaign.SpecCreator = nil

	{
		var response struct{ Node apitest.Campaign }
		apitest.MustExec(ctx, t, s, input, &response, queryCampaign)

		if diff := cmp.Diff(wantCampaign, response.Node); diff != "" {
			t.Fatalf("wrong campaign response (-want +got):\n%s", diff)
		}
	}

	// Now hard-delete the user and check we can still access the campaign in the org namespace.
	err = db.Users.HardDelete(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	{
		var response struct{ Node apitest.Campaign }
		apitest.MustExec(ctx, t, s, input, &response, queryCampaign)

		if diff := cmp.Diff(wantCampaign, response.Node); diff != "" {
			t.Fatalf("wrong campaign response (-want +got):\n%s", diff)
		}
	}
}

const queryCampaign = `
fragment u on User { databaseID, siteAdmin }
fragment o on Org  { id, name }

query($campaign: ID!){
  node(id: $campaign) {
    ... on Campaign {
      id, name, description
      initialApplier { ...u }
      lastApplier    { ...u }
      specCreator    { ...u }
      lastAppliedAt
      createdAt
      updatedAt
      closedAt
      namespace {
        ... on User { ...u }
        ... on Org  { ...o }
      }
      url
    }
  }
}
`
const queryCampaignByName = `
fragment u on User { databaseID, siteAdmin }
fragment o on Org  { id, name }

query($namespace: ID!, $name: String!){
  campaign(namespace: $namespace, name: $name) {
    id, name, description
    initialApplier { ...u }
    lastApplier    { ...u }
    specCreator    { ...u }
    lastAppliedAt
    createdAt
    updatedAt
    closedAt
    namespace {
      ... on User { ...u }
      ... on Org  { ...o }
    }
    url
  }
}
`
