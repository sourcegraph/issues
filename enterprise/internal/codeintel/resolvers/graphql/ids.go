package graphql

import (
	"strconv"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
)

func marshalLSIFUploadGQLID(uploadID int64) graphql.ID {
	return relay.MarshalID("LSIFUpload", uploadID)
}

func unmarshalLSIFUploadGQLID(id graphql.ID) (uploadID int64, err error) {
	// First, try to unmarshal the ID as a string and then convert it to an
	// integer. This is here to maintain backwards compatibility with the
	// src-cli lsif upload command, which constructs its own relay identifier
	// from a the string payload returned by the upload proxy.

	var idString string
	err = relay.UnmarshalSpec(id, &idString)
	if err == nil {
		uploadID, err = strconv.ParseInt(idString, 10, 64)
		return
	}

	// If it wasn't unmarshal-able as a string, it's a new-style int identifier
	err = relay.UnmarshalSpec(id, &uploadID)
	return uploadID, err
}

//
//

func marshalLSIFIndexGQLID(indexID int64) graphql.ID {
	return relay.MarshalID("LSIFIndex", indexID)
}

func unmarshalLSIFIndexGQLID(id graphql.ID) (indexID int64, err error) {
	err = relay.UnmarshalSpec(id, &indexID)
	return indexID, err
}
