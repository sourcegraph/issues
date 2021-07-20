package jsonc

import (
	"encoding/json"
	"strings"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/jsonx"
)

// Unmarshal unmarshals the JSON using a fault-tolerant parser that allows comments and trailing
// commas. If any unrecoverable faults are found, an error is returned.
func Unmarshal(text string, v interface{}) error {
	data, err := Parse(text)
	if err != nil {
		return err
	}
	if strings.TrimSpace(text) == "" {
		return nil
	}
	return json.Unmarshal(data, v)
}

// Parse converts JSON with comments, trailing commas, and some types of syntax errors into standard
// JSON. If there is an error that it can't unambiguously resolve, it returns the error.
func Parse(text string) ([]byte, error) {
	data, errs := jsonx.Parse(text, jsonx.ParseOptions{Comments: true, TrailingCommas: true})
	if len(errs) > 0 {
		return data, errors.Errorf("failed to parse JSON: %v", errs)
	}
	return data, nil
}

// Normalize is like Parse, except it ignores errors and always returns valid JSON, even if that
// JSON is a subset of the input.
func Normalize(input string) []byte {
	output, _ := jsonx.Parse(input, jsonx.ParseOptions{Comments: true, TrailingCommas: true})
	if len(output) == 0 {
		return []byte("{}")
	}
	return output
}

var DefaultFormatOptions = jsonx.FormatOptions{InsertSpaces: true, TabSize: 2}

// Remove returns the input JSON with the given path removed.
func Remove(input string, path ...string) (string, error) {
	edits, _, err := jsonx.ComputePropertyRemoval(input,
		jsonx.PropertyPath(path...),
		DefaultFormatOptions,
	)
	if err != nil {
		return input, err
	}

	return jsonx.ApplyEdits(input, edits...)
}

// Edit returns the input JSON with the given path set to v.
func Edit(input string, v interface{}, path ...string) (string, error) {
	edits, _, err := jsonx.ComputePropertyEdit(input,
		jsonx.PropertyPath(path...),
		v,
		nil,
		DefaultFormatOptions,
	)
	if err != nil {
		return input, err
	}

	return jsonx.ApplyEdits(input, edits...)
}

// ReadProperty attempts to read the value of the specified path, ignoring parse errors. it will only error if the path
// doesn't exist
func ReadProperty(input, path string) (interface{}, error) {
	root, _ := jsonx.ParseTree(input, jsonx.ParseOptions{Comments: true, TrailingCommas: true})
	node := jsonx.FindNodeAtLocation(root, jsonx.PropertyPath(path))
	if node == nil {
		return nil, errors.Errorf("couldn't find node: %s", path)
	}
	return node.Value, nil
}

// Format returns the input JSON formatted with the given options.
func Format(input string, opt *jsonx.FormatOptions) (string, error) {
	if opt == nil {
		opt = &DefaultFormatOptions
	}
	return jsonx.ApplyEdits(input, jsonx.Format(input, *opt)...)
}
