package graphqlbackend

import (
	"encoding/json"
	"strconv"

	"github.com/cockroachdb/errors"
)

// BigInt implements the BigInt GraphQL scalar type.
type BigInt struct{ Int int64 }

// BigIntOrNil is a helper function that returns nil for int == nil and otherwise wraps int in
// BigInt.
func BigIntOrNil(int *int64) *BigInt {
	if int == nil {
		return nil
	}
	return &BigInt{Int: *int}
}

func (BigInt) ImplementsGraphQLType(name string) bool {
	return name == "BigInt"
}

func (v BigInt) MarshalJSON() ([]byte, error) {
	return json.Marshal(strconv.FormatInt(v.Int, 10))
}

func (v *BigInt) UnmarshalGraphQL(input interface{}) error {
	s, ok := input.(string)
	if !ok {
		return errors.Errorf("invalid GraphQL BigInt scalar value input (got %T, expected string)", input)
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	*v = BigInt{Int: n}
	return nil
}
