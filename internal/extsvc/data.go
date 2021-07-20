package extsvc

import (
	"encoding/json"

	"github.com/cockroachdb/errors"
)

func setJSONOrError(field **json.RawMessage, value interface{}) {
	if value == nil {
		*field = nil
		return
	}

	b, err := json.Marshal(value)
	if err != nil {
		b, _ = json.Marshal(struct {
			Error string `json:"__jsonError"`
		}{Error: err.Error()})
	}
	*field = (*json.RawMessage)(&b)
}

// SetAccountData sets the Data field to the (JSON-encoded) value. If an error occurs during
// JSON encoding, a JSON object describing the error is written to the field, instead.
func (d *AccountData) SetAccountData(v interface{}) {
	setJSONOrError(&d.Data, v)
}

// SetAuthData sets the AuthData field to the (JSON-encoded) value. If an error occurs during JSON
// encoding, a JSON object describing the error is written to the field, instead.
func (d *AccountData) SetAuthData(v interface{}) {
	setJSONOrError(&d.AuthData, v)
}

// GetAccountData reads the Data field into the value. The value should be a pointer type to
// the type that was passed to SetAccountData.
func (d *AccountData) GetAccountData(v interface{}) error {
	return getJSONOrError(d.Data, v)
}

// GetAuthData reads the AuthData field into the value. The value should be a pointer type to the
// type that was passed to SetAuthData.
func (d *AccountData) GetAuthData(v interface{}) error {
	return getJSONOrError(d.AuthData, v)
}

func getJSONOrError(field *json.RawMessage, v interface{}) error {
	if field == nil {
		return errors.New("field was nil")
	}

	if err := json.Unmarshal(*field, v); err != nil {
		var jsonErr jsonError
		if err := json.Unmarshal(*field, &jsonErr); err != nil {
			return errors.Errorf("could not parse field as JSON: %s", err)
		}
		return errors.New(jsonErr.Error)
	}
	return nil
}

type jsonError struct {
	Error string `json:"__jsonError"`
}
