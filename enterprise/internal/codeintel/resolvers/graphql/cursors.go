package graphql

import (
	"encoding/base64"
	"strconv"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/graphqlutil"
)

// encodeCursor creates a PageInfo object from the given cursor. If the cursor is not
// defined, then an object indicating the end of the result set is returned. The cursor
// is base64 encoded for transfer, and should be decoded using the function decodeCursor.
func encodeCursor(val *string) *graphqlutil.PageInfo {
	if val != nil {
		return graphqlutil.NextPageCursor(base64.StdEncoding.EncodeToString([]byte(*val)))
	}

	return graphqlutil.HasNextPage(false)
}

// decodeCursor decodes the given cursor value. It is assumed to be a value previously
// returned from the function encodeCursor. An empty string is returned if no cursor is
// supplied. Invalid cursors return errors.
func decodeCursor(val *string) (string, error) {
	if val == nil {
		return "", nil
	}

	decoded, err := base64.StdEncoding.DecodeString(*val)
	if err != nil {
		return "", err
	}

	return string(decoded), nil
}

// encodeIntCursor creates a PageInfo object from the given new offset value. If the
// new offset value, then an object indicating the end of the result set is returned.
// The cursor is base64 encoded for transfer, and should be decoded using the function
// decodeIntCursor.
func encodeIntCursor(val *int32) *graphqlutil.PageInfo {
	if val == nil {
		return encodeCursor(nil)
	}

	str := strconv.FormatInt(int64(*val), 10)
	return encodeCursor(&str)
}

// decodeIntCursor decodes the given integer cursor value. It is assumed to be a value
// previously returned from the function encodeIntCursor. The zero value is returned if
// no cursor is supplied. Invalid cursors return errors.
func decodeIntCursor(val *string) (int, error) {
	cursor, err := decodeCursor(val)
	if err != nil || cursor == "" {
		return 0, err
	}

	return strconv.Atoi(cursor)
}
