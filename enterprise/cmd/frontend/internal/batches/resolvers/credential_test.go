package resolvers

import (
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
)

func TestUnmarshalBatchChangesCredentialID(t *testing.T) {
	siteCred := marshalBatchChangesCredentialID(123, true)
	userCred := marshalBatchChangesCredentialID(123, false)
	tcs := []struct {
		id               graphql.ID
		credentialID     int64
		isSiteCredential bool
		wantErr          bool
	}{
		{
			id:               siteCred,
			credentialID:     123,
			isSiteCredential: true,
		},
		{
			id:               userCred,
			credentialID:     123,
			isSiteCredential: false,
		},
		// Encoded value is not a string.
		{
			id:      relay.MarshalID("BatchChangesCredential", 123),
			wantErr: true,
		},
		// Encoded value does not conform to `<scope>:<int_id>` pattern.
		{
			id:      relay.MarshalID("BatchChangesCredential", "site123"),
			wantErr: true,
		},
		// Encoded value does not contain valid scope.
		{
			id:      relay.MarshalID("BatchChangesCredential", "invalidkind:1"),
			wantErr: true,
		},
		// Encoded value does not contain valid int id.
		{
			id:      relay.MarshalID("BatchChangesCredential", "user:invalidid"),
			wantErr: true,
		},
	}
	for _, tc := range tcs {
		haveCredentialID, haveIsSiteCredential, haveErr := unmarshalBatchChangesCredentialID(tc.id)
		if haveCredentialID != tc.credentialID {
			t.Errorf("invalid credential ID returned for %q: want=%d have=%d", tc.id, tc.credentialID, haveCredentialID)
		}
		if haveIsSiteCredential != tc.isSiteCredential {
			t.Errorf("invalid isSiteCredential returned for %q: want=%t have=%t", tc.id, tc.isSiteCredential, haveIsSiteCredential)
		}
		if (haveErr != nil) != tc.wantErr {
			t.Errorf("invalid error %+v", haveErr)
		}
	}
}
