package secret

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"

	"github.com/pkg/errors"
)

var (
	_ driver.Valuer    = StringValue{}
	_ sql.Scanner      = (*StringValue)(nil)
	_ json.Marshaler   = StringValue{}
	_ json.Unmarshaler = (*StringValue)(nil)
)

// StringValue is a string type secret that implements driver.Valuer, sql.Scanner,
// json.Marshaler and json.Unmarshaler for transparent encryption and decryption
// during database access and JSON serialization.
type StringValue struct {
	S *string
}

func (v StringValue) Value() (driver.Value, error) {
	if v.S == nil {
		return nil, errors.New("unable to encrypt a nil string pointer")
	}
	return Encrypt(*v.S)
}

func (v *StringValue) Scan(src interface{}) (err error) {
	if v.S == nil {
		return errors.New("unable to decrypt to a nil string pointer")
	}

	var plaintext string
	switch src := src.(type) {
	case string: // expect a base64 encoded string
		plaintext, err = Decrypt(src)
		if err != nil {
			return err
		}

	default:
		return errors.Errorf("incompatible type %T for StringValue", src)
	}

	*v.S = plaintext
	return nil
}

func (v StringValue) MarshalJSON() ([]byte, error) {
	if v.S == nil {
		return nil, errors.New("unable to encrypt a nil string pointer")
	}

	ciphertext, err := Encrypt(*v.S)
	if err != nil {
		return nil, errors.Wrap(err, "encrypt")
	}
	return json.Marshal(ciphertext)
}

func (v *StringValue) UnmarshalJSON(src []byte) error {
	if v.S == nil {
		// It happens very often that fields are having zero values when unmarshal a JSON blob.
		var s string
		v.S = &s
	}

	var ciphertext string
	err := json.Unmarshal(src, &ciphertext)
	if err != nil {
		return errors.Wrap(err, "unmarshal")
	}

	plaintext, err := Decrypt(ciphertext)
	if err != nil {
		return errors.Wrap(err, "decrypt")
	}

	*v.S = plaintext
	return nil
}

var (
	_ driver.Valuer = NullStringValue{}
	_ sql.Scanner   = (*NullStringValue)(nil)
)

// NullStringValue is a NULLABLE string type secret that implements driver.Valuer and sql.Scanner
// for transparent encryption and decryption during database access.
type NullStringValue struct {
	S     *string
	Valid bool // Valid is true if String is not NULL
}

func (nv *NullStringValue) Scan(value interface{}) error {
	switch v := value.(type) {
	case string:
		es := StringValue{S: nv.S}
		err := es.Scan(v)
		if err != nil {
			return err
		}
		nv.Valid = true
	}
	return nil
}

func (nv NullStringValue) Value() (driver.Value, error) {
	if nv.S == nil {
		return nil, nil
	}

	return StringValue{S: nv.S}.Value()
}
