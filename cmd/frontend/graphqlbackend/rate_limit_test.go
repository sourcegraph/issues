package graphqlbackend

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestEstimateQueryCost(t *testing.T) {
	for _, tc := range []struct {
		name      string
		query     string
		variables map[string]interface{}
		want      queryCost
	}{
		{
			name: "Multiple top level queries",
			query: `query {
  thing
}
query{
  thing
}
`,
			want: queryCost{
				FieldCount: 2,
				MaxDepth:   1,
			},
		},
		{
			name: "Simple query, no variables",
			query: `
query SiteProductVersion {
                site {
                    productVersion
                    buildVersion
                    hasCodeIntelligence
                }
            }
`,
			want: queryCost{
				FieldCount: 4,
				MaxDepth:   2,
			},
		},
		{
			name: "nodes field should not be counted",
			query: `
query{
  externalServices(first: 10){
    nodes{
      displayName
      webhookURL
    }
  }
}
`,
			want: queryCost{
				FieldCount: 21,
				MaxDepth:   3,
			},
		},
		{
			name: "Query with variables",
			query: `
query Extensions($first: Int!, $prioritizeExtensionIDs: [String!]!) {
                    extensionRegistry {
                        extensions(first: $first, prioritizeExtensionIDs: $prioritizeExtensionIDs) {
                            nodes {
                                id
                                extensionID
                                url
                                manifest {
                                    raw
                                }
                                viewerCanAdminister
                            }
                        }
                    }
                }
`,
			variables: map[string]interface{}{
				"first": 10,
			},
			want: queryCost{
				FieldCount: 62,
				MaxDepth:   5,
			},
		},
		{
			name: "Query with default variables",
			query: `
query fetchExternalServices($first: Int = 10){
  externalServices(first: $first){
    nodes{
      displayName
      webhookURL
    }
  }
}
`,
			variables: map[string]interface{}{
				"first": 5,
			},
			want: queryCost{
				FieldCount: 11,
				MaxDepth:   3,
			},
		},
		{
			name: "Query with fragments",
			query: `
query StatusMessages {
	 statusMessages {
		 ...StatusMessageFields
	 }
 }
 fragment StatusMessageFields on StatusMessage {
	 __typename
	 ... on CloningProgress {
		 message
	 }
	 ... on SyncError {
		 message
	 }
	 ... on ExternalServiceSyncError {
		 message
		 externalService {
			 id
			 displayName
		 }
	 }
 }
`,
			want: queryCost{
				FieldCount: 5,
				MaxDepth:   2,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			have, err := estimateQueryCost(tc.query, tc.variables)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.want, *have); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
