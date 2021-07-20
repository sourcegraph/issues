package batches

import (
	"encoding/json"

	"github.com/cockroachdb/errors"
)

// PublishedValue is a wrapper type that supports the quadruple `true`, `false`,
// `"draft"`, `nil`.
type PublishedValue struct {
	Val interface{}
}

// True is true if the enclosed value is a bool being true.
func (p *PublishedValue) True() bool {
	if b, ok := p.Val.(bool); ok {
		return b
	}
	return false
}

// False is true if the enclosed value is a bool being false.
func (p PublishedValue) False() bool {
	if b, ok := p.Val.(bool); ok {
		return !b
	}
	return false
}

// Draft is true if the enclosed value is a string being "draft".
func (p PublishedValue) Draft() bool {
	if s, ok := p.Val.(string); ok {
		return s == "draft"
	}
	return false
}

// Nil is true if the enclosed value is a null or omitted.
func (p PublishedValue) Nil() bool {
	return p.Val == nil
}

// Valid returns whether the enclosed value is of any of the permitted types.
func (p *PublishedValue) Valid() bool {
	return p.True() || p.False() || p.Draft() || p.Nil()
}

// Value returns the underlying value stored in this wrapper.
func (p *PublishedValue) Value() interface{} {
	return p.Val
}

func (p PublishedValue) MarshalJSON() ([]byte, error) {
	if p.Nil() {
		v := "null"
		return []byte(v), nil
	}
	if p.True() {
		v := "true"
		return []byte(v), nil
	}
	if p.False() {
		v := "false"
		return []byte(v), nil
	}
	if p.Draft() {
		v := `"draft"`
		return []byte(v), nil
	}
	return nil, errors.Errorf("invalid PublishedValue: %s (%T)", p.Val, p.Val)
}

func (p *PublishedValue) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &p.Val)
}

// UnmarshalYAML unmarshalls a YAML value into a Publish.
func (p *PublishedValue) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := unmarshal(&p.Val); err != nil {
		return err
	}

	return nil
}

func (p *PublishedValue) UnmarshalGraphQL(input interface{}) error {
	p.Val = input
	if !p.Valid() {
		return errors.Errorf("invalid PublishedValue: %v", input)
	}
	return nil
}

// ImplementsGraphQLType lets GraphQL-go tell apart the corresponding GraphQL scalar.
func (p *PublishedValue) ImplementsGraphQLType(name string) bool {
	return name == "PublishedValue"
}
